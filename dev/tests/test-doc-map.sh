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
#   5.  静的リストの健全性 — test-inventory.sh の FLAKE_CHECKS が flake ファイルの
#       checks 定義と一致する（片側更新漏れの検出）
#   6.  自己検証 — 合成フィクスチャで §1〜§3 のアサーションが期待どおり FAIL する
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

# shellcheck source=dev/scripts/lib-testdoc.sh
. "$(dirname "$0")/../scripts/lib-testdoc.sh"

require_commands "test-doc-map.sh（nix develop '.?dir=dev#sara' から実行する）" yq git || exit 1
require_yq_go test-doc-map.sh || exit 1

cd "$(testdoc_repo_root)" || exit 1

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
#
# yq の失敗（frontmatter が壊れている等）と「target が空」は区別する。畳むと
# 「target が無い（model.yaml で required なので sara check も落ちる）」と診断されて、
# sara check が実は緑な状況で誤った調査先へ案内してしまう。
: > "$work/case-targets"
unreadable=0
while IFS= read -r file; do
  if ! target=$(yq --front-matter=extract '.target // ""' "$file" 2>/dev/null); then
    fault "入力: $file の frontmatter を yq で読めなかった（YAML の構文を確認する）"
    unreadable=1
    continue
  fi
  printf '%s\t%s\n' "$file" "$target" >> "$work/case-targets"
done < <(grep -rl '^type: test_case' docs/test/ | LC_ALL=C sort)

if [ ! -s "$work/case-targets" ]; then
  echo "test-doc-map.sh: CASE を 1 件も読めなかった" >&2
  exit 1
fi

if [ "$unreadable" -eq 0 ]; then
  pass "全 CASE の frontmatter を読める"
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

# --- 5. 静的リストの健全性（FLAKE_CHECKS ⟷ flake ファイル） ------------------

# test-inventory.sh は flake check の一覧を静的リストで持つ（--static を純ファイル走査に
# 保つため nix eval を呼べない）。この突合が無いと片側更新漏れが検知できない: check を
# flake へ足してリストへ足し忘れると列挙に現れず、順方向・逆方向のどちらも触れないまま
# 黙って緑になる（実際に踏んだ）。sara-id.sh §6b が prefix マップを model.yaml と
# 突合するのと同じ発想で、flake ファイルの定義行を grep して集合比較する。

# 静的リストから flake check の識別子を採る。`dev:` 接頭辞で flake ファイルを振り分ける。
bash "$inventory_sh" --static |
  awk -F'\t' '$2 == "flake-check" { print $1 }' |
  LC_ALL=C sort > "$work/listed-checks"

# flakeModule 生成分（flake ファイルに定義行が無い）は grep 突合の対象外。
bash "$inventory_sh" --module-generated-checks | LC_ALL=C sort > "$work/module-checks"

# flake ファイル側の定義。`checks.<name> =` の行から名前を採る。
extract_checks() {
  grep -oE '^[[:space:]]*checks\.[A-Za-z0-9_-]+[[:space:]]*=' "$1" |
    sed -E 's/^[[:space:]]*checks\.([A-Za-z0-9_-]+)[[:space:]]*=.*/\1/'
}

{
  extract_checks flake.nix | sed 's|^|checks.|'
  extract_checks dev/flake.nix | sed 's|^|dev:checks.|'
} | LC_ALL=C sort -u > "$work/defined-checks"

if [ ! -s "$work/defined-checks" ]; then
  fault "静的リスト: flake ファイルから checks 定義を 1 件も抽出できなかった（grep の前提が崩れた）"
else
  # 静的リスト（module 生成分を除く）⟷ flake の定義。
  LC_ALL=C comm -23 "$work/listed-checks" "$work/module-checks" > "$work/listed-hand"

  only_flake=$(LC_ALL=C comm -13 "$work/listed-hand" "$work/defined-checks" | tr '\n' ' ')
  only_list=$(LC_ALL=C comm -23 "$work/listed-hand" "$work/defined-checks" | tr '\n' ' ')

  if [ -z "$only_flake" ] && [ -z "$only_list" ]; then
    pass "FLAKE_CHECKS が flake ファイルの checks 定義と一致する"
  fi
  if [ -n "$only_flake" ]; then
    fault "静的リスト: flake に在るが FLAKE_CHECKS に無い（$only_flake）— $inventory_sh へ追加する"
  fi
  if [ -n "$only_list" ]; then
    fault "静的リスト: FLAKE_CHECKS に在るが flake の定義に無い（$only_list）— 消えた check の残骸か、flakeModule 生成分なら --module-generated-checks 側へ移す"
  fi

  # module 生成分は逆に flake ファイルへ定義行を持たないはず（持つなら手書き側の管理へ移す）。
  wrongly_module=$(LC_ALL=C comm -12 "$work/module-checks" "$work/defined-checks" | tr '\n' ' ')
  if [ -z "$wrongly_module" ]; then
    pass "flakeModule 生成分が flake ファイルに定義行を持たない"
  else
    fault "静的リスト: flakeModule 生成分として除外しているが flake に定義行がある（$wrongly_module）"
  fi
fi

# --- 6. 自己検証（アサーションが本当に落ちるか） -----------------------------

# §1〜§3 は「今のリポジトリが正しい」ことだけを見るため、アサーションが誤って常に真を
# 返す退行が起きても緑のまま通る。合成フィクスチャで各判定ロジックが期待どおり FAIL 側へ
# 倒れることを確かめる（リポジトリの実ファイルには一切触らない）。
#
# ここで検証するのは判定の骨（grep -qxF による集合演算・uniq -d による重複検出）で、
# §1〜§3 が使っているものと同じ手段を同じ向きで叩く。

self_fail=0

# 順方向: 実在しない target が assets に無いと判定されること。
if grep -qxF 'internal/does/not/exist_test.go' "$work/assets"; then
  fault "自己検証: 実在しない target が資産として在ると判定された"
  self_fail=1
fi

# 逆方向: 除外にも CASE にも無い架空の資産が未カバーと判定されること。
if grep -qxF 'internal/phantom/phantom_test.go' "$work/covered" ||
  grep -qxF 'internal/phantom/phantom_test.go' "$work/exclusions"; then
  fault "自己検証: 架空の資産がカバー済み / 除外済みと判定された"
  self_fail=1
fi

# 1:1: 同じ target を 2 度並べたら uniq -d が拾うこと。
synthetic_dup=$(printf 'a\na\nb\n' | LC_ALL=C sort | uniq -d)
if [ "$synthetic_dup" != "a" ]; then
  fault "自己検証: 重複検出（uniq -d）が機能していない（実際: $synthetic_dup）"
  self_fail=1
fi

# 入力の非空性: 検査対象の 3 集合が空だと全アサーションが空虚に真になる。
for name in assets covered exclusions; do
  if [ ! -s "$work/$name" ]; then
    fault "自己検証: $name が空（アサーションが空虚に真になる）"
    self_fail=1
  fi
done

if [ "$self_fail" -eq 0 ]; then
  pass "自己検証: 判定ロジックが合成フィクスチャで期待どおり倒れる"
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
