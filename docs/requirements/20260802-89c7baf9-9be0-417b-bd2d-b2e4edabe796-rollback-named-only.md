---
id: "REQ-89c7baf9-9be0-417b-bd2d-b2e4edabe796"
type: requirement
name: "rollback は名指し必須で --all に対応しない"
derives_from:
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
specification: |
  `nput rollback` SHALL require an explicitly named config and SHALL NOT support `--all`,
  because rolling every config back at once is destructive and a footgun, and a failure
  partway through can leave the state inconsistent.
specification_ja: |
  `nput rollback` は名指し必須とし、`--all` に対応してはならない。全 config を一斉に
  戻すのは破壊的で footgun であり、途中失敗で状態が不揃いになり得るため。
---
# REQ-89c7baf9-9be0-417b-bd2d-b2e4edabe796: rollback は名指し必須で --all に対応しない

## 仕様

`rollback` は **名指し必須**（`--all` 非対応）。全 config を一斉に戻すのは破壊的で
footgun、途中失敗で状態が不揃いになり得るため。

同じ理由による `reset` の名指し必須は REQ-a8edc58f-4adc-4637-b888-ab8ccc7e73e4 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `rollback` 名指し必須の箇条書き。

決定の実体は ADR-0018「`--all` のサブコマンド対応範囲」（`rollback --all` は非対応）。
