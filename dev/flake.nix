{
  description = "nput development environment";

  inputs = {
    root.url = "path:../";
    nixpkgs.follows = "root/nixpkgs";
    flake-parts.follows = "root/flake-parts";

    # Claude Code 用スキル集（mattpocock/skills）。
    # 従来は vercel の skills コマンド + skills-lock.json で .claude/skills/ に展開していたが、
    # nput のドッグフーディングとして project mode の nput apply で配置する（flake.lock が rev を pin）。
    matt-skills = {
      url = "github:mattpocock/skills";
      flake = false;
    };
  };

  outputs =
    inputs@{ flake-parts, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      inherit systems;
      imports = [
        inputs.root.flakeModules.default
        # nput dogfood config（perSystem.nput.skills）を flake-parts module として切り出す。
        ./nput.nix
      ];
      perSystem =
        { inputs', pkgs, ... }:
        {
          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              statix
              nixd
              inputs'.root.formatter
              inputs'.root.packages.nput
              gopls
            ];
            shellHook = ''
              export REPO_ROOT=$(git rev-parse --show-superproject-working-tree --show-toplevel)
              # mattpocock/skills を .claude/skills/ に dogfood 配置する（project mode）。
              # 競合時は待たず skip（--no-wait）し、no-op
              nput apply skills -f "$REPO_ROOT/dev" --no-wait
            '';
          };

          # 非 NixOS E2E ハーネス（tests/e2e/run.sh）専用の最小 CI シェル（→ ADR-0012 §2）。
          # dev 専用ツール（statix / nixd / gopls 等）と dogfood の shellHook を持たず、
          # ハーネスが要する nput バイナリ + bash / git / jq / coreutils だけを提供する。
          # nix / nix-env は install-nix-action が入れた ambient nix を使う（pkgs.nix を載せて
          # 上書きしない）。TERM=dumb で対話 UI を抑える。
          devShells.ci = pkgs.mkShell {
            packages = with pkgs; [
              inputs'.root.packages.nput
              bash
              git
              jq
              coreutils
            ];
            env.TERM = "dumb";
          };
        };
    };
}
