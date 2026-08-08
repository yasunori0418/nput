---
id: "QA-a5f7f088-a459-4bb2-9674-82b1a4a52053"
type: quality
name: "サポート対象の全プラットフォームでの自動検証をマージの必須条件にする"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  Every change to `main` SHALL be verified automatically on all supported platforms before
  it is merged, and the verification SHALL cover the lib evaluation tests, the engine's Go
  checks and the end-to-end path on a non-NixOS system. Passing that verification SHALL be
  a technical precondition of merging rather than a rule contributors are asked to follow,
  and no participant SHALL be able to bypass it. A change that does not affect the verified
  sources MAY skip the verification work itself, but SHALL NOT thereby become unmergeable.
specification_ja: |
  main へのすべての変更は、マージ前にサポート対象の全プラットフォーム上で自動的に検証され
  なければならず、その検証は lib の評価テスト・engine の Go チェック・非 NixOS 環境での
  一気通貫の経路を対象としなければならない。検証の成功は、参加者に遵守を求める運用ルール
  ではなくマージの技術的な必須条件でなければならず、いかなる参加者もこれを迂回できては
  ならない。検証対象のソースに影響しない変更は検証の実行自体を省いてもよいが、それに
  よってマージ不能になってはならない。
---
# QA-a5f7f088: サポート対象の全プラットフォームでの自動検証をマージの必須条件にする

## 方針

nput は複数の OS / system で動くことを主張するツールであり、その主張は「手元の 1 環境で
通った」では裏づけられない。サポート対象の全プラットフォームでの検証を、マージ前に自動で
通すことを必須にする。

必須化を**技術的な条件**として課す点が要点になる。「main へ直接コミットしない」「PR 経由で
マージする」といった運用ルールは、テストが失敗していてもマージできる状態を残す。方針として
求めるのはその状態を塞ぐことであり、bypass の余地を残さないことを含む。

検証対象のソースに影響しない変更（ドキュメントのみの変更など）まで全環境の検証を実走させる
必要はない。ただし「実走を省く」ことが「マージできなくなる」に転化してはならない — 省略の
手段はこの制約と両立するものに限られる。

## 実現する基盤

- INF-d1230e1f（CI パイプライン）— 検証そのものを実行する
- INF-8b97573f（マージゲート）— 検証の成功をマージの技術的条件に変換する

## 出典

ADR-0012（CI・テスト実行基盤の確定）、ADR-0027（flake check の CI マトリクスとトリガ）、
ADR-0030（テスト成功を main マージの必須条件にする）が置いた方針を、基盤（INF）から分離して
quality item として立てたもの。具体的なジョブ構成・check 名・ruleset の設定は上記 INF が持つ。
