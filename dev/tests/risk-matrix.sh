#!/usr/bin/env bash
# risk の level 導出マトリクス整合の契約テスト（→ Issue #303）。
#
# 実行:
#   nix develop '.?dir=dev#sara' -c dev/tests/risk-matrix.sh   # devShell / CI から直接
#   nix flake check ./dev                                       # checks.risk-matrix 経由
#
# 検証対象は 1 つだけ:
#   docs/risks/*.md 全件で、frontmatter の level が likelihood × impact の
#   マトリクス導出（下の LEVEL_MATRIX）と一致する。
#
# フィールドの存在・enum 妥当性（high / medium / low のいずれか）は docs/model.yaml が
# 宣言し `sara check` が検証する担当なので、ここでは重複させない。ここが見るのは
# 3 フィールドの「関係」だけで、値が enum 外・欠損のときはマトリクスを引けないため
# その旨を FAIL として報告する（sara check の代替をするのではなく、判定不能を黙って
# 素通りさせないための扱い）。
#
# ## なぜ機械化するか
#
# level 導出は `docs/agents/sara-graph.md` の規約のうち唯一「散文の解釈を伴わない
# 純粋な写像」で、スクリプトが判定しきれる。目視レビューはこの検査の最も弱い形。

# -e は使わない（sara-id.sh・test-doc-map.sh と同じ理由）。このテストは「1 回の実行で
# 全失敗を報告する」集計方式で、-e があると最初の非ゼロ終了で以降のアサーションが
# 走らず、退行時の診断が先頭 1 件で切れる。
set -uo pipefail

fail=0
pass() { printf 'ok   - %s\n' "$1"; }
fault() {
  printf 'FAIL - %s\n' "$1"
  fail=1
}

# level 導出マトリクス。正本は docs/agents/sara-graph.md「How a risk is scored」節の
# `level` 表で、ここはその 9 セルを機械可読に写したもの。表を変えるときは両方を同じ
# コミットで揃える（9 セル固定・共用する消費者もいないため TSV 等へ外出ししない）。
#
#   likelihood \ impact | high   | medium | low
#   high                | high   | high   | medium
#   medium              | high   | medium | low
#   low                 | medium | low    | low
declare -A LEVEL_MATRIX=(
  [high:high]=high
  [high:medium]=high
  [high:low]=medium
  [medium:high]=high
  [medium:medium]=medium
  [medium:low]=low
  [low:high]=medium
  [low:medium]=low
  [low:low]=low
)

# 走査基点をリポジトリルートへ解決する。git 管理外ではカレント基準へフォールバックする
# （checks 派生のサンドボックスは作業ツリーを持たないためこの経路を通る）。
repo_root=$(git rev-parse --show-toplevel 2>/dev/null || printf '.')

# docs/risks の在り処は 2 経路ある（sara-id.sh §6b と同じ扱い）:
#   1. RISK_DOCS_DIR（checks.risk-matrix のサンドボックス。作業ツリーが無いので
#      nix が store path を渡す）
#   2. git のリポジトリルート基準（`nix develop` からの直接実行・CI の sara job）
risks_dir="${RISK_DOCS_DIR:-}"
[[ -d "$risks_dir" ]] || risks_dir="$repo_root/docs/risks"

if [[ ! -d "$risks_dir" ]]; then
  echo "risk-matrix.sh: risk item のディレクトリを解決できない（$risks_dir）" >&2
  exit 1
fi

# frontmatter（先頭の `---` から次の `---` まで）から 1 フィールドを取り出す。
# yq を足さず sed で済ませる。本文にも `likelihood: medium — …` の形の根拠記述が
# あるため、範囲を frontmatter に限らないと本文側を拾って判定が狂う。
field_of() {
  sed -n '1{/^---$/!q}; 1,/^---$/{ /^---$/d; s/^'"$2"':[[:space:]]*//p }' "$1" | head -1
}

# ファイル一覧は改行区切りで受ける（この corpus のファイル名は日付 + UUID8 + slug で
# 空白・改行を含まない → CLAUDE.md の ID 規約）。
mapfile -t risk_files < <(find "$risks_dir" -maxdepth 1 -type f -name '*.md' | LC_ALL=C sort)

# 対象 0 件を緑にしない。走査先の解決を誤ったとき（ディレクトリ移動・環境変数の
# 誤配線）、件数を見ないと「全件合格」と報告してしまう。
if [[ "${#risk_files[@]}" -eq 0 ]]; then
  fault "docs/risks に risk item が 1 件も無い（走査先: $risks_dir）"
  exit "$fail"
fi

checked=0
for f in "${risk_files[@]}"; do
  name="$(basename "$f")"
  likelihood="$(field_of "$f" likelihood)"
  impact="$(field_of "$f" impact)"
  level="$(field_of "$f" level)"

  if [[ -z "$likelihood" || -z "$impact" || -z "$level" ]]; then
    fault "$name: frontmatter から 3 フィールドを読めない（likelihood=$likelihood impact=$impact level=$level）"
    continue
  fi

  want="${LEVEL_MATRIX[$likelihood:$impact]:-}"
  if [[ -z "$want" ]]; then
    fault "$name: likelihood=$likelihood impact=$impact に対応するマトリクスのセルが無い"
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
