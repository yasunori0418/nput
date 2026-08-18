---
id: "QA-0d42104c-a3c2-4bd1-9cad-a42d9e5a56a1"
type: quality
name: "リファレンスドキュメントは記述対象のソースから生成し、生成物を持たない"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  Reference documentation SHALL be generated from the source it documents at build time and
  SHALL NOT be committed in generated form, so that it cannot drift from that source. Its
  buildability SHALL be verified in CI rather than on request, so that a break in the
  extraction is caught before it is merged.
specification_ja: |
  リファレンスドキュメントはビルド時に記述対象のソースから生成されなければならず、生成物を
  コミットしてはならない（ソースとの乖離が起こり得ないようにするため）。そのビルド可能性は、
  要求されたときではなく CI で検証されなければならない（抽出の破損をマージ前に捕まえる
  ため）。
---
# QA-0d42104c-a3c2-4bd1-9cad-a42d9e5a56a1: リファレンスドキュメントは記述対象のソースから生成し、生成物を持たない

## 仕様

記述対象を変えたのに記述を直し忘れる、というずれ方は「気づいた人が直す」運用では検出できない。
ソースの doc-comment から毎回生成し、生成物をコミットしないことで、乖離が起こり得る状態
そのものを作らない。

生成できなくなる形（doc-comment の構文エラーなど）は残るため、ビルド可能性の検証を CI に
置いてマージ前に捕まえる。

抽出元・抽出ツール・掲載範囲・サイトの構成は本 item の規範に含めない。INF-0865477b-21b2-4feb-9c06-1a882a4e595a が持つ。

## 出典

ADR-0037（ドキュメントサイトは Astro Starlight + 同一リポジトリ + Cloudflare Pages で構築する）が
置いた方針を、基盤（INF）から分離して quality item として立てたもの。当初は
QA-6bf957d9-17d9-4660-92b7-ebd6eeb71a8c（ドキュメントグラフの機械検証）と 1 件にまとめていたが、根拠 ADR も支える基盤も
別で片方だけ改訂され得るため、Issue #242 のレビューで 2 件へ分けた。
