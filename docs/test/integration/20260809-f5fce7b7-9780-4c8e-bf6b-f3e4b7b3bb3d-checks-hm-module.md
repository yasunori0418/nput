---
id: "CASE-f5fce7b7-9780-4c8e-bf6b-f3e4b7b3bb3d"
type: test_case
name: "checks.hm-module — activation の配線を評価アサートで検証する flake check"
target: "checks.hm-module"
covers:
  - "TC-5a72b00e-2c03-4703-acdc-33cffb6fa994"
---
# CASE-f5fce7b7-9780-4c8e-bf6b-f3e4b7b3bb3d: checks.hm-module — activation の配線を評価アサートで検証する flake check

## 対象

`flake.nix` の `checks.hm-module`（対象のモジュール実体は `modules/home-manager.nix`）

## 検証内容

standalone な home-manager configuration をビルドの sandbox 内で評価し、生成された
activation スクリプトと、それが参照する文書に対してアサートする。activate はしない。

- activation が統合先のファイル配置機構へ翻訳せず、ビルド済みの配置物を渡して engine を
  `apply --manifest` の形で起動していること
- 起動に渡る文書が実在し、root の種別として home を固定していること
- モジュール上で宣言した entry が、その属性キーを配置先として文書へ流れていること
- 退避のオプションを有効にし、かつサフィックスを省略したとき、既定のサフィックスが同じ起動へ
  フラグとして乗ること

配置元は固定の store パスを持つ test double で与え、文書の内容をマシンによらず安定させる。
home-manager はこの check のためだけの入力として持ち込み、ライブラリ本体には依存を波及させ
ない。公開用のラッパーを経由せず、モジュール本体へ CLI を直接注入して評価する。
