---
id: "REQ-840b3641-6e76-46da-82e9-680cabd65abe"
type: requirement
name: "失敗時に残る pending gcroot は config あたり最大 1 個に有界とし回収処理を持たない"
derives_from:
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
specification: |
  When an apply fails before reaching `--set`, the `<profileDir>/.pending` gcroot remains
  and keeps holding a built but unused link-farm. Because the next apply overwrites it
  under the same name (`.pending`), it SHALL be bounded to at most one per config, and no
  reclamation process SHALL be provided; this SHALL be accepted.
specification_ja: |
  `--set` 到達前に apply が失敗すると `<profileDir>/.pending` gcroot が残り、ビルド済み
  未使用 link-farm を掴み続ける。次回 apply が同名（`.pending`）で上書きするため config
  あたり最大 1 個に有界でなければならず、回収処理を持ってはならない。これは許容
  しなければならない。
---
# REQ-840b3641-6e76-46da-82e9-680cabd65abe: 失敗時に残る pending gcroot は config あたり最大 1 個に有界とし回収処理を持たない

## 仕様

`--set`（実行フローの f）到達前に apply が失敗すると `<profileDir>/.pending` gcroot が
残り、ビルド済み未使用 link-farm を掴み続けるが、次回 apply が**同名**（`.pending`）で
上書きするため config あたり最大 1 個に有界。回収処理は持たず許容する。

pending out-link を張る段そのものは REQ-60c6b7ea-e936-4ce8-bd75-ad35e9c693b9 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「実行フロー」の `.pending` gcroot 残存の箇条書き。

決定の実体は ADR-0016「pending gcroot」。
