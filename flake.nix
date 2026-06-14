{
  description = "Place fetched git repositories at arbitrary paths via symlink or copy.";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      # nix-unit の flake-parts モジュールは nixpkgs-lib follows を要求する。
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    # lib 評価テスト（→ ADR-0006, ADR-0012）。
    nix-unit = {
      url = "github:nix-community/nix-unit";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    haumea = {
      url = "github:nix-community/haumea";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    namaka = {
      url = "github:nix-community/namaka";
      inputs.haumea.follows = "haumea";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ flake-parts, ... }:
    let
      # flake output（lib）とテスト入力で同一実体を共有する（self 参照を避ける）。
      nputLib = import ./lib;
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.treefmt-nix.flakeModule
        inputs.nix-unit.modules.flake.default
      ];
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
      perSystem =
        { pkgs, ... }:
        {
          treefmt = {
            projectRootFile = "flake.nix";
            programs.nixfmt = {
              enable = true;
              package = pkgs.nixfmt;
            };
          };

          # nix-unit: デフォルト適用・manifest 構造の不変条件をアサート（→ ADR-0006, ADR-0010）。
          # flake-parts モジュールが checks 派生を組み `nix flake check` に載せる。
          # check は sandbox 内で `nix-unit --flake ${self}#tests.systems.<system>` を回し flake を
          # 再 import するため、全 direct input を override-input でローカルに渡しオフライン評価する。
          nix-unit.inputs = {
            inherit (inputs)
              nixpkgs
              flake-parts
              treefmt-nix
              nix-unit
              haumea
              namaka
              ;
          };
          nix-unit.tests = import ./tests/nix-unit.nix {
            inherit (pkgs) lib;
            nput = nputLib;
          };

          # namaka: manifest.json 全体（= normalizeManifest 出力）のスナップショット回帰（→ ADR-0006）。
          # namaka.lib.load は不一致で throw・成功で {} を返す純評価。seq で評価を強制し
          # check 派生に紐付けて `nix flake check` に載せる。
          checks.namaka = builtins.seq (inputs.namaka.lib.load {
            src = ./tests/namaka;
            inputs = {
              inherit (pkgs) lib;
              nput = nputLib;
            };
          }) (pkgs.runCommandLocal "nput-namaka-snapshots" { } "touch \"$out\"");
        };
      flake = {
        lib = nputLib;
        homeManagerModules.default = ./modules/home-manager.nix;
        nixosModules.default = ./modules/nixos.nix;
        darwinModules.default = ./modules/nix-darwin.nix;
      };
    };
}
