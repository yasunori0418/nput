---
id: "QA-58522afb-31d5-4a1f-a7df-0858efa9e44b"
type: quality
name: "最新 main のビルド成果物を再ビルドなしに消費できる状態を保つ"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  The build outputs of the latest `main` SHALL be published to a public binary cache for
  every supported platform, so that a consumer that takes nput as a flake input can obtain
  them without building from source. Publication SHALL be driven by the changes that
  determine those outputs, and SHALL NOT be tied to release tags, so that the cache tracks
  `main` rather than the release cadence.
specification_ja: |
  最新 main のビルド成果物は、サポート対象の全プラットフォームについて公開バイナリ
  キャッシュへ投入されなければならない（nput を flake input として取り込む消費側が、
  ソースからビルドせずに取得できるようにするため）。投入は成果物の内容を決める変更に
  よって駆動されなければならず、リリースタグに紐づけてはならない（キャッシュがリリースの
  頻度ではなく main に追従するようにするため）。
---
# QA-58522afb: 最新 main のビルド成果物を再ビルドなしに消費できる状態を保つ

## 方針

nput は flake input として消費されるツールであり、消費側の CI やローカル環境が nput を使う
たびにソースからビルドするのは実用上のコストになる。最新 main の成果物を公開キャッシュから
引ける状態を保つ。

投入の契機を**リリースタグではなく main の変更**に置く点が要点になる。タグに紐づけると
リリース間の main 変更がキャッシュされず、「flake input として最新 main を追う」という
消費のされ方と噛み合わない。リリース（QA-0949183b）とキャッシュは互いに独立に駆動する。

## 実現する基盤

- INF-af33c5a1（バイナリキャッシュ）— cachix への投入経路とトリガ

## 出典

ADR-0028（cachix push を main push の os マトリクスで実装する）が置いた方針を、基盤（INF）から
分離して quality item として立てたもの。投入先のキャッシュ名・マトリクス構成・トリガの
`paths` は INF-af33c5a1 が持つ。
