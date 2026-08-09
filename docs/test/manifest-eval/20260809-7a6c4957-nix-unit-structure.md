---
id: "CASE-7a6c4957-9365-4490-b48f-9725f42162f2"
type: test_case
name: "nix-unit: structure.nix — manifest 構造の不変条件"
covers:
  - "TC-4e7cfae7-72bc-4af6-a1f5-1ead7db564b1"
---
# CASE-7a6c4957: nix-unit structure.nix

## 対象

`tests/nix-unit/structure.nix`（`tests/nix-unit.nix` のディレクトリ列挙で自動搭載。テスト名の
接頭辞は `test*` でファイル横断に一意）

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

配置元は固定 `outPath` を持つ fake flake-input（store-backed 判定が `? outPath` を見る実装で
あるがゆえに通る正当な double）。
