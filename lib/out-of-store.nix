# Marker constructors (→ ADR-0001, ADR-0004, ADR-0005, ADR-0007, ADR-0010).
#
# A marker is a "container that carries the kind for runtime resolution", not syntactic sugar that returns a path string.
# Since both `src` (a derivation) and a marker are attrsets and cannot be distinguished structurally, a marker
# carries a `_nputMarker` discriminator tag, and the `check` of the custom optionType in `lib/types.nix` distinguishes them.
# `_nputMarker` stays within Nix evaluation and is never leaked into `manifest.json` (the Go contract is a clean enum・→ ADR-0010).
#
# No dependencies (pure attrset constructors that do not even require nixpkgs.lib).
{
  # Marker representing an out-of-store symlink to a local path (→ ADR-0001).
  # The argument is an absolute path string fixed at Nix eval time. Actual link creation is the engine's runtime responsibility.
  mkOutOfStoreSymlink = path: {
    _nputMarker = "outOfStore";
    inherit path;
  };

  # root markers (→ ADR-0004 revised, ADR-0005, ADR-0007). They merely carry kind; concrete path resolution is the engine's runtime responsibility.
  projectRoot = {
    _nputMarker = "root";
    kind = "project";
  };
  homeRoot = {
    _nputMarker = "root";
    kind = "home";
  };
  systemRoot = {
    _nputMarker = "root";
    kind = "system";
  };
}
