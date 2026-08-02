---
id: "REQ-31dae599-f3a3-4bbe-b367-c955535265da"
type: requirement
name: "reset の確認プロンプトは stdin が TTY のときだけ出し、非 TTY で同意が無ければ即エラー停止する"
specification: |
  The confirmation prompt of `nput reset` SHALL be shown only while stdin is a TTY. When
  stdin is not a TTY — under CI, direnv or a pipe — and neither `-y` nor `--yes` has been
  given, the command SHALL NOT show the prompt and SHALL stop immediately with an error
  and exit code 1, so as to prevent both hanging and an accidental deletion caused by
  empty input.
specification_ja: |
  `nput reset` の確認プロンプトは stdin が TTY のときのみ出さなければならない。非 TTY
  （CI / direnv / パイプ）かつ `-y` / `--yes` が未指定なら、プロンプトを出さず即エラー停止
  （exit 1）しなければならない（ハングと空入力による誤削除を防ぐため）。
---
# REQ-31dae599: reset の確認プロンプトは stdin が TTY のときだけ出し、非 TTY で同意が無ければ即エラー停止する

## 仕様

確認プロンプトは **stdin が TTY のときのみ**出す。**非 TTY（CI / direnv / パイプ）かつ
`-y/--yes` 未指定なら、プロンプトを出さず即エラー停止（exit 1）**する（ハングと空入力誤削除を
防ぐ）。

> **上は原文の写しで、規範は frontmatter が正**。データ損失リスクのため確認プロンプトか
> `-y` / `--yes` で同意を要求すること自体は REQ-31f2882e の担当で、本 item はその同意取得が
> TTY 有無でどう振る舞うかだけを規定する。`--json` 指定時に TTY でもプロンプトを出さない
> ことは REQ-2a613337 の担当。

## 出典

`docs/spec.md`「配置動作仕様」→「recopy・reset」節の reset 箇条書き第 1 項。

決定の実体は ADR-0025「実装前残セマンティクス第7巡」の「reset 非TTY」。
