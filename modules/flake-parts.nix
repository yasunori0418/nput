# flake-parts module: transpose perSystem.nput.<name> to flake.nput.<system>.<name> (→ ADR-0029).
# Reuse flake-parts' official mkTransposedPerSystemModule (the same transposition mechanism as packages / legacyPackages),
# rather than hand-writing our own mkPerSystemOption + transposition (ADR-0029 remaining task 1).
#
# The option is lazyAttrsOf package. The consumer writes, in perSystem,
#   nput.<name> = inputs.nput.lib.mkManifest { inherit pkgs; root = ...; entries = { ... }; };
# The transposition target flake.nput.<system>.<name> must be a buildable derivation that the CLI invokes
# via `nix build .#nput.<system>.<name>` (it stores the result of mkManifest = a derivation・ADR decision 2).
#
# The module does pure transposition only. It does not inject mkManifest or the markers into perSystem arguments
# (ADR decision 5). The consumer references nput.lib of the same input directly.
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
