{
  description = "Place fetched git repositories at arbitrary paths via symlink or copy.";

  # nput 自体を clone して nix develop / build / flake check する際に cachix からビルド済み
  # バイナリを引くための設定（trusted-user / accept-flake-config 前提）。flake の nixConfig は
  # input に伝播しないため、nput を flake input として消費する側のキャッシュ取得には効かない。
  nixConfig = {
    extra-substituters = [
      "https://cache.nixos.org/"
      "https://nix-community.cachix.org"
      "https://yasunori0418.cachix.org"
    ];
    extra-trusted-public-keys = [
      "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
      "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
      "yasunori0418.cachix.org-1:mC1j+M5A6063OHaOB5bH2nS0BiCW/BJsSRiOWjLeV9o="
    ];
  };

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

    # HM モジュール統合の評価テスト専用（→ Issue #17・checks.hm-module）。
    # lib/ は home-manager に依存しない（→ ADR-0006）。本 input は checks でのみ使う。
    home-manager = {
      url = "github:nix-community/home-manager";
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
            # cgo 未使用。サンドボックスに C コンパイラを持ち込まずピュア Go で検査する。
            export CGO_ENABLED=0
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

          # HM モジュール統合の評価アサート（→ Issue #17 AC・NixOS VM / 実 activate は #19 E2E）。
          # standalone な homeManagerConfiguration を評価し、(1) activation が home.file へ翻訳せず
          # `nput apply --manifest` で engine を起動する配線であること、(2) 渡す manifest が
          # root=homeRoot を pin すること、(3) nput.entries が manifest に流れることをアサートする。
          # 実 activate（nix-env --set・FS 配置）は build sandbox では行えないため E2E（#19）へ回す。
          checks.hm-module =
            let
              # store hash 揺れを避ける fake な flake-input 相当（nix-unit / namaka と同じ test double）。
              fakeSrc = {
                outPath = "/nix/store/00000000000000000000000000000000-fake-src";
              };
              hm = inputs.home-manager.lib.homeManagerConfiguration {
                inherit pkgs;
                modules = [
                  ./modules/home-manager.nix
                  {
                    # ラッパー（flake.homeManagerModules.default）と同じく pin 版 nput を注入する。
                    _module.args.nputPackage = config.packages.nput;
                    home.username = "nput-test";
                    home.homeDirectory = "/home/nput-test";
                    home.stateVersion = "24.05";
                    # nixpkgs=unstable と HM=master の release 文字列ずれによる無害な
                    # warning を抑制する（packages は nixpkgs follows で一致・→ #17 レビュー）。
                    home.enableNixpkgsReleaseCheck = false;
                    nput.enable = true;
                    nput.entries.".claude/skills/nix" = {
                      src = fakeSrc;
                      subpath = "skills/nix";
                    };
                  }
                ];
              };
              # home.activation の dag entry の生スクリプト。
              activationScript = pkgs.writeText "nput-activation" hm.config.home.activation.nput.data;
            in
            pkgs.runCommandLocal "nput-hm-module-check" { } ''
              script=${activationScript}

              # (1) home.file へ翻訳せず engine を --manifest 経路で起動する配線であること。
              grep -q 'apply --manifest /nix/store/' "$script" \
                || { echo "FAIL: activation が nput apply --manifest を起動していません"; cat "$script"; exit 1; }

              # (2) 渡す manifest が root=homeRoot を pin していること（mkManifest が記録）。
              manifest=$(grep -oE '/nix/store/[a-z0-9]+-nput-manifest' "$script" | head -n1)
              test -n "$manifest" || { echo "FAIL: manifest の store パスを抽出できません"; cat "$script"; exit 1; }
              test -f "$manifest/manifest.json" || { echo "FAIL: $manifest/manifest.json がありません"; exit 1; }
              grep -q '"rootKind":"home"' "$manifest/manifest.json" \
                || { echo "FAIL: manifest が homeRoot を pin していません"; cat "$manifest/manifest.json"; exit 1; }

              # (3) nput.entries が manifest に流れていること（target = 属性キー）。
              grep -q '".claude/skills/nix"' "$manifest/manifest.json" \
                || { echo "FAIL: nput.entries が manifest に反映されていません"; cat "$manifest/manifest.json"; exit 1; }

              touch "$out"
            '';
        };
      flake = {
        lib = nputLib;

        # `nput init <template>` / `nix flake init -t <ref>#<template>` で展開する starter テンプレ。
        # default = project（spec が project mode を canonical と明記・最も完備した例を渡す）。
        templates = {
          standalone = {
            path = ./templates/standalone;
            description = "nput standalone config（homeRoot 例 + バリエーションコメント）";
          };
          project = {
            path = ./templates/project;
            description = "nput project config（projectRoot + devShell + shellHook + .gitignore）";
          };
          default = inputs.self.templates.project;
        };

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

        # HM モジュール本体（modules/home-manager.nix）は engine をキックするのに pin 版 nput
        # CLI を要する。利用者システムの packages.nput を _module.args として注入する薄い
        # ラッパーで包む（利用者は import するだけ・→ ADR-0007, modules/home-manager.nix）。
        homeManagerModules.default =
          { pkgs, ... }:
          {
            imports = [ ./modules/home-manager.nix ];
            _module.args.nputPackage = inputs.self.packages.${pkgs.stdenv.hostPlatform.system}.nput;
          };
        nixosModules.default = ./modules/nixos.nix;
        darwinModules.default = ./modules/nix-darwin.nix;
      };
    };
}
