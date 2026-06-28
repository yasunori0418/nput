# Private helpers extracted from manifest.nix for unit-test reachability (→ #71).
#
# NOT part of the public API. Reached via `nput.__internal.<name>` and used internally by
# manifest.nix. Each helper takes nixpkgs.lib explicitly because the nput lib attrset is
# unparameterized (`import ./lib` with no args), so we cannot pre-bind lib at this layer.
let
  # Determine whether following `..` makes the depth go negative (escapes outside base) (→ ADR-0019).
  escapesBase =
    lib: p:
    let
      comps = lib.filter (c: c != "" && c != ".") (lib.splitString "/" p);
      step =
        acc: c:
        if acc.bad then
          acc
        else if c == ".." then
          {
            bad = acc.depth == 0;
            depth = if acc.depth == 0 then 0 else acc.depth - 1;
          }
        else
          {
            bad = false;
            depth = acc.depth + 1;
          };
    in
    (lib.foldl' step {
      bad = false;
      depth = 0;
    } comps).bad;

  # target is root-relative, subpath is relative within src. Absolute paths (leading `/`) and
  # paths that escape outward via `..` are rejected at eval time (→ ADR-0019).
  pathChecks = lib: {
    isUnsafe = p: lib.hasPrefix "/" p || escapesBase lib p;
  };

  # GC anchor name for the symlink farm = sha256 short hex of target (fixed length, FS-safe, collision-free・→ ADR-0016).
  anchorName = lib: target: lib.substring 0 32 (builtins.hashString "sha256" target);

  # entry marker tag → clean enum + resolved src string (→ ADR-0010).
  resolveEntry =
    lib: e:
    let
      t = import ./types.nix lib;
      srcInfo =
        if t.isOutOfStoreMarker e.src then
          {
            srcKind = "outOfStore";
            src = e.src.path;
          }
        else
          {
            srcKind = "store";
            src = toString e.src;
          };
    in
    {
      inherit (srcInfo) srcKind src;
      inherit (e) subpath target method;
    };

  # Farm anchors are limited to entries that are "store-backed and method = symlink" (→ ADR-0016, ADR-0019).
  # out-of-store / copy have no farm anchor (copy is out-of-generation, place-once, and independent of the store).
  farmEntries = lib: entries: lib.filter (e: e.srcKind == "store" && e.method == "symlink") entries;
in
{
  inherit
    escapesBase
    pathChecks
    anchorName
    resolveEntry
    farmEntries
    ;
}
