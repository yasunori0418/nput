---
id: "CASE-3a1f403d-3c7b-4221-a32a-2ddc272a58a3"
type: test_case
name: "nix-unit: farm-entries.nix — アンカー対象の抽出と anchorLines の組み立て"
covers:
  - "TC-1d69350e-db3c-4d74-a24e-7a3fabb31b0a"
---
# CASE-3a1f403d: nix-unit farm-entries.nix

## 対象

`tests/nix-unit/farm-entries.nix`（TP-403c55c7 のテスト seam `nput.__internal.farmEntries` /
`nput.__internal.anchorName` を直接叩き、`lib/manifest.nix` の anchorLines 生成式を同じ形で
再構成して突き合わせる）

## 検証内容

- **抽出条件**: store×symlink / store×copy / out-of-store×symlink が混在する manifest を
  1 つ与え、採用されるのが store×symlink の 2 件だけであること（target 列で確認）
- **空になる条件**: copy と out-of-store しか無い manifest では抽出結果が空リストになること
- **アンカー名の固定**: 対象 target の 1 つ（`.config/sym`）についてアンカー名を `anchorName`
  に単独で問い、既知の sha256 短縮 hex リテラルに一致すること。次項の合成結果の期待値を作る
  ための足場で、farm を経由した適用そのものを見ているのは次項の方である。アンカー名の形式・
  決定性・特殊文字耐性は CASE-ead15d61 の担当
- **組み立て結果**: 抽出とアンカー名の合成が
  `ln -s <escapeShellArg src> "$out/<anchorName>"` を改行連結した文字列になり、**target ごとに**
  anchor 名が変わること（2 entry の `src` は同一の fake store パスなので、変わるのは
  anchor 側だけである）

配置元は TP-d3d06fe4 の fake flake-input double イディオムに従う。

## 出典

`tests/nix-unit/farm-entries.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
