#!/usr/bin/env bash
# sara-gap（グラフ未カバー 3 段の列挙コマンド）の検証。
#
# 実行:
#   nix develop ./dev -c dev/tests/sara-gap.sh   # devShell から直接
#   nix flake check ./dev                         # checks.sara-gap 経由
#
# 検証対象。番号は下の節見出しに対応する:
#   1.  ギャップのある fixture で 3 段（unthreatened / unmitigated / uncovered）を
#       検出して exit 1 を返す（張り先を持つ item は列挙しない）
#   2.  text 出力の行形式（<ID 前方8>\t<name>\t<file>、file は docs/ 前置）
#   3.  --json の 3 キーと行の形（ref / name / file）。exit code は text と同じ
#   4.  ギャップの無いグラフでは 3 セクション「なし」・空配列で exit 0
#   5.  sara check が失敗するグラフ（broken reference）では exit 2
#   6.  JSON 形状異常（.items 不在・valid: false）では exit 2（seam で決定論再現）
#   7.  走査先をリポジトリルート基準で解決する（SARA_GAP_ROOT 無し・サブディレクトリから）
#   8.  引数の異常系（--help = 0 / 未知の引数・引数過多 = 2）
#
# 担保できる範囲: 逆引きロジック（宣言辺 → to 集合 → 未カバー）と exit code 契約、
# 出力形式。fixture は実物の sara バイナリ + 実物の docs/model.yaml で検証する
# （モデルは fixture に写しを持たず、実行時に実物を重ねる — 二重管理の回避。型や
# 必須フィールドの変更で fixture item が実モデルに合わなくなれば、このテストが落ちて
# 追随を要求する）。sara の JSON 形状が変われば lock bump の PR でこのテストが落ちる
# （変わった「後」にしか現れない点で事前検知ではない）。
#
# 担保できない範囲: 実リポジトリ docs/ との整合（実グラフのどの item がギャップかは
# ここでは見ない。実グラフは fixture と違い件数が動き続けるため契約にできない）。
#
# -e は使わない。sara-id.sh と同じく「1 回の実行で全失敗を報告する」集計方式のため
# （-e があると最初の非ゼロ終了で以降のアサーションが走らない）。
set -uo pipefail

fail=0
pass() { printf 'ok   - %s\n' "$1"; }
fault() {
  printf 'FAIL - %s\n' "$1"
  fail=1
}

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# fixture はスクリプト自身からの相対で解決する（checks.sara-gap のサンドボックスは
# dev/tests/ の木を作ってから走らせる。devShell / CI 経路はリポジトリそのまま）。
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fixture="$script_dir/fixtures/sara-gap"

if [[ ! -d "$fixture/docs" ]]; then
  fault "fixture を解決できない（$fixture）"
  exit 1
fi

# モデルの正本は実リポジトリの docs/model.yaml（fixture は写しを持たない）。
# 解決は 2 経路: checks.sara-gap のサンドボックスは作業ツリーが無いので nix が
# SARA_GAP_MODEL_YAML で store path を渡す。devShell / CI 経路は git ルート基準。
# どちらでも解決できなければ skip せず失敗させる（黙って素通りさせない）。
model_yaml="${SARA_GAP_MODEL_YAML:-}"
contract_root="$(git rev-parse --show-toplevel 2>/dev/null || printf '.')"
[[ -f "$model_yaml" ]] || model_yaml="$contract_root/docs/model.yaml"
if [[ ! -f "$model_yaml" ]]; then
  fault "docs/model.yaml を解決できない（model.yaml=$model_yaml）"
  exit 1
fi

# fixture の複製に実物モデルを重ねて、実行可能な sara リポジトリの形を作る。
make_fixture() {
  mkdir -p "$1"
  cp -r "$fixture"/. "$1"
  cp "$model_yaml" "$1/model.yaml"
}

base="$work/base"
make_fixture "$base"

# fixture は 3 段のギャップを 1 件ずつ持つ:
#   unthreatened: REQ-bbbb0002（risk 無し要求）・DSG-dd000001（risk 無し設計）
#   unmitigated:  RISK-22220002（TC 無しリスク）
#   uncovered:    TC-44440002（CASE 無しテスト条件）
# 張り先を持つ側: REQ-aaaa0001 / RISK-11110001 / TC-33330001。

run_gap() {
  stdout_capture="$("$@" 2>/dev/null)" && status_capture=0 || status_capture=$?
}

# --- 1. 3 段の検出と exit 1 ---------------------------------------------------

run_gap env SARA_GAP_ROOT="$base" sara-gap
gap_out="$stdout_capture"

if [[ "$status_capture" -eq 1 ]]; then
  pass "ギャップがあれば exit 1"
else
  fault "ギャップがあれば exit 1（実際: exit=$status_capture）"
fi

for want in REQ-bbbb0002 DSG-dd000001 RISK-22220002 TC-44440002; do
  if grep -q "^$want" <<<"$gap_out"; then
    pass "$want を列挙する"
  else
    fault "$want を列挙する（出力に無い）"
  fi
done

for nowant in REQ-aaaa0001 RISK-11110001 TC-33330001 CASE-55550001; do
  if ! grep -q "$nowant" <<<"$gap_out"; then
    pass "張り先を持つ $nowant を列挙しない"
  else
    fault "張り先を持つ $nowant を列挙しない（出力に出ている）"
  fi
done

# 3 セクションの見出しが揃っている（片方の段の検出が黙って消える退行を防ぐ）。
if [[ "$(grep -c '^## ' <<<"$gap_out")" -eq 3 ]]; then
  pass "セクション見出しを 3 つ出す"
else
  fault "セクション見出しを 3 つ出す（実際: $(grep -c '^## ' <<<"$gap_out") 個）"
fi

# --- 2. text の行形式 ----------------------------------------------------------

req_line="$(grep '^REQ-bbbb0002' <<<"$gap_out")"
if [[ "$req_line" == REQ-bbbb0002$'\t'*$'\t'docs/*.md ]]; then
  pass "行が <ID 前方8>\\t<name>\\t<docs/ 前置の file> の形"
else
  fault "行が <ID 前方8>\\t<name>\\t<docs/ 前置の file> の形（実際: $req_line）"
fi

# --- 3. --json -----------------------------------------------------------------

run_gap env SARA_GAP_ROOT="$base" sara-gap --json
json_out="$stdout_capture"

if [[ "$status_capture" -eq 1 ]]; then
  pass "--json でも exit code は同じ（ギャップあり = 1）"
else
  fault "--json でも exit code は同じ（実際: exit=$status_capture）"
fi

if [[ "$(jq -r 'keys | join(",")' <<<"$json_out" 2>/dev/null)" == "uncovered,unmitigated,unthreatened" ]]; then
  pass "--json は 3 キーのオブジェクトを返す"
else
  fault "--json は 3 キーのオブジェクトを返す（実際: $(jq -r 'keys | join(",")' <<<"$json_out" 2>/dev/null)）"
fi

counts="$(jq -r '[.unthreatened, .unmitigated, .uncovered | length] | join(",")' <<<"$json_out" 2>/dev/null)"
if [[ "$counts" == "2,1,1" ]]; then
  pass "--json の件数が 3 段で 2,1,1"
else
  fault "--json の件数が 3 段で 2,1,1（実際: $counts）"
fi

row_ok="$(jq -r '.unmitigated[0] | (.ref == "RISK-22220002") and (.name != "") and (.file | startswith("docs/"))' <<<"$json_out" 2>/dev/null)"
if [[ "$row_ok" == "true" ]]; then
  pass "--json の行が ref / name / file（docs/ 前置）を持つ"
else
  fault "--json の行が ref / name / file（docs/ 前置）を持つ（実際: $(jq -c '.unmitigated[0]' <<<"$json_out" 2>/dev/null)）"
fi

# --- 4. ギャップなしのグラフでは exit 0 ----------------------------------------
#
# ギャップ側の item（と、それにぶら下がるものが無い item）を除いた複製を作る。
# 残るのは SOL → UC → REQ-aaaa0001 ← RISK-11110001 ← TC-33330001 ← CASE-55550001 の
# 一本鎖で、3 段とも未カバーが無い。

clean="$work/clean"
make_fixture "$clean"
rm "$clean/docs/req-gap.md" "$clean/docs/dsg-gap.md" "$clean/docs/risk-gap.md" "$clean/docs/tc-gap.md"

run_gap env SARA_GAP_ROOT="$clean" sara-gap
clean_out="$stdout_capture"
if [[ "$status_capture" -eq 0 ]]; then
  pass "ギャップが無ければ exit 0"
else
  fault "ギャップが無ければ exit 0（実際: exit=$status_capture 出力: $clean_out）"
fi

if [[ "$(grep -c '^なし$' <<<"$clean_out")" -eq 3 ]]; then
  pass "0 件のセクションは「なし」と出す"
else
  fault "0 件のセクションは「なし」と出す（実際: $(grep -c '^なし$' <<<"$clean_out") 個）"
fi

run_gap env SARA_GAP_ROOT="$clean" sara-gap --json
if [[ "$status_capture" -eq 0 && "$(jq -r '[.[] | length] | add' <<<"$stdout_capture" 2>/dev/null)" == "0" ]]; then
  pass "--json でも空配列 3 本で exit 0"
else
  fault "--json でも空配列 3 本で exit 0（exit=$status_capture 出力: $stdout_capture）"
fi

# --- 5. sara check が失敗するグラフでは exit 2 ---------------------------------
#
# broken reference（実在しない risk への mitigates）を混ぜる。壊れたグラフから
# ギャップ一覧を出さない契約（部分的な一覧を信用して工程を進める事故の防止）。

broken="$work/broken"
make_fixture "$broken"
cat >"$broken/docs/broken.md" <<'MD'
---
id: "TC-99990009-0000-4000-8000-00000000000b"
type: test_condition
name: "壊れた参照を持つテスト条件"
mitigates:
  - "RISK-deadbeef-0000-4000-8000-00000000000c"
---
# TC-99990009: 壊れた参照を持つテスト条件
MD

run_gap env SARA_GAP_ROOT="$broken" sara-gap
if [[ "$status_capture" -eq 2 ]]; then
  pass "sara check が失敗するグラフでは exit 2"
else
  fault "sara check が失敗するグラフでは exit 2（実際: exit=$status_capture）"
fi
if [[ -z "$stdout_capture" ]]; then
  pass "sara check 失敗時はギャップ一覧を stdout に出さない"
else
  fault "sara check 失敗時はギャップ一覧を stdout に出さない（実際: $stdout_capture）"
fi

# --- 6. JSON 形状異常では exit 2（seam）----------------------------------------
#
# sara のバージョン更新で JSON 形状が変わったとき、黙って「ギャップなし」を返す
# 事故を防ぐガード。実物の sara に形状を変えさせることはできないので、
# SARA_GAP_SARA seam で偽 sara を差し込んで決定論的に検証する。
# shebang は実行中の bash の絶対パスを埋め込む（sara-id.sh と同じ理由:
# nix のビルドサンドボックスに /usr/bin/env が無い）。

fake_noitems="$work/fake-sara-noitems"
{
  printf '#!%s\n' "$BASH"
  printf 'printf %%s "{}"\n'
} >"$fake_noitems"
chmod +x "$fake_noitems"

run_gap env SARA_GAP_ROOT="$base" SARA_GAP_SARA="$fake_noitems" sara-gap
if [[ "$status_capture" -eq 2 ]]; then
  pass ".items 配列が無い JSON では exit 2"
else
  fault ".items 配列が無い JSON では exit 2（実際: exit=$status_capture）"
fi

fake_invalid="$work/fake-sara-invalid"
{
  printf '#!%s\n' "$BASH"
  printf "printf %%s '{\"items\": [], \"valid\": false}'\n"
} >"$fake_invalid"
chmod +x "$fake_invalid"

run_gap env SARA_GAP_ROOT="$base" SARA_GAP_SARA="$fake_invalid" sara-gap
if [[ "$status_capture" -eq 2 ]]; then
  pass "exit 0 でも valid: false の JSON では exit 2"
else
  fault "exit 0 でも valid: false の JSON では exit 2（実際: exit=$status_capture）"
fi

# --- 7. 走査先をリポジトリルート基準で解決する ---------------------------------
#
# SARA_GAP_ROOT 無しの通常経路。サブディレクトリから叩いても git ルートの
# sara.toml / docs を対象にすることを固定する（sara-id の 4b と同じ論点）。

gitrepo="$work/gitrepo"
make_fixture "$gitrepo"
git -C "$gitrepo" init -q
mkdir -p "$gitrepo/sub"
sub_status=0
(cd "$gitrepo/sub" && sara-gap >/dev/null 2>&1) || sub_status=$?
if [[ "$sub_status" -eq 1 ]]; then
  pass "サブディレクトリから叩いても git ルートのグラフを対象にする（ギャップ検出 = exit 1）"
else
  fault "サブディレクトリから叩いても git ルートのグラフを対象にする（実際: exit=$sub_status）"
fi

# --- 8. 引数の異常系 ------------------------------------------------------------

run_gap sara-gap --help
if [[ "$status_capture" -eq 0 ]]; then
  pass "--help は exit 0"
else
  fault "--help は exit 0（実際: exit=$status_capture）"
fi

run_gap sara-gap --bogus
if [[ "$status_capture" -eq 2 ]]; then
  pass "未知の引数は exit 2"
else
  fault "未知の引数は exit 2（実際: exit=$status_capture）"
fi

run_gap sara-gap --json extra
if [[ "$status_capture" -eq 2 ]]; then
  pass "引数過多は exit 2"
else
  fault "引数過多は exit 2（実際: exit=$status_capture）"
fi

exit "$fail"
