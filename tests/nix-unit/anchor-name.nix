# nix-unit: GC anchor 名（`__internal.anchorName`）の不変条件をアサートする（→ #58, #73, ADR-0016）。
#
# anchorName = sha256(target) の先頭 32 hex。symlink farm の衝突しない FS-safe な anchor 名として使う。
# 検証する性質: (1) 常に 32 文字の hex / (2) 同一 target は同一 hash（決定性）/ (3) 特殊文字を
# 含む target でも安定して 32 hex を返す。
#
# 期待 hash は `lib.substring 0 32 (builtins.hashString "sha256" target)` を nix で評価した実値を
# 直書きする（関数の再実装ではなく外部に固定した ground-truth との一致を見る）。
{ lib, nput }:
let
  an = nput.__internal.anchorName lib;

  # cyrillic / 日本語 / 空白 / 記号（& * " |）を含む target。FS 名として直に使えない文字を含んでも
  # sha256 経由で安定した hex に潰れることを確認する。
  specialTarget = ".config/нвим/ファイル space&sym*\"|";

  isHex32 = s: builtins.match "[0-9a-f]{32}" s != null;
in
{
  # 出力は常に 32 文字。
  testAnchorNameLength = {
    expr = lib.stringLength (an ".claude/skills/nix");
    expected = 32;
  };

  # 出力は 32 桁の小文字 hex のみで構成される。
  testAnchorNameAllHex = {
    expr = isHex32 (an ".claude/skills/nix");
    expected = true;
  };

  # 同一 target は外部固定の既知 hash に一致する（決定性 + 値の正しさ）。
  testAnchorNameDeterministic = {
    expr = an ".claude/skills/nix";
    expected = "6742682e75579724615f6b3800237eb4";
  };

  # 同じ target を 2 回適用しても同値（純粋関数としての決定性を明示）。
  testAnchorNameStableAcrossCalls = {
    expr = an ".claude/skills/nix" == an ".claude/skills/nix";
    expected = true;
  };

  # 異なる target は異なる hash になる（衝突しない anchor 名であることの示唆）。
  testAnchorNameDistinctTargets = {
    expr = an ".claude/a" == an ".claude/b";
    expected = false;
  };

  # 特殊文字を含む target でも既知 hash に安定一致する。
  testAnchorNameSpecialCharsStable = {
    expr = an specialTarget;
    expected = "84bbba52edcecff0154de08d41a8259e";
  };

  # 特殊文字を含む target でも長さ 32 を保つ。
  testAnchorNameSpecialCharsLength = {
    expr = lib.stringLength (an specialTarget);
    expected = 32;
  };

  # 特殊文字を含む target でも 32 桁 hex のまま。
  testAnchorNameSpecialCharsAllHex = {
    expr = isHex32 (an specialTarget);
    expected = true;
  };
}
