# namaka: manifest.json 全体（= normalizeManifest 出力）のスナップショット回帰（→ ADR-0006）。
# src は toString が安定する fake な flake-input 相当（`{ outPath = …; }`）を使い、
# store hash 揺れでスナップショットが壊れないようにする。
{ lib, nput }:
nput.normalizeManifest {
  inherit lib;
  root = nput.projectRoot;
  entries = {
    ".claude/skills/nix" = {
      src = {
        outPath = "/nix/store/00000000000000000000000000000000-fake-src";
      };
      subpath = "skills/nix";
    };
    # subpath / target / method 省略 → デフォルト適用
    ".config/foo" = {
      src = {
        outPath = "/nix/store/11111111111111111111111111111111-other";
      };
    };
  };
}
