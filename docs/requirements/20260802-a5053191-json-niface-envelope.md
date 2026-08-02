---
id: "REQ-a5053191-1c6a-449b-9c5e-5ff49dc5aead"
type: requirement
name: "--json は niface 規約準拠のエンベロープを出す第 2 契約とする"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  When `--json` is given, the CLI SHALL write exactly one niface envelope to stdout. Its
  top level SHALL carry `specVersion` / `tool` / `command` / `status` / `dryRun` /
  `startedAt` / `finishedAt` / `errors[]` / `results[]` in camelCase, with times in
  RFC 3339 using a `T` separator and a mandatory offset (`Z` for UTC). The top level SHALL
  always be `results[]` regardless of single or batch execution, its length being zero or
  more, and SHALL NOT carry a discriminator field for the execution form; `items` /
  `changes` / `info` SHALL live under each `results[i].result`. The default line-oriented
  stdout SHALL be unchanged, `--json` being an opt-in second contract. Errors SHALL be
  structured into the envelope while the human-facing text on stderr SHALL always
  coexist. The exit code table SHALL be unchanged, and the niface `status` SHALL follow it:
  exit 0 maps to `success`, exit 1 and 2 map to `error`. `tool.version` SHALL be the
  ldflags-embedded `main.version`. The JSON output of nput SHALL conform to the niface
  convention, both now and for future features.
specification_ja: |
  `--json` 指定時、CLI は stdout に niface エンベロープを 1 文書だけ出さなければならない。
  トップレベルは `specVersion` / `tool` / `command` / `status` / `dryRun` / `startedAt` /
  `finishedAt` / `errors[]` / `results[]`（camelCase・時刻は RFC 3339・`T` 区切り・
  オフセット必須・UTC は `Z`）とする。single / batch を問わずトップレベルは常に
  `results[]`（要素数 0 以上）とし、実行形態の判別子フィールドを持ってはならない。
  `items` / `changes` / `info` は各 `results[i].result` 配下に置く。デフォルトの行指向
  stdout は不変で、`--json` は opt-in の第 2 契約とする。エラーはエンベロープに構造化
  しつつ、stderr の人間向けテキストを常時併存させなければならない。終了コード表は不変で、
  niface `status` は exit 0 → `success` / 1・2 → `error` に連動させる。`tool.version` は
  ldflags 埋め込みの `main.version` とする。nput の JSON 出力は現在も将来の機能も niface
  規約に準拠しなければならない。
---
# REQ-a5053191: --json は niface 規約準拠のエンベロープを出す第 2 契約とする

## 仕様

`--json`（機械可読出力）は **niface 規約準拠のエンベロープ**を stdout に出す（opt-in の
第 2 契約・niface specVersion 1）。デフォルトのテキスト出力 + ストリーム規律は不変で、
`--json` 時のみ niface エンベロープに切り替わる。

トップレベルは `specVersion` / `tool` / `command` / `status` / `dryRun` / `startedAt` /
`finishedAt` / `errors[]` / **`results[]`**（camelCase・時刻 RFC 3339）。**single / batch を
問わずトップレベルは常に `results[]`**（要素数は 0 以上・実行形態の判別子フィールドは
持たない）で、`items` / `changes` / `info` は各 `results[i].result` 配下に入る。

エラーは niface エンベロープに構造化（前段の全体エラーはトップ `errors[]`・主体起因は
`results[i].errors[]`・item 起因は `item.error`）しつつ **stderr の人間向けテキストも
常時併存**する。終了コード表 0 / 1 / 2 は不変で、niface `status` は exit 0 → `success` /
1・2 → `error` に連動する。`--all` は `results[]` に config ごとの `SubjectResult` を
列挙する（形状は単一実行と同一。`subject` は全 `SubjectResult` で常時必須・
`specVersion` / `tool` / `command` はトップに 1 度だけ）。

**時刻**: RFC 3339・`T` 区切り・オフセット必須（ローカルオフセット・UTC は `Z`）。
`tool.version` は ldflags 埋め込みの `main.version`（VERSION ファイル由来）。

read-only 列挙（`list-generations` の世代・`gitignore` のパス）は
`results[i].result.info` のツール固有インベントリに置き id 導出 item にはしない。

**nput の JSON 出力は現在も将来の機能も niface 規約に準拠する**（エコシステム合成の
北極星要件）。

> **上は原文の写しで、規範は frontmatter が正**。原文が併記する次の点は本 item の
> 規範ではない。
>
> - エラー層の振り分け規則の詳細 → REQ-9341fa5d
> - `--all` の `SubjectResult` の積み方・集約 `status`・順序 → REQ-059eb4d5
> - 変更系 / 読み取り系のペイロードの中身 → REQ-2ea19863 / REQ-fa181aa6
> - 実装 issue（#130 / #131 / #132 / #164）の進捗記述は要求ではなく履歴の注記

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」の「niface 準拠の `--json`
出力」箇条書き（親項目）と、そのサブ項目「時刻」、およびサブコマンド体系節の
`--json` に関する blockquote。

決定の実体は ADR-0043「`--json` 機械可読出力を niface 規約準拠にし、JSON 出力の niface
準拠を恒常原則とする」。`tool.version` の供給元は ADR-0042。
