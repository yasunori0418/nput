---
id: "CASE-823a65c6-e9d2-404f-99ed-85d09376afea"
type: test_case
name: "nix-unit: fixed-root.nix — fixed root の絶対パス併記"
target: "tests/nix-unit/fixed-root.nix"
covers:
  - "TC-f9e927d0-8e10-4b8e-9870-5b5486949af6"
---
# CASE-823a65c6: nix-unit fixed-root.nix

## 対象

`tests/nix-unit/fixed-root.nix`（`tests/nix-unit.nix` のディレクトリ列挙で自動搭載。TP-36e90d5d
が定めるとおり、テスト名はファイル横断で一意でなければならない。本ファイルは
`testFixedRoot*` 接頭辞を使う）

## 検証内容

`normalizeManifest` の `root` に marker でない絶対パス文字列を渡し、出力の `root`
オブジェクトを値を固定してアサートする。

- `root.rootKind` が `"fixed"`
- `root.root` に渡した絶対パスがそのまま写る
- `root` オブジェクトが `{ rootKind = "fixed"; root = <絶対パス>; }` に exact 一致
  （余分なキーが残れば落ちる）
- 別の絶対パスでも同じ形になる（特定の値に依存した通り方をしない）
- fixed root でも store-backed entry の正規化結果は project root のときと同じ形
- `homeRoot` marker を渡した場合は `{ rootKind = "home"; }` で、fixed へ倒れず
  絶対パスも持たない

project root の否定側（固定 root を持たない）は CASE-7a6c4957 が持つ。本 CASE はその対に
なる肯定側で、両者で REQ-dd10d820 の評価層分を挟む（→ TC-f9e927d0）。

配置元は TP-d3d06fe4 の fake flake-input double イディオムに従う。

## 出典

Issue #288（REQ-dd10d820 の肯定側が 8 区分のどこにも無い。検出元は Issue #273 L1 レーンの
逆算）。root オブジェクトの設計判断は ADR-0010 が持つ。
