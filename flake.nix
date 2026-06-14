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
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.treefmt-nix.flakeModule
        inputs.nix-unit.modules.flake.default
      ];
      inherit systems;
      perSystem =
        {
          config,
          pkgs,
          lib,
          ...
        }:
        let
          # Go ビルド・lint の入力（go.mod + go.sum + internal/ + cmd/。docs 変更で再ビルドしないよう絞る）。
          goSrc = lib.fileset.toSource {
            root = ./.;
            fileset = lib.fileset.unions [
              ./go.mod
              ./go.sum
              ./internal
              ./cmd
            ];
          };
          # nix sandbox（ネットワーク遮断）で go ツールを回すための環境。
          # CLI 層が cobra に依存するため（→ ADR-0011）、buildGoModule が固定 hash で取得した
          # vendored deps（goModules）を vendor/ に展開し GOFLAGS=-mod=vendor でオフライン解決する。
          # build dir 直下は Go が temp root とみなし go.mod を無視するため、サブディレクトリで作業する。
          goToolEnv = ''
            export HOME="$TMPDIR"
            export GOCACHE="$TMPDIR/go-cache"
            export GOTOOLCHAIN=local
            export GOFLAGS=-mod=vendor
            export GOPROXY=off
            mkdir -p build && cd build
            cp -r --no-preserve=mode ${goSrc}/. .
            cp -r --no-preserve=mode ${config.packages.nput.goModules} vendor
          '';
        in
        {
          # nput CLI（cmd/nput）+ 配置エンジン（internal/）を含む Go モジュール（→ ADR-0006, ADR-0011）。
          # CLI 層が cobra に依存するため vendorHash 文字列を pin する（依存変更時に更新）。
          # doCheck で go test（engine の unit + tmpdir 統合テスト）を回す。
          packages.nput = pkgs.buildGoModule {
            pname = "nput";
            version = "0.0.0";
            src = goSrc;
            vendorHash = "sha256-7K17JaXFsjf163g5PXCb5ng2gYdotnZ2IDKk8KFjNj0=";
            doCheck = true;
            env.GOTOOLCHAIN = "local";
            meta = {
              description = "Place fetched git repositories at arbitrary paths via symlink or copy.";
              mainProgram = "nput";
            };
          };

          treefmt = {
            projectRootFile = "flake.nix";
            programs.nixfmt = {
              enable = true;
              package = pkgs.nixfmt;
            };
            # Go 整形（→ ADR-0025）。
            programs.gofmt.enable = true;
          };

          # 静的解析を flake check に載せる（→ ADR-0025）。stdlib-only ゆえ依存検出は軽い。
          checks.go-vet = pkgs.runCommandLocal "nput-go-vet" { nativeBuildInputs = [ pkgs.go ]; } ''
            ${goToolEnv}
            go vet ./...
            touch "$out"
          '';
          checks.golangci-lint =
            pkgs.runCommandLocal "nput-golangci-lint"
              {
                nativeBuildInputs = [
                  pkgs.go
                  pkgs.golangci-lint
                ];
              }
              ''
                ${goToolEnv}
                export GOLANGCI_LINT_CACHE="$TMPDIR/golangci-cache"
                golangci-lint run ./...
                touch "$out"
              '';
          # go test（unit + tmpdir 統合テスト）も flake check で回す。
          checks.nput = config.packages.nput;

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

        # ドッグフーディング用の project mode config（→ Issue #7・AC e2e 経路）。
        # `nput apply default` で git toplevel 配下の .nput-example/docs に
        # 本 repo（self）の docs を store-symlink 配置する最小 example。
        # `nput.<system>.<name>` は標準 flake output ではないため `nix flake check` で
        # `warning: unknown flake output 'nput'`（exit 0・想定内）が出る（→ docs/spec.md）。
        nput = inputs.nixpkgs.lib.genAttrs systems (system: {
          default = nputLib.mkManifest {
            pkgs = inputs.nixpkgs.legacyPackages.${system};
            root = nputLib.projectRoot;
            entries.".nput-example/docs" = {
              src = inputs.self;
              subpath = "docs";
            };
          };
        });

        homeManagerModules.default = ./modules/home-manager.nix;
        nixosModules.default = ./modules/nixos.nix;
        darwinModules.default = ./modules/nix-darwin.nix;
      };
    };
}
