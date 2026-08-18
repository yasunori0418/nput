---
id: "QA-67715bb3-1162-4ccf-8441-2f44257a57da"
type: quality
name: "規範は 1 ファイル 1 item のグラフが持ち、概要文書は索引に縮退させる"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  Normative content SHALL be held by the item graph alone, one item per file, and the
  overview documents SHALL be reduced to indexes that link to those items rather than
  restating them. Each item SHALL be placed under the type its norm binds, following the
  project's discrimination rule, and its identifier SHALL be drawn from the project's
  numbering command rather than assigned by hand, so that parallel lanes cannot collide on
  a number. An item whose type carries a specification SHALL state that norm in English and
  SHALL carry the corresponding Japanese norm alongside it; neither field SHALL be treated
  as a gloss of the other.
specification_ja: |
  規範的な内容は 1 ファイル 1 item の item グラフだけが持たなければならず、概要文書は
  それらの item へリンクする索引へ縮退させなければならない（内容を書き写してはならない）。
  各 item は、その規範が何を縛るかに応じてプロジェクトの型判別規約に従った型の下へ配置され
  なければならず、その識別子は手で振るのではなくプロジェクトの採番コマンドから採らなければ
  ならない（並列レーンで採番が衝突しないようにするため）。specification を持つ型の item は、
  その規範を英語で述べなければならず、対応する日本語の規範文を併記しなければならない。
  いずれの欄も他方の訳注として扱ってはならない。
---
# QA-67715bb3-1162-4ccf-8441-2f44257a57da: 規範は 1 ファイル 1 item のグラフが持ち、概要文書は索引に縮退させる

## 仕様

規範を概要文書と item の両方に書くと二重管理になり、改訂で片方だけが古くなる。規範の所在を
グラフ側に一本化し、概要文書は通読の入口としての索引に徹する。この分担は文書の分量ではなく
**どちらが正典か**の宣言であり、概要文書へ仕様を書き足すことは規範の複製にあたる。

**配置と採番を規約で固定する**のが要点になる。型を取り違えた item は形が正しい限り機械検証を
素通りするため（QA-6bf957d9-17d9-4660-92b7-ebd6eeb71a8c の検証は参照の解決・一意性・非循環しか見ない）、正しい型へ置く
ことはレビューで守られる判断になる。採番を手で行わないのは、並列に走るレーンが同じ番号を
採る事故を構造的に避けるため。

二言語で書く規範は、英語と日本語のどちらも正典として扱うことを含む。片方を訳注に降格させると、
実際に読まれる側の言語で規範の強度が落ちる。強度の対応（SHALL と「〜しなければならない」など）
まで含めた執筆規約は `docs/agents/sara-graph.md` が持ち、これも機械検証されないためレビューで
守る。

型の判別基準の具体・採番コマンドの実装・強度の対応表は本 item の規範に含めない。判別と執筆の
規約は `docs/agents/sara-graph.md` が、採番の仕組みは INF-659b139d-0cf8-4c65-b30d-93c5ee2dfc71 が持つ。

## 出典

`docs/spec.md` 冒頭の書き方の規約、`docs/agents/sara-graph.md` の型判別規約と
`specification` / `specification_ja` の執筆規約、`CLAUDE.md` の ID 規約節、
`docs/dev/definition-of-done.md` の DOD-04 が実運用してきた規範を、Issue #272 で quality item
として立てたもの。
