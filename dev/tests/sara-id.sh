#!/usr/bin/env bash
# sara-id（UUIDv4 二層 ID の採番コマンド）の検証。
#
# 実行:
#   nix develop ./dev -c dev/tests/sara-id.sh   # devShell から直接
#   nix flake check ./dev                        # checks.sara-id 経由
#
# 検証対象（→ Issue #207・ADR-0048）。番号は下の節見出しに対応する:
#   1.  正式 ID・ファイル名素材・散文参照の 3 形式を出力し、UUIDv4 として妥当
#       （version nibble = 4・variant nibble ∈ {8,9,a,b}）で、省略形が前方一致する
#   2.  呼ぶたびに異なる ID を返す
#   3.  8 文字 prefix が docs/ に既出なら再生成する（再帰走査・呼び出し回数）
#   4.  再生成は 16 回で打ち切り exit 1（重複 ID を出さない）
#   4b. 走査先を git のリポジトリルート基準で解決する
#   4c. git 管理外ではカレント基準へフォールバックする
#   5.  docs/ が無いリポジトリでも採番できる
#   6.  型名・別名から prefix を引き、未知の入力はフォールバックで大文字化する
#   6b. prefix マップが docs/model.yaml の型・prefix と双方向に一致する（契約テスト）
#   7.  slug 未指定でもファイル名素材を出す
#   8.  ADR は連番維持のため exit 2 で拒否する
#   9.  異常系の終了コードを区別する（引数不正 = 2 / --help = 0）

# -e は使わない。このテストは「1 回の実行で全失敗を報告する」集計方式を採っており、
# -e があると最初の非ゼロ終了でスクリプトごと打ち切られて以降のアサーションが走らない
# （退行時の診断が先頭 1 件で切れる）。SUT の非ゼロ終了自体は run_sara_id が受けるが、
# アサーション周辺の想定外終了まで含めて集計を守るため -e を外す。既存の
# tests/e2e/run.sh も同じ理由で set -uo pipefail を採っている。
set -uo pipefail

fail=0
pass() { printf 'ok   - %s\n' "$1"; }
fault() {
  printf 'FAIL - %s\n' "$1"
  fail=1
}

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# 実装は走査先を `git rev-parse --show-toplevel` で解決し、失敗時のみカレント基準へ
# フォールバックする。以降の重複検出系アサーションは「$work/docs を走査する」ことを
# 前提に既出ファイルを置くため、$work 自身を git repo のルートに固定して前提を明示する
# （TMPDIR が git 作業ツリー内に置かれた環境だと、固定しない限り走査先が実リポジトリの
# docs/ に解決され、アサーションが一斉に偽陽性化する）。
git -C "$work" init -q

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
# shebang は実行中の bash の絶対パスを埋め込む。`#!/usr/bin/env bash` だと
# nix のビルドサンドボックス（checks.sara-id 経由）に /usr/bin/env が無く
# exit 126 になる。
{
  printf '#!%s\n' "$BASH"
  cat <<'FAKE'
n=$(cat "$SARA_ID_FAKE_COUNT" 2>/dev/null || echo 0)
n=$((n + 1))
printf '%s' "$n" >"$SARA_ID_FAKE_COUNT"
if [ "$n" -le "$SARA_ID_FAKE_SWITCH" ]; then
  printf '%s\n' "$SARA_ID_FAKE_TAKEN"
else
  printf '%s\n' "$SARA_ID_FAKE_FRESH"
fi
FAKE
} >"$fake"
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

# ここから先は本物の uuidgen を使う（形式・一意性の検査が固定値で素通りしないよう、
# seam をグローバルから外す）。以降で偽 uuidgen を使うケースは呼び出しごとに
# 環境変数を明示的に渡すこと。
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

# --- 4c. git 管理外ではカレント基準へフォールバックする ----------------------
#
# `git rev-parse` が失敗する経路。カレント基準で重複検出が働くことを固定する
# （この経路は他ケースが暗黙に依存している前提でもあるので明示的に押さえる）。

nogit="$(mktemp -d)"
mkdir -p "$nogit/docs"
printf 'REQ-%s\n' "$taken" >"$nogit/docs/existing.md"
: >"$count_file"
nogit_out="$(cd "$nogit" &&
  SARA_ID_UUIDGEN="$fake" SARA_ID_FAKE_COUNT="$count_file" \
    SARA_ID_FAKE_TAKEN="$taken" SARA_ID_FAKE_FRESH="$fresh" SARA_ID_FAKE_SWITCH=1 \
    sara-id req x 2>/dev/null)"
nogit_id="$(printf '%s\n' "$nogit_out" | sed -n 's/^id:[[:space:]]*//p')"
rm -rf "$nogit"
if [[ "$nogit_id" == "REQ-$fresh" ]]; then
  pass "git 管理外ではカレント基準の docs/ を走査する"
else
  fault "git 管理外ではカレント基準の docs/ を走査する（実際: $nogit_id）"
fi

# --- 5. docs/ が無いリポジトリでも動く ---------------------------------------
#
# 実装の `[ ! -d "$scan_dir" ]` 早期 break を固定する。ルート解決が成功したうえで
# docs/ が無い状態を作る（git init 済み・docs/ なし）。git 管理外の一時ディレクトリ
# だと、TMPDIR の置き場所次第で走査先が実リポジトリへ解決されこの分岐を通らない。

nodocs="$(mktemp -d)"
git -C "$nodocs" init -q
nodocs_out="$(cd "$nodocs" && sara-id req x 2>/dev/null)" && nodocs_status=0 || nodocs_status=$?
rm -rf "$nodocs"
nodocs_id="$(printf '%s\n' "$nodocs_out" | sed -n 's/^id:[[:space:]]*//p')"
if [[ "$nodocs_status" -eq 0 && "$nodocs_id" =~ ^REQ-${uuid_re}$ ]]; then
  pass "docs/ が無いリポジトリでも採番できる"
else
  fault "docs/ が無いリポジトリでも採番できる（exit=$nodocs_status id=$nodocs_id）"
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

# --- 6b. prefix マップと docs/model.yaml の突合（契約テスト）-----------------
#
# 型を足すたび model.yaml / dev/flake.nix の case 分岐 / CLAUDE.md の 3 箇所を手で
# 同期する構図に対し、前 2 者の機械的な突合を置く（→ Issue #250）。§6 の代表サンプル
# 3 件では、マップ行が失われても sara が prefix を検証しない（model.yaml 冒頭:
# 検証は「非空」「英数字 / ハイフン / アンダースコア」のみ）ため、誤 prefix の ID が
# 黙って frontmatter へ流入する。
#
# 突合は 3 つに分ける。正本が別々なので担保できる範囲も違う:
#   - 6b-1: model.yaml の型 ⊆ case アーム（型を足して case へ足し忘れると落ちる）
#   - 6b-2: case アームの正式型名 ⊆ model.yaml（case へ足して model.yaml へ
#           足し忘れる逆方向。正式型名だけを見る。別名は model.yaml に載らない）
#   - 6b-3: case アームの全入力（別名含む）を実起動し、宣言どおりの prefix を返す
#
# 6b-3 の検知力は入力によって違う。`*)` フォールバックが大文字化した結果と期待 prefix が
# 一致する入力（risk → RISK・adr → ADR・qa → QA・tp → TC 等の短縮別名）は、case アームを
# 削っても同じ値が返るため退行を検知できない。一方 quality → QUALITY・use-case →
# USE-CASE のように一致しない入力は検知する。除外リストを手で持つとその一覧自体が
# 事実と乖離するので、6b-1 / 6b-2 の集合突合で漏れを押さえたうえで 6b-3 は全入力を回す。
#
# ADR は「採番せず exit 2 で拒否する」が期待動作なので 6b-3 の実起動からは外す
# （拒否経路は §8 が担保する）。集合突合の 6b-1 / 6b-2 には含める。
#
# 正本 2 ファイルの在り処は 2 経路ある。どちらでも解決できなければ skip せず失敗させる
# （黙って素通りすると、この節が存在するのに何も検証していない状態を緑で隠す）:
#   1. SARA_MODEL_YAML / SARA_DEV_FLAKE（checks.sara-id のサンドボックス。リポジトリの
#      作業ツリーが無いので nix が store path を渡す）
#   2. git のリポジトリルート基準（`nix develop` からの直接実行・CI の sara job）

model_yaml="${SARA_MODEL_YAML:-}"
dev_flake="${SARA_DEV_FLAKE:-}"
# テスト自身の cwd（$work）ではなくスクリプトを起動した元のリポジトリを見るため、
# run_sara_id とは違い cd しない。
contract_root="$(git rev-parse --show-toplevel 2>/dev/null)"
[[ -f "$model_yaml" ]] || model_yaml="$contract_root/docs/model.yaml"
[[ -f "$dev_flake" ]] || dev_flake="$contract_root/dev/flake.nix"

if [[ ! -f "$model_yaml" || ! -f "$dev_flake" ]]; then
  fault "契約テストの正本を解決できない（model.yaml=$model_yaml flake.nix=$dev_flake）"
else
  # model.yaml の item_types: 〜 relations: から `- id:` と直後の `prefix:` を組にする。
  # yq を devShell に足さず awk で済ませる（並びが崩れれば下の件数突合が検知する）。
  mapfile -t model_pairs < <(
    awk '
      /^item_types:/  { in_types = 1; next }
      /^relations:/   { in_types = 0; next }
      !in_types       { next }
      /^  - id: /     { type = $3; next }
      /^    prefix: / { if (type != "") { print type, $2; type = "" } }
    ' "$model_yaml"
  )

  # 抽出の健全性を先に固定する。件数は直書きせず model.yaml 側の `- id:` 行数と
  # 突き合わせる（型を足したとき、この節が余計な更新点にならないようにする）。
  model_type_lines="$(awk '
    /^item_types:/ { in_types = 1; next }
    /^relations:/  { in_types = 0; next }
    in_types && /^  - id: / { n++ }
    END { print n + 0 }
  ' "$model_yaml")"

  # case アームから「入力名 → prefix」を全て取り出す（正式型名・別名の両方）。
  # `*)` のフォールバック行は契約の対象外なので落とす。
  mapfile -t arm_pairs < <(
    awk '
      /case "\$1" in/ { in_case = 1; next }
      /^ *esac/       { in_case = 0 }
      !in_case        { next }
      /^ *\*\)/       { next }
      /prefix=/ {
        line = $0
        sub(/^ */, "", line)
        split(line, halves, ")")
        prefix = line
        sub(/.*prefix=/, "", prefix)
        sub(/[^A-Za-z0-9_].*/, "", prefix)
        n = split(halves[1], names, "|")
        for (i = 1; i <= n; i++) {
          gsub(/ /, "", names[i])
          if (names[i] != "") { print names[i], prefix }
        }
      }
    ' "$dev_flake"
  )

  extract_ok=1
  if [[ "${#model_pairs[@]}" -ne "$model_type_lines" || "$model_type_lines" -eq 0 ]]; then
    fault "model.yaml の全型から prefix を抽出する（型 $model_type_lines 件に対し抽出 ${#model_pairs[@]} 件）"
    extract_ok=0
  fi
  if [[ "${#arm_pairs[@]}" -eq 0 ]]; then
    fault "dev/flake.nix の case アームから prefix マップを抽出する（0 件）"
    extract_ok=0
  fi

  if [[ "$extract_ok" -eq 0 ]]; then
    # 抽出が壊れた状態で突合を回すと、集合が空なので全て緑に見える。
    fault "prefix マップの突合を実行できない（抽出が壊れているため中断した）"
  else
    declare -A arm_prefix=()
    for pair in "${arm_pairs[@]}"; do
      arm_prefix["${pair%% *}"]="${pair##* }"
    done

    declare -A model_prefix_of=()
    for pair in "${model_pairs[@]}"; do
      model_prefix_of["${pair%% *}"]="${pair##* }"
    done

    # --- 6b-1. model.yaml の型が case アームに揃っているか -------------------
    forward_ok=1
    for model_type in "${!model_prefix_of[@]}"; do
      want_prefix="${model_prefix_of[$model_type]}"
      if [[ -z "${arm_prefix[$model_type]:-}" ]]; then
        fault "型 $model_type が dev/flake.nix の case アームに無い（model.yaml は $want_prefix）"
        forward_ok=0
      elif [[ "${arm_prefix[$model_type]}" != "$want_prefix" ]]; then
        fault "型 $model_type の prefix が食い違う（model.yaml=$want_prefix case=${arm_prefix[$model_type]}）"
        forward_ok=0
      fi
    done
    if [[ "$forward_ok" -eq 1 ]]; then
      pass "model.yaml の全 $model_type_lines 型が case アームに同じ prefix で載っている"
    fi

    # --- 6b-2. case アームの正式型名が model.yaml にあるか -------------------
    #
    # 別名（req / uc …）は model.yaml に載らないので、正式型名だけを対象にする。
    # 「別名かどうか」は model.yaml 側の型集合で判定できないため、同じ prefix を持つ
    # アームのうち model.yaml に存在するものが 1 つでもあれば残りは別名とみなす。
    backward_ok=1
    for arm_name in "${!arm_prefix[@]}"; do
      [[ -n "${model_prefix_of[$arm_name]:-}" ]] && continue
      arm_p="${arm_prefix[$arm_name]}"
      known_prefix=0
      for model_type in "${!model_prefix_of[@]}"; do
        [[ "${model_prefix_of[$model_type]}" == "$arm_p" ]] && known_prefix=1 && break
      done
      if [[ "$known_prefix" -eq 0 ]]; then
        fault "case アームの $arm_name（prefix=$arm_p）に対応する型が model.yaml に無い"
        backward_ok=0
      fi
    done
    if [[ "$backward_ok" -eq 1 ]]; then
      pass "case アームの全 ${#arm_prefix[@]} 入力が model.yaml のいずれかの prefix に対応する"
    fi

    # --- 6b-3. 全入力を実起動して宣言どおりの prefix を返すか ----------------
    #
    # UUID 部分は検証しないので偽 uuidgen で決定論化する（本物の乱数を
    # 入力数ぶん引かない）。既出チェックに当たらないよう docs/ を空にしておく。
    rm -f "$work/docs"/*.md
    : >"$count_file"
    export SARA_ID_UUIDGEN="$fake" SARA_ID_FAKE_COUNT="$count_file" \
      SARA_ID_FAKE_TAKEN="$fresh" SARA_ID_FAKE_FRESH="$fresh" SARA_ID_FAKE_SWITCH=0

    invoke_ok=1
    for arm_name in "${!arm_prefix[@]}"; do
      # ADR は採番せず exit 2 で拒否する（§8）。prefix 一致では検証できない。
      [[ "${arm_prefix[$arm_name]}" == "ADR" ]] && continue

      run_sara_id sara-id "$arm_name" probe
      got_id="$(field id)"
      if [[ "$status_capture" -ne 0 || "$got_id" != "${arm_prefix[$arm_name]}-"* ]]; then
        fault "sara-id $arm_name が ${arm_prefix[$arm_name]}- を返す（exit=$status_capture 実際: $got_id）"
        invoke_ok=0
      fi
    done

    unset SARA_ID_UUIDGEN SARA_ID_FAKE_COUNT SARA_ID_FAKE_TAKEN SARA_ID_FAKE_FRESH SARA_ID_FAKE_SWITCH

    if [[ "$invoke_ok" -eq 1 ]]; then
      pass "case アームの全入力が宣言どおりの prefix を返す（ADR の拒否経路は §8）"
    fi
  fi
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

# 境界は 2/3 の間（実装は $# -gt 2 で弾く）。無効側と有効側の両方を押さえる。
run_sara_id sara-id req slug extra
if [[ "$status_capture" -eq 2 ]]; then
  pass "引数 3 個は exit 2"
else
  fault "引数 3 個は exit 2（実際: exit=$status_capture）"
fi

run_sara_id sara-id req slug
if [[ "$status_capture" -eq 0 ]]; then
  pass "引数 2 個は受理する（境界の有効側）"
else
  fault "引数 2 個は受理する（実際: exit=$status_capture）"
fi

run_sara_id sara-id --help
if [[ "$status_capture" -eq 0 ]]; then
  pass "--help は exit 0"
else
  fault "--help は exit 0（実際: exit=$status_capture）"
fi

exit "$fail"
