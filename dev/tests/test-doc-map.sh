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
#   6.  自己検証 — §1〜§3 が呼ぶ judge_* 関数そのものを合成フィクスチャへ当て、
#       違反なしで真・違反ありで偽の両側へ倒れることを確かめる
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

# --- 判定（§1〜§3 の本体。§6 が同じ関数を合成フィクスチャで叩く） --------------
#
# 判定を関数へ切り出して §1〜§3 と §6 の双方から呼ぶ。§6 が判定を再実装すると
# 「アサーションが常に真を返す退行」を検知できない（プリミティブの動作確認になるだけで、
# §1 の条件を `if false` へ差し替えても緑のまま通る → review 2 周目で実証された）。
#
# 各関数は診断行を stdout へ出し、違反があれば 1 を返す。呼び出し側が pass / fault へ
# 振り分ける。ファイルパスを引数で受けるので、実データでも合成フィクスチャでも同じ経路。

# 順方向: case_targets（<ファイル>\t<target>）の target が assets に在るか。
# $3 は診断メッセージ用の除外リストパス（案内文にのみ使う）。
judge_forward() {
  local case_targets=$1 assets=$2
  local violated=0 file target
  while IFS=$'\t' read -r file target; do
    if [ -z "$target" ]; then
      echo "順方向: $file に target が無い（docs/model.yaml で required なので sara check も落ちる）"
      violated=1
      continue
    fi
    if ! grep -qxF "$target" "$assets"; then
      echo "順方向: $file の target '$target' に対応するテスト資産が無い（リネーム / 削除の追従漏れ）"
      violated=1
    fi
  done < "$case_targets"
  return "$violated"
}

# 逆方向: assets の各要素が covered か exclusions のどちらかに在るか。
judge_reverse() {
  local assets=$1 covered=$2 exclusions=$3 exclusions_hint=${4:-$3}
  local violated=0 asset
  while IFS= read -r asset; do
    if grep -qxF "$asset" "$covered"; then
      continue
    fi
    if grep -qxF "$asset" "$exclusions"; then
      continue
    fi
    echo "逆方向: テスト資産 '$asset' に CASE が無い（CASE を起こすか $exclusions_hint へ除外理由付きで追加する）"
    violated=1
  done < "$assets"
  return "$violated"
}

# 1:1: case_targets の target に重複が無いか。
judge_unique() {
  local case_targets=$1
  local dups target owners
  dups=$(cut -f2 "$case_targets" | grep -v '^$' | LC_ALL=C sort | uniq -d)
  if [ -z "$dups" ]; then
    return 0
  fi
  while IFS= read -r target; do
    owners=$(awk -F'\t' -v t="$target" '$2 == t { printf "%s ", $1 }' "$case_targets")
    echo "1:1 違反: '$target' に複数の CASE が張られている（$owners）"
  done <<< "$dups"
  return 1
}

# 判定関数を実データへ当て、診断行を fault へ流す。
run_judge() {
  local ok_message=$1
  shift
  local diagnostics
  if diagnostics=$("$@"); then
    pass "$ok_message"
    return 0
  fi
  while IFS= read -r line; do
    [ -n "$line" ] && fault "$line"
  done <<< "$diagnostics"
  return 1
}

# --- 1. 順方向（CASE の target が実在する） ----------------------------------

run_judge "全 CASE の target が実在するテスト資産を指す" \
  judge_forward "$work/case-targets" "$work/assets"

# --- 2. 逆方向（全テスト資産に CASE がある） ---------------------------------

cut -f2 "$work/case-targets" | grep -v '^$' | LC_ALL=C sort -u > "$work/covered"

run_judge "全テスト資産に CASE がある（除外リストを除く）" \
  judge_reverse "$work/assets" "$work/covered" "$work/exclusions" "$exclusions_tsv"

# --- 3. 1:1 一意性 -----------------------------------------------------------

# 順方向の 1:1（1 CASE = 1 target）は frontmatter が単一 text であることで型に担保
# されている（!list text ではない）。ここで見るのは逆方向の重複だけ。
run_judge "1 テスト資産に張られた CASE は高々 1 件（1:1）" \
  judge_unique "$work/case-targets"

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

# 静的リストの flake check 識別子。冒頭で保存した --static の出力から採る
# （再実行すると終了ステータスの扱いが冒頭と二重になる）。
awk -F'\t' '$2 == "flake-check" { print $1 }' "$work/inventory" |
  LC_ALL=C sort > "$work/listed-checks"

# flakeModule 生成分（flake ファイルに定義行が無い）は grep 突合の対象外。
bash "$inventory_sh" --module-generated-checks | LC_ALL=C sort > "$work/module-checks" || {
  echo "test-doc-map.sh: $inventory_sh --module-generated-checks が失敗した" >&2
  exit 1
}

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

  # module 生成分は only_flake から除く。除かないと「flake に在るが FLAKE_CHECKS に無い
  # → 追加せよ」と出るが、実際は既に FLAKE_CHECKS に在るので指示に従っても直らない
  # （正しい診断は下の wrongly_module 側が出す）。
  LC_ALL=C comm -13 "$work/listed-hand" "$work/defined-checks" |
    LC_ALL=C comm -23 - "$work/module-checks" > "$work/only-flake"

  only_flake=$(tr '\n' ' ' < "$work/only-flake")
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

# --- 6. 自己検証（判定関数が両側へ倒れるか） ---------------------------------
#
# §1〜§3 は「今のリポジトリが正しい」ことだけを見るため、判定が誤って常に真を返す退行が
# 起きても緑のまま通る。ここでは §1〜§3 が実際に呼んでいる judge_* 関数そのものを、
# 合成フィクスチャ（$work/self/ 配下・リポジトリの実ファイルには触らない）へ当てて、
# 違反なしで真・違反ありで偽の**両側へ**倒れることを確かめる。
#
# 判定を §6 内で再実装してはいけない。再実装すると grep / uniq の動作確認になるだけで、
# §1 の条件を `if false` へ差し替えても緑のまま通る（2 周目のレビューで実証された）。

self=$work/self
mkdir -p "$self"

# 違反のないフィクスチャ。
printf 'a_test.go\nb_test.go\n' > "$self/assets-ok"
printf 'CASE-a.md\ta_test.go\nCASE-b.md\tb_test.go\n' > "$self/case-targets-ok"
printf 'a_test.go\nb_test.go\n' > "$self/covered-ok"
printf '\n' > "$self/exclusions-empty"

# 違反のあるフィクスチャ。判定内の分岐ごとに 1 つずつ用意する。1 つのフィクスチャで
# 複数の分岐を同時に踏ませると、片方の分岐が死んでももう片方が違反を返して緑になり、
# 「常に真になる退行」を取りこぼす（実際に踏んだ）。
# 順方向 (a): 実在しない target を指す。
printf 'CASE-a.md\tgone_test.go\n' > "$self/case-targets-dangling"
# 順方向 (b): target が空。
printf 'CASE-b.md\t\n' > "$self/case-targets-empty"
# 逆方向: covered にも exclusions にも無い資産。
printf 'a_test.go\nc_test.go\n' > "$self/assets-uncovered"
# 逆方向: 除外リストだけで消える資産（除外経路の単独検証用）。
printf 'c_test.go\n' > "$self/assets-c-only"
# 1:1: 同じ target を 2 CASE が張る。
printf 'CASE-a.md\ta_test.go\nCASE-b.md\ta_test.go\n' > "$self/case-targets-dup"

self_fail=0

# 期待どおりに真（違反なし）を返すか。
expect_judge_pass() {
  local label=$1
  shift
  if ! "$@" >/dev/null; then
    fault "自己検証: $label — 違反のないフィクスチャで偽を返した（判定が過剰）"
    self_fail=1
  fi
}

# 期待どおりに偽（違反あり）を返し、診断行を出すか。
expect_judge_fail() {
  local label=$1
  shift
  local diagnostics
  if diagnostics=$("$@"); then
    fault "自己検証: $label — 違反のあるフィクスチャで真を返した（判定が常に真になる退行）"
    self_fail=1
  elif [ -z "$diagnostics" ]; then
    fault "自己検証: $label — 偽を返したが診断行が空（失敗の原因が報告されない）"
    self_fail=1
  fi
}

expect_judge_pass "judge_forward" \
  judge_forward "$self/case-targets-ok" "$self/assets-ok"
expect_judge_fail "judge_forward（実在しない target）" \
  judge_forward "$self/case-targets-dangling" "$self/assets-ok"
# target が空の分岐は下の grep 分岐と重なる（空文字は assets にも無いので、この分岐が
# 死んでも違反自体は報告される。失われるのは「target が無い」という具体的な診断だけ）。
# よってこのアサーションは分岐の生死ではなく、空 target が違反として扱われることを固定する。
expect_judge_fail "judge_forward（target が空）" \
  judge_forward "$self/case-targets-empty" "$self/assets-ok"

# 逆方向は「covered に在る」「exclusions に在る」の 2 経路で違反を消す。両経路を
# それぞれ単独で確かめる（片方が死んでももう片方が拾って緑になるのを防ぐ）。
expect_judge_pass "judge_reverse（covered 経路）" \
  judge_reverse "$self/assets-ok" "$self/covered-ok" "$self/exclusions-empty"
printf 'c_test.go\n' > "$self/exclusions-c"
printf '\n' > "$self/covered-empty"
expect_judge_pass "judge_reverse（除外リスト経路）" \
  judge_reverse "$self/assets-c-only" "$self/covered-empty" "$self/exclusions-c"
expect_judge_fail "judge_reverse（どちらにも無い）" \
  judge_reverse "$self/assets-uncovered" "$self/covered-ok" "$self/exclusions-empty"

expect_judge_pass "judge_unique" \
  judge_unique "$self/case-targets-ok"
expect_judge_fail "judge_unique" \
  judge_unique "$self/case-targets-dup"

# 実データ側の入力の非空性。assets / covered が空だと §1〜§3 が空虚に真になる
# （exclusions は空でも §2 が単に厳しく判定するだけなので、ここには含めない）。
for name in assets covered case-targets; do
  if [ ! -s "$work/$name" ]; then
    fault "自己検証: $name が空（§1〜§3 が空虚に真になる）"
    self_fail=1
  fi
done

if [ "$self_fail" -eq 0 ]; then
  pass "自己検証: judge_* が合成フィクスチャで真偽の両側へ倒れる"
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
