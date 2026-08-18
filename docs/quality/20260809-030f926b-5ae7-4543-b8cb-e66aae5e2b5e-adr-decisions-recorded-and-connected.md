---
id: "QA-030f926b-5ae7-4543-b8cb-e66aae5e2b5e"
type: quality
name: "設計判断は ADR に記録し、改訂の書き戻しと item への接続を同じ変更の中で完了させる"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
depends_on:
  - "QA-6bf957d9-17d9-4660-92b7-ebd6eeb71a8c"
specification: |
  A change that carries a design decision SHALL record that decision as an ADR. Where a new
  ADR revises the decision of an existing one, the back-reference note on the revised ADR
  SHALL be written within the same change; deferring it SHALL NOT be treated as acceptable.
  Every ADR SHALL connect to at least one item it justifies, and where no such item exists
  yet, it SHALL be created within the same change rather than the connection being left
  empty, so that an empty connection remains readable as an omission. Because the mutual
  consistency of a revision and its back-reference note is not mechanically verified, it
  SHALL be upheld by review.
specification_ja: |
  設計判断を伴う変更は、その判断を ADR として記録しなければならない。新しい ADR が既存 ADR の
  決定を改訂する場合、改訂される側の ADR への書き戻し注記を同じ変更の中で書かなければならず、
  これを先送りしてよいものとして扱ってはならない。すべての ADR は、自身が裏づける item へ
  1 本以上接続しなければならず、接続先の item がまだ無い場合は、接続を空のまま残すのではなく
  同じ変更の中で item を起こさなければならない（空の接続が埋め忘れとして読み取れる状態を
  保つため）。改訂と書き戻し注記の相互整合は機械的に検証されないため、レビューで守られなければ
  ならない。
---
# QA-030f926b-5ae7-4543-b8cb-e66aae5e2b5e: 設計判断は ADR に記録し、改訂の書き戻しと item への接続を同じ変更の中で完了させる

## 仕様

決定を記録しない、あるいは記録したが後続の改訂が旧文書へ反映されない状態は、旧 ADR だけを
読んだ人に「上書きされた内容」を最新仕様と誤読させる。実際にこの漏れが積み重なった経緯が
あり、本 item はその再発防止を規範として固定する。

**「同じ変更の中で」**を条件に含めるのが要点になる。書き戻しも item への接続も、遅らせられる
形にした瞬間に崩壊する。接続先の item が無いことは接続を空にする理由にならず、item を先に
起こすか、決定が定まっていないものとして ADR の粒度を解き直すかのどちらかになる。

接続を空のまま残さない規範は、**空を埋め忘れとして機械判定できる状態**を保つためにある。
接続の非空性は QA-6bf957d9-17d9-4660-92b7-ebd6eeb71a8c の機械検証が拾うが、その検証は frontmatter しか見ない。旧 ADR
本文の注記が書かれているかはその帰結として視野の外にあり、そこは人手のレビューで守る
（確認の手順は `docs/adr/README.md` のセルフチェック節が持つ）。

改訂の表現形式（supersede を使わず blockquote 注記で誘導する・注記の書式・接続先の型の
一覧）は本 item の規範に含めない。運用手順は `docs/adr/README.md` が、接続可能な型は
`docs/model.yaml` が持つ。

## 出典

`docs/adr/README.md`（supersede 不使用の運用規約・`justifies` 必須の規約・注記漏れは目視で
確認するというセルフチェック手順）、`docs/dev/definition-of-done.md` の DOD-03、
`.github/PULL_REQUEST_TEMPLATE.md` の「docs / ADR 影響」欄が実運用してきた規範を、Issue #272 で
quality item として立てたもの。
