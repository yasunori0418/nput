---
id: "RISK-09df40d3-2752-433e-9ab0-2816fbd14969"
type: risk
name: "評価時に止めるべき入力が素通りして engine 実行時まで到達する"
likelihood: medium
impact: high
level: high
threatens:
  - "REQ-6911eab6-12b4-457c-9db4-d7430a9e9b3f"
  - "REQ-d1b5b3f5-10a0-400d-9f03-ba00c63d1c34"
  - "REQ-3e446ad9-a6f4-4229-b5c5-184754b0ef51"
  - "REQ-a33a11e3-830d-4142-88ed-4c1fc35e7f74"
  - "REQ-16faf428-77f3-492f-b858-222c5274cbf7"
  - "REQ-5c6b07da-3d06-414d-8770-4f438234b322"
  - "REQ-99ca5381-6c53-426c-b145-7b4297c53868"
  - "REQ-4ec3accc-8bb6-461f-9024-dcf0027849e4"
  - "REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10"
---
# RISK-09df40d3: 評価時に止めるべき入力が素通りして engine 実行時まで到達する

## リスク

`mkManifest` は `evalModules` による入力検査の単一ゲート（REQ-d1b5b3f5）で、パス安全性
（REQ-6911eab6）・未知キーと旧名の拒否（REQ-3e446ad9）・entry のフィールドを 4 つに限ること
（REQ-a33a11e3）・意図が矛盾する組み合わせ（REQ-16faf428）・同一 manifest 内の target 衝突
（REQ-5c6b07da）・素の文字列 src の拒否（REQ-99ca5381）・未実装 root の拒否（REQ-4ec3accc の
明示必須と併せた systemRoot）を全てここで止める。この検査は `modules/common.nix` が共有する
entriesType にも効く（REQ-fc1c7ce6）。

**顕在化したときに起きること**: ゲートが緩むと、`..` や絶対パスで root の外を指す target が
engine まで届き、意図しない場所を書き換える。タイポした旧名は黙って無視され、ユーザーは
宣言したはずの entry が配置されない理由を追えない。target 衝突は最後に評価された側が勝ち、
どちらが配置されたか宣言からは読み取れない。いずれも「評価時に止まる」ことが安全の前提に
なっており、実行時まで到達した時点で被害は FS に出る。

## 評価

- likelihood: medium — 境界条件（`..` の深さ・0 ちょうど）は実装変更で崩れやすい一方、
  ゲート自体は型定義とクロスフィールド検査に集約されている
- impact: high — パス安全性の破れは root 外への書き込みで、原状回復の対象にすらならない
- level: high

## 対処

TC-e7ff0e6d で緩和する。同 TC は各ゲートが `ThrownError` を投げることと、`escapesBase` の
深さ 0 境界を内側・外側の対でアサートすることの両方を持つ。
