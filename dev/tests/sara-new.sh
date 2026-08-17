#!/usr/bin/env bash
# sara-new（sara init を包む item 起票ラッパー）の検証。
#
# 実行:
#   nix develop ./dev -c dev/tests/sara-new.sh   # devShell から直接
#   nix flake check ./dev                         # checks.sara-new 経由
#
# 検証対象（→ Issue #367・epic #364）。番号は下の節見出しに対応する:
#   1.  item を起票し `<YYYYMMDD>-<フル UUID>-<slug>.md` へ rename する。
#       frontmatter の id と、ファイル名の UUID 部が一致する
#   2.  採番 ID とファイルパスを機械可読な 2 行（id: / file:）で出力する
#   3.  sara init へオプションを透過する（-- 以降）
#   4.  配置ディレクトリを作る（無ければ mkdir -p）。型名はアンダースコア表記
#       （model.yaml・規約文書）とハイフン表記（sara init のサブコマンド名）の両方を受ける
#   5.  slug の検査（空・不正文字は exit 2 で、ファイルを残さない）
#   6.  ADR は連番維持のため exit 2 で拒否する
#   7.  sara init が失敗したら非ゼロで落ち、一時ファイルを残さない（seam で再現）
#   8.  sara init の出力から ID を読めなければ失敗し、一時ファイルを残さない（seam）
#   9.  出力先が既存なら上書きせず exit 1（起票済み item を潰さない）
#   10. 引数の異常系（引数不足 = 2 / --help = 0）
#
# 担保できる範囲: ラッパーの責務（パス組み立て・rename・ID 読み取り・異常系の後片付け）。
# UUID の採番そのもの・8 文字 prefix の重複可否は検証しない（sara init の領分であり、
# ラッパーは採番を持たない設計 → Issue #367）。
#
# fixture は実物の docs/model.yaml を重ねて作る（sara-gap.sh と同じく写しを持たない。
# 型や必須フィールドが変わって fixture が実モデルに合わなくなれば、このテストが落ちて
# 追随を要求する）。
#
# -e は使わない。sara-id.sh / sara-gap.sh と同じく「1 回の実行で全失敗を報告する」
# 集計方式のため（-e があると最初の非ゼロ終了で以降のアサーションが走らない）。
set -uo pipefail

fail=0
pass() { printf 'ok   - %s\n' "$1"; }
fault() {
  printf 'FAIL - %s\n' "$1"
  fail=1
}

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# モデルの正本は実リポジトリの docs/model.yaml。解決は 2 経路: checks.sara-new の
# サンドボックスは作業ツリーが無いので nix が SARA_NEW_MODEL_YAML で store path を
# 渡す。devShell / CI 経路は git ルート基準。どちらでも解決できなければ skip せず
# 失敗させる（黙って素通りさせない）。
model_yaml="${SARA_NEW_MODEL_YAML:-}"
contract_root="$(git rev-parse --show-toplevel 2>/dev/null || printf '.')"
[[ -f "$model_yaml" ]] || model_yaml="$contract_root/docs/model.yaml"
if [[ ! -f "$model_yaml" ]]; then
  fault "docs/model.yaml を解決できない（model.yaml=$model_yaml）"
  exit 1
fi

# sara が動くリポジトリの最小形（sara.toml + model.yaml + 空の docs/）を作る。
# 各節が互いの残骸に依存しないよう、節ごとに作り直せる関数にしておく。
make_repo() {
  local root="$1"
  mkdir -p "$root/docs"
  cp "$model_yaml" "$root/docs/model.yaml"
  cat >"$root/sara.toml" <<'EOF'
model_schema = "docs/model.yaml"

[repositories]
paths = ["./docs"]
EOF
}

uuid_re='[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}'

# SUT を repo のルートで呼ぶ（sara.toml を既定パスで引かせるため）。
# 非ゼロ終了でもここでは落とさない（判定は各アサーションが行う）。
stdout_capture=""
status_capture=0
run_sara_new() {
  local root="$1"
  shift
  stdout_capture="$(cd "$root" && "$@" 2>/dev/null)" && status_capture=0 || status_capture=$?
}

field() { printf '%s\n' "$stdout_capture" | sed -n "s/^$1:[[:space:]]*//p"; }

# --- 1. 起票と rename ---------------------------------------------------------

repo="$work/basic"
make_repo "$repo"
run_sara_new "$repo" sara-new requirement lock-ordering docs/requirements

created="$(field file)"
created_id="$(field id)"

if [[ "$status_capture" -eq 0 ]]; then
  pass "起票に成功して exit 0"
else
  fault "起票に成功して exit 0（実際: exit=$status_capture 出力: $stdout_capture）"
fi

if [[ "$created_id" =~ ^REQ-${uuid_re}$ ]]; then
  pass "採番 ID が <PREFIX>-<フル UUIDv4> 形式"
else
  fault "採番 ID が <PREFIX>-<フル UUIDv4> 形式（実際: $created_id）"
fi

uuid="${created_id#REQ-}"
today="$(date +%Y%m%d)"
want_name="$today-$uuid-lock-ordering.md"

if [[ "$created" == "docs/requirements/$want_name" ]]; then
  pass "ファイル名が <YYYYMMDD>-<フル UUID>-<slug>.md で、出力の file: と一致する"
else
  fault "ファイル名が <YYYYMMDD>-<フル UUID>-<slug>.md（期待: docs/requirements/$want_name 実際: $created）"
fi

if [[ -f "$repo/docs/requirements/$want_name" ]]; then
  pass "その名前のファイルが実在する"
else
  fault "その名前のファイルが実在する（$repo/docs/requirements/$want_name が無い）"
fi

# frontmatter の id と、ファイル名に埋めた UUID が一致する（rename 先の組み立てに
# 別の UUID を混ぜていないことを固定する）。
if grep -q "id: \"$created_id\"" "$repo/docs/requirements/$want_name" 2>/dev/null; then
  pass "frontmatter の id とファイル名の UUID が同じ item を指す"
else
  fault "frontmatter の id とファイル名の UUID が同じ item を指す（id=$created_id）"
fi

# 一時ファイル名が残っていない（rename であって copy ではない）。
leftovers="$(find "$repo/docs/requirements" -name '*.md' -type f | wc -l)"
if [[ "$leftovers" -eq 1 ]]; then
  pass "生成物は 1 ファイルだけ（一時ファイルを残さない）"
else
  fault "生成物は 1 ファイルだけ（実際: $leftovers 件）"
fi

# --- 2. 出力形式 --------------------------------------------------------------
#
# 呼び出し側（人・エージェント）が採番結果を機械的に拾えることを固定する。
# sara init の装飾付き出力をそのまま流すと、後続の自動処理が壊れる。

if [[ "$(printf '%s\n' "$stdout_capture" | grep -c '^id: \|^file: ')" -eq 2 ]]; then
  pass "id: / file: の 2 行を出力する"
else
  fault "id: / file: の 2 行を出力する（実際: $stdout_capture）"
fi

# --- 3. sara init へのオプション透過 ------------------------------------------
#
# ラッパーが sara init のオプション面を塞ぐと、起票のたびに手で frontmatter を
# 埋め直すことになる（--name / --specification 等）。`--` 以降を透過する。

repo3="$work/passthru"
make_repo "$repo3"
run_sara_new "$repo3" sara-new requirement passthru docs/requirements -- --name "透過テスト"
passthru_file="$(field file)"
if [[ "$status_capture" -eq 0 ]] && grep -q 'name: "透過テスト"' "$repo3/$passthru_file" 2>/dev/null; then
  pass "-- 以降のオプションを sara init へ透過する"
else
  fault "-- 以降のオプションを sara init へ透過する（exit=$status_capture file=$passthru_file）"
fi

# --- 4. 配置ディレクトリの自動作成と型名の表記ゆれ ----------------------------
#
# 新しい区分（docs/test/<対象>/ 等）を起こすとき、mkdir を別途踏ませない。
# あわせて型名のアンダースコア表記（model.yaml・規約文書の書き方）を受けることを
# 固定する。sara init のサブコマンドはハイフン（test-case）なので、素通しにすると
# 規約文書どおりに叩いた呼び出しが unrecognized subcommand で落ちる。

repo4="$work/mkdir"
make_repo "$repo4"
run_sara_new "$repo4" sara-new test_case first-case docs/test/new-area
mkdir_file="$(field file)"
if [[ "$status_capture" -eq 0 && -f "$repo4/$mkdir_file" && "$mkdir_file" == docs/test/new-area/* ]]; then
  pass "存在しない配置ディレクトリを作って起票する（型名はアンダースコア表記）"
else
  fault "存在しない配置ディレクトリを作って起票する（exit=$status_capture file=$mkdir_file）"
fi

# ハイフン表記（sara init のサブコマンド名そのもの）も同じ型として通る。
run_sara_new "$repo4" sara-new test-case second-case docs/test/new-area
hyphen_file="$(field file)"
hyphen_id="$(field id)"
if [[ "$status_capture" -eq 0 && "$hyphen_id" == CASE-* ]]; then
  pass "型名のハイフン表記も同じ型として通る（test-case → CASE）"
else
  fault "型名のハイフン表記も同じ型として通る（exit=$status_capture id=$hyphen_id file=$hyphen_file）"
fi

# --- 5. slug の検査 -----------------------------------------------------------
#
# slug はファイル名へそのまま入る。`../` やスペースを通すと配置先が黙ってずれる。
# 検査は起票の前に行い、失敗時にファイルを残さない。

repo5="$work/slug"
make_repo "$repo5"
for bad in "" "has space" "../escape" "UPPER" "under_score"; do
  run_sara_new "$repo5" sara-new requirement "$bad" docs/requirements
  bad_status="$status_capture"
  bad_files="$(find "$repo5/docs" -name '*.md' -type f ! -name 'model.yaml' | wc -l)"
  if [[ "$bad_status" -eq 2 && "$bad_files" -eq 0 ]]; then
    pass "不正な slug '$bad' を exit 2 で拒否し、ファイルを残さない"
  else
    fault "不正な slug '$bad' を exit 2 で拒否し、ファイルを残さない（exit=$bad_status 残 $bad_files 件）"
  fi
done

# 有効側（英小数字とハイフン）は通す。境界の両側を押さえる。
run_sara_new "$repo5" sara-new requirement a1-b2 docs/requirements
if [[ "$status_capture" -eq 0 ]]; then
  pass "英小文字・数字・ハイフンの slug は受理する（境界の有効側）"
else
  fault "英小文字・数字・ハイフンの slug は受理する（実際: exit=$status_capture）"
fi

# --- 6. ADR は連番維持のため拒否する ------------------------------------------
#
# ADR だけ id_format が {prefix}-{seq:04} で、ファイル名規約も別（→ ADR-0053）。
# ラッパーは UUID 採番の 10 型だけを担い、ADR は `sara init adr` 直呼びへ委ねる。

repo6="$work/adr"
make_repo "$repo6"
run_sara_new "$repo6" sara-new adr some-decision docs/adr
adr_status="$status_capture"
adr_files="$(find "$repo6/docs" -name '*.md' -type f ! -name 'model.yaml' | wc -l)"
if [[ "$adr_status" -eq 2 ]]; then
  pass "ADR は exit 2 で拒否する（連番維持）"
else
  fault "ADR は exit 2 で拒否する（実際: exit=$adr_status）"
fi
if [[ "$adr_files" -eq 0 ]]; then
  pass "ADR 拒否時はファイルを作らない"
else
  fault "ADR 拒否時はファイルを作らない（実際: $adr_files 件）"
fi

# --- 7. sara init の失敗を伝播する --------------------------------------------
#
# sara 呼び出しは seam（SARA_NEW_SARA）経由。失敗を握り潰して 0 を返すと、
# 起票できていないのに成功したと誤認する。

fake_sara="$work/fake-sara"
# shebang は実行中の bash の絶対パスを埋め込む。`#!/usr/bin/env bash` だと
# nix のビルドサンドボックス（checks.sara-new 経由）に /usr/bin/env が無く
# exit 126 になる（sara-id.sh の偽 uuidgen と同じ事情）。
{
  printf '#!%s\n' "$BASH"
  cat <<'FAKE'
# 引数末尾がファイルパスとは限らないので、`init` の次の次を拾わず、
# SARA_NEW_FAKE_MODE に応じた振る舞いだけを行う。
case "$SARA_NEW_FAKE_MODE" in
  fail)
    echo "fake sara: boom" >&2
    exit 3
    ;;
  no-id)
    echo "[OK] Created something with Requirement template"
    exit 0
    ;;
esac
FAKE
} >"$fake_sara"
chmod +x "$fake_sara"

repo7="$work/initfail"
make_repo "$repo7"
run_sara_new "$repo7" env SARA_NEW_SARA="$fake_sara" SARA_NEW_FAKE_MODE=fail \
  sara-new requirement boom docs/requirements
initfail_status="$status_capture"
initfail_files="$(find "$repo7/docs" -name '*.md' -type f ! -name 'model.yaml' | wc -l)"
if [[ "$initfail_status" -ne 0 ]]; then
  pass "sara init が失敗したら非ゼロで落ちる"
else
  fault "sara init が失敗したら非ゼロで落ちる（実際: exit=$initfail_status）"
fi
if [[ "$initfail_files" -eq 0 ]]; then
  pass "sara init 失敗時に一時ファイルを残さない"
else
  fault "sara init 失敗時に一時ファイルを残さない（実際: $initfail_files 件）"
fi

# --- 8. ID を読めなければ失敗する ---------------------------------------------
#
# sara の出力形式が変わった場合。ID 無しで rename を続行すると、規約に反した
# ファイル名（UUID 部が空）の item が黙って生まれる。

repo8="$work/noid"
make_repo "$repo8"
run_sara_new "$repo8" env SARA_NEW_SARA="$fake_sara" SARA_NEW_FAKE_MODE=no-id \
  sara-new requirement noid docs/requirements
noid_status="$status_capture"
noid_files="$(find "$repo8/docs" -name '*.md' -type f ! -name 'model.yaml' | wc -l)"
if [[ "$noid_status" -ne 0 ]]; then
  pass "sara init の出力から ID を読めなければ非ゼロで落ちる"
else
  fault "sara init の出力から ID を読めなければ非ゼロで落ちる（実際: exit=$noid_status）"
fi
if [[ "$noid_files" -eq 0 ]]; then
  pass "ID 読み取り失敗時に一時ファイルを残さない"
else
  fault "ID 読み取り失敗時に一時ファイルを残さない（実際: $noid_files 件）"
fi

# --- 9. 既存ファイルを上書きしない --------------------------------------------
#
# 同じ日・同じ slug で 2 度叩いても UUID が違うので通常は衝突しないが、
# rename 先が既存なら潰さず落とす（起票済み item の消失を構造的に避ける）。

repo9="$work/clobber"
make_repo "$repo9"
run_sara_new "$repo9" sara-new requirement dup docs/requirements
first_file="$(field file)"
if [[ -z "$first_file" ]]; then
  fault "§9 の前提（1 件目の起票）が失敗した（$stdout_capture）"
else
  # 2 件目の rename 先を 1 件目と同じにするため、SARA_NEW_SARA で 1 件目と
  # 同じ ID を返す偽 sara を使う（実 sara は毎回別の UUID を採るため衝突を作れない）。
  clobber_sara="$work/clobber-sara"
  {
    printf '#!%s\n' "$BASH"
    cat <<'FAKE'
# 最終引数を初期化対象のファイルとみなして空ファイルを作り、既定の ID を返す。
for target; do :; done
: >"$target"
printf '  ID:   %s\n' "$SARA_NEW_FAKE_ID"
FAKE
  } >"$clobber_sara"
  chmod +x "$clobber_sara"

  first_id="$(sed -n 's/^id: "\(.*\)"$/\1/p' "$repo9/$first_file")"
  run_sara_new "$repo9" env SARA_NEW_SARA="$clobber_sara" SARA_NEW_FAKE_ID="$first_id" \
    sara-new requirement dup docs/requirements
  clobber_status="$status_capture"
  clobber_files="$(find "$repo9/docs/requirements" -name '*.md' -type f | wc -l)"
  if [[ "$clobber_status" -eq 1 ]]; then
    pass "rename 先が既存なら exit 1 で拒否する"
  else
    fault "rename 先が既存なら exit 1 で拒否する（実際: exit=$clobber_status）"
  fi
  if [[ "$clobber_files" -eq 1 ]]; then
    pass "既存ファイルを上書きせず、一時ファイルも残さない"
  else
    fault "既存ファイルを上書きせず、一時ファイルも残さない（実際: $clobber_files 件）"
  fi
fi

# --- 10. 引数の異常系 ---------------------------------------------------------

repo10="$work/args"
make_repo "$repo10"

run_sara_new "$repo10" sara-new
if [[ "$status_capture" -eq 2 ]]; then
  pass "引数なしは exit 2"
else
  fault "引数なしは exit 2（実際: exit=$status_capture）"
fi

# 境界は 2/3 の間（型・slug・dir の 3 個が必須）。無効側を押さえる
# （有効側は §1 が押さえている）。
run_sara_new "$repo10" sara-new requirement only-slug
if [[ "$status_capture" -eq 2 ]]; then
  pass "配置ディレクトリを省くと exit 2（境界の無効側）"
else
  fault "配置ディレクトリを省くと exit 2（実際: exit=$status_capture）"
fi

run_sara_new "$repo10" sara-new --help
if [[ "$status_capture" -eq 0 ]]; then
  pass "--help は exit 0"
else
  fault "--help は exit 0（実際: exit=$status_capture）"
fi

exit "$fail"
