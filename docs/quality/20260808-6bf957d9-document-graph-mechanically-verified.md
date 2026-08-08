---
id: "QA-6bf957d9-17d9-4660-92b7-ebd6eeb71a8c"
type: quality
name: "ドキュメントグラフのトレーサビリティは機械的に検証する"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  The traceability of the document graph — that every reference resolves to an item that
  exists, that identifiers are unique, and that the graph is acyclic — SHALL be verified
  mechanically rather than by manual inspection, and that verification SHALL run in CI
  rather than on request. A check that does not resolve the reference it inspects SHALL NOT
  be treated as satisfying this.
specification_ja: |
  ドキュメントグラフのトレーサビリティ（すべての参照が実在する item を指すこと・ID が一意で
  あること・グラフが非循環であること）は、人手の点検ではなく機械的に検証されなければならず、
  その検証は要求されたときではなく CI で実行されなければならない。参照先を解決しない検査を
  もってこれを満たしたとみなしてはならない。
---
# QA-6bf957d9: ドキュメントグラフのトレーサビリティは機械的に検証する

## 仕様

文書間の参照が指す先を失う、というずれ方は「気づいた人が直す」運用では検出できない。存在
しない ID への参照・ID の重複・循環は、人が読んで気づく類の誤りではないため、機械検証に置く。

**参照先の実在を確かめる**ことを条件に含めるのが要点になる。ID を文字列として突き合わせる
だけの検査（grep による照合など）は、存在しない ID を書いても検出できず、カバー率だけが
実態より高く出る穴を残す。

検証の**強度**（未接続を失敗とするか警告に留めるか）は本 item の規範に含めない。移行の
進行度に応じて変わる設定であり、現在の値と strict 化の判断時期は INF-659b139d が持つ。

**スコープ境界**: 本 item が対象とするのは**ドキュメント間**のグラフ整合のみ。公開契約と
コードの対応関係（cobra の `Use:` や `mkOption` からの母集団の機械抽出と突合）はドキュメント
グラフの視野外で、quality-observability 側の課題として残る（→ Issue #203 のスコープ境界）。

## 出典

ADR-0048（ドキュメントは sara でグラフ構造化する）が置いた方針を、基盤（INF）から分離して
quality item として立てたもの。model.yaml の型定義・`strict_mode` の設定・採番コマンドは
INF-659b139d が持つ。当初はリファレンスの生成（QA-0d42104c）と 1 件にまとめていたが、
根拠 ADR も支える基盤も別で片方だけ改訂され得るため、Issue #242 のレビューで 2 件へ分けた。
