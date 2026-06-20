#!/usr/bin/env bash
# HM module: home-manager standalone configuration を非 NixOS で評価・activate し、
# activation（home.activation.nput）が engine を起動して \$HOME 配下へ配置することをアサート。
# モジュール経路は CLI と mkManifest が同一 flake input 由来で schemaVersion skew が起きない（→ ADR-0026）。
set -euo pipefail
source "$(dirname "$0")/../lib.sh"
e2e_isolate

# standalone HM の activate は configured username == \$USER を要求するため実ユーザー名を使う。
USERNAME="$(id -un)"

PROJ="$E2E_WORK/cfg"
mkdir -p "$PROJ/srcrepo/skills/nix"
echo "SKILLBODY" >"$PROJ/srcrepo/skills/nix/SKILL.md"

cat >"$PROJ/flake.nix" <<EOF
{
$(e2e_flake_inputs with-hm)
  outputs = { self, nixpkgs, home-manager, nput }:
    let
      system = "$E2E_SYSTEM";
      pkgs = nixpkgs.legacyPackages.\${system};
    in {
      homeConfigurations.e2e = home-manager.lib.homeManagerConfiguration {
        inherit pkgs;
        modules = [
          nput.homeManagerModules.default
          {
            home.username = "$USERNAME";
            home.homeDirectory = "$HOME";
            home.stateVersion = "24.05";
            home.enableNixpkgsReleaseCheck = false;
            nput.enable = true;
            nput.entries.".cfg/skill" = { src = ./srcrepo; subpath = "skills/nix"; };
          }
        ];
      };
    };
}
EOF

cd "$PROJ"
git init -q
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm init

e2e_step "HM standalone activationPackage をビルド"
ACT="$(nix build ".#homeConfigurations.e2e.activationPackage" --no-link --print-out-paths)"
e2e_log "activationPackage: $ACT"

e2e_step "activate を実行（home.activation.nput が engine を起動）"
"$ACT/activate"

e2e_step "HM 経由で \$HOME 配下に配置されたか"
assert_symlink "$HOME/.cfg/skill"
assert_file_eq "$HOME/.cfg/skill/SKILL.md" "SKILLBODY"
case "$(readlink "$HOME/.cfg/skill")" in
	/nix/store/*) e2e_pass "store symlink を指す" ;;
	*) e2e_fail "store symlink を指すべき: $(readlink "$HOME/.cfg/skill")" ;;
esac

e2e_finish
