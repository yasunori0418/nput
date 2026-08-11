---
id: "TC-f9e927d0-8e10-4b8e-9870-5b5486949af6"
type: test_condition
name: "fixed root で rootKind が fixed になり評価時確定の絶対パスを併記する"
mitigates:
  - "RISK-5df2d02b-e5d4-40eb-86ad-e8bc96e4c34d"
---
# TC-f9e927d0: fixed root で絶対パスを併記する

## テスト条件

`normalizeManifest` の `root` に marker でない絶対パス文字列を渡したとき、出力の `root`
オブジェクトが REQ-dd10d820 の fixed 側の規定どおりになることをアサートする。

- `root.rootKind` が `"fixed"` になること
- `root.root` に渡した絶対パスがそのまま写ること。特定の値でだけ通る実装にならないよう、
  異なるパスでも同じ形になることを確かめる
- `root` オブジェクトが `rootKind` / `root` のちょうど 2 フィールドであること
  （exact 一致で見るので余分なキーの混入は落ちる）
- root 種別を fixed にしても entry の正規化結果は変わらないこと
- root marker（`homeRoot`）は fixed へ倒れず絶対パスも持たないこと。fixed 判定が
  marker 側へ誤って広がらないことの担保

**この条件は REQ-dd10d820 の肯定側（fixed）を持つ**。同じ REQ の否定側のうち project root
分は TC-4e7cfae7 が持ち、`homeRoot` を実際の HM 統合で見る分は `checks.hm-module`
（`integration` 区分）が担当する。

## 覆う CASE

- CASE-823a65c6（`tests/nix-unit/fixed-root.nix`）

## 出典

REQ-dd10d820 の肯定側が 8 区分のどこにも無いという検出（→ Issue #288。検出元は Issue #273
L1 レーンの逆算）。RISK-5df2d02b が残余として記録していたものを本条件で埋める。
