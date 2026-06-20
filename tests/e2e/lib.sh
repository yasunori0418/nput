# shellcheck shell=bash
# E2E ハーネス共通ライブラリ（→ docs/design.md「テスト戦略」・ADR-0012）。
#
# 各シナリオはこのファイルを source し、隔離した一時 HOME / XDG_STATE_HOME 下で
# 実 nix を使って `nput` を駆動し、FS / profile / 世代の結果をアサートする。
# 偽 src は fixture flake ディレクトリ内の相対パス（eval 時に store へコピー）か、
# out-of-store の live ディレクトリ（store 外）として用意する。

# 多重 source を防ぐ。
if [ -n "${_NPUT_E2E_LIB:-}" ]; then return 0; fi
_NPUT_E2E_LIB=1

# lib.sh の位置からリポジトリルートを解決する（cwd に依存しない）。
_E2E_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$_E2E_LIB_DIR/../.." && pwd)"
export REPO_ROOT

# nput バイナリ。CI では ci devShell が PATH に載せる。NPUT で上書き可能。
NPUT="${NPUT:-nput}"
export NPUT

# 実行環境の nix system 名（fixture flake の system 次元・HM activationPackage の選択に使う）。
E2E_SYSTEM="$(nix eval --impure --raw --expr 'builtins.currentSystem')"
export E2E_SYSTEM

# 全 fixture flake が共有する systems リスト（4 system・root flake と揃える）。
E2E_SYSTEMS='[ "x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin" ]'

# ---- ログ / アサーション -----------------------------------------------------

_E2E_FAILS=0

e2e_log()  { printf '    %s\n' "$*"; }
e2e_step() { printf '\n>>> %s\n' "$*"; }

e2e_pass() { printf '    \033[32mok\033[0m   %s\n' "$*"; }
e2e_fail() {
	printf '    \033[31mFAIL\033[0m %s\n' "$*" >&2
	_E2E_FAILS=$((_E2E_FAILS + 1))
}

# シナリオ末尾で呼ぶ。1 件でも失敗していれば非ゼロ終了。
e2e_finish() {
	if [ "$_E2E_FAILS" -ne 0 ]; then
		printf '\n%d 件のアサーションが失敗しました\n' "$_E2E_FAILS" >&2
		exit 1
	fi
}

assert_symlink() {
	# $1: パス, $2(任意): 期待するリンク先
	if [ ! -L "$1" ]; then
		e2e_fail "symlink であるべき: $1"
		return
	fi
	if [ "$#" -ge 2 ]; then
		local got; got="$(readlink "$1")"
		if [ "$got" != "$2" ]; then
			e2e_fail "symlink 先が不一致: $1 -> $got (期待: $2)"
			return
		fi
	fi
	e2e_pass "symlink: $1"
}

assert_real_dir() {
	# 通常ディレクトリ（symlink ではない）であること。
	if [ -d "$1" ] && [ ! -L "$1" ]; then
		e2e_pass "通常ディレクトリ: $1"
	else
		e2e_fail "通常ディレクトリであるべき（symlink 不可）: $1"
	fi
}

assert_exists() {
	if [ -e "$1" ]; then e2e_pass "存在: $1"; else e2e_fail "存在すべき: $1"; fi
}

assert_absent() {
	# symlink が壊れていても検知できるよう -e ではなく -e || -L で判定。
	if [ ! -e "$1" ] && [ ! -L "$1" ]; then
		e2e_pass "不在: $1"
	else
		e2e_fail "不在であるべき: $1"
	fi
}

assert_file_eq() {
	# $1: パス, $2: 期待する内容（行）
	local got; got="$(cat "$1" 2>/dev/null || true)"
	if [ "$got" = "$2" ]; then
		e2e_pass "内容一致: $1"
	else
		e2e_fail "内容不一致: $1 = '$got' (期待: '$2')"
	fi
}

assert_writable() {
	if [ -w "$1" ]; then e2e_pass "書込可: $1"; else e2e_fail "書込可であるべき: $1"; fi
}

# ---- 環境の隔離 --------------------------------------------------------------

# 隔離した一時 HOME / XDG_STATE_HOME を作り export する。EXIT trap で消す。
# profile（nix-env --set）は XDG_STATE_HOME 配下に作られるため、ランナーの実 state を汚さない。
e2e_isolate() {
	E2E_WORK="$(mktemp -d)"
	export HOME="$E2E_WORK/home"
	export XDG_STATE_HOME="$E2E_WORK/state"
	mkdir -p "$HOME" "$XDG_STATE_HOME"
	# 一時 HOME には nix の設定が無いため、ランナーの実設定（experimental-features 等）を引き継ぐ。
	export NIX_CONFIG="${NIX_CONFIG:-}
experimental-features = nix-command flakes"
	# shellcheck disable=SC2064
	trap "rm -rf '$E2E_WORK'" EXIT
	e2e_log "work=$E2E_WORK home=$HOME state=$XDG_STATE_HOME"
}

# fixture flake の inputs 定義を出力する（nput を REPO_ROOT の path: input で参照し、
# nixpkgs / home-manager は nput の flake.lock pin に follows させてオフライン評価する）。
# 引数: $1 = 追加 input 行（任意・例: home-manager follows）
e2e_flake_inputs() {
	printf '  inputs.nput.url = "path:%s";\n' "$REPO_ROOT"
	printf '  inputs.nixpkgs.follows = "nput/nixpkgs";\n'
	if [ "${1:-}" = "with-hm" ]; then
		printf '  inputs.home-manager.follows = "nput/home-manager";\n'
	fi
}

# nput を実行する（dirty git tree 警告など stderr ノイズは通すが、結果はアサーションで判定）。
nput() { command "$NPUT" "$@"; }
