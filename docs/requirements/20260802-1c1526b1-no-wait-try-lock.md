---
id: "REQ-1c1526b1-59e3-4264-bb7c-65a10a4aa461"
type: requirement
name: "--no-wait は flock 競合時に待たずスキップする"
specification: |
  The `--no-wait` flag SHALL make the command skip instead of waiting when the flock is
  contended, for use from a shellHook. The default SHALL be that an explicit apply waits
  for the lock (blocking wait).
specification_ja: |
  `--no-wait` は flock 競合時に待たずスキップさせるフラグでなければならない
  （shellHook 用）。既定は明示 apply が待つ（blocking wait）ものとする。
---
# REQ-1c1526b1: --no-wait は flock 競合時に待たずスキップする

## 仕様

```bash
--no-wait           # flock 競合時に待たず skip（shellHook 用。既定は明示 apply=blocking wait）
```

try-lock の具体的な挙動（`LOCK_NB` で保持中ならスキップし stderr に 1 行通知する）は
実行フローの規範であり REQ-60c6b7ea の担当。skip 通知が既定沈黙の対象であることは
REQ-8ef34101、skip が終了コード 0 になることは REQ-2c5a10d8 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」のグローバルフラグ表 `--no-wait`。

決定の実体は ADR-0013（flock の待ち方と `--no-wait` の位置づけ）。
