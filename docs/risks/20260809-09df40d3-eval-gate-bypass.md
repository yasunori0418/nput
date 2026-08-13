---
id: "RISK-09df40d3-2752-433e-9ab0-2816fbd14969"
type: risk
name: "評価時に止めるべき入力が素通りして engine 実行時まで到達する"
likelihood: low
impact: high
level: medium
threatens:
  - "REQ-6911eab6-12b4-457c-9db4-d7430a9e9b3f"
  - "REQ-d1b5b3f5-10a0-400d-9f03-ba00c63d1c34"
  - "REQ-3e446ad9-a6f4-4229-b5c5-184754b0ef51"
  - "REQ-a33a11e3-830d-4142-88ed-4c1fc35e7f74"
  - "REQ-16faf428-77f3-492f-b858-222c5274cbf7"
  - "REQ-5c6b07da-3d06-414d-8770-4f438234b322"
  - "REQ-99ca5381-6c53-426c-b145-7b4297c53868"
  - "REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10"
---
# RISK-09df40d3: 評価時に止めるべき入力が素通りして engine 実行時まで到達する

## リスク

`mkManifest` は `evalModules` による入力検査の単一ゲート（REQ-d1b5b3f5）で、パス安全性
（REQ-6911eab6）・未知キーと旧名の拒否（REQ-3e446ad9）・entry のフィールドを 4 つに限ること
（REQ-a33a11e3）・意図が矛盾する組み合わせ（REQ-16faf428）・同一 manifest 内の target 衝突
（REQ-5c6b07da）・素の文字列 src の拒否（REQ-99ca5381）を全てここで止める。この検査は
`modules/common.nix` が共有する entriesType にも効く（REQ-fc1c7ce6）。

**`systemRoot` の未実装拒否はここに含めない**。この決定（ADR-0013 §5）は ADR-0036 が撤回済み
で、現行の規範は `rootKind = "system"` を正規の値とする（REQ-37b56673）。既存 REQ-16faf428 /
REQ-c5dfcae6 も同じ理由でこの拒否を規範から外している。実装（`lib/manifest.nix` の throwIf と
`gates.nix` の拒否テスト）に残っているのは ADR-0036 が指示した更新の未了であり、逆算元とすべき
規範ではない。同様に、root 省略時のエラー（REQ-4ec3accc の明示必須）を見るアサートは評価
テストに存在しないため、本 risk は REQ-4ec3accc へ `threatens` を張らない。

**顕在化したときに起きること**: ゲートが緩むと、`..` や絶対パスで root の外を指す target が
engine まで届き、意図しない場所を書き換える。タイポした旧名は黙って無視され、ユーザーは
宣言したはずの entry が配置されない理由を追えない。target 衝突は最後に評価された側が勝ち、
どちらが配置されたか宣言からは読み取れない。いずれも「評価時に止まる」ことが安全の前提に
なっており、実行時まで到達した時点で被害は FS に出る。

## 評価

- likelihood: low — ゲートは型定義とクロスフィールド検査の 1 箇所に集約されており、
  境界条件（`..` の深さ・0 ちょうど）まで含めて nix-unit が対で覆っている。本 risk が
  扱う「止めるべき入力が素通りする」方向、すなわち throw しなくなる退行はこの検査が
  確実に捕まえる（どのゲートが throw したかの取り違えは CASE-879a93da の部分一致では
  区別できないが、それは素通りとは別の失敗モードで RISK-3de9753f の担当）
- impact: high — 評価時のゲートが形骸化する類型なので、規約の継承則に従い「そのゲートが
  捕まえるはずの regression の impact」を継ぐ。ここで継ぐのはパス安全性の破れ、すなわち
  `..` や絶対パスが root の外を指したまま engine へ届く経路で、書き込み先は nput の記録に
  残らず原状回復の対象にすらならない。ゲート自体が評価時に閉じることを理由に low へは落とさない

## 対処

TC-e7ff0e6d（各ゲートが評価時に `ThrownError` を投げること。主経路は `normalizeManifest`
だが、型共有の 1 件は `entriesType` を直接評価する）・TC-311ca3b2（private helper を単体で
叩いてパス脱出判定の深さ 0 境界を内側・外側の対で見ること）で緩和する。前者はエラーの到達性、
後者は判定ロジック単体という軸の違いで 2 つの TC に分ける。

## 出典

`tests/nix-unit/gates.nix` / `tests/nix-unit/escapes-base.nix` の現行実装からの逆算
（→ Issue #273「L1〜L4」節）。各ゲートの設計判断は ADR-0008 / ADR-0010 / ADR-0013 /
ADR-0019 / ADR-0024 が持つ（ADR-0013 のうち §5 は ADR-0036 が撤回済み）。
