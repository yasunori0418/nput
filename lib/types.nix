# entry submodule + srcType / rootType / marker custom type (→ ADR-0010, ADR-0014).
#
# Written as a pure function that takes nixpkgs.lib and returns the full set of type definitions,
# so it can be shared by `mkManifest`'s `evalModules` and `modules/common.nix`'s `attrsOf (submodule …)`.
# `lib.types` / `mkOption` / `evalModules` are core to nixpkgs.lib, so this satisfies
# "lib depends on nixpkgs.lib only" (no dependency on home-manager / NixOS / nix-darwin).
lib:
let
  inherit (lib) types mkOption mkDefault;
  inherit (lib.options) mergeEqualOption;
  inherit (builtins) isAttrs isString;

  # Marker discrimination (distinguished by the `_nputMarker` tag attached by `out-of-store.nix`).
  isOutOfStoreMarker = x: isAttrs x && (x._nputMarker or null) == "outOfStore";
  isRootMarker = x: isAttrs x && (x._nputMarker or null) == "root";

  # store-backed src: collapse path / derivation / flake input (`{ outPath = …; }`) into a single branch.
  # Reject bare strings and forbid the implicit out-of-store branch at the type level (→ ADR-0001).
  # path and set behave identically (both are store links), so they are not split at the type level (→ ADR-0010).
  isStoreBacked =
    x:
    lib.isPath x || lib.isDerivation x || (isAttrs x && x ? outPath && (x._nputMarker or null) == null);

  # srcType = either storeBacked outOfStoreMarker (→ ADR-0010).
  srcType = types.mkOptionType {
    name = "nputSrc";
    description = "store-backed source (path / derivation / flake input) or out-of-store marker";
    check = x: isStoreBacked x || isOutOfStoreMarker x;
    merge = mergeEqualOption;
  };

  # rootType = either str rootMarker (→ ADR-0010). Used only by `mkManifest`, not shared with modules
  # (modules pin root・→ ADR-0003).
  rootType = types.mkOptionType {
    name = "nputRoot";
    description = "absolute path string or root marker (projectRoot / homeRoot / systemRoot)";
    check = x: isString x || isRootMarker x;
    merge = mergeEqualOption;
  };

  # entry submodule (→ ADR-0014). The attribute key = target is the identifier. strict (unknown keys rejected).
  entryModule =
    { name, ... }:
    {
      options = {
        src = mkOption {
          type = srcType;
          description = "Placement source. A store-backed value (path / set) or an out-of-store marker.";
        };
        subpath = mkOption {
          type = types.str;
          default = ".";
          description = "Relative path inside src. Omitted = the whole repository (→ ADR-0008).";
        };
        target = mkOption {
          type = types.str;
          # Default = attribute key (→ ADR-0014).
          default = name;
          defaultText = "attribute key";
          description = "Placement target relative to root. Defaults to the attribute key when omitted.";
        };
        method = mkOption {
          type = types.enum [
            "symlink"
            "copy"
          ];
          default = "symlink";
          description = "Placement method (formerly named mode; → ADR-0015).";
        };
      };
    };

  entriesType = types.attrsOf (types.submodule entryModule);
in
{
  inherit
    isOutOfStoreMarker
    isRootMarker
    isStoreBacked
    srcType
    rootType
    entryModule
    entriesType
    ;
}
