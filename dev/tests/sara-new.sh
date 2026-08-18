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
#   1b. --name 未指定の既定経路でも name が slug 由来になる（仮ファイル名が漏れない）
#   1c. 一時ファイル・一時ディレクトリ（.sara-new-*）を残さない
#   2.  採番 ID とファイルパスを機械可読な 2 行（id: / file:）で出力する
#   3.  sara init へオプションを透過する（-- 以降）
#   4.  配置ディレクトリを作る（無ければ mkdir -p）
#   4b. 型名はアンダースコア表記（model.yaml・規約文書）とハイフン表記
#       （sara init のサブコマンド名）の両方を受ける
#   5.  slug の検査（空・不正文字は exit 2 で、ファイルを残さない）
#   5b. 英小文字・数字・ハイフンの slug を受理し、ファイル名へ入れる（境界の有効側）
#   6.  ADR は連番維持のため exit 2 で拒否する
#   7.  sara init の失敗（exit 3）をそのまま伝播し、一時ファイルを残さない（seam で再現）
#   8.  sara init の出力から ID を読めなければ exit 1 で落ち、一時ファイルを残さない（seam）
#   8b. 採番 ID から UUID 部を取り出せなければ exit 1 で落ちる（prefix がハイフンを
#       含む型。正常系では踏まない経路なので seam で押さえる）
#   9.  出力先が既存なら上書きせず exit 1（起票済み item を潰さない）
#   10. 引数の異常系（引数不足 = 2 / -- 区切り無しの余分引数 = 2 / --help = 0）
#
# 偽 sara（seam）は SUT の呼び出し形（--no-color --no-emoji init <型> <仮パス>）も
# 検査する。素通しの偽物にすると、SUT がその形を崩す退行を吸収して緑のまま通る。
#
# 担保できる範囲: ラッパーの責務（パス組み立て・rename・ID 読み取り・異常系の後片付け）。
# UUID の採番そのもの・8 文字 prefix の重複可否は検証しない（sara init の領分であり、
# ラッパーは採番を持たない設計 → Issue #367）。
#
# fixture は実物の docs/model.yaml を重ねて作る（sara-gap.sh と同じく写しを持たない。
# 型や必須フィールドが変わって fixture が実モデルに合わなくなれば、このテストが落ちて
# 追随を要求する）。
#
# -e は使わない。sara-gap.sh と同じく「1 回の実行で全失敗を報告する」
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

# SUT が PATH に無いまま走ると、異常系の節（§5〜§10）は「非ゼロで落ちる」「ファイルを
# 残さない」がどちらも自明に成立して緑になる。全節の前提としてここで存在を確かめ、
# 無ければ以降のアサーションを回さず落とす（偽の緑を構造的に防ぐ）。
if ! command -v sara-new >/dev/null 2>&1; then
  fault "SUT（sara-new）が PATH に無い。devShell 経由で実行すること"
  exit 1
fi

# SUT を repo のルートで呼ぶ（sara.toml を既定パスで引かせるため）。
# 非ゼロ終了でもここでは落とさない（判定は各アサーションが行う）。
# stderr は捨てずに拾う（失敗時の fault メッセージに SUT の診断を載せるため。
# 捨てると CI ログに終了コードしか残らず原因追跡がリポジトリ外から不可能になる）。
stdout_capture=""
stderr_capture=""
status_capture=0
run_sara_new() {
  local root="$1"
  shift
  # 固定パスを毎回切って使い回す。stderr_capture が保持するのは常に「直前の 1 回分」
  # で、呼び出しは全て逐次（並列化していない）。
  local err_file="$work/stderr"
  : >"$err_file"
  stdout_capture="$(cd "$root" && "$@" 2>"$err_file")" && status_capture=0 || status_capture=$?
  stderr_capture="$(cat "$err_file" 2>/dev/null)"
}

field() { printf '%s\n' "$stdout_capture" | sed -n "s/^$1:[[:space:]]*//p"; }

# --- 1. 起票と rename ---------------------------------------------------------

repo="$work/basic"
make_repo "$repo"
# 日付は SUT 実行の前後で採る。実行中に日付が変わる（深夜 0 時跨ぎ）と、後だけで
# 採った期待値が実際のファイル名と食い違って偽陽性になる。
date_before="$(date +%Y%m%d)"
run_sara_new "$repo" sara-new requirement lock-ordering docs/requirements
date_after="$(date +%Y%m%d)"

created="$(field file)"
created_id="$(field id)"

if [[ "$status_capture" -eq 0 ]]; then
  pass "起票に成功して exit 0"
else
  fault "起票に成功して exit 0（実際: exit=$status_capture 出力: $stdout_capture stderr: $stderr_capture）"
fi

if [[ "$created_id" =~ ^REQ-${uuid_re}$ ]]; then
  pass "採番 ID が <PREFIX>-<フル UUIDv4> 形式"
else
  fault "採番 ID が <PREFIX>-<フル UUIDv4> 形式（実際: $created_id）"
fi

uuid="${created_id#REQ-}"
want_before="docs/requirements/$date_before-$uuid-lock-ordering.md"
want_after="docs/requirements/$date_after-$uuid-lock-ordering.md"

if [[ "$created" == "$want_before" || "$created" == "$want_after" ]]; then
  pass "ファイル名が <YYYYMMDD>-<フル UUID>-<slug>.md で、出力の file: と一致する"
else
  fault "ファイル名が <YYYYMMDD>-<フル UUID>-<slug>.md（期待: $want_before 実際: $created）"
fi

if [[ -n "$created" && -f "$repo/$created" ]]; then
  pass "その名前のファイルが実在する"
else
  fault "その名前のファイルが実在する（$repo/$created が無い）"
fi

# frontmatter の id と、ファイル名に埋めた UUID が一致する（rename 先の組み立てに
# 別の UUID を混ぜていないことを固定する）。
if grep -q "id: \"$created_id\"" "$repo/$created" 2>/dev/null; then
  pass "frontmatter の id とファイル名の UUID が同じ item を指す"
else
  fault "frontmatter の id とファイル名の UUID が同じ item を指す（id=$created_id）"
fi

# --name を渡さない既定経路で、item の name が意味のある値になる。
#
# sara init は --name 未指定のときファイル名の stem から name を導出するため、
# ラッパーが仮ファイルを `.sara-new-<pid>.md` のような名前で作ると
# name: ".sara-new-12345" が frontmatter へ焼き付き、rename しても直らない
# （sara check はこの値を検証しないので機械検出にも載らない）。実運用の主経路が
# これなので、§3 の --name 透過とは別に固定する。
actual_name="$(sed -n 's/^name: "\(.*\)"$/\1/p' "$repo/$created" 2>/dev/null)"
if [[ "$actual_name" == "lock-ordering" ]]; then
  pass "--name 未指定でも name が slug 由来になる（仮ファイル名が漏れない）"
else
  fault "--name 未指定でも name が slug 由来になる（期待: lock-ordering 実際: $actual_name）"
fi

# 一時ファイル・一時ディレクトリが残っていない（rename であって copy ではない）。
# 件数だけでなく名指しでも見る（件数だけだと「tmp が残る」と「rename 先が増える」を
# 区別できず、失敗時の診断が数字しか出ない）。
leftovers="$(find "$repo/docs/requirements" -name '*.md' -type f | wc -l)"
if [[ "$leftovers" -eq 1 ]]; then
  pass "生成物は 1 ファイルだけ（一時ファイルを残さない）"
else
  fault "生成物は 1 ファイルだけ（実際: $leftovers 件）"
fi

tmp_left="$(find "$repo/docs" -name '.sara-new-*' | wc -l)"
if [[ "$tmp_left" -eq 0 ]]; then
  pass "一時ファイル・一時ディレクトリ（.sara-new-*）を残さない"
else
  fault "一時ファイル・一時ディレクトリ（.sara-new-*）を残さない（実際: $tmp_left 件）"
fi

# --- 2. 出力形式 --------------------------------------------------------------
#
# 呼び出し側（人・エージェント）が採番結果を機械的に拾えることを固定する。
# sara init の装飾付き出力をそのまま流すと、後続の自動処理が壊れる。

# 2 行を合計で数えると、id: が 2 行出て file: が 0 行でも通ってしまう。個別に見る。
id_lines="$(printf '%s\n' "$stdout_capture" | grep -c '^id: ')"
file_lines="$(printf '%s\n' "$stdout_capture" | grep -c '^file: ')"
if [[ "$id_lines" -eq 1 && "$file_lines" -eq 1 ]]; then
  pass "id: / file: をそれぞれ 1 行ずつ出力する"
else
  fault "id: / file: をそれぞれ 1 行ずつ出力する（id: $id_lines 行 file: $file_lines 行）"
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

# 無効側の同値クラスは、実装の文字集合検査のどれが効いたか切り分けられるよう分ける
# （`../escape` は `.` と `/` を同時に含むので、単独では切り分けにならない）。
repo5="$work/slug"
make_repo "$repo5"
for bad in "" "has space" "../escape" "a/b" "UPPER" "under_score" "dot.ted"; do
  run_sara_new "$repo5" sara-new requirement "$bad" docs/requirements
  bad_status="$status_capture"
  bad_files="$(find "$repo5/docs" -name '*.md' -type f | wc -l)"
  if [[ "$bad_status" -eq 2 && "$bad_files" -eq 0 ]]; then
    pass "不正な slug '$bad' を exit 2 で拒否し、ファイルを残さない"
  else
    fault "不正な slug '$bad' を exit 2 で拒否し、ファイルを残さない（exit=$bad_status 残 $bad_files 件）"
  fi
done

# 有効側（英小数字とハイフン）は通す。境界の両側を押さえる。
# 受理するだけでなく、その slug がファイル名へ入ることまで見る（exit 0 だけだと
# slug がファイル名から欠落・変形する退行を通してしまう）。
run_sara_new "$repo5" sara-new requirement a1-b2 docs/requirements
valid_slug_file="$(field file)"
if [[ "$status_capture" -eq 0 && "$valid_slug_file" == *-a1-b2.md ]]; then
  pass "英小文字・数字・ハイフンの slug を受理し、ファイル名へ入れる（境界の有効側）"
else
  fault "英小文字・数字・ハイフンの slug を受理し、ファイル名へ入れる（exit=$status_capture file=$valid_slug_file）"
fi

# --- 6. ADR は連番維持のため拒否する ------------------------------------------
#
# ADR だけ id_format が {prefix}-{seq:04} で、ファイル名規約も別（→ ADR-0053）。
# ラッパーは UUID 採番の 10 型だけを担い、ADR は `sara init adr` 直呼びへ委ねる。

repo6="$work/adr"
make_repo "$repo6"
run_sara_new "$repo6" sara-new adr some-decision docs/adr
adr_status="$status_capture"
adr_files="$(find "$repo6/docs" -name '*.md' -type f | wc -l)"
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
# exit 126 になる（sara-gap.sh の偽 sara と同じ事情）。
#
# 偽 sara は「SUT がどう呼んだか」も検査する。ここを素通しにすると、SUT が
# --no-color を落とす・仮ファイルの stem を slug 以外にするといった退行を偽 sara が
# 吸収して緑のまま通してしまう（実 sara を使う §1〜§4 は、装飾を出さない環境だと
# --no-color の欠落を検知できない）。
#
# 仮ファイルの stem を検査するのは、それが name の由来だから。stem == slug は
# 「--name 未指定でも意味のある name が入る」の前提だが、§1b の name アサーションは
# 「sara が stem から name を導出する」という sara 側の挙動に依存している。sara が
# その導出をやめると §1b は緑のまま検知力だけ失うので、ここで stem 自体も固定する。
#
# 加えて、失敗する前に必ず仮ファイルを作る。作らないと「一時ファイルを残さない」の
# アサーションは SUT の trap cleanup が壊れていても自明に成立する（空振りする）。
{
  printf '#!%s\n' "$BASH"
  cat <<'FAKE'
# 引数契約: sara-new は必ず `--no-color --no-emoji init <型> <仮パス> [透過分...]`
# の順で呼ぶ。違えば偽 sara 自身が非ゼロで落ちて、テスト側の期待コードと食い違う。
if [ "$1" != "--no-color" ] || [ "$2" != "--no-emoji" ] || [ "$3" != "init" ]; then
  echo "fake sara: 想定外の呼ばれ方: $*" >&2
  exit 90
fi
if [ "$4" != "$SARA_NEW_FAKE_WANT_TYPE" ]; then
  echo "fake sara: 型が想定と違う（期待 $SARA_NEW_FAKE_WANT_TYPE 実際 $4）" >&2
  exit 91
fi

# 仮ファイル（第 5 引数）を実際に作る。SUT の後片付けを検証可能にするため。
tmp_path=$5
if [ -z "$tmp_path" ]; then
  echo "fake sara: 仮ファイルのパスが渡っていない" >&2
  exit 92
fi

# stem が slug と一致すること（name の由来なので規約の一部）。
tmp_stem=${tmp_path##*/}
tmp_stem=${tmp_stem%.md}
if [ "$tmp_stem" != "$SARA_NEW_FAKE_WANT_STEM" ]; then
  echo "fake sara: 仮ファイルの stem が想定と違う（期待 $SARA_NEW_FAKE_WANT_STEM 実際 $tmp_stem）" >&2
  exit 94
fi

: >"$tmp_path"

case "${SARA_NEW_FAKE_MODE:-}" in
  fail)
    echo "fake sara: boom" >&2
    exit 3
    ;;
  no-id)
    echo "[OK] Created something with Requirement template"
    exit 0
    ;;
  id)
    printf '  ID:   %s\n' "$SARA_NEW_FAKE_ID"
    exit 0
    ;;
  *)
    # モード指定漏れを成功扱いにしない（この偽 sara を別の節から流用したとき、
    # 指定漏れが黙って exit 0 になると偽の緑になる）。
    echo "fake sara: 未知のモード: ${SARA_NEW_FAKE_MODE:-（未設定）}" >&2
    exit 93
    ;;
esac
FAKE
} >"$fake_sara"
chmod +x "$fake_sara"

repo7="$work/initfail"
make_repo "$repo7"
run_sara_new "$repo7" env SARA_NEW_SARA="$fake_sara" SARA_NEW_FAKE_MODE=fail \
  SARA_NEW_FAKE_WANT_TYPE=requirement SARA_NEW_FAKE_WANT_STEM=boom \
  sara-new requirement boom docs/requirements
initfail_status="$status_capture"
initfail_files="$(find "$repo7/docs" -name '*.md' -type f | wc -l)"
initfail_tmp="$(find "$repo7/docs" -name '.sara-new-*' | wc -l)"
# 期待コードは具体値で押さえる。`-ne 0` だと SUT 不在（127）・偽 sara の引数契約違反
# （90 番台）まで pass してしまう。SUT は set -e 配下で sara の非ゼロをそのまま伝播する
# ので、偽 sara の exit 3 がそのまま出る。
if [[ "$initfail_status" -eq 3 ]]; then
  pass "sara init の失敗（exit 3）をそのまま伝播する"
else
  fault "sara init の失敗（exit 3）をそのまま伝播する（実際: exit=$initfail_status stderr: $stderr_capture）"
fi
if [[ "$initfail_files" -eq 0 && "$initfail_tmp" -eq 0 ]]; then
  pass "sara init 失敗時に一時ファイル・一時ディレクトリを残さない"
else
  fault "sara init 失敗時に一時ファイル・一時ディレクトリを残さない（md $initfail_files 件 / tmp $initfail_tmp 件）"
fi

# --- 8. ID を読めなければ失敗する ---------------------------------------------
#
# sara の出力形式が変わった場合。ID 無しで rename を続行すると、規約に反した
# ファイル名（UUID 部が空）の item が黙って生まれる。

repo8="$work/noid"
make_repo "$repo8"
run_sara_new "$repo8" env SARA_NEW_SARA="$fake_sara" SARA_NEW_FAKE_MODE=no-id \
  SARA_NEW_FAKE_WANT_TYPE=requirement SARA_NEW_FAKE_WANT_STEM=noid \
  sara-new requirement noid docs/requirements
noid_status="$status_capture"
noid_files="$(find "$repo8/docs" -name '*.md' -type f | wc -l)"
noid_tmp="$(find "$repo8/docs" -name '.sara-new-*' | wc -l)"
# ID を読めない経路は SUT 自身の判断で落ちるので exit 1（具体値で押さえる）。
if [[ "$noid_status" -eq 1 ]]; then
  pass "sara init の出力から ID を読めなければ exit 1 で落ちる"
else
  fault "sara init の出力から ID を読めなければ exit 1 で落ちる（実際: exit=$noid_status stderr: $stderr_capture）"
fi
if [[ "$noid_files" -eq 0 && "$noid_tmp" -eq 0 ]]; then
  pass "ID 読み取り失敗時に一時ファイル・一時ディレクトリを残さない"
else
  fault "ID 読み取り失敗時に一時ファイル・一時ディレクトリを残さない（md $noid_files 件 / tmp $noid_tmp 件）"
fi

# --- 8b. UUID 部を取り出せなければ失敗する ------------------------------------
#
# SUT は正式 ID の最初のハイフンで切って UUID 部を得る。prefix 自体がハイフンを
# 含む型（横展開先で TEST-CASE のような prefix を定義した場合）では残りが
# `CASE-<uuid>` に化けるので、UUID の形をしているか検査して落とす。この分岐が
# 無いと規約違反のファイル名が黙って生まれる（本ラッパーの存在意義そのものが
# 崩れる）ため、正常系では踏まない経路だが seam で押さえる。

repo8b="$work/badprefix"
make_repo "$repo8b"
run_sara_new "$repo8b" env SARA_NEW_SARA="$fake_sara" SARA_NEW_FAKE_MODE=id \
  SARA_NEW_FAKE_ID='TEST-CASE-11111111-2222-4333-8444-555555555555' \
  SARA_NEW_FAKE_WANT_TYPE=requirement SARA_NEW_FAKE_WANT_STEM=badprefix \
  sara-new requirement badprefix docs/requirements
badprefix_status="$status_capture"
badprefix_files="$(find "$repo8b/docs" -name '*.md' -type f | wc -l)"
badprefix_tmp="$(find "$repo8b/docs" -name '.sara-new-*' | wc -l)"
if [[ "$badprefix_status" -eq 1 ]]; then
  pass "UUID 部を取り出せない ID なら exit 1 で落ちる"
else
  fault "UUID 部を取り出せない ID なら exit 1 で落ちる（実際: exit=$badprefix_status stderr: $stderr_capture）"
fi
if [[ "$badprefix_files" -eq 0 && "$badprefix_tmp" -eq 0 ]]; then
  pass "UUID 部の検査で落ちたときにファイル・一時ディレクトリを残さない"
else
  fault "UUID 部の検査で落ちたときにファイル・一時ディレクトリを残さない（md $badprefix_files 件 / tmp $badprefix_tmp 件）"
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
  # 引数契約を持つ共通の偽 sara を id モードで使い回す（専用の緩い偽物を別に置くと、
  # そちらだけ SUT の呼び出し形の退行を吸収してしまう）。
  first_id="$(sed -n 's/^id: "\(.*\)"$/\1/p' "$repo9/$first_file")"
  run_sara_new "$repo9" env SARA_NEW_SARA="$fake_sara" SARA_NEW_FAKE_MODE=id \
    SARA_NEW_FAKE_ID="$first_id" SARA_NEW_FAKE_WANT_TYPE=requirement \
    SARA_NEW_FAKE_WANT_STEM=dup \
    sara-new requirement dup docs/requirements
  clobber_status="$status_capture"
  clobber_files="$(find "$repo9/docs/requirements" -name '*.md' -type f | wc -l)"
  clobber_tmp="$(find "$repo9/docs" -name '.sara-new-*' | wc -l)"

  # 衝突は「日付 + UUID + slug」が揃って初めて成立する。1 件目と 2 件目の間で
  # 日付が変わると rename 先が別名になり、衝突しないのが正しい挙動になる。
  # §1 と同じ日付境界の配慮だが、こちらは前提が崩れるので判定自体を見送る
  # （偽陽性で赤くするより、成立しなかったことを明示する）。
  first_date="${first_file##*/}"
  first_date="${first_date%%-*}"
  if [[ "$first_date" != "$(date +%Y%m%d)" ]]; then
    pass "§9 は日付が跨いだため判定を見送る（衝突の前提が崩れる）"
  elif [[ "$clobber_status" -eq 1 ]]; then
    pass "rename 先が既存なら exit 1 で拒否する"
    if [[ "$clobber_files" -eq 1 && "$clobber_tmp" -eq 0 ]]; then
      pass "既存ファイルを上書きせず、一時ファイル・一時ディレクトリも残さない"
    else
      fault "既存ファイルを上書きせず、一時ファイル・一時ディレクトリも残さない（md $clobber_files 件 / tmp $clobber_tmp 件）"
    fi
  else
    fault "rename 先が既存なら exit 1 で拒否する（実際: exit=$clobber_status stderr: $stderr_capture）"
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

# `--` 区切り無しの余分な引数は受け付けない（タイポした引数が黙って捨てられるより、
# 使い方を出して落とす設計）。この分岐を削っても §3 の正常系は通るので個別に押さえる。
run_sara_new "$repo10" sara-new requirement stray-arg docs/requirements extra
stray_status="$status_capture"
stray_files="$(find "$repo10/docs" -name '*.md' -type f | wc -l)"
if [[ "$stray_status" -eq 2 && "$stray_files" -eq 0 ]]; then
  pass "-- 区切り無しの余分な引数は exit 2 で拒否し、ファイルを残さない"
else
  fault "-- 区切り無しの余分な引数は exit 2 で拒否し、ファイルを残さない（exit=$stray_status 残 $stray_files 件）"
fi

run_sara_new "$repo10" sara-new --help
if [[ "$status_capture" -eq 0 ]]; then
  pass "--help は exit 0"
else
  fault "--help は exit 0（実際: exit=$status_capture）"
fi

exit "$fail"
