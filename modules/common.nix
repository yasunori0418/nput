# nput option definitions common to all modules (→ ADR-0003, ADR-0007, ADR-0010, ADR-0014).
#
# The common options imported by HM / NixOS / nix-darwin. They hold no placement logic and
# only let the user declare the data (entries) of "what, where, and how to place". Each module
# pins root according to its own nature (HM → homeRoot), so the user does not re-specify root.
#
# The entry submodule type is shared with lib/types.nix (the same entriesType as mkManifest's
# evalModules). This way, unknown keys (typos, old names) become eval errors via the strict submodule,
# avoiding a duplicate definition of validation (→ ADR-0010, docs/spec.md "module option spec").
#
# > Via the HM module, the MVP is limited to a single nput.entries = 1 profile (fixed name default),
# > and role separation (multiple profiles) is not possible. Users who need role separation use the
# > standalone CLI path (entrypoint's nput.<name>). Multiple profiles are a future seam (→ ADR-0024, ADR-0025).
{ lib, ... }:
let
  nputTypes = import ../lib/types.nix lib;
in
{
  options.nput = {
    enable = lib.mkEnableOption "nput (symlink / copy placement of fetched git repositories)";

    entries = lib.mkOption {
      type = nputTypes.entriesType;
      default = { };
      example = lib.literalExpression ''
        {
          # attribute key = root-relative target (identifier; → ADR-0014)
          ".claude/skills/nix" = {
            src = inputs.claude-skills;
            subpath = "skills/nix";
          };
        }
      '';
      description = ''
        Attrset of placement definitions. The attribute key = the placement target
        (identifier), and each value is an entry submodule (src / subpath / target /
        method). The type is shared with lib/types.nix, and unknown keys are an eval
        error (→ docs/spec.md "entries schema").
      '';
    };
  };
}
