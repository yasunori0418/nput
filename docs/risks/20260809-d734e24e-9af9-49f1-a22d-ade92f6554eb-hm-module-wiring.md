---
id: "RISK-d734e24e-9af9-49f1-a22d-ade92f6554eb"
type: risk
name: "home-manager モジュールが engine をキックする配線から外れ native 機構へ翻訳される"
threatens:
  - "REQ-8085f194-c903-4ecb-abd8-c719fe7b3292"
  - "REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9"
  - "REQ-e1e1114b-ba07-4d57-8e04-6e30e39a5da3"
  - "REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a"
  - "REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10"
likelihood: medium
impact: high
level: high
---
# RISK-d734e24e-9af9-49f1-a22d-ade92f6554eb: home-manager モジュールが engine をキックする配線から外れ native 機構へ翻訳される

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
  分離が崩れる。ただし現状の hm 経路の検証（評価アサート・実 activate とも）は単一 config
  しか通しておらず、この崩れ方だけは下流のテスト条件で受けられていない

これらは評価の時点で決まるが、実際に activate するまで観測できない性質を持つ。activate を
伴わない検証層が無ければ、統合先の実環境でしか壊れが露見しない。

## 影響

宣言的に管理しているつもりの配置が engine の保証を外れる。崩れ方のうち軽い側——root が
固定されない・entry やオプションが届かない——は配線を直したうえでの再 activation が engine
経由の配置へ戻すので medium に留まるが、最も重い「翻訳への退化」はそうではない。engine を
呼ばず統合先の native な配置機構へ entry を流し込むと、配置も撤去も engine の外で行われ、
home-manager の readlink パターンによる cleanup が記録外の実体を消す経路が開く（この取り違えを
避けるために nput が manifest 記録による分類を採っていることは RISK-e3d42a21-1f43-4ac6-835e-a5caf8d86363 が述べる。ただし
同 item の射程は engine 自身の除去実装で、engine を経由しない配置は射程外なので、この経路の
impact は本 item が持つ）。消えるのは nput の記録に無い実体なので再 activation では戻らない。
規約の最重方向則に従い impact を high とする。home-manager 経路が主要な利用導線でありながら
崩れても単体では動いてしまい、保証を失ったことに利用者が気づく手段が乏しい点は沈黙性として
残る。

likelihood を medium とするのは、モジュールが配線するオプションが増えるたびに engine の
起動フラグへの受け渡し漏れが入りうる一方、これらは評価の時点で決まるため、評価アサートを
持つ検証層が退行の多くを捕まえるため。
