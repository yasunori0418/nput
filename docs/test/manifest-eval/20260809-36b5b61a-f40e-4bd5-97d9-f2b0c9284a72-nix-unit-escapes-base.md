---
id: "CASE-36b5b61a-f40e-4bd5-97d9-f2b0c9284a72"
type: test_case
name: "nix-unit: escapes-base.nix — `..` 深さ判定の境界網羅"
target: "tests/nix-unit/escapes-base.nix"
covers:
  - "TC-311ca3b2-d6d1-4712-9e2a-9027941d3527"
---
# CASE-36b5b61a-f40e-4bd5-97d9-f2b0c9284a72: nix-unit escapes-base.nix

## 対象

`tests/nix-unit/escapes-base.nix`（TP-403c55c7-d996-4951-8e6b-c3a7dddd387c のテスト seam `nput.__internal` 経由で private
helper `escapesBase` / `pathChecks.isUnsafe` を直接叩く）

## 検証内容

`escapesBase` を、深さ 0 の境界を挟んで内側と外側の対でアサートする。

- 深さを動かさない入力: `.` / 空文字 → 脱出しない
- 単体の `..` と先頭の `../a` → 深さ 0 で `..` に当たり即脱出
- 通常の下降 `a/b` → 脱出しない
- ちょうど 0 まで戻る `a/b/../..` → 脱出しない（境界の内側）
- 1 つ外側の `a/b/../../..` → 脱出（境界の外側）
- 途中で `..` に当たるが深さが残る `a/../b` → 脱出しない
- 途中で負に踏み込む `a/../../b` → 後続が安全でも脱出確定

`isUnsafe` については、脱出しない相対パスが安全、絶対パス（`/etc/x`）が `escapesBase` とは
独立に拒否、脱出する相対パス（`../../etc`）が `escapesBase` 経由で拒否されることを見る。

## 出典

`tests/nix-unit/escapes-base.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
パス安全性検査の設計判断は ADR-0019 が持つ。
