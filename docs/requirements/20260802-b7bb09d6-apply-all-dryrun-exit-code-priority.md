---
id: "REQ-b7bb09d6-74c4-44d6-905f-cb5e8383ea32"
type: requirement
name: "apply --all --dryrun の終了コードは error を conflict より優先する"
derives_from:
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
specification: |
  The exit code of `apply --all --dryrun` SHALL be 1 when any config errored, 2 when there
  was no error but a conflict, and 0 when there was neither; that is, error (1) SHALL take
  priority over conflict (2). A plain maximum (which would give conflict priority since
  2 > 1) SHALL NOT be used, because it would hide the more serious eval / engine errors in
  CI. A non-dryrun `--all` has no notion of conflict and SHALL yield only 0 or 1.
specification_ja: |
  `apply --all --dryrun` の終了コードは、いずれかが error なら 1、error が無く conflict が
  あれば 2、どちらも無ければ 0 でなければならない（error(1) を conflict(2) より優先する）。
  単純な最大値（2 > 1 で conflict 優先）を採ってはならない。より深刻な eval / engine
  エラーを CI で隠すため。非 dryrun の `--all` は conflict 概念が無く 0 / 1 のみとする。
---
# REQ-b7bb09d6: apply --all --dryrun の終了コードは error を conflict より優先する

## 仕様

**`apply --all --dryrun` の終了コードは「いずれかが error なら 1、error が無く conflict が
あれば 2、どちらも無ければ 0」**（error(1) 最優先 → conflict(2) → 0）。単純な最大値
（2 > 1 で conflict 優先）は採らない（より深刻な eval / engine エラーを CI で隠すため）。
非 dryrun の `--all` は conflict 概念が無く 0 / 1 のみ。

終了コード表そのものは REQ-2c5a10d8 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」の最終箇条書き。

決定の実体は ADR-0024「終了コード優先」。
