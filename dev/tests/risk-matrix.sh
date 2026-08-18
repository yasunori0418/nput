#!/usr/bin/env bash
# risk の level 導出マトリクス整合の契約テスト（→ Issue #303）。
#
# 実行:
#   nix develop '.?dir=dev#sara' -c dev/tests/risk-matrix.sh   # devShell / CI から直接
#   nix flake check ./dev                                       # checks.risk-matrix 経由
#
# 検証対象:
#   docs/risks/*.md 全件で、frontmatter の level が likelihood × impact の
#   マトリクス導出（dev/tests/risk-matrix.tsv）と一致する。
#
# マトリクスの正本は dev/tests/risk-matrix.tsv（機械可読）。docs/agents/sara-graph.md の
# level 表は読み物としての要約で、規範は TSV 側にある（test-categories.tsv と CLAUDE.md の
# 8 区分表と同じ構図）。当初は 9 セルをスクリプトへ直書きし、sara-graph.md の Markdown
# テーブルを awk で解析して突き合わせていたが、その処理がスクリプトの複雑さの大半を
# 占めたため正本を TSV へ移して廃止した（→ PR #352）。
#
# フィールドの存在・enum 妥当性（high / medium / low のいずれか）は docs/model.yaml が
# 宣言し `sara check` が検証する担当なので、ここでは重複させない。ここが見るのは
# 3 フィールドの「関係」だけで、値が enum 外・欠損のときはマトリクスを引けないため
# その旨を FAIL として報告する（sara check の代替をするのではなく、判定不能を黙って
# 素通りさせないための扱い）。
#
# frontmatter の読み取りは yq（mikefarah/yq v4）で行う。docs/ の frontmatter を読む
# 既存スクリプト（test-doc-map.sh・test-doc-matrix.sh）と同じ手段で、devShells.sara にも
# 既に載っているので依存は増えない。手書きのパーサだと引用符・空白・複数行スカラといった
# YAML の表記ゆれを自前で潰すことになり、その取りこぼしが「マトリクスのセルが無い」と
# いう誤った診断で表面化する。
#
# ## なぜ機械化するか
#
# level 導出は `docs/agents/sara-graph.md` の規約のうち唯一「散文の解釈を伴わない
# 純粋な写像」で、スクリプトが判定しきれる。目視レビューはこの検査の最も弱い形。

# -e は使わない（test-doc-map.sh と同じ理由）。このテストは「1 回の実行で
# 全失敗を報告する」集計方式で、-e があると最初の非ゼロ終了で以降のアサーションが
# 走らず、退行時の診断が先頭 1 件で切れる。
set -uo pipefail

fail=0
pass() { printf 'ok   - %s\n' "$1"; }
fault() {
  printf 'FAIL - %s\n' "$1"
  fail=1
}

# shellcheck source=dev/scripts/lib-testdoc.sh
. "$(dirname "$0")/../scripts/lib-testdoc.sh"

require_commands "risk-matrix.sh（nix develop '.?dir=dev#sara' から実行する）" yq || exit 1
require_yq_go risk-matrix.sh || exit 1

# 走査基点をリポジトリルートへ解決する。git 管理外ではカレント基準へフォールバックする
# （checks 派生のサンドボックスは作業ツリーを持たないためこの経路を通る）。
repo_root=$(git rev-parse --show-toplevel 2>/dev/null || printf '.')

# 走査対象 docs/risks の在り処は 2 経路ある:
#   1. RISK_DOCS_DIR（checks.risk-matrix のサンドボックス。作業ツリーが無いので
#      nix が store path を渡す）
#   2. git のリポジトリルート基準（`nix develop` からの直接実行・CI の sara job）
risks_dir="${RISK_DOCS_DIR:-}"
[[ -d "$risks_dir" ]] || risks_dir="$repo_root/docs/risks"

if [[ ! -d "$risks_dir" ]]; then
  echo "risk-matrix.sh: risk item のディレクトリを解決できない（$risks_dir）" >&2
  exit 1
fi

# --- level 導出マトリクスを正本の TSV から読む -------------------------------
#
# TSV はこのスクリプトと同じディレクトリに置く（checks 派生も dev/ の木ごと配置するので
# スクリプト基準で解決できる。docs/risks のように環境変数の受け口は要らない）。
matrix_tsv="$(dirname "$0")/risk-matrix.tsv"

if [[ ! -f "$matrix_tsv" ]]; then
  echo "risk-matrix.sh: マトリクスの正本が無い（$matrix_tsv）" >&2
  exit 1
fi

declare -A LEVEL_MATRIX=()
tsv_rows=0
tsv_ok=1
while IFS=$'\t' read -r likelihood impact level; do
  tsv_rows=$((tsv_rows + 1))
  if [[ -z "$likelihood" || -z "$impact" || -z "$level" ]]; then
    fault "risk-matrix.tsv の $tsv_rows 行目に空の列がある（likelihood='$likelihood' impact='$impact' level='$level'）"
    tsv_ok=0
    continue
  fi
  key="$likelihood:$impact"
  if [[ -n "${LEVEL_MATRIX[$key]:-}" ]]; then
    fault "risk-matrix.tsv で $key が 2 回現れる（既出: ${LEVEL_MATRIX[$key]}・再出: $level）"
    tsv_ok=0
    continue
  fi
  LEVEL_MATRIX["$key"]="$level"
done < <(read_tsv "$matrix_tsv")

# 9 セルが揃っていることを確かめる。TSV が空・行が削れた状態で本体の走査を回すと、
# 該当セルを持つ risk が「セルが無い」で落ちるだけで、TSV 側の欠損だと分からないまま
# 調査先を誤る。3 値（high / medium / low）の直積なので 9 が期待値。
if [[ "${#LEVEL_MATRIX[@]}" -ne 9 ]]; then
  fault "risk-matrix.tsv のセルが 9 件ではない（実際: ${#LEVEL_MATRIX[@]} 件・データ行 $tsv_rows 行）"
  tsv_ok=0
fi

# マトリクスを引けない状態で走査を回しても全件 FAIL するだけで診断の役に立たない。
if [[ "$tsv_ok" -eq 0 ]]; then
  exit 1
fi

pass "risk-matrix.tsv から ${#LEVEL_MATRIX[@]} セルのマトリクスを読んだ"

# --- 全 risk item の level が導出と一致する -----------------------------------

# ファイル一覧は改行区切りで受ける（この corpus のファイル名は日付 + UUID8 + slug で
# 空白・改行を含まない → CLAUDE.md の ID 規約）。
mapfile -t risk_files < <(find "$risks_dir" -maxdepth 1 -type f -name '*.md' | LC_ALL=C sort)

# 対象 0 件を緑にしない。走査先の解決を誤ったとき（ディレクトリ移動・環境変数の
# 誤配線）、件数を見ないと「全件合格」と報告してしまう。
if [[ "${#risk_files[@]}" -eq 0 ]]; then
  fault "docs/risks に risk item が 1 件も無い（走査先: $risks_dir）"
  exit "$fail"
fi

# 1 ファイル 1 回の yq で 3 フィールドを採る（test-doc-matrix.sh と同じ形）。32 件規模なら
# ファイルごとに yq を起こすコストは問題にならない。
#
# yq の失敗（YAML が壊れている）と「値が空」は区別する。畳むと、frontmatter の構文エラーが
# 「フィールドが無い」と診断されて調査先を誤らせる（フィールドの欠損自体は model.yaml の
# required 宣言により sara check が落とす担当）。
checked=0
for f in "${risk_files[@]}"; do
  name="$(basename "$f")"

  if ! fields="$(yq --front-matter=extract -r \
    '[(.likelihood // ""), (.impact // ""), (.level // "")] | @tsv' "$f" 2>/dev/null)"; then
    fault "$name: frontmatter を yq で読めなかった（YAML の構文を確認する）"
    continue
  fi

  IFS=$'\t' read -r likelihood impact level <<<"$fields"

  if [[ -z "$likelihood" || -z "$impact" || -z "$level" ]]; then
    fault "$name: frontmatter に 3 フィールドが揃っていない（likelihood='$likelihood' impact='$impact' level='$level'）"
    continue
  fi

  want="${LEVEL_MATRIX[$likelihood:$impact]:-}"
  if [[ -z "$want" ]]; then
    # 生値を引用符で囲んで出す。enum 外の値であることを目視で確かめられるようにする
    # （引用符・空白の表記ゆれは yq が正規化するのでここには現れない）。
    fault "$name: likelihood='$likelihood' impact='$impact' に対応するマトリクスのセルが無い（enum 外の値）"
    continue
  fi

  if [[ "$level" != "$want" ]]; then
    fault "$name: level が導出と食い違う（likelihood=$likelihood impact=$impact → $want、実際: $level）"
    continue
  fi

  checked=$((checked + 1))
done

if [[ "$checked" -eq "${#risk_files[@]}" ]]; then
  pass "risk $checked 件の level が likelihood × impact の導出と一致する"
fi

exit "$fail"
