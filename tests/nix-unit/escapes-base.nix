# nix-unit: `escapesBase` の `..` 深さ判定の境界を網羅する（→ #72・ADR-0019）。
#
# #71 で `lib/__internal.nix` に切り出した private helper を `nput.__internal` 経由で直接叩く。
# `escapesBase lib p` は path を `/` 分割し `""`・`.` を捨てた上で depth を辿り、depth 0 で `..` に
# 当たった瞬間に escape（base の外へ出る）と判定する。`isUnsafe` はそれに絶対パス（先頭 `/`）拒否を
# OR したもの。テスト名は他ファイルと衝突しない `testEscapesBase*` 接頭辞を付ける（`//` は後勝ちのため）。
{ lib, nput }:
let
  escapesBase = nput.__internal.escapesBase lib;
  isUnsafe = (nput.__internal.pathChecks lib).isUnsafe;
in
{
  # `.` は現在地。comps から `.` は捨てられ depth を動かさないので escape しない。
  testEscapesBaseDot = {
    expr = escapesBase ".";
    expected = false;
  };

  # `..` 単体は depth 0 で `..` に当たるので即 escape。
  testEscapesBaseDotDot = {
    expr = escapesBase "..";
    expected = true;
  };

  # 空文字は comps が空になり escape しない（depth 0 のまま）。
  testEscapesBaseEmpty = {
    expr = escapesBase "";
    expected = false;
  };

  # 通常の下降 `a/b` は depth を増やすだけで escape しない。
  testEscapesBaseDescend = {
    expr = escapesBase "a/b";
    expected = false;
  };

  # `a/b/../..` は depth 2 → 0 まで戻るが負にはならないので escape しない（境界・ちょうど 0）。
  testEscapesBaseBackToZero = {
    expr = escapesBase "a/b/../..";
    expected = false;
  };

  # `a/b/../../..` は 0 まで戻った後さらに `..` で escape（境界の 1 つ外側）。
  testEscapesBaseBelowZero = {
    expr = escapesBase "a/b/../../..";
    expected = true;
  };

  # `a/../b` は depth 1 → 0 → 1 と推移し、途中で `..` に当たっても depth > 0 なので escape しない。
  testEscapesBaseDownUpDown = {
    expr = escapesBase "a/../b";
    expected = false;
  };

  # 先頭の `../a` は depth 0 で `..` に当たるので escape。
  testEscapesBaseLeadingDotDot = {
    expr = escapesBase "../a";
    expected = true;
  };

  # `a/../../b` は depth 1 → 0 → escape（負に踏み込む）。後続が安全でも先に escape 確定。
  testEscapesBaseMidEscape = {
    expr = escapesBase "a/../../b";
    expected = true;
  };

  # isUnsafe: escape しない相対パスは安全。
  testEscapesBaseUnsafeSafeRelative = {
    expr = isUnsafe "a/b";
    expected = false;
  };

  # isUnsafe: 絶対パス（先頭 `/`）は escapesBase とは独立に拒否。
  testEscapesBaseUnsafeAbsolute = {
    expr = isUnsafe "/etc/x";
    expected = true;
  };

  # isUnsafe: escape する相対パスは escapesBase 経由で拒否。
  testEscapesBaseUnsafeEscaping = {
    expr = isUnsafe "../../etc";
    expected = true;
  };
}
