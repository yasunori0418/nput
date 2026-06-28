# Summary of the nput lib public API (→ docs/design.md "flake.nix outputs design").
#
# Depends on nixpkgs.lib only (no dependency on home-manager / NixOS / nix-darwin・→ ADR-0006).
# The markers are pure attrset constructors with no dependencies. mkManifest takes `pkgs`
# from its argument attrset to obtain the linkFarm equivalent (runCommandLocal) and `pkgs.lib`
# to build the derivation.
let
  markers = import ./out-of-store.nix;
  manifest = import ./manifest.nix;
in
{
  # Private helpers (escapesBase / pathChecks / anchorName / resolveEntry / farmEntries) exposed
  # solely as a unit-test seam (→ #71). NOT a stable public API; the `__internal` name marks intent.
  # Each helper takes nixpkgs.lib explicitly (the nput lib attrset is unparameterized).
  __internal = import ./__internal.nix;

  # lib.mkManifest { pkgs, root, entries } -> derivation (manifest.json + symlink farm・→ ADR-0006, ADR-0023)
  inherit (manifest) mkManifest;

  # Pure data functions for validation/normalization. Unit-test target for nix-unit / namaka, plus future reuse from modules (→ ADR-0010).
  # normalizeManifest { lib, root, entries } -> attrset
  inherit (manifest) normalizeManifest;

  # lib.mkOutOfStoreSymlink "/abs/path" -> marker (passed to src・→ ADR-0001)
  # lib.projectRoot / homeRoot / systemRoot -> marker (passed to root・→ ADR-0004, ADR-0005, ADR-0007)
  inherit (markers)
    mkOutOfStoreSymlink
    projectRoot
    homeRoot
    systemRoot
    ;
}
