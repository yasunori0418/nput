# nix-unit アグリゲータの組み立てロジック（→ TP-36e90d5d, RISK-3de9753f）。
#
# `tests/nix-unit.nix` から分離してあるのは、衝突検査自身をダミー入力で検証できるように
# するため（→ Issue #308）。アグリゲータは `builtins.readDir` に直結していて注入点を
# 持たず、既存の leaf に衝突が無い現状では throw 側の経路が一度も実行されない。
# 検査が壊れて常に「衝突なし」を返すようになっても気づけないため、readDir を切り離した
# 純関数として置く。
#
# **`tests/nix-unit/` 配下に置かないこと**。同ディレクトリはアグリゲータが readDir で
# 列挙するので、置くとこのファイル自身がテストファイルとして import される。
{ lib }:
{
  # modules: `{ file, tests }` のリスト（file = 定義元ファイル名、tests = テスト attrset）。
  # 返り値: 全 tests を `//` マージした attrset。テスト名がファイル横断で衝突していれば
  # マージせず throw する。
  mergeTests =
    modules:
    let
      # テスト名 -> それを定義しているファイル名のリスト。名前だけを見るので、
      # 衝突が無ければ各テストの値（expr / expected）は評価しない。
      ownersByTest = lib.zipAttrs (map (m: lib.mapAttrs (_testName: _: m.file) m.tests) modules);

      collisions = lib.filterAttrs (_testName: owners: lib.length owners > 1) ownersByTest;

      collisionReport = lib.concatStringsSep "\n" (
        lib.mapAttrsToList (
          testName: owners: "  - ${testName}: ${lib.concatStringsSep ", " owners}"
        ) collisions
      );

      merged = lib.foldl' (acc: m: acc // m.tests) { } modules;
    in
    if collisions == { } then
      merged
    else
      throw ''
        tests/nix-unit: テスト名がファイル横断で衝突しています（`//` の後勝ちマージで
        片方のアサートが実行されないまま緑になります）。ファイル固有の接頭辞を付けて
        一意にしてください（→ TP-36e90d5d）。
        ${collisionReport}'';
}
