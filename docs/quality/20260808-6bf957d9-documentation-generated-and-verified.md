---
id: "QA-6bf957d9-17d9-4660-92b7-ebd6eeb71a8c"
type: quality
name: "ドキュメントの正しさは人手の点検ではなく生成と機械検証で担保する"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  The documentation of nput SHALL NOT rely on manual inspection to stay consistent with
  what it describes. Reference documentation SHALL be generated from the source it
  documents at build time and SHALL NOT be committed in generated form, so that it cannot
  drift from that source. The traceability of the document graph — that every reference
  resolves to an item that exists, that identifiers are unique, and that the graph is
  acyclic — SHALL be verified mechanically, and both that verification and the buildability
  of the published documentation SHALL run in CI rather than on request.
specification_ja: |
  nput のドキュメントは、記述対象との整合性の維持を人手の点検に依存してはならない。
  リファレンスドキュメントはビルド時に記述対象のソースから生成されなければならず、生成物を
  コミットしてはならない（ソースとの乖離が起こり得ないようにするため）。ドキュメント
  グラフのトレーサビリティ（すべての参照が実在する item を指すこと・ID が一意であること・
  グラフが非循環であること）は機械的に検証されなければならず、その検証と公開ドキュメントの
  ビルド可能性はいずれも、要求されたときではなく CI で実行されなければならない。
---
# QA-6bf957d9: ドキュメントの正しさは人手の点検ではなく生成と機械検証で担保する

## 仕様

「ドキュメントの正しさを人手の点検に委ねない」という 1 つの方針を、ずれ方の 2 つの形に当てて
述べたもの。記述対象を変えたのに記述を直し忘れる形と、文書間の参照が指す先を失う形は、
どちらも「気づいた人が直す」運用では検出できない。

- **リファレンスは生成する**。ソースの doc-comment から毎回生成し、生成物をコミットしない。
  乖離が起こり得る状態そのものを作らない
- **グラフは機械検証する**。存在しない ID への参照・ID の重複・循環は、人が読んで気づく類の
  誤りではない。参照先の実在を検証しない突合（grep による ID 照合など）では穴が残る

検証の**強度**（未接続を失敗とするか警告に留めるか）は本 item の規範に含めない。移行の
進行度に応じて変わる設定であり、現在の値と strict 化の判断時期は INF-659b139d が持つ。

## 出典

ADR-0037（ドキュメントサイトは Astro Starlight + 同一リポジトリ + Cloudflare Pages で構築する）、
ADR-0048（ドキュメントは sara でグラフ構造化する）が置いた方針を、基盤（INF）から分離して
quality item として立てたもの。SSG の選定・ホスティング・i18n 構成・model.yaml の型定義・
`strict_mode` の設定は上記 INF が持つ。
