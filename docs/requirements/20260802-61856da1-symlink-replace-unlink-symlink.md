---
id: "REQ-61856da1-8883-401e-ad57-9f326b96d400"
type: requirement
name: "既存 symlink の張替えは unlink + symlink の 2 操作で行い冪等な再実行で収束させる"
specification: |
  Replacing an existing symlink SHALL be carried out as two operations, an unlink followed
  by a symlink, and an atomic swap based on rename SHALL NOT be adopted. A crash between
  the two operations may make the target vanish for a while, and this SHALL be accepted on
  the grounds that an idempotent re-run converges, consistently with the guarantee that
  the generations that get stacked are always fully applied. When the run itself detects
  an error at a later stage — that is, in any case other than a crash such as SIGKILL or
  power loss — the destination captured before the unlink SHALL be recorded in the undo
  journal and restored by re-creating the symlink.
specification_ja: |
  既存 symlink の張替えは unlink + symlink の 2 操作で行わなければならず、rename ベースの
  atomic swap は採らない。2 操作の間でクラッシュすると target が一時消失しうるが、冪等な
  再実行で収束することを根拠にこれを受け入れる（「積まれる世代は常に完全適用済み」と整合）。
  クラッシュ（SIGKILL・電源断）以外で、この run 自身が後続の段でエラーを検知した場合は、
  unlink 前に捕捉した張替え先を undo ジャーナルへ記録し、symlink の再作成で復元しなければ
  ならない。
---
# REQ-61856da1: 既存 symlink の張替えは unlink + symlink の 2 操作で行い冪等な再実行で収束させる

## 仕様

既存 symlink の張替えは **unlink + symlink の 2 操作**で行う（rename ベースの atomic swap は
採らない）。間でクラッシュすると target が一時消失しうるが、**冪等な再実行で収束**する
（「積まれる世代は常に完全適用済み」と整合）。クラッシュ（SIGKILL・電源断）以外でこの run
自身が後続の段でエラーを検知した場合は、unlink 前の張替え先を undo ジャーナルが記録しており
re-symlink で復元する。

> **上は原文の写しで、規範は frontmatter が正**。undo ジャーナルそのものの規範は
> REQ-5e75aabc の担当で、本 item は張替えを 2 操作で行うことと、その復元素材（unlink 前の
> 張替え先）を捕捉することだけを規定する。「積まれる世代は常に完全適用済み」（コミット
> 最後）は REQ-1be4d678 の担当。

## 出典

`docs/spec.md`「配置動作仕様」→「symlink モード」の箇条書き第 1 項。

決定の意思決定は ADR-0017「実装前レビュー第 3 巡で surfaced した残セマンティクス」の
「張替えの atomic 性」で、冪等再実行による収束の前提は ADR-0006 が、undo ジャーナルによる
復元は ADR-0044 が定めている。
