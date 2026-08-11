---
id: "CASE-3a1f403d-3c7b-4221-a32a-2ddc272a58a3"
type: test_case
name: "nix-unit: farm-entries.nix — アンカー対象の抽出と anchorLines の組み立て"
target: "tests/nix-unit/farm-entries.nix"
covers:
  - "TC-1d69350e-db3c-4d74-a24e-7a3fabb31b0a"
---
# CASE-3a1f403d: nix-unit farm-entries.nix

## 対象

`tests/nix-unit/farm-entries.nix`（TP-403c55c7 のテスト seam `nput.__internal.farmEntries` /
`nput.__internal.anchorName` / `nput.__internal.anchorLines` を直接叩く。anchorLines は
`lib/manifest.nix` が `mkManifest` から呼ぶ生成式そのもので、テスト側に複製は持たない
→ Issue #289）

## 検証内容

- **抽出条件**: store×symlink / store×copy / out-of-store×symlink が混在する manifest を
  1 つ与え、採用されるのが store×symlink の 2 件だけであること（target 列で確認）
- **空になる条件**: copy と out-of-store しか無い manifest では抽出結果が空リストになること
- **アンカー名の固定**: 対象 target の 1 つ（`.config/sym`）についてアンカー名を `anchorName`
  に単独で問い、既知の sha256 短縮 hex リテラルに一致すること。以降の生成結果の期待値を作る
  ための足場である。アンカー名の形式・決定性・特殊文字耐性は CASE-ead15d61 の担当
- **生成式の単体**: `anchorLines` を最小の手組みエントリ列へ適用し、リテラルの期待値で押さえる
  （manifest を経由しないので期待値を式で組まずに済む）。1 行が
  `ln -s <escapeShellArg src> "$out/<anchorName>"` の形になること、複数エントリが末尾改行なしで
  改行連結され **target ごとに** anchor 名が変わること（2 entry の `src` は同一の fake store
  パスなので変わるのは anchor 側だけである）、空白・記号を含む `src` が `escapeShellArg` で
  quote されること、エントリが空なら空文字列になること
- **farm への配線**: 混在 manifest を `normalizeManifest` → `farmEntries` → `anchorLines` の
  経路へ通し、生成の入力が全 entry ではなく farm 対象の 2 件だけであること。期待値は同じ
  `anchorLines` を独立に選んだ対象 target 列へ適用して組むので、生成式を変えれば両辺が揃って
  動き、ここで落ちるのは抽出と生成の繋ぎ違いだけである（内容の正しさは前項が固定する）

配置元は TP-d3d06fe4 の fake flake-input double イディオムに従う。

## 出典

`tests/nix-unit/farm-entries.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
