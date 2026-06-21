# flake-parts module: perSystem.nput.<name> を flake.nput.<system>.<name> へ転置する（→ ADR-0029）。
# flake-parts 公式の mkTransposedPerSystemModule（packages / legacyPackages と同じ転置機構）を流用し、
# 自前の mkPerSystemOption + transposition は手書きしない（ADR-0029 残作業1）。
#
# option は lazyAttrsOf package。consumer は perSystem で
#   nput.<name> = inputs.nput.lib.mkManifest { inherit pkgs; root = ...; entries = { ... }; };
# と書く。転置先 flake.nput.<system>.<name> は CLI が `nix build .#nput.<system>.<name>` で叩く
# build 可能な derivation でなければならない（mkManifest の結果＝derivation を格納する・ADR 決定2）。
#
# module は純粋な transposition のみ。mkManifest やマーカー群を perSystem 引数に注入しない
# （ADR 決定5）。consumer は同じ input の nput.lib を直接参照する。
{ lib, flake-parts-lib, ... }:
let
  inherit (lib)
    mkOption
    types
    ;
  inherit (flake-parts-lib)
    mkTransposedPerSystemModule
    ;
in
mkTransposedPerSystemModule {
  name = "nput";
  option = mkOption {
    type = types.lazyAttrsOf types.package;
    default = { };
    description = ''
      nput の named manifest（`nput.lib.mkManifest` の結果＝derivation）を公開する属性集合（→ ADR-0007, ADR-0029）。

      `perSystem.nput.<name>` に宣言すると top-level の `flake.nput.<system>.<name>` へ自動転置され、
      CLI が `nix build .#nput.<system>.<name>` で叩く build 可能な derivation になる。

      ```nix
      perSystem = { pkgs, ... }: {
        nput.default = inputs.nput.lib.mkManifest {
          inherit pkgs;
          root = inputs.nput.lib.projectRoot;
          entries.".config/foo" = { src = inputs.foo; subpath = "."; };
        };
      };
      ```
    '';
  };
  file = ./flake-parts.nix;
}
