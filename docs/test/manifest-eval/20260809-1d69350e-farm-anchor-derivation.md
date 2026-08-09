---
id: "TC-1d69350e-db3c-4d74-a24e-7a3fabb31b0a"
type: test_condition
name: "GC アンカー対象の抽出条件とアンカー名の決定性をアサートする"
mitigates:
  - "RISK-3de9753f-3fd3-4364-b1ab-64c68c15ec77"
---
# TC-1d69350e: アンカー対象の抽出とアンカー名の決定性をアサートする

## テスト条件

symlink farm が GC アンカー専用であることを、抽出条件と名前生成の 2 段でアサートする。

**抽出条件**: store-backed かつ `method = "symlink"` の entry だけがアンカー対象になること。
copy（世代外・置き切り）と out-of-store（store 非依存）は対象外であること。対象が皆無なら
空になること。store×symlink / store×copy / out-of-store×symlink が混在する manifest を 1 つ
用意し、採用と除外を同時に見る。

**アンカー名**: target の sha256 先頭 32 hex であること。長さが常に 32、構成が小文字 hex のみ、
同一 target は同値（呼び出しをまたいで安定）、異なる target は異なる名前になること。FS 名として
直に使えない文字（非 ASCII・空白・記号）を含む target でも同じ性質を保つこと。期待値は関数の
再実装ではなく、外部に固定した既知 hash のリテラルと突き合わせる。

**組み立て結果**: 抽出とアンカー名の合成が farm derivation の `ln -s <src> "$out/<anchor>"`
行として期待どおりに並ぶこと。

## 覆う CASE

- CASE-ead15d61（`tests/nix-unit/anchor-name.nix`）
- CASE-3a1f403d（`tests/nix-unit/farm-entries.nix`）
