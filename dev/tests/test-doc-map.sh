#!/usr/bin/env bash
# テストコード ⇔ テストドキュメント（CASE）対応の契約テスト（→ Issue #304、epic #283）。
#
# 実行:
#   nix develop '.?dir=dev#sara' -c dev/tests/test-doc-map.sh   # devShell / CI から直接
#   nix flake check ./dev                                        # checks.test-doc-map 経由
#
# 検証対象。番号は下の節見出しに対応する:
#   1.  順方向 — 全 CASE の target が実在するテスト資産を指す
#   2.  逆方向 — 全テスト資産に CASE がある（除外リストにあるものを除く）
#   3.  1:1 一意性 — 1 資産に 2 つ以上の CASE が張られていない
#   4.  データファイルの健全性 — 区分表が docs/test/ のディレクトリと一致し、
#       除外リストが実在する資産だけを挙げている
#
# ## 純静的であること
#
# この検証は毎 PR で回る（CI の sara ジョブ）。入力は dev/scripts/test-inventory.sh
# --static（find / glob のみ）と CASE frontmatter（yq）だけで、go test も nix eval も
# 呼ばない。sara devShell（sara / yq / git 等）だけで完結する。
#
# ## 規範は frontmatter にある
#
# CASE 本文の「## 対象」節は人間向けの補足で、ここでは照合しない。照合するのは
# frontmatter の target（docs/model.yaml で required: true）。

# -e は使わない（sara-id.sh と同じ理由）。このテストは「1 回の実行で全失敗を報告する」
# 集計方式で、-e があると最初の非ゼロ終了で以降のアサーションが走らず、退行時の診断が
# 先頭 1 件で切れる。
set -uo pipefail

fail=0
pass() { printf 'ok   - %s\n' "$1"; }
fault() {
  printf 'FAIL - %s\n' "$1"
  fail=1
}

for cmd in yq git; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "test-doc-map.sh: $cmd が要る（nix develop '.?dir=dev#sara' から実行する）" >&2
    exit 1
  fi
done

# yq は mikefarah/yq（Go 実装・v4）を前提にする。python-yq（v2/v3 系）が PATH に
# 居ると eval コマンドの構文が違って黙って空を返すため、実装を明示的に確かめる。
if ! yq --version 2>&1 | grep -q 'mikefarah\|version v4'; then
  echo "test-doc-map.sh: mikefarah/yq v4 が要る（実際: $(yq --version 2>&1)）" >&2
  exit 1
fi

repo_root=$(git rev-parse --show-toplevel 2>/dev/null || printf '.')
cd "$repo_root" || exit 1

inventory_sh=dev/scripts/test-inventory.sh
categories_tsv=dev/tests/test-categories.tsv
exclusions_tsv=dev/tests/test-doc-exclusions.tsv

for f in "$inventory_sh" "$categories_tsv" "$exclusions_tsv"; do
  if [ ! -f "$f" ]; then
    echo "test-doc-map.sh: $f が無い" >&2
    exit 1
  fi
done

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# コメント・空行を落として TSV の実データだけを読む。
read_tsv() { grep -v '^[[:space:]]*#' "$1" | grep -v '^[[:space:]]*$'; }

# --- 入力の収集 --------------------------------------------------------------

bash "$inventory_sh" --static > "$work/inventory" || {
  echo "test-doc-map.sh: $inventory_sh --static が失敗した" >&2
  exit 1
}
cut -f1 "$work/inventory" | LC_ALL=C sort > "$work/assets"

if [ ! -s "$work/assets" ]; then
  echo "test-doc-map.sh: テスト資産を 1 件も列挙できなかった" >&2
  exit 1
fi

# CASE の <ファイルパス>\t<target> 表。frontmatter の target を yq で読む
# （sara report matrix --format json は関係だけを返し、カスタムフィールドを含まない）。
# ファイルごとに yq を起こすのは 47 件規模なら実用上のコストにならない。
: > "$work/case-targets"
while IFS= read -r file; do
  target=$(yq --front-matter=extract '.target // ""' "$file" 2>/dev/null)
  printf '%s\t%s\n' "$file" "$target" >> "$work/case-targets"
done < <(grep -rl '^type: test_case' docs/test/ | LC_ALL=C sort)

if [ ! -s "$work/case-targets" ]; then
  echo "test-doc-map.sh: CASE を 1 件も読めなかった" >&2
  exit 1
fi

read_tsv "$exclusions_tsv" | cut -f1 | LC_ALL=C sort > "$work/exclusions"

# --- 1. 順方向（CASE の target が実在する） ----------------------------------

missing_target=0
dangling=0
while IFS=$'\t' read -r file target; do
  if [ -z "$target" ]; then
    fault "順方向: $file に target が無い（docs/model.yaml で required なので sara check も落ちる）"
    missing_target=1
    continue
  fi
  if ! grep -qxF "$target" "$work/assets"; then
    fault "順方向: $file の target '$target' に対応するテスト資産が無い（リネーム / 削除の追従漏れ）"
    dangling=1
  fi
done < "$work/case-targets"

if [ "$missing_target" -eq 0 ]; then
  pass "全 CASE が target を持つ"
fi
if [ "$dangling" -eq 0 ]; then
  pass "全 CASE の target が実在するテスト資産を指す"
fi

# --- 2. 逆方向（全テスト資産に CASE がある） ---------------------------------

cut -f2 "$work/case-targets" | grep -v '^$' | LC_ALL=C sort -u > "$work/covered"

uncovered=0
while IFS= read -r asset; do
  if grep -qxF "$asset" "$work/covered"; then
    continue
  fi
  if grep -qxF "$asset" "$work/exclusions"; then
    continue
  fi
  fault "逆方向: テスト資産 '$asset' に CASE が無い（CASE を起こすか $exclusions_tsv へ除外理由付きで追加する）"
  uncovered=1
done < "$work/assets"

if [ "$uncovered" -eq 0 ]; then
  pass "全テスト資産に CASE がある（除外リストを除く）"
fi

# --- 3. 1:1 一意性 -----------------------------------------------------------

# 順方向の 1:1（1 CASE = 1 target）は frontmatter が単一 text であることで型に担保
# されている（!list text ではない）。ここで見るのは逆方向の重複だけ。
dup_targets=$(cut -f2 "$work/case-targets" | grep -v '^$' | LC_ALL=C sort | uniq -d)
if [ -z "$dup_targets" ]; then
  pass "1 テスト資産に張られた CASE は高々 1 件（1:1）"
else
  while IFS= read -r target; do
    owners=$(awk -F'\t' -v t="$target" '$2 == t { printf "%s ", $1 }' "$work/case-targets")
    fault "1:1 違反: '$target' に複数の CASE が張られている（$owners）"
  done <<< "$dup_targets"
fi

# --- 4. データファイルの健全性 -----------------------------------------------

# 区分表 ⟷ docs/test/ のディレクトリ。区分を増減したときの片側更新漏れを検出する。
read_tsv "$categories_tsv" | cut -f1 | LC_ALL=C sort > "$work/categories"
find docs/test -mindepth 1 -maxdepth 1 -type d 2>/dev/null |
  sed 's|.*/||' |
  LC_ALL=C sort > "$work/case-dirs"

if diff -q "$work/categories" "$work/case-dirs" >/dev/null 2>&1; then
  pass "区分表が docs/test/ のディレクトリと一致する"
else
  only_table=$(comm -23 "$work/categories" "$work/case-dirs" | tr '\n' ' ')
  only_dirs=$(comm -13 "$work/categories" "$work/case-dirs" | tr '\n' ' ')
  fault "区分表と docs/test/ が不一致（表のみ: ${only_table:-なし}/ ディレクトリのみ: ${only_dirs:-なし}）"
fi

# 区分表の 2 列目（説明）が空でないこと。区分名だけの行は表の意味を成さない。
empty_desc=$(read_tsv "$categories_tsv" | awk -F'\t' 'NF < 2 || $2 == "" { print $1 }' | tr '\n' ' ')
if [ -z "$empty_desc" ]; then
  pass "区分表の全行が説明を持つ"
else
  fault "区分表に説明の無い行がある（$empty_desc）"
fi

# 除外リストが実在する資産だけを挙げていること。消えた資産の除外が残ると、
# 同名の資産を後で追加したときに CASE 無しのまま黙って通る。
stale_exclusion=0
while IFS= read -r asset; do
  if ! grep -qxF "$asset" "$work/assets"; then
    fault "除外リストの '$asset' は列挙されるテスト資産に無い（stale な除外）"
    stale_exclusion=1
  fi
done < "$work/exclusions"

if [ "$stale_exclusion" -eq 0 ]; then
  pass "除外リストが実在するテスト資産だけを挙げている"
fi

# 除外リストの 2 列目（理由）が空でないこと。理由の無い除外は後から判断できない。
empty_reason=$(read_tsv "$exclusions_tsv" | awk -F'\t' 'NF < 2 || $2 == "" { print $1 }' | tr '\n' ' ')
if [ -z "$empty_reason" ]; then
  pass "除外リストの全行が理由を持つ"
else
  fault "除外リストに理由の無い行がある（$empty_reason）"
fi

# CASE と除外リストの重複。除外しつつ CASE を持つのは意図が二重で、どちらかが古い。
both=$(LC_ALL=C comm -12 "$work/covered" "$work/exclusions" | tr '\n' ' ')
if [ -z "$both" ]; then
  pass "除外リストと CASE の target が排他である"
else
  fault "除外されているのに CASE がある（$both）"
fi

# --- 結果 -------------------------------------------------------------------

echo
if [ "$fail" -eq 0 ]; then
  printf 'test-doc-map: 全アサーション通過（CASE %d 件 / テスト資産 %d 件 / 除外 %d 件）\n' \
    "$(wc -l < "$work/case-targets")" "$(wc -l < "$work/assets")" "$(wc -l < "$work/exclusions")"
  exit 0
fi

printf 'test-doc-map: 失敗あり\n' >&2
exit 1
