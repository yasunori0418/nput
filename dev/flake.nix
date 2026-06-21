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
      nputLib = inputs.root.lib;
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];

      # 展開する skill を明示列挙する（mattpocock/skills の skills/ 配下の相対パス）。
      # 本来は skills/<category> を lib.listFilesInSrc で動的列挙したいが、その API は
      # 本ブランチ未実装のため、現行運用（skills-lock.json）の skill 集合を明示列挙して
      # 忠実に再現する。skills-lock.json は vercel skills 用に残置（両者は別経路）。
      skillSubpaths = [
        "engineering/grill-with-docs"
        "engineering/improve-codebase-architecture"
        "engineering/prototype"
        "engineering/setup-matt-pocock-skills"
        "engineering/tdd"
        "engineering/to-issues"
        "engineering/to-prd"
        "engineering/triage"
        "productivity/grill-me"
        "productivity/grilling"
        "productivity/handoff"
      ];

      # skill ごとに { ".claude/skills/<name>" = entry; } を組む。
      # target = .claude/skills/<skill 名>、配置元は skills/<category>/<name> の subpath。
      skillEntries = builtins.listToAttrs (
        map (p: {
          name = ".claude/skills/${baseNameOf p}";
          value = {
            src = inputs.matt-skills;
            subpath = "skills/${p}";
          };
        }) skillSubpaths
      );
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      inherit systems;
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
              # 競合時は待たず skip（--no-wait）し、no-op / 配置レポートは抑制する（--quiet）。
              nput apply skills -f "$REPO_ROOT/dev" --no-wait --quiet
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

      # nput の project mode config（dogfood）。
      # `nput apply skills -f <dev flake>` でビルドし、各 skill を .claude/skills/<name> へ
      # store-symlink 配置する。root = projectRoot（git toplevel）なので配置先は repo root 配下。
      # 配置物は .gitignore 済み（.claude/skills/*）の ephemeral。
      flake.nput = inputs.nixpkgs.lib.genAttrs systems (system: {
        skills = nputLib.mkManifest {
          pkgs = inputs.nixpkgs.legacyPackages.${system};
          root = nputLib.projectRoot;
          entries = skillEntries;
        };
      });
    };
}
