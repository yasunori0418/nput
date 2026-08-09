---
id: "TC-d9175bb5-d7ec-41e0-8bee-71de928a71fb"
type: test_condition
name: "省略時の既定適用・明示上書き・target 辞書順での決定的配列化をアサートする"
mitigates:
  - "RISK-5df2d02b-e5d4-40eb-86ad-e8bc96e4c34d"
---
# TC-d9175bb5: 既定適用と決定的な配列化をアサートする

## テスト条件

attrset で宣言された entries が配列へ正規化される過程を、3 つの観点でアサートする。

- **既定の適用**: `subpath` 省略時は `"."`、`target` 省略時は属性キー、`method` 省略時は
  `"symlink"` になること。既定が変われば世代管理下に置かれる entry の集合が変わるため、
  値そのものを固定する
- **明示上書き**: `target` / `subpath` / `method` を明示したとき、既定ではなく宣言値が
  文書へ載ること
- **決定的な配列化**: 出力の並びが属性キー（= 既定の target）の辞書順であること。attrset は
  宣言順を保たないため、順序を固定しなければ同じ入力から異なる文書が出る

## 覆う CASE

- CASE-59de34d4（`tests/nix-unit/defaults.nix`）
