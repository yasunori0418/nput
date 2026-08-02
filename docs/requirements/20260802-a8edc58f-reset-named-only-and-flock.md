---
id: "REQ-a8edc58f-4adc-4637-b888-ab8ccc7e73e4"
type: requirement
name: "reset は名指し必須で profileDir 単位の blocking flock を取る"
specification: |
  `nput reset` SHALL require an explicitly named config and SHALL NOT support `--all`,
  because removing everything at once is a destructive footgun, consistently with the
  rejection of `rollback --all`; removing several configs SHALL be done by naming each of
  them in turn. `reset` SHALL acquire a blocking flock keyed on the resolved `profileDir`
  so that it is serialized against a concurrent apply / reset. To determine `profileDir`,
  the preliminary rootKind eval and root resolution SHALL precede even for this
  non-building command, and with `--root` the same roothash key SHALL be used. `reset`
  SHALL additionally evaluate the entrypoint in order to read the entries.
specification_ja: |
  `nput reset` は名指し必須とし、`--all` に対応してはならない（一斉撤去は破壊的な
  footgun であり、`rollback --all` の却下と一貫させるため）。複数撤去は名指しを複数回行う。
  `reset` は解決後 `profileDir` 単位の blocking flock を取得して、並行する apply / reset と
  直列化しなければならない。profileDir 確定のため、build しないコマンドでも rootKind
  先取り eval → root 解決を先行させる。`--root` 時は同じ roothash キーを使う。
  `reset` はさらに entries 読みのため entrypoint eval も行う。
---
# REQ-a8edc58f: reset は名指し必須で profileDir 単位の blocking flock を取る

## 仕様

`reset` は **名指し必須（`--all` 非対応）**（一斉撤去は破壊的 footgun・`rollback --all`
却下と一貫）。複数撤去は名指しを複数回。**解決後 `profileDir` 単位の blocking flock を
取得**して並行 apply / reset と直列化する。profileDir 確定のため **rootKind 先取り eval →
root 解決**を build しないコマンドでも先行する（apply と共通の前段・`--root` 時は同じ
roothash キー）。`reset` はさらに entries 読みのため entrypoint eval も行う。

非 build コマンド一般の eval 先行は REQ-9c111c32 が規定し、本 item は `reset` が
それに従い entrypoint eval も加えることを規定する。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `reset` 名指し必須の箇条書き。

決定の実体は ADR-0021「reset の `--all` 非対応・`--dryrun` 対応・flock / recopy 合成を
確定する」と、`--all` 非対応の一貫性を与える ADR-0018。flock の直列化は ADR-0013、
非 build コマンドの eval 先行と `--root` の roothash キーは ADR-0024。
