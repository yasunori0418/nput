# nix-unit アグリゲータ: `tests/nix-unit/` 配下の全 `*.nix` を列挙し、各々を `{ lib, nput }` で
# import して `//` マージした単一の test attrset を返す（→ ADR-0006, ADR-0010）。
#
# leaf なテストファイルを `tests/nix-unit/` に追加するだけで自動的に nix-unit に載るため、
# 集約ファイル・flake.nix の編集が不要になる（flake.nix の `import ./tests/nix-unit.nix
# { lib, nput }` シグネチャは不変）。テスト名はファイル横断で一意であること（`//` は後勝ち）。
# 一意性は規約に留めず、マージ前の重複検査で機械的に強制する（→ TP-36e90d5d, RISK-3de9753f）。
#
# 本ファイルはディレクトリ列挙（readDir）だけを担い、検査を含むマージ自体は
# `tests/nix-unit-lib.nix` の `mergeTests` が持つ。ダミー入力で検査自身を検証できる
# ようにするための分離で、その leaf テストは `tests/nix-unit/aggregator-merge.nix`
# （→ Issue #308）。
{ lib, nput }:
let
  inherit (import ./nix-unit-lib.nix { inherit lib; }) mergeTests;

  dir = ./nix-unit;
  testFiles = lib.filterAttrs (name: type: type == "regular" && lib.hasSuffix ".nix" name) (
    builtins.readDir dir
  );
  # { file, tests } の組で持ち回り、衝突時にどのファイル同士かを示せるようにする。
  modules = lib.mapAttrsToList (name: _type: {
    file = name;
    tests = import (dir + "/${name}") { inherit lib nput; };
  }) testFiles;
in
mergeTests modules
