---
id: "REQ-2b48620a-abaa-43df-a106-954bbba3de56"
type: requirement
derives_from:
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
name: "method 変更は symlink→copy のみ配置前除去で移行し、copy→symlink は移行しない"
specification: |
  When the method for the same target changes from symlink to copy, and the previous
  generation recorded a symlink whose on-disk destination matches that record, the engine
  SHALL remove it before placement (PreRemove) and then newly place the copy; this removal
  SHALL NOT be reported as a warning, being an intended migration, and SHALL instead fall
  under the ordinary output discipline of the placement report.
  When the on-disk link has drifted from the record, the engine SHALL NOT migrate it and
  SHALL fall back to the ordinary foreign classification. A method change from copy to
  symlink SHALL NOT be migrated automatically and SHALL remain a conflict, giving priority
  to protecting copied data the user may have edited.
specification_ja: |
  同一 target で method が symlink→copy に変わり、前世代が記録した symlink の on-disk が
  記録通りであるとき、engine は配置前に除去（PreRemove）してから copy を新規配置しなければ
  ならない。この除去は意図された移行であり warning にしてはならず、配置レポートの通常の出力
  規律に従わせる。on-disk が記録と不一致（readlink drift）のときは移行せず、通常の
  foreign 判定へフォールバックする。method 変更 copy→symlink は自動移行してはならず、
  conflict のままとする（ユーザーが編集し得る copy データの保護を優先するため）。
---
# REQ-2b48620a: method 変更は symlink→copy のみ配置前除去で移行し、copy→symlink は移行しない

## 仕様

```
target と同一 target で method が symlink→copy に変わるとき、前世代が記録した symlink で on-disk が
記録通りなら → 配置前に除去（PreRemove・silent）してから copy を新規配置
readlink drift（on-disk が記録と不一致）は移行せず通常の foreign 判定へフォールバックする
```

method 変更 copy→symlink は**自動移行しない**（ユーザー編集済み copy データの保護を優先し、
従来通り conflict のまま）。

> **上は原文の写しで、規範は frontmatter が正**。copy→symlink の conflict は `--backup`
> を付ければ退避 + 配置される（REQ-5dd5a4e9）。退避が配置手順のどの段に入るかは
> REQ-9b0046e0 の担当。

祖先 symlink の migration は REQ-c9ab91c1、実 dir target の migration は REQ-7cee95dd の
担当。除去を既定 silent とし `-v` で可視化するという出力規律そのもの（→ ADR-0031）は
REQ-8ef34101 / REQ-0a123b89 の担当で、本 item は「この除去を warning にせず配置レポート側で
扱う」ことだけを規範とする。

## 出典

`docs/spec.md`「配置動作仕様」→「symlink モード」の手順 0.6 と、同節の箇条書き
「method 変更 copy→symlink は自動移行しない」。

決定の実体は ADR-0047「配置前除去（PreRemove）の一般化」の D5。
