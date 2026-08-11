# nix-unit: fixed root（root に絶対パス文字列を渡す）の肯定側をアサートする
# （→ REQ-dd10d820・ADR-0010・Issue #288）。
#
# root marker（projectRoot / homeRoot / systemRoot）は実行時解決なのでパスを持たないが、
# marker でない文字列を渡した場合だけ `rootKind = "fixed"` になり評価時に確定した絶対パスを
# `root` に併記する。structure.nix は project root（否定側 = 固定 root を持たない）を見るので、
# その対になる肯定側をここで見る。
#
# store パスの hash 揺れを避けるため src には toString が安定する fake な flake-input 相当
# （`{ outPath = …; }`）を使う。これは srcType の store-backed 判定（`? outPath`）を通る正当な test double。
{ lib, nput }:
let
  fakeSrc = {
    outPath = "/nix/store/00000000000000000000000000000000-fake-src";
  };
  norm = root: entries: nput.normalizeManifest { inherit lib root entries; };

  fixed = norm "/srv/deploy" {
    ".config/foo" = {
      src = fakeSrc;
    };
  };
in
{
  # marker でない文字列を root に渡すと fixed になる（→ REQ-dd10d820, ADR-0010）。
  testFixedRootKind = {
    expr = fixed.root.rootKind;
    expected = "fixed";
  };

  # fixed のときだけ絶対パスを併記する。project（structure.nix の否定側）との対。
  testFixedRootPath = {
    expr = fixed.root.root;
    expected = "/srv/deploy";
  };

  # root オブジェクトは rootKind と root のちょうど 2 フィールド。exact 一致で見るので
  # 余分なキーが混ざれば落ちる（entry 側の shape アサートと同じ方針）。
  testFixedRootObjectShape = {
    expr = fixed.root;
    expected = {
      rootKind = "fixed";
      root = "/srv/deploy";
    };
  };

  # 渡した文字列がそのまま写ることを別のパスでも確かめる（特定の値に依存した通り方を避ける）。
  # entries は上と同条件に揃え、差分がパス値の 1 軸だけになるようにする。
  testFixedRootPathVerbatim = {
    expr =
      (norm "/opt/nput" {
        ".config/foo" = {
          src = fakeSrc;
        };
      }).root;
    expected = {
      rootKind = "fixed";
      root = "/opt/nput";
    };
  };

  # entry の正規化は root 種別に依らない。既定値そのものは defaults.nix が持つので、
  # ここは project root との同値比較で独立性だけを主張する（期待値を写すと既定値の
  # 知識が二重管理になる）。真偽値へは畳まない（畳むと落ちたとき `expected: true /
  # got: false` しか出ず、どのフィールドがずれたか分からない）。
  # 現行の resolveEntry は root を取らないのでこの結合は構造的に起こり得ず、entry の
  # 解決へ root を渡す設計変更が入ったときに初めて働く回帰網である。
  testFixedRootEntryUnaffected = {
    expr = fixed.entries;
    expected =
      (norm nput.projectRoot {
        ".config/foo" = {
          src = fakeSrc;
        };
      }).entries;
  };

  # 対になる否定側の再確認: homeRoot marker は fixed にならず絶対パスも持たない
  # （project 分は structure.nix が見る。ここは fixed 判定が marker へ誤って広がらないことの
  # 担保）。exact 一致なので `root` フィールドの不在まで見る。
  testFixedRootHomeMarkerShape = {
    expr = (norm nput.homeRoot { }).root;
    expected = {
      rootKind = "home";
    };
  };
}
