#!/usr/bin/env bash
# project mode: 一時 git repo で `nput apply` → git toplevel 配下に store symlink 配置されることをアサート。
set -euo pipefail
source "$(dirname "$0")/../lib.sh"
e2e_isolate

PROJ="$E2E_WORK/proj"
mkdir -p "$PROJ/srcrepo/skills/nix"
echo "SKILLBODY" >"$PROJ/srcrepo/skills/nix/SKILL.md"

cat >"$PROJ/flake.nix" <<EOF
{
$(e2e_flake_inputs)
  outputs = { self, nixpkgs, nput }: {
    nput = nixpkgs.lib.genAttrs $E2E_SYSTEMS (system: {
      docs = nput.lib.mkManifest {
        pkgs = nixpkgs.legacyPackages.\${system};
        root = nput.lib.projectRoot;
        entries.".nput-out/docs" = { src = ./srcrepo; subpath = "skills/nix"; };
      };
    });
  };
}
EOF

cd "$PROJ"
git init -q
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm init

e2e_step "nput apply docs（project mode）"
nput apply docs

e2e_step "git toplevel 配下に配置されたか"
TARGET="$PROJ/.nput-out/docs"
assert_symlink "$TARGET"
assert_file_eq "$TARGET/SKILL.md" "SKILLBODY"
# store symlink であること（/nix/store を指す）。
case "$(readlink "$TARGET")" in
	/nix/store/*) e2e_pass "store symlink を指す" ;;
	*) e2e_fail "store symlink を指すべき: $(readlink "$TARGET")" ;;
esac

e2e_step "再 apply は冪等（配置が壊れない）"
nput apply docs
assert_symlink "$TARGET"
assert_file_eq "$TARGET/SKILL.md" "SKILLBODY"

e2e_finish
