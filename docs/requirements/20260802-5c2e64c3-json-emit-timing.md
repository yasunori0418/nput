---
id: "REQ-5c2e64c3-09a7-4ae8-b60c-4f1ccabce4fd"
type: requirement
name: "エンベロープはコマンド完了時に 1 回だけ出し成立条件を満たさない実行では出さない"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The envelope SHALL be written to stdout exactly once, at command completion (one
  document plus a trailing newline, with nothing else on stdout). It SHALL be emitted only
  for an execution of a nput subcommand proper, that is, one that passed flag parsing and
  argument validation and reached RunE. `--help` / `--version` and the cobra-generated
  `help` / `completion` SHALL NOT emit an envelope, since their own text occupies stdout.
  A failure of flag parsing or argument validation SHALL likewise emit no envelope,
  because whether `--json` was given cannot be determined at that point, and the exit code
  plus stderr SHALL be the signal instead. A failure to write the envelope itself (EPIPE
  and the like) SHALL cause a non-zero exit even when the command itself succeeded, so
  that a missing document is not read as success.
specification_ja: |
  エンベロープはコマンド完了時に stdout へ 1 回だけ書かなければならない（1 文書 +
  末尾改行・それ以外 stdout には何も出さない）。emit するのは nput 自身のサブコマンド
  実行（フラグ解析・引数検証を通過して RunE に到達したもの）のみでなければならない。`--help` /
  `--version`・cobra 自動生成の `help` / `completion` はエンベロープを出してはならない
  （それぞれのテキストが stdout を使うため）。フラグ解析・引数検証の失敗もエンベロープ
  なしとしなければならず（`--json` の指定有無をその時点で確定できないため）、終了コード +
  stderr をシグナルとしなければならない。エンベロープ書き込み自体の失敗（EPIPE 等）は
  コマンド成功時でも
  非ゼロ終了にしなければならない（欠損文書を成功と読ませないため）。
---
# REQ-5c2e64c3: エンベロープはコマンド完了時に 1 回だけ出し成立条件を満たさない実行では出さない

## 仕様

**emit タイミングと成立条件**: エンベロープは**コマンド完了時に 1 回だけ** stdout へ書く
（1 文書 + 末尾改行・それ以外 stdout には何も出ない）。emit するのは nput 自身の
サブコマンド実行（フラグ解析・引数検証を通過して RunE に到達したもの）のみ。
`--help` / `--version`・cobra 自動生成の `help` / `completion` はエンベロープを出さない
（それぞれのテキストが stdout を使う）。フラグ解析・引数検証の失敗もエンベロープなし
（`--json` の指定有無をその時点で確定できないため）で、終了コード + stderr がシグナル。
エンベロープ書き込み自体の失敗（EPIPE 等）はコマンド成功時でも非ゼロ終了にする
（欠損文書を成功と読ませない）。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」→「niface 準拠の `--json`
出力」のサブ項目「emit タイミングと成立条件」。

決定の実体は ADR-0043。
