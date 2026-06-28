# normalizeManifest (pure data, validation gate) + mkManifest (derivation generation) (→ ADR-0006, ADR-0010, ADR-0016, ADR-0019, ADR-0023).
#
# Split into two stages (→ ADR-0010):
#   - normalizeManifest { lib, root, entries } -> attrset
#       evalModules validation, default application, path safety / cross-field throwIf, marker tag → clean enum conversion.
#       Kept outside the derivation so it becomes a unit-test target for nix-unit / namaka.
#   - mkManifest { pkgs, root, entries } -> derivation
#       Writes normalizeManifest's output to manifest.json and builds a symlink farm to the store src.
#
# Minimal scope of this slice: root = projectRoot only / src = store-backed symlink entries of path/set only
# (→ Issue #5). Types and throwIf are defined in full form, anticipating future slices (home / copy / out-of-store).
#
# Private helpers (escapesBase / pathChecks / anchorName / resolveEntry / farmEntries) live in
# ./__internal.nix so they stay unit-test reachable via `nput.__internal.<name>` (→ #71).
let
  internal = import ./__internal.nix;

  normalizeManifest =
    {
      lib,
      root,
      entries ? { },
    }:
    let
      t = import ./types.nix lib;
      checks = internal.pathChecks lib;

      # mkManifest itself runs evalModules so validation applies on both paths (direct CLI call / module) (→ ADR-0010).
      evaluated = lib.evalModules {
        modules = [
          {
            options = {
              root = lib.mkOption { type = t.rootType; };
              entries = lib.mkOption {
                type = t.entriesType;
                default = { };
              };
            };
            config = { inherit root entries; };
          }
        ];
      };
      cfg = evaluated.config;

      # root marker tag → clean enum (→ ADR-0010).
      rootInfo =
        if t.isRootMarker cfg.root then
          { rootKind = cfg.root.kind; }
        else
          {
            rootKind = "fixed";
            root = cfg.root;
          };

      # entry marker tag → clean enum + resolved src string (→ ADR-0010).
      # Since the attribute key = target, serialize to an array deterministically in attrNames lexical order (Go reads the array・→ ADR-0014).
      normEntries = map (key: internal.resolveEntry lib cfg.entries.${key}) (lib.attrNames cfg.entries);

      targets = map (e: e.target) normEntries;

      # ---- Cross-field / path validation (→ ADR-0013, ADR-0019, ADR-0024)-------
      assertions = lib.concatLists [
        # systemRoot is not implemented (→ ADR-0013).
        (lib.optional (
          rootInfo.rootKind == "system"
        ) "nput: root = systemRoot (system mode) is not implemented (→ ADR-0013)")
        # method = "copy" combined with an out-of-store marker is a contradiction of intent (→ ADR-0013).
        (map (
          e:
          "nput: method = \"copy\" cannot be combined with an out-of-store marker (target: ${e.target}; → ADR-0013)"
        ) (lib.filter (e: e.method == "copy" && e.srcKind == "outOfStore") normEntries))
        # Collision from explicitly overriding target to the same value under a different key (→ ADR-0024).
        (lib.optional (
          lib.length targets != lib.length (lib.unique targets)
        ) "nput: multiple entries resolve to the same target (→ ADR-0024)")
        # Reject absolute paths / `..` escapes in target / subpath (→ ADR-0019).
        (map (
          e: "nput: invalid target (absolute path or escapes root via `..`): ${e.target} (→ ADR-0019)"
        ) (lib.filter (e: checks.isUnsafe e.target) normEntries))
        (map (
          e:
          "nput: invalid subpath (absolute path or escapes src via `..`): ${e.subpath} (target: ${e.target}; → ADR-0019)"
        ) (lib.filter (e: checks.isUnsafe e.subpath) normEntries))
      ];

      result = {
        schemaVersion = 1;
        root = rootInfo;
        entries = normEntries;
      };
    in
    # Run every assertion through the evaluation gate with throwIf.
    lib.foldl' (acc: msg: lib.throwIf true msg acc) result assertions;

  mkManifest =
    {
      pkgs,
      root,
      entries ? { },
    }:
    let
      lib = pkgs.lib;
      norm = normalizeManifest { inherit lib root entries; };

      manifestJson = pkgs.writeText "manifest.json" (builtins.toJSON norm);

      # Farm anchors are limited to entries that are "store-backed and method = symlink" (→ ADR-0016, ADR-0019).
      # out-of-store / copy have no farm anchor (copy is out-of-generation, place-once, and independent of the store).
      farmEntries = internal.farmEntries lib norm.entries;

      anchorLines = lib.concatMapStringsSep "\n" (
        e: "ln -s ${lib.escapeShellArg e.src} \"$out/${internal.anchorName lib e.target}\""
      ) farmEntries;
    in
    # The derivation contains manifest.json (the engine's input contract) + a symlink farm to the store src (GC anchors) (→ ADR-0006).
    pkgs.runCommandLocal "nput-manifest"
      {
        # The CLI reads this via `nix eval … .rootKind` before build (→ ADR-0023).
        passthru = {
          inherit (norm.root) rootKind;
        }
        // lib.optionalAttrs (norm.root ? root) { inherit (norm.root) root; };
      }
      ''
        mkdir -p "$out"
        cp ${manifestJson} "$out/manifest.json"
        ${anchorLines}
      '';
in
{
  inherit normalizeManifest mkManifest;
}
