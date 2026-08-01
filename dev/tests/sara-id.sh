#!/usr/bin/env bash
# sara-id（UUIDv4 二層 ID の採番コマンド）の検証。
#
# devShell から実行する:
#   nix develop ./dev -c dev/tests/sara-id.sh
#
# 検証対象（→ Issue #207・ADR-0048）:
#   1. 正式 ID（<PREFIX>-<フル UUIDv4>）とファイル名素材（<YYYYMMDD>-<前方8文字>-<slug>.md）を出力する
#   2. UUIDv4 として妥当（version nibble = 4・variant nibble ∈ {8,9,a,b}）
#   3. 呼ぶたびに異なる ID を返す
#   4. 8 文字 prefix が docs/ に既出なら再生成する
#   5. prefix 引数を大文字化して正式 ID・散文参照に使う
#   6. slug 未指定でもファイル名素材を出す

set -euo pipefail

fail=0
pass() { printf 'ok   - %s\n' "$1"; }
fault() {
  printf 'FAIL - %s\n' "$1"
  fail=1
}

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# --- 1. 出力形式 -------------------------------------------------------------

out="$(cd "$work" && mkdir -p docs && sara-id req lock-ordering)"

formal="$(printf '%s\n' "$out" | sed -n 's/^id:[[:space:]]*//p')"
filename="$(printf '%s\n' "$out" | sed -n 's/^filename:[[:space:]]*//p')"
shortref="$(printf '%s\n' "$out" | sed -n 's/^ref:[[:space:]]*//p')"

if [[ -n "$formal" && -n "$filename" && -n "$shortref" ]]; then
  pass "id / filename / ref の 3 行を出力する"
else
  fault "id / filename / ref の 3 行を出力する（実際: $out）"
fi

uuid_re='[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}'

if [[ "$formal" =~ ^REQ-${uuid_re}$ ]]; then
  pass "正式 ID が <PREFIX>-<フル UUIDv4> 形式（version 4 / variant 準拠）"
else
  fault "正式 ID が <PREFIX>-<フル UUIDv4> 形式（実際: $formal）"
fi

if [[ "$filename" =~ ^[0-9]{8}-[0-9a-f]{8}-lock-ordering\.md$ ]]; then
  pass "ファイル名素材が <YYYYMMDD>-<前方8文字>-<slug>.md 形式"
else
  fault "ファイル名素材が <YYYYMMDD>-<前方8文字>-<slug>.md 形式（実際: $filename）"
fi

if [[ "$shortref" =~ ^REQ-[0-9a-f]{8}$ ]]; then
  pass "散文参照が <PREFIX>-<前方8文字> 形式"
else
  fault "散文参照が <PREFIX>-<前方8文字> 形式（実際: $shortref）"
fi

# 3 つの出力の 8 文字が一致する（省略形は正式 ID の前方一致でなければ grep が破綻する）。
uuid8="${formal#REQ-}"
uuid8="${uuid8:0:8}"
if [[ "$shortref" == "REQ-$uuid8" && "$filename" == *"-$uuid8-"* ]]; then
  pass "省略形が正式 ID の前方 8 文字と一致する"
else
  fault "省略形が正式 ID の前方 8 文字と一致する（uuid8=$uuid8 ref=$shortref filename=$filename）"
fi

# --- 2. 一意性 ---------------------------------------------------------------

a="$(cd "$work" && sara-id req x | sed -n 's/^id:[[:space:]]*//p')"
b="$(cd "$work" && sara-id req x | sed -n 's/^id:[[:space:]]*//p')"
if [[ "$a" != "$b" ]]; then
  pass "呼ぶたびに異なる ID を返す"
else
  fault "呼ぶたびに異なる ID を返す（両方 $a）"
fi

# --- 3. 8 文字 prefix の重複時に再生成する -----------------------------------
#
# 乱数のままだと「重複チェックを持たない実装」でも偶然通ってしまうため、
# UUID 生成を SARA_ID_UUIDGEN（seam）で差し替えて決定論的に検証する。
# 1 回目は docs/ に既出の 8 文字、2 回目は未出のものを返す偽 uuidgen を注入し、
# コマンドが 1 回目を捨てて 2 回目を採用することを見る。

taken='aaaaaaaa-1111-4111-8111-111111111111'
fresh='bbbbbbbb-2222-4222-9222-222222222222'
printf 'REQ-%s\n' "$taken" >"$work/docs/existing.md"

fake="$work/fake-uuidgen"
cat >"$fake" <<EOF
#!/usr/bin/env bash
# 呼ばれた回数を数え、1 回目は既出・2 回目以降は未出の UUID を返す。
n_file="\$SARA_ID_FAKE_COUNT"
n=\$(cat "\$n_file" 2>/dev/null || echo 0)
n=\$((n + 1))
printf '%s' "\$n" >"\$n_file"
if [[ "\$n" -eq 1 ]]; then
  printf '%s\n' '$taken'
else
  printf '%s\n' '$fresh'
fi
EOF
chmod +x "$fake"

count_file="$work/fake-count"
: >"$count_file"
got="$(cd "$work" && SARA_ID_UUIDGEN="$fake" SARA_ID_FAKE_COUNT="$count_file" sara-id req x |
  sed -n 's/^id:[[:space:]]*//p')"
calls="$(cat "$count_file")"

if [[ "$got" == "REQ-$fresh" ]]; then
  pass "docs/ に既出の 8 文字 prefix なら再生成した ID を返す"
else
  fault "docs/ に既出の 8 文字 prefix なら再生成した ID を返す（実際: $got）"
fi

if [[ "$calls" -ge 2 ]]; then
  pass "重複検出時に uuidgen を呼び直す（呼び出し回数 $calls）"
else
  fault "重複検出時に uuidgen を呼び直す（呼び出し回数 $calls）"
fi

# 未出なら 1 回で確定する（無駄な再生成をしない）。
: >"$count_file"
printf 'nothing here\n' >"$work/docs/existing.md"
got2="$(cd "$work" && SARA_ID_UUIDGEN="$fake" SARA_ID_FAKE_COUNT="$count_file" sara-id req x |
  sed -n 's/^id:[[:space:]]*//p')"
calls2="$(cat "$count_file")"
if [[ "$got2" == "REQ-$taken" && "$calls2" -eq 1 ]]; then
  pass "未出なら 1 回で確定する"
else
  fault "未出なら 1 回で確定する（id=$got2 calls=$calls2）"
fi

# 走査は docs/ 配下を再帰する（サブディレクトリの既出も拾う）。
: >"$count_file"
mkdir -p "$work/docs/requirements"
printf 'REQ-%s\n' "$taken" >"$work/docs/requirements/nested.md"
printf 'nothing here\n' >"$work/docs/existing.md"
got3="$(cd "$work" && SARA_ID_UUIDGEN="$fake" SARA_ID_FAKE_COUNT="$count_file" sara-id req x |
  sed -n 's/^id:[[:space:]]*//p')"
if [[ "$got3" == "REQ-$fresh" ]]; then
  pass "docs/ を再帰走査してサブディレクトリの既出も検出する"
else
  fault "docs/ を再帰走査してサブディレクトリの既出も検出する（実際: $got3）"
fi
rm -f "$work/docs/requirements/nested.md" "$work/docs/existing.md"

# --- 4. prefix の大文字化 ----------------------------------------------------

lower="$(cd "$work" && sara-id test_condition probe | sed -n 's/^id:[[:space:]]*//p')"
if [[ "$lower" =~ ^TC- ]]; then
  pass "型名から prefix を引いて大文字で使う（test_condition → TC）"
else
  fault "型名から prefix を引いて大文字で使う（実際: $lower）"
fi

# --- 5. slug 省略 ------------------------------------------------------------

noslug="$(cd "$work" && sara-id design | sed -n 's/^filename:[[:space:]]*//p')"
if [[ "$noslug" =~ ^[0-9]{8}-[0-9a-f]{8}\.md$ ]]; then
  pass "slug 未指定でもファイル名素材を出す"
else
  fault "slug 未指定でもファイル名素材を出す（実際: $noslug）"
fi

# --- 6. 引数なしは使い方を出して失敗する -------------------------------------

if (cd "$work" && sara-id >/dev/null 2>&1); then
  fault "引数なしは非ゼロで終了する"
else
  pass "引数なしは非ゼロで終了する"
fi

exit "$fail"
