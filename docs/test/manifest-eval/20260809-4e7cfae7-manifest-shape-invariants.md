---
id: "TC-4e7cfae7-72bc-4af6-a1f5-1ead7db564b1"
type: test_condition
name: "manifest 文書の形（schemaVersion / root / entry 5 フィールド）を不変条件としてアサートする"
mitigates:
  - "RISK-5df2d02b-e5d4-40eb-86ad-e8bc96e4c34d"
---
# TC-4e7cfae7: manifest 文書の形を不変条件としてアサートする

## テスト条件

`normalizeManifest` が返す文書のトップレベルと entry の形を、値ごとに名前の付いた不変条件
としてアサートする。

- `schemaVersion` が 1 であること
- project root について、`root.rootKind` が `"project"` になり、実行時解決の root なので
  固定の絶対パス（`root.root`）を持たないこと
- store-backed な entry が `srcKind` / `src` / `subpath` / `target` / `method` の 5 フィールド
  ちょうどであること（exact 一致で余分なキーの混入を落とす）
- out-of-store marker を与えた entry が `srcKind = "outOfStore"` と marker の絶対パスへ
  変換され、判別タグ `_nputMarker` が文書に漏れないこと。こちらも exact 一致で見るので、
  余分なキーの混入は store-backed 側と同じく検出される

**この条件の適用範囲は project root に限る**。`homeRoot` の `rootKind` と fixed root での
絶対パス併記は評価テストに無く、`checks.hm-module` と E2E（`integration` 区分）が担当する。
ここを「全 root 種別を見ている」と読ませないために範囲を明示する。

スナップショット（TC-de6514e2）とは役割が重ならない。役割分担の規範は TP-36e90d5d が持つ。

## 覆う CASE

- CASE-7a6c4957（`tests/nix-unit/structure.nix`）

## 出典

`tests/nix-unit/structure.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
