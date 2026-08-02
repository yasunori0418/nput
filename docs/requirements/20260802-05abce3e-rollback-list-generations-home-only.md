---
id: "REQ-05abce3e-9797-432b-b93f-37c55d09afde"
type: requirement
name: "rollback と list-generations は home mode 限定にする"
derives_from:
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
specification: |
  `nput rollback` and `nput list-generations` SHALL be restricted to home mode. In project
  mode generations SHALL be kept as an internal mechanism and SHALL NOT be exposed to the
  user. `list-generations --all` SHALL list the generations of every home mode config and
  SHALL be read-only.
specification_ja: |
  `nput rollback` と `nput list-generations` は home mode 限定でなければならない。
  project mode では世代を内部機構に留め、ユーザーに公開してはならない。
  `list-generations --all` は home mode の全 config の世代を一覧する読み取り専用の
  操作とする。
---
# REQ-05abce3e: rollback と list-generations は home mode 限定にする

## 仕様

```bash
nput rollback <name>           # 前世代へ戻す（home mode 限定・名指し必須）
nput list-generations <name>   # 世代一覧を表示（home mode 限定）
nput list-generations --all    # home mode の全 config の世代を一覧
```

`rollback` / `list-generations` は **home mode 限定**。project mode は世代を内部機構に
留めユーザーに公開しない。`list-generations --all` は home mode の全 config の世代を
一覧（読み取り専用）。

`rollback` が名指し必須である点は REQ-89c7baf9 の担当。

> **原文「standalone（CLI）」節の残る規範の所在**: 同節の第 1 文（`nput apply <name>` の
> 明示実行と、CLI による entrypoint 発見・manifest ビルド・エンジン駆動）は REQ-f4d7d4ab /
> REQ-1cc080f6、世代管理の機構そのものは REQ-1be4d678 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `rollback` / `list-generations` の
箇条書き、および「モジュール別動作仕様」→「standalone（CLI）」節の第 2 文（home mode で
nix profile による世代管理を行い `rollback` / `list-generations` を提供する）。後者は
独立 item を立てず本 item に畳んだ。

決定の実体は ADR-0005「project mode は世代を内部機構に留めユーザーに公開しない」と、
`list-generations --all` を追加した ADR-0018。
