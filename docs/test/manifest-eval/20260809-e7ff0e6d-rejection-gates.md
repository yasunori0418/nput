---
id: "TC-e7ff0e6d-32d7-4ed6-8c2f-449dba34b2f6"
type: test_condition
name: "拒否すべき入力が評価時に throw することをアサートする"
mitigates:
  - "RISK-09df40d3-2752-433e-9ab0-2816fbd14969"
---
# TC-e7ff0e6d: 拒否ゲートが評価時に throw することをアサートする

## テスト条件

「この入力は評価が失敗する」を、エラー種別と拒否されたフィールドの名指しで直接アサートする。
正常系のアサートでは、ゲートが外れたことを検出できないため。判定ロジックそのものを private
helper 越しに単体で見るのは TC-311ca3b2 の担当で、検証境界が異なる。

主たる経路は `normalizeManifest`（evalModules）だが、型の共有を確かめる 1 件だけは
`entriesType` を直接 `evalModules` して見る（`normalizeManifest` を通さない）。同じ型定義が
モジュール経路でも strict であることは、公開経路のアサートからは導けないため。

**`systemRoot` の未実装拒否はこの条件に含めない**。実装（`gates.nix`）には当該アサートが
残っているが、その決定（ADR-0013 §5）は ADR-0036 が撤回済みで、現行の規範は
`rootKind = "system"` を正規の値とする。これは ADR-0036 が指示したテスト更新（拒否テスト →
変換テスト）の未了であり、規範として条件化しない（→ CASE-879a93da の該当箇所も同じ扱い）。

**ゲートごとの拒否**:

- `method = "copy"` と out-of-store marker の組み合わせ（意図の矛盾）
- 絶対パスの target / `..` で root の外へ出る target / `..` で src の外へ出る subpath
- 別キーから同一 target を指す衝突
- 未知キー・旧名（strict submodule）。`modules/common.nix` が共有する entriesType でも
  同じく eval エラーになること
- 素の文字列 src（out-of-store は marker による opt-in であって暗黙変換しない）

## 覆う CASE

- CASE-879a93da（`tests/nix-unit/gates.nix`）

## 出典

`tests/nix-unit/gates.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
