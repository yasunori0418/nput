---
id: "TC-5a72b00e-2c03-4703-acdc-33cffb6fa994"
type: test_condition
name: "activation が評価時点で engine 起動へ配線され root と entry とオプションを届ける"
mitigates:
  - "RISK-d734e24e-9af9-49f1-a22d-ade92f6554eb"
  - "RISK-7808768d-bd87-429a-af42-52e8559d940c"
---
# TC-5a72b00e: activation が評価時点で engine 起動へ配線され root と entry とオプションを届ける

## 条件

統合層のモジュールを評価し、生成された activation の内容に対してアサートする。実際に
activate はしない。この層が独立して必要なのは、ここで確かめる性質が**評価時に決まるのに
activate 後にしか観測できない**ためで、評価だけで先に捕まえられる。

- **engine キックへの配線** — activation が entry を統合先のファイル配置機構へ翻訳せず、
  ビルド済みの配置物を渡して engine を起動する形になっていること
- **root の pin** — engine へ渡す文書が home root を固定していること
- **entry の到達** — モジュール上で宣言した entry が、その属性キーを配置先として文書へ
  流れていること
- **オプションの配線** — モジュールのオプションが対応するフラグとして同じ起動に乗ること。
  サフィックスを省略したときに既定値が入ること

実 activate を要するもの（profile の世代コミット・実際の配置）はこの条件の外に置く。ビルドの
sandbox には変更すべき profile が無いため。

配置元は固定の store パスを持つ test double で与え、評価結果をマシンによらず安定させる。
