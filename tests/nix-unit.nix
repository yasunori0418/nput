# nix-unit アグリゲータ: `tests/nix-unit/` 配下の全 `*.nix` を列挙し、各々を `{ lib, nput }` で
# import して `//` マージした単一の test attrset を返す（→ ADR-0006, ADR-0010）。
#
# leaf なテストファイルを `tests/nix-unit/` に追加するだけで自動的に nix-unit に載るため、
# 集約ファイル・flake.nix の編集が不要になる（flake.nix の `import ./tests/nix-unit.nix
# { lib, nput }` シグネチャは不変）。テスト名はファイル横断で一意であること（`//` は後勝ち）。
{ lib, nput }:
let
  dir = ./nix-unit;
  testFiles = lib.filterAttrs (name: type: type == "regular" && lib.hasSuffix ".nix" name) (
    builtins.readDir dir
  );
  modules = lib.mapAttrsToList (
    name: _type: import (dir + "/${name}") { inherit lib nput; }
  ) testFiles;
in
lib.foldl' (acc: m: acc // m) { } modules
