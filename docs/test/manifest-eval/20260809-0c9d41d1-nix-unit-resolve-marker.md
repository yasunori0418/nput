---
id: "CASE-0c9d41d1-33fd-4982-aed9-fa575a56ee01"
type: test_case
name: "nix-unit: resolve-marker.nix — src 解決と marker 判別述語"
covers:
  - "TC-81be084d-709f-481b-9b61-5d2d11c317a0"
---
# CASE-0c9d41d1: nix-unit resolve-marker.nix

## 対象

`tests/nix-unit/resolve-marker.nix`（テスト seam `nput.__internal.resolveEntry` と
`lib/types.nix` の `isRootMarker` / `isOutOfStoreMarker` を直接叩く。manifest 全体を介さず
単一 entry の src 種別判定・文字列化だけを見る最小の境界）

## 検証内容

- **store src**: `srcKind = "store"`、`src` が store パスへ文字列化、entry 全体が
  5 フィールドに exact 一致
- **out-of-store marker**: `srcKind = "outOfStore"`、`src` が marker の絶対パス、entry 全体が
  exact 一致（`method = "copy"` を明示した形で）
- **判別タグの非漏洩**: store / out-of-store のどちらの解決結果にも `_nputMarker` が無いこと
- **`isOutOfStoreMarker`**: out-of-store marker には true、root マーカー・store-backed な
  attrset には false
- **`isRootMarker`**: projectRoot / homeRoot / systemRoot には true、out-of-store marker と
  絶対パス文字列には false

配置元は固定 `outPath` を持つ fake flake-input。
