#!/usr/bin/env bash
# sara-id（UUIDv4 二層 ID の採番コマンド）の検証。
#
# 実行:
#   nix develop ./dev -c dev/tests/sara-id.sh   # devShell から直接
#   nix flake check ./dev                        # checks.sara-id 経由
#
# 検証対象（→ Issue #207・ADR-0048）:
#   1. 正式 ID（<PREFIX>-<フル UUIDv4>）とファイル名素材（<YYYYMMDD>-<前方8文字>-<slug>.md）を出力する
#   2. UUIDv4 として妥当（version nibble = 4・variant nibble ∈ {8,9,a,b}）
#   3. 呼ぶたびに異なる ID を返す
#   4. 8 文字 prefix が docs/ に既出なら再生成する（上限 16 回で打ち切る）
#   5. 型名から prefix を引き、未知の入力はフォールバックで大文字化する
#   6. slug 未指定でもファイル名素材を出す
#   7. ADR は連番維持のため exit 2 で拒否する
#   8. 異常系の終了コードを区別する（引数不正 = 2 / 候補枯渇 = 1）

# -e は使わない。SUT（sara-id）の非ゼロ終了はこのテストが検知したい退行そのもので、
# -e があるとその時点でスクリプトが打ち切られ、以降のアサーションが走らず
# 「1 回の実行で全失敗を報告する」という集計設計が壊れる（tests/e2e/run.sh も同じ理由で
# -e を採らない）。SUT 呼び出しは全て `|| true` 相当で受けて fault に流す。
set -uo pipefail

fail=0
pass() { printf 'ok   - %s\n' "$1"; }
fault() {
  printf 'FAIL - %s\n' "$1"
  fail=1
}

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# SUT を呼び、stdout を stdout_capture に、終了コードを status_capture に入れる。
# 非ゼロ終了でもここでは落とさない（判定は各アサーションが行う）。
stdout_capture=""
status_capture=0
run_sara_id() {
  stdout_capture="$(cd "$work" && "$@" 2>/dev/null)" && status_capture=0 || status_capture=$?
}

field() { printf '%s\n' "$stdout_capture" | sed -n "s/^$1:[[:space:]]*//p"; }

# --- 1. 出力形式 -------------------------------------------------------------

mkdir -p "$work/docs"
run_sara_id sara-id req lock-ordering

formal="$(field id)"
filename="$(field filename)"
shortref="$(field ref)"

if [[ "$status_capture" -eq 0 && -n "$formal" && -n "$filename" && -n "$shortref" ]]; then
  pass "id / filename / ref の 3 行を出力する"
else
  fault "id / filename / ref の 3 行を出力する（exit=$status_capture 実際: $stdout_capture）"
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

run_sara_id sara-id req x
a="$(field id)"
run_sara_id sara-id req x
b="$(field id)"
if [[ -n "$a" && "$a" != "$b" ]]; then
  pass "呼ぶたびに異なる ID を返す"
else
  fault "呼ぶたびに異なる ID を返す（a=$a b=$b）"
fi

# --- 3. 8 文字 prefix の重複時に再生成する -----------------------------------
#
# 乱数のままだと「重複チェックを持たない実装」でも偶然通ってしまうため、
# UUID 生成を SARA_ID_UUIDGEN（seam）で差し替えて決定論的に検証する。

taken='aaaaaaaa-1111-4111-8111-111111111111'
fresh='bbbbbbbb-2222-4222-9222-222222222222'

# 呼ばれた回数を数え、n 回目までは既出・それ以降は未出の UUID を返す偽 uuidgen。
# SARA_ID_FAKE_SWITCH で切り替え点を、SARA_ID_FAKE_COUNT で計数ファイルを受ける。
# ヒアドキュメントはクオートして内側を一切展開させず、可変値は環境変数で渡す
# （エスケープ漏れ 1 個で偽 uuidgen が壊れるのを避ける）。
fake="$work/fake-uuidgen"
cat >"$fake" <<'FAKE'
#!/usr/bin/env bash
n=$(cat "$SARA_ID_FAKE_COUNT" 2>/dev/null || echo 0)
n=$((n + 1))
printf '%s' "$n" >"$SARA_ID_FAKE_COUNT"
if [ "$n" -le "$SARA_ID_FAKE_SWITCH" ]; then
  printf '%s\n' "$SARA_ID_FAKE_TAKEN"
else
  printf '%s\n' "$SARA_ID_FAKE_FRESH"
fi
FAKE
chmod +x "$fake"

count_file="$work/fake-count"
export SARA_ID_UUIDGEN="$fake"
export SARA_ID_FAKE_COUNT="$count_file"
export SARA_ID_FAKE_TAKEN="$taken"
export SARA_ID_FAKE_FRESH="$fresh"

calls_made() {
  local n
  n="$(cat "$count_file" 2>/dev/null)"
  printf '%s' "${n:-0}"
}

# 1 回目だけ既出を返す設定。
export SARA_ID_FAKE_SWITCH=1
printf 'REQ-%s\n' "$taken" >"$work/docs/existing.md"
: >"$count_file"
run_sara_id sara-id req x
got="$(field id)"
calls="$(calls_made)"

if [[ "$got" == "REQ-$fresh" ]]; then
  pass "docs/ に既出の 8 文字 prefix なら再生成した ID を返す"
else
  fault "docs/ に既出の 8 文字 prefix なら再生成した ID を返す（実際: $got）"
fi

if [[ "$calls" -eq 2 ]]; then
  pass "重複検出時に uuidgen をちょうど 1 回呼び直す"
else
  fault "重複検出時に uuidgen をちょうど 1 回呼び直す（呼び出し回数 $calls）"
fi

# 未出なら 1 回で確定する（無駄な再生成をしない）。
printf 'nothing here\n' >"$work/docs/existing.md"
: >"$count_file"
run_sara_id sara-id req x
got2="$(field id)"
calls2="$(calls_made)"
if [[ "$got2" == "REQ-$taken" && "$calls2" -eq 1 ]]; then
  pass "未出なら 1 回で確定する"
else
  fault "未出なら 1 回で確定する（id=$got2 calls=$calls2）"
fi

# 走査は docs/ 配下を再帰する（サブディレクトリの既出も拾う）。
mkdir -p "$work/docs/requirements"
printf 'REQ-%s\n' "$taken" >"$work/docs/requirements/nested.md"
printf 'nothing here\n' >"$work/docs/existing.md"
: >"$count_file"
run_sara_id sara-id req x
got3="$(field id)"
if [[ "$got3" == "REQ-$fresh" ]]; then
  pass "docs/ を再帰走査してサブディレクトリの既出も検出する"
else
  fault "docs/ を再帰走査してサブディレクトリの既出も検出する（実際: $got3）"
fi
rm -f "$work/docs/requirements/nested.md"
rmdir "$work/docs/requirements" 2>/dev/null || true

# --- 4. 再生成ループの上限（16 回で打ち切り exit 1）--------------------------
#
# 乱数源が壊れて同じ値を返し続けるケース。黙って重複 ID を返さず失敗することを固定する。

export SARA_ID_FAKE_SWITCH=9999 # 常に既出を返す
printf 'REQ-%s\n' "$taken" >"$work/docs/existing.md"
: >"$count_file"
run_sara_id sara-id req x
exhausted_status="$status_capture"
exhausted_out="$stdout_capture"
exhausted_calls="$(calls_made)"

if [[ "$exhausted_status" -eq 1 ]]; then
  pass "候補が枯渇したら exit 1 で失敗する"
else
  fault "候補が枯渇したら exit 1 で失敗する（実際: exit=$exhausted_status）"
fi

if [[ -z "$exhausted_out" ]]; then
  pass "候補枯渇時は重複 ID を stdout に出さない"
else
  fault "候補枯渇時は重複 ID を stdout に出さない（実際: $exhausted_out）"
fi

if [[ "$exhausted_calls" -eq 16 ]]; then
  pass "打ち切りまでにちょうど 16 回試行する"
else
  fault "打ち切りまでにちょうど 16 回試行する（実際: $exhausted_calls）"
fi

unset SARA_ID_UUIDGEN SARA_ID_FAKE_COUNT SARA_ID_FAKE_TAKEN SARA_ID_FAKE_FRESH SARA_ID_FAKE_SWITCH
rm -f "$work/docs/existing.md"

# --- 4b. 走査先をリポジトリルート基準で解決する ------------------------------
#
# カレント相対だと dev/ 等のサブディレクトリから叩いたときに docs/ を見つけられず、
# 重複チェックが黙って外れる。git 管理下ではルート基準で解決することを固定する。

gitrepo="$(mktemp -d)"
git -C "$gitrepo" init -q
mkdir -p "$gitrepo/docs" "$gitrepo/sub"
printf 'REQ-%s\n' "$taken" >"$gitrepo/docs/existing.md"
: >"$count_file"
sub_out="$(cd "$gitrepo/sub" &&
  SARA_ID_UUIDGEN="$fake" SARA_ID_FAKE_COUNT="$count_file" \
    SARA_ID_FAKE_TAKEN="$taken" SARA_ID_FAKE_FRESH="$fresh" SARA_ID_FAKE_SWITCH=1 \
    sara-id req x 2>/dev/null)"
sub_id="$(printf '%s\n' "$sub_out" | sed -n 's/^id:[[:space:]]*//p')"
rm -rf "$gitrepo"
if [[ "$sub_id" == "REQ-$fresh" ]]; then
  pass "サブディレクトリから叩いてもルートの docs/ を走査する"
else
  fault "サブディレクトリから叩いてもルートの docs/ を走査する（実際: $sub_id）"
fi

# --- 5. docs/ が無いリポジトリでも動く ---------------------------------------

nodocs="$(mktemp -d)"
nodocs_out="$(cd "$nodocs" && sara-id req x 2>/dev/null)" && nodocs_status=0 || nodocs_status=$?
rm -rf "$nodocs"
nodocs_id="$(printf '%s\n' "$nodocs_out" | sed -n 's/^id:[[:space:]]*//p')"
if [[ "$nodocs_status" -eq 0 && "$nodocs_id" =~ ^REQ-${uuid_re}$ ]]; then
  pass "docs/ が無いディレクトリでも採番できる"
else
  fault "docs/ が無いディレクトリでも採番できる（exit=$nodocs_status id=$nodocs_id）"
fi

# --- 6. prefix の解決 --------------------------------------------------------

run_sara_id sara-id test_condition probe
lower="$(field id)"
if [[ "$lower" =~ ^TC- ]]; then
  pass "型名から prefix を引いて大文字で使う（test_condition → TC）"
else
  fault "型名から prefix を引いて大文字で使う（実際: $lower）"
fi

run_sara_id sara-id inf probe
alias_id="$(field id)"
if [[ "$alias_id" =~ ^INF- ]]; then
  pass "短縮別名から prefix を引く（inf → INF）"
else
  fault "短縮別名から prefix を引く（実際: $alias_id）"
fi

# 未知の入力は prefix そのものとみなして大文字化する（フォールバック経路）。
run_sara_id sara-id xyz probe
fallback_id="$(field id)"
if [[ "$fallback_id" =~ ^XYZ- ]]; then
  pass "未知の入力は prefix とみなして大文字化する"
else
  fault "未知の入力は prefix とみなして大文字化する（実際: $fallback_id）"
fi

# --- 7. slug 省略 ------------------------------------------------------------

run_sara_id sara-id design
noslug="$(field filename)"
if [[ "$noslug" =~ ^[0-9]{8}-[0-9a-f]{8}\.md$ ]]; then
  pass "slug 未指定でもファイル名素材を出す"
else
  fault "slug 未指定でもファイル名素材を出す（実際: $noslug）"
fi

# --- 8. ADR は連番維持のため拒否する -----------------------------------------
#
# ADR-0048 §4「ADR のみ連番を維持する」に対応する分岐。UUID を振ると
# 既存 48 本の相互参照・docs/adr/README.md の運用が壊れる。

run_sara_id sara-id adr
adr_status="$status_capture"
adr_out="$stdout_capture"
if [[ "$adr_status" -eq 2 ]]; then
  pass "ADR は exit 2 で拒否する（連番維持）"
else
  fault "ADR は exit 2 で拒否する（実際: exit=$adr_status）"
fi
if [[ -z "$adr_out" ]]; then
  pass "ADR 拒否時は ID を出力しない"
else
  fault "ADR 拒否時は ID を出力しない（実際: $adr_out）"
fi

# --- 9. 引数の異常系（終了コードを区別する）----------------------------------

run_sara_id sara-id
if [[ "$status_capture" -eq 2 ]]; then
  pass "引数なしは exit 2"
else
  fault "引数なしは exit 2（実際: exit=$status_capture）"
fi

run_sara_id sara-id req slug extra
if [[ "$status_capture" -eq 2 ]]; then
  pass "引数過多は exit 2"
else
  fault "引数過多は exit 2（実際: exit=$status_capture）"
fi

run_sara_id sara-id --help
if [[ "$status_capture" -eq 0 ]]; then
  pass "--help は exit 0"
else
  fault "--help は exit 0（実際: exit=$status_capture）"
fi

exit "$fail"
