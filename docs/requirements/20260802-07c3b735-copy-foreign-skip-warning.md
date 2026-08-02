---
id: "REQ-07c3b735-3744-4778-a640-8c6fb66f4aa7"
type: requirement
derives_from:
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
name: "copy が foreign 実ファイルを skip したときは warning で可視化する"
specification: |
  When the target of a copy entry exists and the previous-generation manifest does not
  hold that copy entry — that is, the file was not placed by nput and is foreign — the
  engine SHALL skip the copy under place-once but SHALL emit a warning stating that an
  existing file was present at the target and the copy was skipped, symmetrically with the
  foreign warning for symlinks, so as to prevent the user from mistaking the content for
  something nput put there. The engine SHALL NOT overwrite it and SHALL NOT stop the apply
  as a whole. Whether nput placed it SHALL be decided by the presence of the entry in the
  previous-generation manifest and SHALL NOT be decided from the content.
specification_ja: |
  copy entry の target が存在し、かつ前世代 manifest にこの copy entry が無いとき（= nput が
  置いていない foreign なファイル）、engine は place-once により copy を skip しつつ、
  「target に既存ファイルがあり copy をスキップした」という warning を出さなければならない
  （symlink の foreign 警告と対称化し、「nput が中身を置いた」という誤認を防ぐため）。上書きは
  してはならず、apply 全体を止めてもならない。「自分が置いたか」は前世代 manifest の entry
  有無で判別し、内容では判別しない。
---
# REQ-07c3b735: copy が foreign 実ファイルを skip したときは warning で可視化する

## 仕様

**foreign 実ファイルの skip は warning で可視化する**: target が存在し**かつ前世代 manifest に
この copy entry が無い**（= nput が置いていない foreign ファイル）のとき、place-once により
skip するが **warning を出す**（「target に既存ファイルがあり copy をスキップした」）。
symlink の foreign 警告と対称化し、「nput が中身を置いた」と誤認する masking を防ぐ。
上書きはせず apply 全体も止めない。「自分が置いたか」は前世代 manifest の entry 有無で判別し、
内容は判別しない。

> **上は原文の写しで、規範は frontmatter が正**。place-once そのものは REQ-d2277c7a、
> symlink 側の foreign 警告は REQ-622787dc の担当。`apply --backup` 有効時にこの skip +
> warning が退避 + copy 新規配置へ変わることは REQ-5dd5a4e9 / REQ-9b0046e0 の担当。

## 出典

`docs/spec.md`「配置動作仕様」→「copy モード（place-once・ユーザー管理）」節の箇条書き第 1 項。

決定の実体は ADR-0022「実装前残セマンティクス第4巡」の copy foreign 衝突で、対称化の元に
なる symlink の foreign 警告は ADR-0015 が定めている。
