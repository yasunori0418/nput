---
id: "REQ-1c1526b1-59e3-4264-bb7c-65a10a4aa461"
type: requirement
name: "flock の取得は既定 blocking とし --no-wait のときだけ try-lock でスキップする"
derives_from:
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
specification: |
  Without the flag, an explicit apply / rollback SHALL wait for the flock (LOCK_EX,
  blocking), and while waiting it SHALL display that it is waiting for another apply to
  finish. The `--no-wait` flag SHALL make the command skip instead of waiting when the
  flock is contended, for use from a shellHook: it SHALL acquire the lock as a try-lock
  (LOCK_NB), and when the lock is held it SHALL skip, print a one-line notice to stderr,
  and SHALL NOT block entry into the shell.
specification_ja: |
  フラグ無しの既定では、明示 apply / rollback は flock を待たなければならない
  （`LOCK_EX`・blocking）。待っている間は、他の apply の完了待ちである旨を表示しなければ
  ならない。`--no-wait` は flock 競合時に待たずスキップさせるフラグでなければならない
  （shellHook 用）。ロックは try-lock（`LOCK_NB`）で取得しなければならず、保持中なら
  スキップして stderr に 1 行通知しなければならず、シェル入室をブロックしてはならない。
---
# REQ-1c1526b1: flock の取得は既定 blocking とし --no-wait のときだけ try-lock でスキップする

## 仕様

```bash
--no-wait           # flock 競合時に待たず skip（shellHook 用。既定は明示 apply=blocking wait）
```

shellHook 経路（`--no-wait`）は try-lock（`LOCK_NB`）で保持中ならスキップし、stderr に
1 行通知する（例: `nput: another apply in progress, skipped (run \`nput apply\` manually)`・
シェル入室はブロックしない）。明示 apply / rollback は blocking（`LOCK_EX`・取得まで待ち
「他の apply 完了待ち」を表示）。

この try-lock / blocking の取得が実行フローのどの段（2a）で行われるかは REQ-60c6b7ea の
担当。skip 通知が既定沈黙の対象であることは REQ-8ef34101、skip が終了コード 0 になることは
REQ-2c5a10d8 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」のグローバルフラグ表 `--no-wait` と、
「実行フロー」2a の flock 取得（blocking / try-lock の別・待機表示・skip 通知）。

決定の実体は ADR-0013（flock の待ち方と `--no-wait` の位置づけ）と、shellHook の
try-lock skip を無言にせず通知すると定めた ADR-0022 §5。
