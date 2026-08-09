---
id: "QA-5ecd74ba-889a-4e06-b32b-e67f10a45051"
type: quality
name: "公開ドキュメントは英語を canonical とし、日本語版を対で保守する"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
depends_on:
  - "QA-4a623664-650d-4a08-800f-691f4ea6ff91"
specification: |
  Documentation addressed to users outside this repository SHALL take English as its
  canonical language, and a Japanese counterpart SHALL be maintained alongside it. Where a
  counterpart has not caught up, the reader SHALL be led to the canonical text rather than
  to a missing page. Design records addressed to contributors are outside this norm and
  SHALL NOT be required to carry an English counterpart.
specification_ja: |
  このリポジトリの外の利用者に向けたドキュメントは、英語を canonical な言語としなければ
  ならず、日本語版が対で保守されなければならない。対の側が追いついていない箇所では、
  読者は欠落したページではなく canonical な文へ導かれなければならない。貢献者に向けた
  設計記録は本規範の対象外であり、英語の対を持つことを求められてはならない。
---
# QA-5ecd74ba: 公開ドキュメントは英語を canonical とし、日本語版を対で保守する

## 仕様

公開されたリポジトリの読者は日本語話者に限らない。利用者に向けた面は英語を canonical に
置き、日本語版はその対として保守する。どちらが原本かを固定しないと、片方だけ更新されたときに
どちらが正しいかを読者が判断できなくなる。

**未追従時の振る舞いまで規範に含める**のが要点になる。対で保守すると決めても訳の遅れは必ず
生じるため、遅れがリンク切れや空ページに転化しない形（canonical へ導く）を条件にする。
これがないと「対で持つ」規範が翻訳を終えるまで公開を止める運用に化ける。

**対象は外向きの面に限る。** ADR や設計文書は貢献者に向けたもので、英語の対を求めると、
記録すべき判断を書く速度がそのまま落ちる。境界は「ソース・利用者向けの面は英語、設計根拠の
ドキュメントは日本語」で引く。

どの文書を公開面に載せるか（掲載範囲）・ロケールの構成・フォールバックの実装は本 item の規範に
含めない。INF-0865477b が持つ。用語の正名は QA-4a623664 が固定し、本 item はその上で言語の
canonical 性と対の保守だけを定める。

## 出典

`docs/publication-roadmap.md` §1（英語化する面と日本語のまま維持する面の境界）が置いた方針と、
INF-0865477b の i18n 節（root ロケール = 英語・日本語は別ロケール・未翻訳ページは英語へ
フォールバック）、README / README.ja・glossary / glossary.ja の対が実運用してきた規範を、
Issue #272 で quality item として立てたもの。
