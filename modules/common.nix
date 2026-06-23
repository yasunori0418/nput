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
    enable = lib.mkEnableOption "nput（フェッチ済み git リポジトリの symlink / copy 配置）";

    entries = lib.mkOption {
      type = nputTypes.entriesType;
      default = { };
      example = lib.literalExpression ''
        {
          # 属性キー = root 相対 target（識別子・→ ADR-0014）
          ".claude/skills/nix" = {
            src = inputs.claude-skills;
            subpath = "skills/nix";
          };
        }
      '';
      description = ''
        配置定義の attrset。属性キー = 配置先 target（識別子）で、各値は entry submodule
        （src / subpath / target / method）。型は lib/types.nix と共有し、未知キーは eval
        エラーになる（→ docs/spec.md「entries スキーマ仕様」）。
      '';
    };
  };
}
