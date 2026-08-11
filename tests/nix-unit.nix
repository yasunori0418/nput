# nix-unit アグリゲータ: `tests/nix-unit/` 配下の全 `*.nix` を列挙し、各々を `{ lib, nput }` で
# import して `//` マージした単一の test attrset を返す（→ ADR-0006, ADR-0010）。
#
# leaf なテストファイルを `tests/nix-unit/` に追加するだけで自動的に nix-unit に載るため、
# 集約ファイル・flake.nix の編集が不要になる（flake.nix の `import ./tests/nix-unit.nix
# { lib, nput }` シグネチャは不変）。テスト名はファイル横断で一意であること（`//` は後勝ち）。
# 一意性は規約に留めず、マージ前の重複検査で機械的に強制する（→ TP-36e90d5d, RISK-3de9753f）。
{ lib, nput }:
let
  dir = ./nix-unit;
  testFiles = lib.filterAttrs (name: type: type == "regular" && lib.hasSuffix ".nix" name) (
    builtins.readDir dir
  );
  # { file, tests } の組で持ち回り、衝突時にどのファイル同士かを示せるようにする。
  modules = lib.mapAttrsToList (name: _type: {
    file = name;
    tests = import (dir + "/${name}") { inherit lib nput; };
  }) testFiles;

  # テスト名 -> それを定義しているファイル名のリスト。
  ownersByTest = lib.foldl' (
    acc: m:
    acc
    // lib.mapAttrs (
      testName: _: (acc.${testName} or [ ]) ++ [ m.file ]
    ) m.tests
  ) { } modules;

  collisions = lib.filterAttrs (_testName: owners: lib.length owners > 1) ownersByTest;

  collisionReport = lib.concatMapStringsSep "\n" (
    testName: "  - ${testName}: ${lib.concatStringsSep ", " collisions.${testName}}"
  ) (lib.attrNames collisions);

  merged = lib.foldl' (acc: m: acc // m.tests) { } modules;
in
if collisions == { } then
  merged
else
  throw ''
    tests/nix-unit: テスト名がファイル横断で衝突しています（`//` の後勝ちマージで
    片方のアサートが実行されないまま緑になります）。ファイル固有の接頭辞を付けて
    一意にしてください（→ TP-36e90d5d）。
    ${collisionReport}''
