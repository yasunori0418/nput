#!/usr/bin/env bash
# home mode: 仮 $HOME（+ XDG_STATE_HOME）で apply → $HOME 配下配置 + profile 世代コミットを確認し、
# 世代をまたいで（entry 入替）`nput rollback` で前世代の配置へ復帰することをアサート。
set -euo pipefail
source "$(dirname "$0")/../lib.sh"
e2e_isolate

PROJ="$E2E_WORK/cfg"
mkdir -p "$PROJ/srcrepo/a" "$PROJ/srcrepo/b"
echo "AAA" >"$PROJ/srcrepo/a/file"
echo "BBB" >"$PROJ/srcrepo/b/file"

# target / subpath を引数に取り fixture flake を書き出す（世代ごとに entry を入れ替える）。
write_flake() {
	local target="$1" sub="$2"
	cat >"$PROJ/flake.nix" <<EOF
{
$(e2e_flake_inputs)
  outputs = { self, nixpkgs, nput }: {
    nput = nixpkgs.lib.genAttrs $E2E_SYSTEMS (system: {
      home = nput.lib.mkManifest {
        pkgs = nixpkgs.legacyPackages.\${system};
        root = nput.lib.homeRoot;
        entries."$target" = { src = ./srcrepo; subpath = "$sub"; };
      };
    });
  };
}
EOF
}

cd "$PROJ"
write_flake ".cfg/a" "a"
git init -q
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm gen1

e2e_step "世代 1: apply（entry a）→ \$HOME 配下に配置"
nput apply home
assert_symlink "$HOME/.cfg/a"
assert_file_eq "$HOME/.cfg/a/file" "AAA"

e2e_step "profile 世代がコミットされたか（home mode の profile レイアウト）"
PROFILE="$XDG_STATE_HOME/nix/profiles/nput/home/profile"
assert_symlink "$PROFILE"
GENS="$(nput list-generations home)"
echo "$GENS"
if [ "$(printf '%s\n' "$GENS" | grep -c .)" -ge 1 ]; then
	e2e_pass "list-generations が世代を返す"
else
	e2e_fail "世代が 1 つも無い"
fi

e2e_step "世代 2: entry を b に入替えて apply（a は stale 除去）"
write_flake ".cfg/b" "b"
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm gen2
nput apply home
assert_symlink "$HOME/.cfg/b"
assert_file_eq "$HOME/.cfg/b/file" "BBB"
assert_absent "$HOME/.cfg/a"

e2e_step "2 世代以上あること"
if [ "$(nput list-generations home | grep -c .)" -ge 2 ]; then
	e2e_pass "世代が 2 つ以上ある"
else
	e2e_fail "世代が 2 つ未満"
fi

e2e_step "nput rollback で前世代（entry a）へ復帰"
nput rollback home
assert_symlink "$HOME/.cfg/a"
assert_file_eq "$HOME/.cfg/a/file" "AAA"
assert_absent "$HOME/.cfg/b"

e2e_finish
