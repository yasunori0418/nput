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
`nput.__internal.anchorName` / `nput.__internal.anchorLines` を直接叩き、加えて `mkManifest` の
builder への配線を fake pkgs 経由で見る。anchorLines は `lib/manifest.nix` が `mkManifest` から
呼ぶ生成式そのもので、テスト側に複製は持たない → Issue #289）

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
- **farm derivation への配線**: 混在 manifest を `mkManifest` へ与え、その builder スクリプト
  本文が `mkdir` → `manifest.json` のコピー → アンカー行の順で組まれ、アンカー行が farm 対象の
  2 件だけから成ること。copy / out-of-store しか無い manifest では builder にアンカー行が
  現れないこと。期待するアンカー行は同じ `anchorLines` で組むので、生成式を変えれば両辺が
  揃って動き、ここで落ちるのは配線の誤り（フィルタ漏れ・埋め込み忘れ・順序の崩れ）だけである
  （内容の正しさは前項が固定する）

`src` の配置元は TP-d3d06fe4 の fake flake-input double イディオムに従う。配線検証の `pkgs` も
同じイディオムで、`mkManifest` が使う `writeText` / `runCommandLocal` を引数を持ち帰る double へ
差し替えて builder 本文を純評価で取り出す（derivation の実ビルドは評価テストの枠を超えるため
採らない → Issue #289）。

## 出典

`tests/nix-unit/farm-entries.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
