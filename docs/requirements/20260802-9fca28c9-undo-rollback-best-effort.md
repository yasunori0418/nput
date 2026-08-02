---
id: "REQ-9fca28c9-d3b1-4ad7-8f24-13b2ec7aeab2"
type: requirement
derives_from:
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
name: "巻き戻し自体の失敗は best-effort で続行し、全件を stderr へ報告して停止する"
specification: |
  A failure of an individual inverse operation SHALL NOT abort the rollback: the remaining
  entries of the undo journal SHALL still be rolled back. Once every rollback attempt has
  been made, the original apply error together with a list of the items that could not be
  rolled back SHALL be reported in full on stderr, and the command SHALL then stop, taking
  the same shape as conflict reporting — enumerate everything, then raise one aggregated
  error. This report SHALL always go to stderr and SHALL NOT be subject to the
  silent-on-success discipline, being a failure path.
specification_ja: |
  個々の逆操作の失敗によって巻き戻しを中断してはならず、undo ジャーナルの残りの巻き戻しは
  続行しなければならない。全ての巻き戻し試行が終わった時点で、元の apply エラーと巻き戻せ
  なかった項目の一覧を stderr へ全報告して停止する（conflict 報告と同じ「全件列挙してから
  1 本の集約エラー」の形）。この報告は失敗経路のため常時 stderr とし、成功時沈黙の対象に
  してはならない。
---
# REQ-9fca28c9: 巻き戻し自体の失敗は best-effort で続行し、全件を stderr へ報告して停止する

## 仕様

- **巻き戻し自体の失敗は best-effort 続行**: 個々の逆操作が失敗してもジャーナルの残りの
  巻き戻しは続行する。全ての巻き戻し試行が終わった時点で、元の apply エラーと巻き戻せ
  なかった項目の一覧を stderr へ全報告して停止する（`reportConflicts` と同じ「全件列挙して
  から 1 本の集約エラー」の形）
- **報告は常時 stderr**（失敗経路のため沈黙対象外・既定 silent の対象にしない）

> **上は原文の写しで、規範は frontmatter が正**。undo ジャーナルそのものの規範は
> REQ-5e75aabc、成功時沈黙の出力規律そのものは REQ-8ef34101 の担当。conflict の全件報告
> （`reportConflicts`）は REQ-95e97d01 の担当で、本 item はその形を巻き戻し失敗の報告にも
> 用いることだけを規定する。

## 出典

`docs/spec.md`「配置動作仕様」→「途中失敗時の巻き戻し」節の箇条書き第 1・2 項。

決定の実体は ADR-0044「apply 途中失敗の完全巻き戻し — インメモリ undo ジャーナル」で、
失敗経路を沈黙の対象にしない出力規律は ADR-0031 が定めている。
