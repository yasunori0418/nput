---
id: "REQ-8ef34101-8150-4124-92d5-94fabe6b5d90"
type: requirement
name: "成功時はデフォルト沈黙とし warning と error は常時 stderr へ出す"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
specification: |
  The default SHALL be silence on success. The success of `apply` / `reset` / `rollback`
  SHALL be conveyed by exit code 0, and the placement report (summary plus per-target
  lines), the try-lock skip notice and the `apply --all` completion summary SHALL NOT be
  emitted by default. Warnings (such as a foreign symlink) and errors SHALL always be
  written to stderr and SHALL be outside the scope of the silencing. The confirmation
  prompt and the abort notice of `reset` SHALL also remain.
specification_ja: |
  成功時はデフォルト沈黙でなければならない。`apply` / `reset` / `rollback` の成功は
  終了コード 0 が語り、配置レポート（サマリ + per-target 行）・try-lock skip 通知・
  `apply --all` 完了サマリは既定では出さない。warning（foreign symlink 等）と error は
  常時 stderr に出さなければならず、沈黙の対象外とする。`reset` の確認プロンプト・
  中止通知も存続させる。
---
# REQ-8ef34101: 成功時はデフォルト沈黙とし warning と error は常時 stderr へ出す

## 仕様

**成功時はデフォルト沈黙**（UNIX 哲学「沈黙は金」）。`apply` / `reset` / `rollback` の
**成功は終了コード 0 が語り、配置レポート（サマリ + per-target 行）・try-lock skip 通知・
`apply --all` 完了サマリは既定では出さない**。**warning（foreign symlink 等）と error は
常時 stderr に出す**（沈黙対象外）。`reset` の確認プロンプト・中止通知も存続する。

`-v` による opt-in と、stdout 専有出力が沈黙の対象外である点は REQ-0a123b89 /
REQ-fea038de の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」の第 1 箇条書き。

決定の実体は ADR-0031「成功時デフォルト沈黙化・`-v` で配置レポート opt-in・`--debug` で
nix コマンド開示・`--quiet` 廃止」。
