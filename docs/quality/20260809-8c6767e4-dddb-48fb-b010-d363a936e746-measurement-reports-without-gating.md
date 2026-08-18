---
id: "QA-8c6767e4-dddb-48fb-b010-d363a936e746"
type: quality
name: "傾向の計測は報告に留め、マージのゲートにしない"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
depends_on:
  - "QA-a5f7f088-a459-4bb2-9674-82b1a4a52053"
specification: |
  A check SHALL be told apart by what its failure means: a check that verifies behaviour
  fails when something is broken, whereas a check that measures a trend, coverage among
  them, moves without anything being broken. A check of the latter kind SHALL report its
  result where it can be read, and SHALL NOT gate merging on a threshold, so that a change
  is never blocked by a figure another change moved. Which of the remaining checks are
  required to pass before merging SHALL be settled by the norm that makes verification a
  precondition, not here. Detection SHALL be mechanical; the judgement of what a
  measurement calls for SHALL rest with people.
specification_ja: |
  チェックは、その失敗が何を意味するかによって判別されなければならない。振る舞いを検証する
  チェックは何かが壊れているときに失敗するのに対し、傾向を計測するチェック（カバレッジは
  これにあたる）は何も壊れていなくても数値が動く。後者の種類のチェックは、その結果を読める
  場所へ報告しなければならず、閾値によってマージを塞いではならない（ある変更が、別の変更が
  動かした数値によって塞がれることがないようにするため）。残るチェックのうちどれがマージ前に
  通ることを必須とされるかは、ここではなく、検証を必須条件とする規範によって定められなければ
  ならない。検出は機械的でなければならず、計測が何を要求しているかの判断は人が負わなければ
  ならない。
---
# QA-8c6767e4-dddb-48fb-b010-d363a936e746: 傾向の計測は報告に留め、マージのゲートにしない

## 仕様

QA-a5f7f088-a459-4bb2-9674-82b1a4a52053 が「マージ前の自動検証を必須にする」側を規範化するのに対し、本 item は
**必須化しない側の境界**を定める。両方が無いと、必須化の規範が「検証と名の付くものは
すべて塞ぐ」と読めてしまう。

**判別の基準は失敗が何を意味するか**になる。傾向の計測は、数値が下がっても何かが壊れたことを
意味せず、下がったこと自体の評価が状況に依存する。評価が状況に依存するものを閾値で塞ぐと、
塞いだ結果を人が毎回覆すことになり、ゲートが形骸化する。

閾値ゲートを置かないもう一つの理由は、**数値がその変更に固有でない**ことになる。カバレッジの
ような指標は他の変更がマージされるだけで動くため、閾値で塞ぐと変更の中身と無関係にマージ順が
結果を決める。塞ぐ代わりに読める場所へ出し、下がったことをどう扱うかは人が判断する。

**「計測でない = 必須」ではない。** 本 item が定めるのは計測を必須条件にしないことだけで、
何を必須にするかは QA-a5f7f088-a459-4bb2-9674-82b1a4a52053 の担当になる。実際、`sara check` は壊れていることを検出する
検証でありながら required status check にしない判断が別途置かれている（→ ADR-0050）。本 item
の規範をその判断の否定として読んではならない。

どの指標を計測するか・報告先・計測の実装は本 item の規範に含めない。`.github/workflows/` と
INF-d1230e1f-8ba8-49d8-8386-409bfbb7dd27 が持つ。

## 出典

ADR-0050（`sara check` を strict 化するが required status check 化はしない — 検出は機械・
判断は人）が置いた方針と、`.github/workflows/test.yml` の go-coverage / sara ジョブ
（閾値ゲートを持たず Job Summary へ報告する）・`docs/dev/definition-of-done.md`（閾値ゲートを
持たない計測は完成の定義に入れない）・`sara.toml` が実運用してきた規範を、Issue #272 で
quality item として立てたもの。
