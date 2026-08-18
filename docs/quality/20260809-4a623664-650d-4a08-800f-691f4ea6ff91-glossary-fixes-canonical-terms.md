---
id: "QA-4a623664-650d-4a08-800f-691f4ea6ff91"
type: quality
name: "用語の正名は glossary が固定し、執筆はそれに従う"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  The canonical spelling of each domain term SHALL be fixed by a single authoritative
  glossary, and any other glossary that states the same terms SHALL derive from it rather
  than fix a spelling of its own. Writing across the repository SHALL use the canonical
  term and SHALL avoid the alternatives that glossary lists as ones to avoid. Where a term
  needed while writing is absent from the glossary, the gap SHALL be resolved by adding the
  term or by choosing an existing one, rather than by coining a spelling in place.
specification_ja: |
  各ドメイン用語の正名は、一次（authoritative）となる単一の glossary が固定しなければならず、
  同じ用語を述べる他の glossary は、独自に綴りを固定するのではなくそこから導出されなければ
  ならない。リポジトリ全体の執筆はその正名を用いなければならず、glossary が避けるべきものと
  して挙げる同義語を避けなければならない。執筆中に必要になった用語が glossary に無い場合、
  その欠落は、その場で綴りを作り出すのではなく、用語を追加するか既存の用語を選ぶことで
  解消されなければならない。
---
# QA-4a623664: 用語の正名は glossary が固定し、執筆はそれに従う

## 仕様

同じ概念が箇所ごとに違う語で呼ばれると、読み手は同じものを指しているのか判断できず、検索も
効かなくなる。正名を 1 箇所で固定し、執筆はそこに従う。

**避けるべき同義語まで列挙する**のが要点になる。正名だけを挙げても、書き手は「これも同じ意味
だろう」と別の語を選べる。実際に揺れた語を避けるべき側として明示することで、規範が判断を
残さない形になる。

glossary に無い語が要るという事態は、それ自体が信号になる。プロジェクトが使わない語彙を
持ち込もうとしているか、本当に語彙が欠けているかのどちらかで、どちらもその場で綴りを決めて
先へ進む扱いにはしない。

**本 item のスコープは執筆規約に限る。** コマンドの出力にどの文言を用いるかは、利用者が
直接触れる振る舞いであり個別の requirement の領分になる。本 item が縛るのは、そこで語を
選ぶときに参照する正名の所在と、リポジトリ全体の執筆でそれに従うことだけになる。

正名を述べる文書が複数あること自体は妨げない。日本語の対訳版が別ファイルとして並び、意味の
記述が別文書にあってもよく、規範が求めるのは**綴りを固定する一次が 1 つであること**になる。
どの文書が一次でどれが導出かの分担、各語の定義、言語ごとの版の関係は本 item の規範に含めず、
`docs/glossary.md` と `CONTEXT.md`・`docs/agents/domain.md` が持つ。

## 出典

`docs/glossary.md` 冒頭（README・コードコメント・コマンド出力の表記基準としての正名宣言と
避けるべき同義語）、`CONTEXT.md` 冒頭（正名と避けるべき同義語の固定）、
`docs/agents/domain.md`（CONTEXT は意味・glossary は綴りの正典という分担）が実運用してきた
規範を、Issue #272 で quality item として立てたもの。
