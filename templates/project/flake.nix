{
  description = "nput project config: フェッチ済み git リポジトリを repo 配下へ symlink / copy 配置する";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    # nput ライブラリ・CLI。nixpkgs を follows させて mkManifest の pkgs と
    # devShell に載せる nput CLI のビルド入力を揃える（schemaVersion 整合）。
    nput = {
      url = "github:yasunori0418/nput";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    # 実用では「配置したい git リポジトリ」を flake = false の input として宣言し、
    # 下の entries の src をこの input へ差し替える（add me）:
    #   my-repo = {
    #     url = "github:you/your-repo";
    #     flake = false;
    #   };
  };

  outputs =
    {
      self,
      nixpkgs,
      nput,
      ...
    }:
    let
      # 4 system に同じ config を展開するヘルパ（flake-parts は starter には過剰なので不使用）。
      forAllSystems = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
    in
    {
      # nput.<system>.<name> namespace。`nput apply <name>` がこの config をビルドして配置する。
      # 標準 flake output ではないため `nix flake check` で
      # `warning: unknown flake output 'nput'`（exit 0・無害）が出る。
      nput = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          # config 名 example は "rename me" の合図。`nput apply example` で配置する。
          example = nput.lib.mkManifest {
            inherit pkgs;

            # repo（git toplevel）配下を root に取る（target は repo ルート相対）。
            root = nput.lib.projectRoot;

            # 属性キー = 配置先（root 相対 target）。ここでは nput 自身の docs/ を例示する。
            # 実用では src を上の my-repo input へ、subpath を配置したいサブディレクトリへ差し替える。
            #
            # 配置物は ephemeral（git 管理せず再生成する前提）。下の .gitignore に無視パターンを
            # 追記済み。target を変えたら `nput gitignore example` の出力で更新する。
            entries.".nput/docs" = {
              src = nput;
              subpath = "docs";
            };
          };
        }
      );

      # devShell。`nix develop` / direnv 入室時に pin 済み nput CLI を PATH に載せ、
      # shellHook で config を自動配置する。.envrc は同梱しない（direnv 導入は利用者判断・ADR-0018）。
      # direnv を使うなら repo に `echo 'use flake' > .envrc && direnv allow` を実行する。
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = [ nput.packages.${system}.nput ];

            # 入室時に example を配置する。--no-wait で flock 競合時は待たず skip（多重入室で固まらない）。
            # 複数 config を一括配置するなら:
            #   nput apply --all --project-root --no-wait
            shellHook = ''
              nput apply example --no-wait
            '';
          };
        }
      );
    };
}
