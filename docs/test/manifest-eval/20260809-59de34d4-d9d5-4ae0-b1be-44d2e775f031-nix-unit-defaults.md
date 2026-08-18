---
id: "CASE-59de34d4-d9d5-4ae0-b1be-44d2e775f031"
type: test_case
name: "nix-unit: defaults.nix — 既定適用・明示上書き・target 辞書順"
target: "tests/nix-unit/defaults.nix"
covers:
  - "TC-d9175bb5-d7ec-41e0-8bee-71de928a71fb"
---
# CASE-59de34d4-d9d5-4ae0-b1be-44d2e775f031: nix-unit defaults.nix

## 対象

`tests/nix-unit/defaults.nix`

## 検証内容

`normalizeManifest` の正規化を 3 本でアサートする。

- **既定適用**: `src` だけを宣言した entry が `subpath = "."` / `target = 属性キー` /
  `method = "symlink"` を得ること（出力 entry 全体の exact 一致で見る）
- **明示上書き**: `target` / `subpath` / `method` を明示した entry で、既定ではなく宣言値が
  文書へ載ること
- **決定的な配列化**: 属性キー `b` / `a` / `c` を与えたとき、出力 entries の `target` 列が
  `[a b c]` の辞書順になること

配置元は TP-d3d06fe4-6940-4df8-b111-bb4096d5444f の fake flake-input double イディオムに従う。

## 出典

`tests/nix-unit/defaults.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。既定と
辞書順の設計判断は ADR-0010 / ADR-0014 / ADR-0016 が持つ。
