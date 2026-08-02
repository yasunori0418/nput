---
id: "INF-659b139d-0cf8-4c65-b30d-93c5ee2dfc71"
type: infrastructure
name: "トレーサビリティ検証基盤（sara によるドキュメントグラフの機械検証）"
depends_on:
  - "INF-d1230e1f-8ba8-49d8-8386-409bfbb7dd27"
---
# INF-659b139d: トレーサビリティ検証基盤（sara によるドキュメントグラフの機械検証）

## 構成

`docs/` を [sara](https://github.com/cledouarec/sara) のナレッジグラフとして扱い、要求・設計・
リスク・テストの紐付けが一貫しているかを機械検証する。要求とリスクの紐付けの一貫性管理が主目的で、
「仕様とテストの全体像把握」を支える。

| 部品 | 役割 |
|---|---|
| `docs/model.yaml` | カスタムスキーマ。10 型と relation を定義し、sara の組み込みモデルを全面置換する |
| `sara.toml` | リポジトリ設定 |
| `sara check` の CI ジョブ | broken reference / duplicate ID / cycles を検出する |
| `sara-id` | devShell 同梱の採番コマンド。UUIDv4 を引き、8 文字 prefix の重複を確認して再生成する |

先行して試作された grep / awk による ID 突合は、参照先 ID の実在を検証しないという穴を抱えていた
（存在しない `TC-999` を書いても検出されずカバー率に計上される）。sara の broken reference 検出が
この穴を塞ぐ。

## ID 規約

正式 ID は `<PREFIX>-<フル UUIDv4>`、人間が触る面（ファイル名・散文中の参照）は前方 8 文字の
省略形を使う二層構成。乱数 ID により並列レーンでの採番衝突を構造的に回避する。省略形は正式 ID の
前方一致なので、8 文字で grep すれば宣言側と参照側の両方にヒットする。

ADR のみ連番（`ADR-NNNN`）を維持する。既存 47 本の相互参照・`docs/adr/README.md` の運用・Issue
言及を壊さないため。

## 検証の強度

**非必須チェック + `strict_mode = false`** から開始する。orphan は警告のみに留める — 移行中の
item やテスト未着手の要求が正常に存在するため。マージゲート（INF-8b97573f）の required status
check には当面載せず、DoD の残り枠も温存する。strict 化・必須化は移行の完了後に判断する。

## 出典

ADR-0048（ドキュメントは sara でグラフ構造化する）。設計は epic GitHub Issue #203 の grilling
セッション（2026-07-31・2026-08-01）で確定した。
