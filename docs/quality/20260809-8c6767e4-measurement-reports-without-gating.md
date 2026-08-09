---
id: "QA-8c6767e4-dddb-48fb-b010-d363a936e746"
type: quality
name: "傾向の計測は報告に留め、マージのゲートにしない"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
depends_on:
  - "QA-a5f7f088-a459-4bb2-9674-82b1a4a52053"
specification: |
  A check that verifies behaviour — one whose failure means something is broken — SHALL be
  a precondition of merging. A check that measures a trend, coverage among them, SHALL
  report its result where it can be read and SHALL NOT gate merging on a threshold, so that
  a change is never blocked by a figure another change moved. Detection SHALL be
  mechanical; the judgement of what a measurement calls for SHALL rest with people.
specification_ja: |
  振る舞いを検証するチェック（失敗が「何かが壊れている」ことを意味するもの）は、マージの
  必須条件でなければならない。傾向を計測するチェック（カバレッジはこれにあたる）は、その
  結果を読める場所へ報告しなければならず、閾値によってマージを塞いではならない（ある変更が、
  別の変更が動かした数値によって塞がれることがないようにするため）。検出は機械的でなければ
  ならず、計測が何を要求しているかの判断は人が負わなければならない。
---
# QA-8c6767e4: 傾向の計測は報告に留め、マージのゲートにしない

## 仕様

QA-a5f7f088 が「マージ前の自動検証を必須にする」側を規範化するのに対し、本 item は
**必須化しない側の境界**を定める。両方が無いと、必須化の規範が「検証と名の付くものは
すべて塞ぐ」と読めてしまう。

**判別の基準は失敗が何を意味するか**になる。振る舞いの検証は、失敗すれば製品が壊れている
ことを意味するので塞ぐ意味がある。傾向の計測は、数値が下がっても壊れてはおらず、下がった
こと自体の評価が状況に依存する。前者は必須、後者は報告に留める。

閾値ゲートを置かないのは、**数値がその変更に固有でない**ことが理由になる。カバレッジのような
指標は他の変更がマージされるだけで動くため、閾値で塞ぐと変更の中身と無関係にマージ順が
結果を決める。塞ぐ代わりに読める場所へ出し、下がったことをどう扱うかは人が判断する。

どの指標を計測するか・報告先・計測の実装は本 item の規範に含めない。`.github/workflows/` と
対応する infrastructure item が持つ。

## 出典

ADR-0050（`sara check` を strict 化するが required status check 化はしない — 検出は機械・
判断は人）が置いた方針と、`.github/workflows/test.yml` の go-coverage / sara ジョブ
（閾値ゲートを持たず Job Summary へ報告する）・`docs/dev/definition-of-done.md`（閾値ゲートを
持たない計測は完成の定義に入れない）・`sara.toml` が実運用してきた規範を、Issue #272 で
quality item として立てたもの。
