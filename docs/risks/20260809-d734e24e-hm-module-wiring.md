---
id: "RISK-d734e24e-9af9-49f1-a22d-ade92f6554eb"
type: risk
name: "home-manager モジュールが engine をキックする配線から外れ native 機構へ翻訳される"
threatens:
  - "REQ-8085f194-c903-4ecb-abd8-c719fe7b3292"
  - "REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9"
  - "REQ-e1e1114b-ba07-4d57-8e04-6e30e39a5da3"
  - "REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a"
likelihood: medium
impact: high
level: high
---
# RISK-d734e24e: home-manager モジュールが engine をキックする配線から外れ native 機構へ翻訳される

## リスク

統合層のモジュールは engine を起動するだけの配線であり、entry を統合先の native な配置機構へ
翻訳しない、というのが設計全体の前提である。この前提が崩れると、配置の意味論が統合層ごとに
分岐し、engine が一手に引き受けているはずの不変条件（保守的な stale 除去・原子性）が経路に
よって成り立たなくなる。

崩れ方は次の形をとる。

- **翻訳への退化** — activation が engine を呼ばず、統合先のファイル配置機構へ entry を
  流し込む。単体では動いてしまうため、engine 側の保証を失ったことに気づきにくい
- **root の取り違え** — モジュール経由の配置先が home root に固定されず、利用者側の指定で
  上書きされる
- **entry の欠落** — モジュール上で宣言した entry が manifest へ届かない
- **オプションの配線漏れ** — モジュールのオプション（退避の有効化とそのサフィックス）が、
  engine の起動時のフラグとして届かない。既定値を省略したときに既定が入らない、という形でも
  起きる
- **config ごとの独立性の喪失** — 名前つきの config ごとに独立した profile を取れず、役割
  分離が崩れる

これらは評価の時点で決まるが、実際に activate するまで観測できない性質を持つ。activate を
伴わない検証層が無ければ、統合先の実環境でしか壊れが露見しない。

## 影響

宣言的に管理しているつもりの配置が engine の保証を外れる。level を high とするのは、
home-manager 経路が nput の主要な利用導線の 1 つであり、崩れたときに利用者が気づく手段が
乏しいため。
