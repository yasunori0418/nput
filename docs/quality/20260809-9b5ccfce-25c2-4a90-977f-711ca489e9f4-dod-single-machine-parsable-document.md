---
id: "QA-9b5ccfce-25c2-4a90-977f-711ca489e9f4"
type: quality
name: "完成の定義は単一の機械パース可能な文書で持ち、項目数を上限で縛る"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
depends_on:
  - "QA-a5f7f088-a459-4bb2-9674-82b1a4a52053"
specification: |
  The definition of done SHALL be held by a single document at a fixed path, and that
  document SHALL preserve the parsing contract that lets automation report each item's
  verdict, so that completion is judged mechanically wherever it can be. Its items SHALL be
  bounded by a fixed maximum, so that adding one forces an existing item to be merged or
  dropped rather than making completion recede. Where an item names a check that gates
  merging, it SHALL be kept in agreement with the checks actually required.
specification_ja: |
  完成の定義は固定されたパスにある単一の文書が持たなければならず、その文書は、自動化が
  項目ごとの判定を報告できるようにするパース契約を保たなければならない（機械的に判定できる
  範囲は機械的に判定されるようにするため）。その項目は固定された上限で縛られなければならず、
  項目を足すことが、完成を遠のかせるのではなく既存項目の統合か削除を迫るようにしなければ
  ならない。マージを塞ぐチェックを名指しする項目は、実際に必須とされているチェックと一致
  させ続けなければならない。
---
# QA-9b5ccfce-25c2-4a90-977f-711ca489e9f4: 完成の定義は単一の機械パース可能な文書で持ち、項目数を上限で縛る

## 仕様

完成の定義が複数箇所に散ると、どれを満たせば完成なのかが読み手ごとに変わる。単一の文書を
固定パスに置き、そこだけを見れば判定できる状態を保つ。

**パース契約を規範に含める**のが要点になる。人が読める形で書くだけなら文書はどう書いてもよいが、
自動化が項目ごとの判定を報告できる形を保つことで、機械判定できる項目に人の確認を使わずに
済む。この契約が壊れると、判定は静かに人手へ戻る。

**上限で縛る**のは、項目が増えるほど各項目のクリアが重くなり完成が遠のくため。上限は「足す
なら何かを畳むか外す」という取捨の強制であり、項目数そのものが目的ではない。

本 item は **DoD 文書の中身を item へ分割するものではない。** 分割はパース契約を壊す risk を
負ってグラフ接続の綺麗さしか得られない。持つべきは「単一の機械パース可能な文書で持つこと」
というメタ規範であり、文書自体は固定パスに残る。

項目の具体・判定手段の語彙・上限の値は本 item の規範に含めない。
`docs/dev/definition-of-done.md` が持つ。必須チェックとの一致を保つ規範は、必須化そのものを
定める QA-a5f7f088-a459-4bb2-9674-82b1a4a52053 に依存する。

## 出典

`docs/dev/definition-of-done.md`（固定パスへの単一配置・項目数の上限・機械パース契約・
required status check との一致・item へ分割しない判断）が実運用してきた規範を、Issue #272 で
quality item として立てたもの。同文書が「quality 型が実運用に乗ってから再判断する」として
保留していた論点（Issue #237）は、分割ではなくメタ規範化で決着する。
