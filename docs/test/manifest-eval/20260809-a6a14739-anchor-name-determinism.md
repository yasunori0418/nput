---
id: "TC-a6a14739-89e5-4eba-acee-3ed5be5a0b7e"
type: test_condition
name: "GC アンカー名の形式・決定性・特殊文字耐性をアサートする"
mitigates:
  - "RISK-f67a0883-950a-4458-8b6d-1f95cb039cb1"
---
# TC-a6a14739: アンカー名の形式と決定性をアサートする

## テスト条件

アンカー名（target の sha256 先頭 32 hex）そのものの性質を、farm の組み立てから切り離して
アサートする。抽出条件と組み立て結果は TC-1d69350e の担当。

- 出力長が常に 32、構成が小文字 hex のみであること
- 同一 target は同値（呼び出しをまたいで安定）であること
- 異なる target は異なる名前になること（衝突しないことの示唆）
- FS 名として直に使えない文字（非 ASCII・空白・記号）を含む target でも、同じ性質を保つこと
- 期待値は関数の再実装ではなく、外部に固定した既知 hash のリテラルと突き合わせること

## 覆う CASE

- CASE-ead15d61（`tests/nix-unit/anchor-name.nix`）

## 出典

`tests/nix-unit/anchor-name.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
