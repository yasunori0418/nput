---
id: "INF-659b139d-0cf8-4c65-b30d-93c5ee2dfc71"
type: infrastructure
name: "トレーサビリティ検証基盤（sara によるドキュメントグラフの機械検証）"
depends_on:
  - "INF-d1230e1f-8ba8-49d8-8386-409bfbb7dd27"
satisfies:
  - "QA-6bf957d9-17d9-4660-92b7-ebd6eeb71a8c"
---
# INF-659b139d-0cf8-4c65-b30d-93c5ee2dfc71: トレーサビリティ検証基盤（sara によるドキュメントグラフの機械検証）

## 構成

`docs/` を [sara](https://github.com/cledouarec/sara)（Rust 製 CLI。個人 NUR 経由で dev flake
へ入れる）のナレッジグラフとして扱い、要求・設計・リスク・テストの紐付けが一貫しているかを
機械検証する。要求とリスクの紐付けの一貫性管理が主目的で、「仕様とテストの全体像把握」を支える。

| 部品 | 役割 |
|---|---|
| `docs/model.yaml` | カスタムスキーマ。11 型と relation を定義し、sara の組み込みモデルを全面置換する |
| `sara.toml` | リポジトリ設定 |
| `sara` ジョブ | `test.yml` 内の 1 ジョブ。`sara check` で broken reference / duplicate ID / cycles を検出し、同じジョブで契約テスト 3 件を実行する |
| `dev/tests/sara-id.sh` | 採番契約。`sara-id` の出力形式・重複時の再生成・prefix マップと `docs/model.yaml` の一致を検証する |
| `dev/tests/test-doc-map.sh` | テスト資産 ⇔ CASE の 1:1 対応を検証する（→ Issue #304）。区分と除外の正本は `dev/tests/` のデータファイルが持つ |
| `dev/tests/risk-matrix.sh` | risk の `level` が `likelihood` × `impact` のマトリクス導出と一致するかを検証する（→ Issue #303）。マトリクスの正本は `dev/tests/risk-matrix.tsv` |
| `sara-id` | devShell 同梱の採番コマンド。UUIDv4 を引き、8 文字 prefix の重複を確認して再生成する |

契約テストを同じジョブへ載せるのは、テストが「人が思い出したときだけ動く」状態を避けるため。
3 件はいずれも dev flake の `checks.*` としても露出しているが、CI の `flake-check` はルート
flake を対象にするため到達しない。CI の明示ステップとして固定する。

`sara` ジョブは `test.yml` に同居するが、CI パイプライン（INF-d1230e1f-8ba8-49d8-8386-409bfbb7dd27）の変更検出ジョブは
**再利用しない**。あちらの filter は nix / Go のソースを対象にするのに対し、この検査が最も効くのは
docs のみの PR だからで、job ローカルの filter を別に置く。

filter が拾うのは 3 種類。検査対象の文書とその設定（`docs/**` / `sara.toml`）、この検査自身の
実行環境を決めるもの（sara 本体と sara-id を供給する dev flake・その lock・契約テストの実装＝
`dev/tests/**` とそれが依存する `dev/scripts/**`・この workflow 自身）、そして**契約テストが
入力に取るテスト資産そのもの**（Go のテストファイル・`tests/**`・`flake.nix` の checks 定義）。
3 つ目は `test-doc-map.sh` が「テスト資産 ⇔ CASE の 1:1」を検査する以上避けられない — テスト
ファイルを追加・改名しても CASE を更新しなければジョブごとスキップされ、検出したいはずの
乖離がそのままマージされてしまう（→ Issue #304）。
CI 用の devShell も sara 専用のものへ分ける。

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

**`strict_mode = true` の非必須チェック**。検査自体は strict で、orphan を含む全 warning が
error となり `sara check` は exit 1 を返す（epic #203 の移行完了・warning 0 到達を受けて有効化
→ ADR-0050）。合格条件は「warning 0 件」に一本化されている。

一方でマージゲート（INF-8b97573f-d4d6-4abf-85e2-d859afbd96c6）の required status check には登録せず、DoD にも入れない。
これは移行中の暫定措置ではなく「**検出は機械・判断は人**」という分担を保つための恒久判断で、
接続漏れの修正が自明でないケース（item 新設を伴う等）を CI が赤いままレビューで扱える余地を
残す。再検討条件は ADR-0050 が持つ。

閾値ゲートを持たない点は CI パイプライン（INF-d1230e1f-8ba8-49d8-8386-409bfbb7dd27）の `go-coverage` ジョブと同方針で、
他 PR とのマージ順依存を避けるためでもある。

## 出典

ADR-0048（ドキュメントは sara でグラフ構造化する）。設計は epic GitHub Issue #203 の grilling
セッション（2026-07-31・2026-08-01）で確定した。
