#!/usr/bin/env bash
# sara init を包む item 起票ラッパー（→ Issue #367・epic #364）。
#
#   sara-new <型> <slug> <配置ディレクトリ> [-- <sara init のオプション>...]
#
# sara init はファイルパスを先に渡す設計なので、素で使うと「仮の名前で init →
# 採番された ID を見てファイル名を組み立て直す」工程が毎回発生する。本スクリプトは
# それを 1 コマンドに畳み、ファイル名規約 `<YYYYMMDD>-<フル UUID>-<slug>.md`
# （→ ADR-0053）を機械的に守らせる。8 文字省略形をファイル名に混ぜる事故は、CI の
# 検査ではなくこの生成の機械化で構造的に防ぐ。
#
# ID は一切生成しない。UUID の採番・重複可否は sara init（model.yaml の id_format）
# の領分で、ここは `sara init` の出力から採番済み ID を読むだけ。型 → prefix の
# 対応表も持たない（model.yaml との二重管理を作らないため）。
#
# ADR は対象外。id_format が {prefix}-{seq:04} で連番を維持する唯一の型であり
# （→ ADR-0053）、`sara init adr <パス>` の直呼びに委ねる。
#
# 他プロジェクトへ横展開する前提でこのファイル 1 つに閉じている。依存は POSIX の
# 範囲 + sara 本体のみで、リポジトリ固有の知識（型ごとの配置先など）は持たない
# （配置ディレクトリを引数で受けるのはそのため。docs/test/<対象>/ のように型から
# 導けない配置があり、対応表を持てば必ず実体とずれる）。

set -euo pipefail

usage() {
  cat >&2 <<'EOF'
usage: sara-new <type> <slug> <dir> [-- <sara init options>...]

  type   sara の型名（requirement / test_case / test-case …）。ADR は対象外
  slug   ファイル名に使う短い識別子（英小文字・数字・ハイフン）
  dir    配置ディレクトリ（リポジトリルートからの相対。無ければ作る）
  --     以降を sara init へそのまま渡す（--name / --specification …）

例:
  sara-new requirement lock-ordering docs/requirements
  sara-new test-case engine-lock docs/test/atomicity -- --name "engine の lock"

出力（機械可読の 2 行）:
  id:   採番された正式 ID
  file: 起票したファイルのパス

注: ADR は連番を維持するため対象外（sara init adr <パス> を直接使う）
EOF
}

# help は引数個数に先立って処理する（`sara-new --help extra` も help になる）。
case "${1-}" in
  -h | --help)
    usage
    exit 0
    ;;
esac

if [ "$#" -lt 3 ]; then
  usage
  exit 2
fi

type=$1
slug=$2
dir=$3
shift 3

# `--` 以降は sara init への透過オプション。区切りが無ければ余分な引数は受け付けない
# （タイポした引数が黙って捨てられるより、使い方を出して落とす）。
if [ "$#" -gt 0 ]; then
  if [ "$1" != "--" ]; then
    usage
    exit 2
  fi
  shift
fi

# ADR は連番を維持する（→ ADR-0053）。ここで UUID 名のファイルを作ると
# docs/adr/README.md の運用・既存 ADR の相互参照が壊れる。
case "$type" in
  adr | ADR)
    echo "sara-new: ADR は連番を維持する（sara init adr <パス> を直接使う）" >&2
    exit 2
    ;;
esac

# slug はファイル名へそのまま入る。`../` やスペースを通すと配置先が黙ってずれるため、
# 起票の前に弾く（弾いた時点でファイルは 1 つも作られていない）。
case "$slug" in
  "" | *[!a-z0-9-]*)
    echo "sara-new: slug は英小文字・数字・ハイフンのみ（実際: '$slug'）" >&2
    exit 2
    ;;
esac

# sara init のサブコマンド名はハイフン（test-case）だが、model.yaml の型名は
# アンダースコア（test_case）で、規約文書も後者で書かれている。どちらで呼んでも
# 通るよう `_` を `-` へ寄せる。型の一覧は持たない（未知の名前は sara init が
# 「unrecognized subcommand」で弾く）ので、model.yaml との二重管理にはならない。
init_subcommand=$(printf '%s' "$type" | tr '_' '-')

mkdir -p "$dir"

# sara 呼び出しの seam（契約テストが失敗経路を決定論的に再現するため）。
sara_cmd=${SARA_NEW_SARA:-sara}

# 採番前の仮ファイル。sara init はパスを先に要求するので避けられない。プロセス ID を
# 混ぜて並列起動でも衝突させない。以降のどの失敗経路でも残さない（trap で回収する）。
tmp_file="$dir/.sara-new-$$.md"
cleanup() { rm -f "$tmp_file"; }
trap cleanup EXIT

# 装飾（色・絵文字）を落として ID 行を安定させる。sara は診断も stdout へ出すため
# stderr は素通しにして、失敗時の診断を握り潰さない。
init_out=$("$sara_cmd" --no-color --no-emoji init "$init_subcommand" "$tmp_file" "$@")

# `  ID:   <ID>` 行から採番結果を採る。読めなければ rename せず落とす
# （sara の出力形式が変わったとき、UUID 部が空のファイル名を黙って作らせない）。
id=$(printf '%s\n' "$init_out" | sed -n 's/^[[:space:]]*ID:[[:space:]]*\([^[:space:]]*\).*/\1/p' | head -n 1)

if [ -z "$id" ]; then
  echo "sara-new: sara init の出力から ID を読めなかった（sara の出力形式が変わった可能性）" >&2
  printf '%s\n' "$init_out" >&2
  exit 1
fi

# ファイル名は正式 ID の UUID 部（prefix を除いた全体）を使う。8 文字の省略形は
# 使わない（→ ADR-0053）。prefix は型ごとに違うので `<PREFIX>-` を前方から落とす。
uuid=${id#*-}

target="$dir/$(date +%Y%m%d)-$uuid-$slug.md"

# 起票済み item を潰さない。UUID が違えば通常は衝突しないが、衝突したときに
# 黙って上書きするより失敗させる。
if [ -e "$target" ]; then
  echo "sara-new: 出力先が既に存在する: $target" >&2
  exit 1
fi

mv "$tmp_file" "$target"

printf 'id: %s\n' "$id"
printf 'file: %s\n' "$target"
