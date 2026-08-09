---
id: "TC-e7ff0e6d-32d7-4ed6-8c2f-449dba34b2f6"
type: test_condition
name: "拒否すべき入力が評価時に throw することと、パス脱出判定の境界をアサートする"
mitigates:
  - "RISK-09df40d3-2752-433e-9ab0-2816fbd14969"
---
# TC-e7ff0e6d: 拒否ゲートとパス脱出判定の境界をアサートする

## テスト条件

「この入力は評価が失敗する」を、エラー種別とメッセージの一部で直接アサートする。正常系の
アサートでは、ゲートが外れたことを検出できないため。

**ゲートごとの拒否**（`normalizeManifest` の evalModules 経路）:

- 未実装の root（systemRoot）
- `method = "copy"` と out-of-store marker の組み合わせ（意図の矛盾）
- 絶対パスの target / `..` で root の外へ出る target / `..` で src の外へ出る subpath
- 別キーから同一 target を指す衝突
- 未知キー・旧名（strict submodule）。`modules/common.nix` が共有する entriesType でも
  同じく eval エラーになること
- 素の文字列 src（out-of-store は marker による opt-in であって暗黙変換しない）

**パス脱出判定の境界**（`escapesBase` / `isUnsafe`）: `.` と空文字が深さを動かさないこと、
深さがちょうど 0 まで戻る場合は脱出としないこと、その 1 つ外側で脱出とすること、途中で
`..` に当たっても深さが残っていれば脱出としないこと、絶対パスは脱出判定とは独立に拒否される
こと。境界の内側と外側を対で置き、片側だけのアサートにしない。

## 覆う CASE

- CASE-879a93da（`tests/nix-unit/gates.nix`）
- CASE-36b5b61a（`tests/nix-unit/escapes-base.nix`）
