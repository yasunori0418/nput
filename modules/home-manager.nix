# home-manager module: thin wiring that kicks the nput engine from home.activation
# (→ ADR-0002, ADR-0003, ADR-0007, docs/spec.md "per-module behavior spec").
#
# It holds no placement logic and does not translate to home.file / systemd.tmpfiles. Placement and stale
# removal are owned by the nput engine common to all layers (→ ADR-0003). This module confines itself to
#   1. pinning root = homeRoot (the user does not re-specify root・→ ADR-0007),
#   2. building a link-farm (manifest.json + GC anchors) from nput.entries via mkManifest,
#   3. passing that link-farm to `nput apply --manifest` from home.activation to kick the engine.
#
# Generations ride on nput's own profile (an internal mechanism, previous-generation manifest + stale tracking). The MVP is a
# single profile with the fixed name `default` (<state>/nix/profiles/nput/default), and role separation is not possible.
# User-facing rollback is unified to the host (home-manager --rollback), and `nput rollback` is not exposed
# (host rollback re-activates the old config and re-kicks nput, automatically converging the FS)
# (→ ADR-0002, ADR-0024, ADR-0025).
#
# nputPackage (the pinned nput CLI) is injected as _module.args by flake.nix's homeManagerModules.default wiring
# (→ flake.nix). nputLib imports lib/ directly
# (depends on nixpkgs.lib only, no dependency on home-manager・→ ADR-0006).
{
  config,
  lib,
  pkgs,
  nputPackage,
  ...
}:
let
  cfg = config.nput;
  nputLib = import ../lib;

  # Generate a link-farm derivation (manifest.json + symlink farm) from nput.entries.
  # root pins homeRoot (the concrete $HOME is resolved at engine runtime; the marker is not expanded at eval time).
  manifest = nputLib.mkManifest {
    inherit pkgs;
    root = nputLib.homeRoot;
    entries = cfg.entries;
  };
in
{
  imports = [ ./common.nix ];

  config = lib.mkIf cfg.enable {
    # Kick the engine from home.activation. Since the pre-built link-farm is passed via --manifest,
    # no nix eval/build is done inside activation (this is not the entrypoint path). Run it after
    # writeBoundary so it does not collide in ordering with home.file's symlink placement. `run` is home-manager's
    # activation helper and honors --dry-run (DRY_RUN_CMD).
    home.activation.nput = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
      run ${lib.getExe nputPackage} apply --manifest ${manifest}
    '';
  };
}
