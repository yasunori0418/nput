---
id: "CASE-7a6c4957-9365-4490-b48f-9725f42162f2"
type: test_case
name: "nix-unit: structure.nix — manifest 構造の不変条件"
target: "tests/nix-unit/structure.nix"
covers:
  - "TC-4e7cfae7-72bc-4af6-a1f5-1ead7db564b1"
---
# CASE-7a6c4957: nix-unit structure.nix

## 対象

`tests/nix-unit/structure.nix`（`tests/nix-unit.nix` のディレクトリ列挙で自動搭載。TP-36e90d5d
が定めるとおり、テスト名はファイル横断で一意でなければならない）

## 検証内容

`normalizeManifest` の出力構造を、値を固定してアサートする。

- `schemaVersion` が 1
- `root.rootKind` が `"project"`
- project root では固定 root パス（`root.root`）を持たない
- store-backed entry が `{ srcKind = "store"; src; subpath; target; method }` に exact 一致
  （余分なキーが残れば落ちる）
- out-of-store marker を与えた entry が `srcKind = "outOfStore"` と marker の絶対パスへ
  変換され、`method` は既定の `"symlink"`
- out-of-store entry に判別タグ `_nputMarker` が漏れていないことを単独でも明示アサート

本 CASE が検証するのは project root のみ（→ TC-4e7cfae7）。fixed root の絶対パス併記と
`homeRoot` marker の評価層での扱いは CASE-823a65c6（→ TC-f9e927d0）、HM 統合での
`homeRoot` は `checks.hm-module`（`integration` 区分）が担当する。
配置元は TP-d3d06fe4 の fake flake-input double イディオムに従う。

## 出典

`tests/nix-unit/structure.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。manifest
構造の設計判断は ADR-0006 / ADR-0010 / ADR-0014 が持つ。
