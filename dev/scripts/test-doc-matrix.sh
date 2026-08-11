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

# shellcheck source=dev/scripts/lib-testdoc.sh
. "$(dirname "$0")/lib-testdoc.sh"

require_commands "test-doc-matrix.sh（nix develop ./dev から実行する）" yq jq git sara || exit 1
require_yq_go test-doc-matrix.sh || exit 1

cd "$(testdoc_repo_root)" || exit 1

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# --- 入力の収集 --------------------------------------------------------------

bash dev/scripts/test-inventory.sh --full > "$work/inventory" || {
  echo "test-doc-matrix.sh: test-inventory.sh --full が失敗した" >&2
  exit 1
}

sara report matrix --format json > "$work/matrix.json" 2>/dev/null || {
  echo "test-doc-matrix.sh: sara report matrix が失敗した" >&2
  exit 1
}

# CASE: <target>\t<CASE フル ID>\t<name>\t<区分>\t<ファイルパス>
#
# 1 ファイル 1 回の yq で frontmatter の 3 列を採る。区分とファイルパスは置き場所
# （docs/test/<区分>/…）から導けるので yq には要らない。ループを 2 本に割ると同じ入力に
# 対する grep 条件が 2 箇所へ散り、片方だけ変えたときに両者の集合がずれる。
: > "$work/cases"
while IFS= read -r file; do
  category=$(printf '%s\n' "$file" | cut -d/ -f3)
  if ! row=$(yq --front-matter=extract -r \
    '[(.target // ""), (.id // ""), (.name // "")] | @tsv' "$file" 2>/dev/null); then
    echo "test-doc-matrix.sh: 警告: $file の frontmatter を読めなかった（対応表から落ちる）" >&2
    continue
  fi
  printf '%s\t%s\t%s\n' "$row" "$category" "$file" >> "$work/cases"
done < <(grep -rl '^type: test_case' docs/test/ | LC_ALL=C sort)

if [ ! -s "$work/cases" ]; then
  echo "test-doc-matrix.sh: CASE を 1 件も読めなかった" >&2
  exit 1
fi

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

# フル ID → 散文用の省略形（<PREFIX>-<前方 8 文字>）。形式に合わない入力（空文字・
# 8 文字 hex でない ID）はそのまま通すと空のコードスパンとして表に出て不整合が見えないため、
# 目に付く形へ置き換える。
short_id() {
  local id=$1
  # 正準形の知識を 1 箇所に留めるため、判定と切り出しを同じ正規表現で行う
  # （BASH_REMATCH の捕獲を使う。sed と二重に書くと片方だけ変える事故が起きる）。
  if [[ "$id" =~ ^([A-Z]+)-([0-9a-f]{8}) ]]; then
    printf '%s-%s\n' "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}"
  else
    printf '(ID 不正: %s)\n' "${id:-空}"
  fi
}

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

# --- 行の事前計算 ------------------------------------------------------------
#
# 区分 × 資産の対応を 1 度だけ計算して両セクション（表・テスト名内訳）が共有する。
# 区分ごとに資産を走査して都度 lookup すると、同じフィルタ条件が 2 箇所に散って
# 片方だけ直したときに対象集合がずれ、awk のフルスキャンも資産数 × 区分数だけ走る。
#
# $work/asset-rows: <区分>\t<資産>\t<CASE フル ID>\t<CASE name>
# $work/assets:     資産の一意リスト（inventory 由来）
cut -f1 "$work/inventory" | LC_ALL=C sort -u > "$work/assets"

# cases（<target>\t<id>\t<name>\t<区分>\t<ファイル>）を資産側から引ける形へ並べ替える。
awk -F'\t' -v OFS='\t' '
  NR == FNR { asset[$1] = 1; next }
  $1 in asset { print $4, $1, $2, $3 }
' "$work/assets" "$work/cases" | LC_ALL=C sort > "$work/asset-rows"

# inventory に無い target を指す CASE（リネーム / 削除の追従漏れ）は上の join で落ちる。
# 「未分類」節はこれを拾わない（あちらは資産側から見た CASE 無し）。落ちた CASE が対応表の
# どこにも現れないまま無警告になるのを避けるため、stderr へ警告し末尾の節にも載せる
# （test-doc-map.sh §1 が本来落とすが、sara ジョブは required check ではない → ADR-0050。
# 未分類節と同じ扱いに揃える）。
#
# 診断は CASE のフル ID ではなくファイルパスで出す（追従漏れを直す人が開く対象）。
# target が空の CASE もここに落ちるので、空は別表記にして調査先を曖昧にしない。
awk -F'\t' -v OFS='\t' '
  NR == FNR { asset[$1] = 1; next }
  !($1 in asset) { print $5, ($1 == "" ? "(target 空)" : $1) }
' "$work/assets" "$work/cases" > "$work/dangling-cases"

if [ -s "$work/dangling-cases" ]; then
  echo "test-doc-matrix.sh: 警告: 実在しない資産を指す CASE がある（対応表の「実在しない資産を指す CASE」節を参照）:" >&2
  sed 's/^/  /; s/\t/ → /' "$work/dangling-cases" >&2
fi

# --- 生成 --------------------------------------------------------------------

emit_header() {
  local asset_total case_total
  asset_total=$(wc -l < "$work/assets")
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
}

# 区分 1 つ分の対応表。行が無ければその旨を書く。
emit_category_table() {
  local category=$1 description=$2
  printf '## %s\n\n%s\n\n' "$category" "$description"

  if ! awk -F'\t' -v c="$category" '$1 == c { found = 1 } END { exit !found }' "$work/asset-rows"; then
    printf '（この区分に対応する CASE が無い）\n\n'
    return
  fi

  printf '| テスト資産 | CASE | covers する TC | 上流 RISK |\n'
  printf '| --- | --- | --- | --- |\n'
  awk -F'\t' -v c="$category" -v OFS='\t' '$1 == c { print $2, $3, $4 }' "$work/asset-rows" |
    while IFS=$'\t' read -r asset case_id case_name; do
      printf '| `%s` | `%s` %s | %s | %s |\n' \
        "$asset" "$(short_id "$case_id")" "$case_name" \
        "$(tc_cell "$case_id")" "$(risk_cell "$case_id")"
    done
  printf '\n'
}

# 区分 1 つ分のテスト名内訳（表のセルに収まらないため表の後に続けて置く）。
emit_category_names() {
  local category=$1
  local emitted=0
  while IFS= read -r asset; do
    local block
    block=$(names_block "$asset")
    [ -z "$block" ] && continue
    if [ "$emitted" -eq 0 ]; then
      printf '### %s のテスト名内訳\n\n' "$category"
      emitted=1
    fi
    printf '**`%s`**\n\n%s\n\n' "$asset" "$block"
  done < <(awk -F'\t' -v c="$category" '$1 == c { print $2 }' "$work/asset-rows")
}

# 除外リスト。CASE を持たないことが意図的な資産。
emit_exclusions() {
  printf '## CASE を持たないテスト資産\n\n'
  printf 'dev/tests/test-doc-exclusions.tsv（契約テストの逆方向除外リスト）の内容。\n\n'
  printf '| テスト資産 | 除外理由 |\n'
  printf '| --- | --- |\n'
  read_tsv dev/tests/test-doc-exclusions.tsv |
    while IFS=$'\t' read -r asset reason; do
      printf '| `%s` | %s |\n' "$asset" "$reason"
    done
  printf '\n'
}

# CASE も除外行も持たない資産。契約テストが本来落とす状態だが、sara ジョブは required
# status check ではない（→ ADR-0050）ため FAIL のまま main へ入りうる。対応表が「全資産を
# 映す」成果物である以上、黙って落とさず節として明示し stderr へも警告する。
emit_unclassified() {
  cut -f2 "$work/asset-rows" | LC_ALL=C sort -u > "$work/covered"
  read_tsv dev/tests/test-doc-exclusions.tsv | cut -f1 | LC_ALL=C sort -u > "$work/excluded"

  local unclassified
  unclassified=$(LC_ALL=C comm -23 "$work/assets" <(LC_ALL=C sort -u "$work/covered" "$work/excluded"))

  if [ -z "$unclassified" ]; then
    return
  fi

  echo "test-doc-matrix.sh: 警告: CASE も除外行も持たない資産がある（対応表の「未分類」節を参照）:" >&2
  printf '%s\n' "$unclassified" | sed 's/^/  /' >&2

  printf '## 未分類のテスト資産\n\n'
  printf 'CASE も除外行も持たない資産。dev/tests/test-doc-map.sh が落とすべき状態で、\n'
  printf 'CASE を起こすか dev/tests/test-doc-exclusions.tsv へ除外理由を書く。\n\n'
  printf '| テスト資産 |\n'
  printf '| --- |\n'
  printf '%s\n' "$unclassified" | sed 's/^/| `/; s/$/` |/'
  printf '\n'
}

# 実在しない資産を指す CASE。上の join で対応表の本体から落ちるため、成果物側にも節を
# 出す（stderr 警告だけだと artifact を読む人には落ちた CASE が見えず、ヘッダの件数と
# 本体の行数が合わない理由が分からない）。未分類節と同じ扱い。
emit_dangling() {
  if [ ! -s "$work/dangling-cases" ]; then
    return
  fi

  printf '## 実在しない資産を指す CASE\n\n'
  printf 'target が inventory に無い CASE。リネーム / 削除の追従漏れで、対応表の本体からは\n'
  printf '落ちている。dev/tests/test-doc-map.sh §1 が落とすべき状態。\n\n'
  printf '| CASE | target |\n'
  printf '| --- | --- |\n'
  while IFS=$'\t' read -r file target; do
    printf '| `%s` | `%s` |\n' "$file" "$target"
  done < "$work/dangling-cases"
  printf '\n'
}

emit() {
  emit_header

  # 区分ごとのセクション（順序は test-categories.tsv の記載順）。
  while IFS=$'\t' read -r category description; do
    emit_category_table "$category" "$description"
    emit_category_names "$category"
  done < <(read_tsv dev/tests/test-categories.tsv)

  emit_exclusions
  emit_unclassified
  emit_dangling
}

if [ -n "$out" ]; then
  emit > "$out" || exit 1
  echo "test-doc-matrix: $out を生成した" >&2
else
  emit
fi
