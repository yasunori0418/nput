# nix-unit: アグリゲータの衝突検査（`tests/nix-unit-lib.nix` の `mergeTests`）自身を
# ダミー入力でアサートする（→ Issue #308, RISK-3de9753f）。
#
# 検査対象は本番の readDir 経路ではなく純関数のほう。実ファイルは衝突しない（衝突したら
# `nix flake check` が落ちる）ため、throw 側の経路は合成した `{ file, tests }` を渡さない
# 限り一度も実行されず、検査が常に「衝突なし」を返す退行を検知できない。
#
# ダミーの tests の値は `mergeTests` が名前しか見ないので任意でよいが、マージ結果の
# 後勝ちを見分けられるようファイルごとに違う値を置く。
#
# `nput` は使わない（検証対象がスイートの組み立てで、manifest 生成関数ではないため）が、
# アグリゲータが全 leaf を `{ lib, nput }` で import するのでシグネチャは他ファイルに揃える。
{ lib, nput }:
let
  inherit (import ../nix-unit-lib.nix { inherit lib; }) mergeTests;

  # 衝突の無い 2 ファイル。
  disjoint = [
    {
      file = "alpha.nix";
      tests = {
        testAlphaOne = 1;
        testAlphaTwo = 2;
      };
    }
    {
      file = "beta.nix";
      tests = {
        testBetaOne = 3;
      };
    }
  ];

  # 同名 `testDup` を 2 ファイルが定義する。
  twoWayCollision = [
    {
      file = "alpha.nix";
      tests = {
        testAlphaOne = 1;
        testDup = 2;
      };
    }
    {
      file = "beta.nix";
      tests = {
        testDup = 3;
      };
    }
  ];

  # 同名 `testDup` を 3 ファイルが定義する。
  threeWayCollision = [
    {
      file = "alpha.nix";
      tests.testDup = 1;
    }
    {
      file = "beta.nix";
      tests.testDup = 2;
    }
    {
      file = "gamma.nix";
      tests.testDup = 3;
    }
  ];

  # 衝突するテスト名が 2 組ある（報告が 2 行になる）。
  twoCollisionGroups = [
    {
      file = "alpha.nix";
      tests = {
        testDupA = 1;
        testDupB = 2;
      };
    }
    {
      file = "beta.nix";
      tests = {
        testDupA = 3;
        testDupB = 4;
      };
    }
  ];
in
{
  # 衝突が無ければ全ファイルのテスト名がマージ結果に揃う。
  testAggregatorMergeUnionsNames = {
    expr = lib.attrNames (mergeTests disjoint);
    expected = [
      "testAlphaOne"
      "testAlphaTwo"
      "testBetaOne"
    ];
  };

  # 値もそのまま写る（名前だけ集めて中身を落とす実装にならない担保）。
  testAggregatorMergeKeepsValues = {
    expr = mergeTests disjoint;
    expected = {
      testAlphaOne = 1;
      testAlphaTwo = 2;
      testBetaOne = 3;
    };
  };

  # 空入力は空 attrset（衝突なしの境界）。
  testAggregatorMergeEmptyInput = {
    expr = mergeTests [ ];
    expected = { };
  };

  # 2 ファイルが同名を定義したら throw する。メッセージに衝突したテスト名と
  # 両方のファイル名が出る（後勝ちで消えた側を追えるようにするため）。
  testAggregatorMergeTwoWayCollisionThrows = {
    expr = mergeTests twoWayCollision;
    expectedError.type = "ThrownError";
    expectedError.msg = "  - testDup: alpha\\.nix, beta\\.nix";
  };

  # 衝突していないテスト名は報告に出ない（無関係なファイルを巻き込んで報告しない）。
  testAggregatorMergeCollisionReportExcludesInnocent = {
    expr = mergeTests twoWayCollision;
    expectedError.type = "ThrownError";
    expectedError.msg = "^(.|\n)*テスト名がファイル横断で衝突((?!testAlphaOne)(.|\n))*$";
  };

  # 3 ファイル以上なら owners が 3 件とも列挙される（2 件で打ち切らない）。
  testAggregatorMergeThreeWayCollisionListsAllOwners = {
    expr = mergeTests threeWayCollision;
    expectedError.type = "ThrownError";
    expectedError.msg = "  - testDup: alpha\\.nix, beta\\.nix, gamma\\.nix";
  };

  # 衝突が複数組あれば報告が複数行になる（最初の 1 組で止めない）。
  testAggregatorMergeMultipleCollisionGroupsReportEachLine = {
    expr = mergeTests twoCollisionGroups;
    expectedError.type = "ThrownError";
    expectedError.msg = "  - testDupA: alpha\\.nix, beta\\.nix\n  - testDupB: alpha\\.nix, beta\\.nix";
  };
}
