#!/usr/bin/env bash
# テスト資産 ⇔ CASE ⇔ TC ⇔ RISK の対応表を Markdown 1 枚で生成する
# （→ Issue #304、epic #283）。
#
# 実行:
#   nix develop ./dev -c dev/scripts/test-doc-matrix.sh [出力パス]
#
# 出力パス省略時は stdout。CI は main push / workflow_dispatch のジョブで生成し
# artifact へ上げる。**リポジトリへはコミットしない**（生成物とソースのドリフト回避）。
#
# ## 入力
#
#   - dev/scripts/test-inventory.sh --full        テスト資産とテスト名（go test -json 実行ベース）
#   - sara report matrix --format json            CASE→TC / TC→RISK の全関係を 1 回で取得
#   - CASE frontmatter の target（yq）            資産 → CASE の join キー
#   - dev/tests/test-categories.tsv               区分のセクション順
#   - dev/tests/test-doc-exclusions.tsv           CASE を持たない資産の理由
#
# 区分（8 区分）は CASE ファイルの置き場所 docs/test/<区分>/ から導く。パス prefix では
# 決まらない（同 prefix が複数区分へ割れる → test-categories.tsv の注記）。
#
# ## 出力形式
#
# 区分ごとにセクションを切り、行 = テスト資産、列 = CASE（省略形 + name）/ covers する TC /
# 上流 RISK。Go サブテスト・nix-unit attr の内訳は <details> で折りたたむ。
# CASE を持たない資産（除外リスト）は末尾に別表で理由付きで載せる。

set -uo pipefail

if [ "$#" -gt 1 ]; then
  cat >&2 <<'EOF'
usage: test-doc-matrix.sh [出力パス]

  出力パス  省略時は stdout へ書く
EOF
  exit 2
fi

out=${1-}

for cmd in yq jq git sara; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "test-doc-matrix.sh: $cmd が要る（nix develop ./dev から実行する）" >&2
    exit 1
  fi
done

# yq は mikefarah/yq（Go 実装・v4）を前提にする。ambient PATH の python-yq（v2/v3 系）を
# 拾うと構文違いで黙って空を返し、CASE 0 件の対応表が出る（実際に踏んだ）。
if ! yq --version 2>&1 | grep -q 'mikefarah\|version v4'; then
  echo "test-doc-matrix.sh: mikefarah/yq v4 が要る（実際: $(yq --version 2>&1)）" >&2
  exit 1
fi

repo_root=$(git rev-parse --show-toplevel 2>/dev/null || printf '.')
cd "$repo_root" || exit 1

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

read_tsv() { grep -v '^[[:space:]]*#' "$1" | grep -v '^[[:space:]]*$'; }

# --- 入力の収集 --------------------------------------------------------------

bash dev/scripts/test-inventory.sh --full > "$work/inventory" || {
  echo "test-doc-matrix.sh: test-inventory.sh --full が失敗した" >&2
  exit 1
}

sara report matrix --format json > "$work/matrix.json" 2>/dev/null || {
  echo "test-doc-matrix.sh: sara report matrix が失敗した" >&2
  exit 1
}

# CASE: <target>\t<CASE フル ID>\t<name>
: > "$work/cases"
while IFS= read -r file; do
  yq --front-matter=extract -r \
    '[(.target // ""), (.id // ""), (.name // "")] | @tsv' "$file" 2>/dev/null |
    awk -F'\t' -v OFS='\t' '{ print $1, $2, $3 }' >> "$work/cases"
done < <(grep -rl '^type: test_case' docs/test/ | LC_ALL=C sort)

# CASE の区分（置き場所 docs/test/<区分>/ 由来）: <CASE フル ID>\t<区分>
: > "$work/case-categories"
while IFS= read -r file; do
  id=$(yq --front-matter=extract -r '.id // ""' "$file" 2>/dev/null)
  category=$(printf '%s\n' "$file" | cut -d/ -f3)
  printf '%s\t%s\n' "$id" "$category" >> "$work/case-categories"
done < <(grep -rl '^type: test_case' docs/test/ | LC_ALL=C sort)

# CASE → TC: <CASE フル ID>\t<TC フル ID>\t<TC name>
jq -r '
  .rows[]
  | select(.source_type == "Test Case")
  | .source_id as $case
  | .targets[]
  | select(.target_type == "Test Condition")
  | [$case, .id, .name] | @tsv
' "$work/matrix.json" > "$work/case-tc"

# TC → RISK: <TC フル ID>\t<RISK フル ID>\t<RISK name>
jq -r '
  .rows[]
  | select(.source_type == "Test Condition")
  | .source_id as $tc
  | .targets[]
  | select(.target_type == "Risk")
  | [$tc, .id, .name] | @tsv
' "$work/matrix.json" > "$work/tc-risk"

# --- 部品 --------------------------------------------------------------------

# フル ID → 散文用の省略形（<PREFIX>-<前方 8 文字>）。
short_id() {
  printf '%s\n' "$1" | sed -E 's/^([A-Z]+)-([0-9a-f]{8}).*/\1-\2/'
}

lookup_case() { awk -F'\t' -v t="$1" '$1 == t { print $2 "\t" $3 }' "$work/cases"; }
lookup_category() { awk -F'\t' -v id="$1" '$1 == id { print $2 }' "$work/case-categories"; }

# セル内で使う <br> 区切りの箇条書き（Markdown の表はセル内改行を許さない）。
join_br() { paste -sd'@' - | sed 's/@/<br>/g'; }

# CASE の covers する TC を「TC-xxxxxxxx name」の <br> 区切りで返す。
tc_cell() {
  awk -F'\t' -v id="$1" '$1 == id { print $2 "\t" $3 }' "$work/case-tc" |
    while IFS=$'\t' read -r tc_id tc_name; do
      printf '`%s` %s\n' "$(short_id "$tc_id")" "$tc_name"
    done | join_br
}

# CASE の上流 RISK（covers する TC が mitigates するもの）を重複排除して返す。
risk_cell() {
  awk -F'\t' -v id="$1" '$1 == id { print $2 }' "$work/case-tc" |
    while IFS= read -r tc_id; do
      awk -F'\t' -v t="$tc_id" '$1 == t { print $2 "\t" $3 }' "$work/tc-risk"
    done | LC_ALL=C sort -u |
    while IFS=$'\t' read -r risk_id risk_name; do
      printf '`%s` %s\n' "$(short_id "$risk_id")" "$risk_name"
    done | join_br
}

# 資産のテスト名内訳。1 件でもあれば <details> で折りたたむ。
names_block() {
  local asset=$1
  local names
  names=$(awk -F'\t' -v a="$asset" '$1 == a && $3 != "" { print $3 }' "$work/inventory" | LC_ALL=C sort)
  if [ -z "$names" ]; then
    return
  fi
  printf '<details><summary>テスト名 %d 件</summary>\n\n' "$(printf '%s\n' "$names" | wc -l)"
  printf '%s\n' "$names" | sed 's/^/- `/; s/$/`/'
  printf '\n</details>\n'
}

# --- 生成 --------------------------------------------------------------------

emit() {
  local asset_total case_total
  asset_total=$(cut -f1 "$work/inventory" | LC_ALL=C sort -u | wc -l)
  case_total=$(wc -l < "$work/cases")

  cat <<EOF
# テストコード ⇔ テストドキュメント対応表

テスト資産 ⇔ CASE ⇔ TC ⇔ 上流 RISK の対応。dev/scripts/test-doc-matrix.sh の生成物で、
リポジトリへはコミットしない（CI の artifact として取得する → Issue #304）。

- テスト資産: ${asset_total} 件 / CASE: ${case_total} 件
- 区分（セクション）は CASE の置き場所 \`docs/test/<区分>/\` 由来
- ID は散文用の省略形（\`<PREFIX>-<前方 8 文字>\`）。フル ID は各 item の frontmatter を参照
- CASE を持たない資産は末尾の「CASE を持たないテスト資産」を参照

EOF

  # 区分ごとのセクション（順序は test-categories.tsv の記載順）。
  while IFS=$'\t' read -r category description; do
    printf '## %s\n\n%s\n\n' "$category" "$description"

    # この区分に属する資産（= 区分内の CASE の target）を列挙する。
    local rows=0
    local buffer="$work/section"
    : > "$buffer"

    while IFS=$'\t' read -r asset _type _name; do
      local case_row case_id case_name
      case_row=$(lookup_case "$asset")
      [ -z "$case_row" ] && continue
      case_id=${case_row%%$'\t'*}
      case_name=${case_row#*$'\t'}
      [ "$(lookup_category "$case_id")" = "$category" ] || continue

      {
        printf '| `%s` | `%s` %s | %s | %s |\n' \
          "$asset" "$(short_id "$case_id")" "$case_name" \
          "$(tc_cell "$case_id")" "$(risk_cell "$case_id")"
      } >> "$buffer"
      rows=$((rows + 1))
    done < <(LC_ALL=C sort -u -t$'\t' -k1,1 "$work/inventory")

    if [ "$rows" -eq 0 ]; then
      printf '（この区分に対応する CASE が無い）\n\n'
      continue
    fi

    printf '| テスト資産 | CASE | covers する TC | 上流 RISK |\n'
    printf '| --- | --- | --- | --- |\n'
    cat "$buffer"
    printf '\n'

    # テスト名の内訳（表のセルに収まらないため区分ごとに続けて置く）。
    local emitted_names=0
    while IFS= read -r asset; do
      local case_row case_id
      case_row=$(lookup_case "$asset")
      [ -z "$case_row" ] && continue
      case_id=${case_row%%$'\t'*}
      [ "$(lookup_category "$case_id")" = "$category" ] || continue

      local block
      block=$(names_block "$asset")
      [ -z "$block" ] && continue
      if [ "$emitted_names" -eq 0 ]; then
        printf '### %s のテスト名内訳\n\n' "$category"
        emitted_names=1
      fi
      printf '**`%s`**\n\n%s\n\n' "$asset" "$block"
    done < <(cut -f1 "$work/inventory" | LC_ALL=C sort -u)
  done < <(read_tsv dev/tests/test-categories.tsv)

  # 除外リスト。CASE を持たないことが意図的な資産。
  printf '## CASE を持たないテスト資産\n\n'
  printf 'dev/tests/test-doc-exclusions.tsv（契約テストの逆方向除外リスト）の内容。\n\n'
  printf '| テスト資産 | 除外理由 |\n'
  printf '| --- | --- |\n'
  read_tsv dev/tests/test-doc-exclusions.tsv |
    while IFS=$'\t' read -r asset reason; do
      printf '| `%s` | %s |\n' "$asset" "$reason"
    done
}

if [ -n "$out" ]; then
  emit > "$out" || exit 1
  echo "test-doc-matrix: $out を生成した" >&2
else
  emit
fi
