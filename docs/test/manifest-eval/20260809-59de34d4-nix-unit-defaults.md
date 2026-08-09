---
id: "CASE-59de34d4-d9d5-4ae0-b1be-44d2e775f031"
type: test_case
name: "nix-unit: defaults.nix — 既定適用・明示上書き・target 辞書順"
covers:
  - "TC-d9175bb5-d7ec-41e0-8bee-71de928a71fb"
---
# CASE-59de34d4: nix-unit defaults.nix

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

配置元は固定 `outPath` を持つ fake flake-input。
