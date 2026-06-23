{
  description = "nput standalone config: フェッチ済み git リポジトリを home 配下へ symlink / copy 配置する";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    # nput ライブラリ・CLI。nixpkgs を follows させて mkManifest が使う pkgs を揃える。
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

            # home 配下を root に取る（target は $HOME 相対）。
            root = nput.lib.homeRoot;

            # 属性キー = 配置先（root 相対 target）。ここでは nput 自身の docs/ を例示する。
            # 実用では src を上の my-repo input へ、subpath を配置したいサブディレクトリへ差し替える。
            entries.".config/nput-docs" = {
              src = nput;
              subpath = "docs";
            };

            # ---- バリエーション例 ------------------------------------------------
            #
            # subpath 省略 = リポジトリ全体を配置:
            #   entries.".config/my-repo" = { src = my-repo; };
            #
            # method = "copy" = symlink ではなく通常ファイルとして配置（書込可・place-once）:
            #   entries.".config/editable" = {
            #     src = my-repo;
            #     subpath = "config";
            #     method = "copy";
            #   };
            #
            # out-of-store symlink = store ではなくローカルの絶対パスへ直接 symlink:
            #   entries.".config/live" = {
            #     src = nput.lib.mkOutOfStoreSymlink "/abs/path/to/dir";
            #   };
            #
            # 複数 entry = 属性をそのまま増やす:
            #   entries.".config/a" = { src = my-repo; subpath = "a"; };
            #   entries.".config/b" = { src = my-repo; subpath = "b"; };
            #
            # 動的 entry 化 = 既 realise の store パス / flake input を builtins.readDir で
            # 走査して entry を組む（IFD 回避のため生 derivation / out-of-store marker は readDir 不可）:
            #   entries = builtins.mapAttrs
            #     (name: _: { src = my-repo; subpath = "skills/${name}"; })
            #     (builtins.readDir "${my-repo}/skills");
          };
        }
      );
    };
}
