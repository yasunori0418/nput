---
id: "CASE-0c9d41d1-33fd-4982-aed9-fa575a56ee01"
type: test_case
name: "nix-unit: resolve-marker.nix — src 解決と marker 判別述語"
target: "tests/nix-unit/resolve-marker.nix"
covers:
  - "TC-81be084d-709f-481b-9b61-5d2d11c317a0"
---
# CASE-0c9d41d1-33fd-4982-aed9-fa575a56ee01: nix-unit resolve-marker.nix

## 対象

`tests/nix-unit/resolve-marker.nix`（TP-403c55c7-d996-4951-8e6b-c3a7dddd387c のテスト seam `nput.__internal.resolveEntry`
と `lib/types.nix` の `isRootMarker` / `isOutOfStoreMarker` を直接叩く。manifest 全体を介さず
単一 entry の src 種別判定・文字列化だけを見る最小の境界）

## 検証内容

src 解決については、`srcKind` 単体・`src` 単体・entry 全体の shape という 3 段のアサートを
store / out-of-store それぞれに重ねて置く。どの段で落ちたかで、種別判定・文字列化・余分な
キーの混入のどれが壊れたかを切り分けられるようにするため。

- **store src**: `srcKind = "store"`、`src` が store パスへ文字列化、entry 全体が
  5 フィールドに exact 一致
- **out-of-store marker**: `srcKind = "outOfStore"`、`src` が marker の絶対パス、entry 全体が
  exact 一致（`method = "copy"` を明示した形で）
- **判別タグの非漏洩**: store / out-of-store のどちらの解決結果にも `_nputMarker` が無いこと
- **`isOutOfStoreMarker`**: out-of-store marker には true、root マーカー・store-backed な
  attrset には false
- **`isRootMarker`**: projectRoot / homeRoot / systemRoot には true、out-of-store marker と
  絶対パス文字列には false

配置元は TP-d3d06fe4-6940-4df8-b111-bb4096d5444f の fake flake-input double イディオムに従う。

## 出典

`tests/nix-unit/resolve-marker.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
marker の設計判断は ADR-0001 / ADR-0010 / ADR-0013 が持つ。
