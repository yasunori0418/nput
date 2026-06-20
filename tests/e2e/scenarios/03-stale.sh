#!/usr/bin/env bash
# stale 除去: 2 entry を配置 → 片方の entry を config から削除 → 再 apply で旧 symlink が消えることをアサート
# （保守的 stale 除去の不変条件: 前世代 manifest に記録され今は無い nput 管理 symlink だけを除去する）。
set -euo pipefail
source "$(dirname "$0")/../lib.sh"
e2e_isolate

PROJ="$E2E_WORK/proj"
mkdir -p "$PROJ/srcrepo/x" "$PROJ/srcrepo/y"
echo "XXX" >"$PROJ/srcrepo/x/file"
echo "YYY" >"$PROJ/srcrepo/y/file"

# keep="both" で x,y 両方、keep="x" で x のみを公開する fixture flake を書く。
write_flake() {
	local entries
	if [ "$1" = "both" ]; then
		entries='entries.".out/x" = { src = ./srcrepo; subpath = "x"; }; entries.".out/y" = { src = ./srcrepo; subpath = "y"; };'
	else
		entries='entries.".out/x" = { src = ./srcrepo; subpath = "x"; };'
	fi
	cat >"$PROJ/flake.nix" <<EOF
{
$(e2e_flake_inputs)
  outputs = { self, nixpkgs, nput }: {
    nput = nixpkgs.lib.genAttrs $E2E_SYSTEMS (system: {
      docs = nput.lib.mkManifest {
        pkgs = nixpkgs.legacyPackages.\${system};
        root = nput.lib.projectRoot;
        $entries
      };
    });
  };
}
EOF
}

cd "$PROJ"
write_flake both
git init -q
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm both

e2e_step "x,y 両方を apply"
nput apply docs
assert_symlink "$PROJ/.out/x"
assert_symlink "$PROJ/.out/y"

e2e_step "y を config から削除して再 apply"
write_flake x
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm "drop y"
nput apply docs

e2e_step "x は残り、y の旧 symlink は stale 除去される"
assert_symlink "$PROJ/.out/x"
assert_file_eq "$PROJ/.out/x/file" "XXX"
assert_absent "$PROJ/.out/y"

e2e_finish
