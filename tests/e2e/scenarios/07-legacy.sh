#!/usr/bin/env bash
# legacy entrypoint (shell.nix, passthru canonical form): `nput apply` / `apply --all` /
# plain `nix-shell` compatibility (→ ADR-0032).
set -euo pipefail
source "$(dirname "$0")/../lib.sh"
e2e_isolate
e2e_pin_nix_path

PROJ="$E2E_WORK/proj"
mkdir -p "$PROJ/srcrepo/skills/nix"
echo "SKILLBODY" >"$PROJ/srcrepo/skills/nix/SKILL.md"

cat >"$PROJ/shell.nix" <<EOF
{ pkgs ? import <nixpkgs> {}
, nput ? { lib = import $REPO_ROOT/lib; }
}:
pkgs.mkShell {
  packages = [ ];
  shellHook = "nput apply docs --no-wait";
  passthru.nput = {
    docs = nput.lib.mkManifest {
      inherit pkgs;
      root = nput.lib.projectRoot;
      entries.".nput-out/docs" = { src = ./srcrepo; subpath = "skills/nix"; };
    };
    extra = nput.lib.mkManifest {
      inherit pkgs;
      root = nput.lib.projectRoot;
      entries.".nput-out/extra" = { src = ./srcrepo; subpath = "skills/nix"; };
    };
  };
}
EOF

cd "$PROJ"
git init -q
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm init

e2e_step "nput apply docs（legacy shell.nix・passthru 形）"
nput apply docs

e2e_step "git toplevel 配下に配置されたか"
TARGET="$PROJ/.nput-out/docs"
assert_symlink "$TARGET"
assert_file_eq "$TARGET/SKILL.md" "SKILLBODY"
# flake（01-project.sh）と異なり、legacy -f eval には flake の事前 store コピーが無いため、素の相対
# path リテラル `src = ./srcrepo` は toString でも store へコピーされず生の作業木パスのまま解決される
# （→ ADR-0007 §5「impure eval を許容」の具体的帰結。reproducible にしたいユーザーは builtins.path /
# fetchTarball 等で明示的に store 化する）。よってここでは store symlink であることまでは assert しない。

e2e_step "nput apply --all（passthru.nput.* を一括適用）"
nput apply --all
assert_symlink "$PROJ/.nput-out/extra"
assert_file_eq "$PROJ/.nput-out/extra/SKILL.md" "SKILLBODY"

e2e_step "素の nix-shell がそのまま動く（passthru に壊されない）＋ shellHook が apply を kick する"
rm -rf "$PROJ/.nput-out"
nix-shell --run true "$PROJ/shell.nix"
assert_symlink "$TARGET"
assert_file_eq "$TARGET/SKILL.md" "SKILLBODY"

e2e_finish
