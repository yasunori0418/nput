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

    # niface エンベロープの E2E 適合検証キット（niface-validate CLI + id-vectors）。
    # 規格由来の検証依存として許容する（→ issue #132・niface ADR-0021 / 0023 / 0025）。
    niface = {
      url = "github:yasunori0418/niface";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-parts.follows = "flake-parts";
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
        let
          # niface の Go 参照実装は subPackages 由来の bin/validate を生むため、
          # 規格上の CLI 名 niface-validate で PATH に載せる薄い wrapper を挟む。
          niface-validate = pkgs.writeShellScriptBin "niface-validate" ''
            exec ${inputs'.niface.packages.validate}/bin/validate "$@"
          '';
        in
        {
          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              statix
              nixd
              inputs'.root.formatter
              inputs'.root.packages.nput
              go
              gopls
              # ローカルでカバレッジ計測する coverage ツール（go test -coverprofile + go tool cover）。
              # func サマリを出し、HTML を見たい場合のコマンドを案内する（閾値ゲートは持たない）。
              (writeShellScriptBin "nput-coverage" ''
                set -euo pipefail
                profile="cover.out"
                go test -coverprofile="$profile" ./...
                go tool cover -func="$profile"
                echo "HTML レポート: go tool cover -html=$profile"
              '')
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
              # --json エンベロープの適合検証（schema〔format assertion 込み〕+ lint MUST・→ issue #132）。
              niface-validate
            ];
            env.TERM = "dumb";
            # E2E の id-vectors 整合チェックが参照する適合ベクタ（niface testdata の正本）。
            env.NIFACE_ID_VECTORS = "${inputs.niface}/testdata/v1/id-vectors.json";
          };
        };
    };
}
