---
id: "QA-d028e302-8262-428c-9030-98d46b4b0cd3"
type: quality
name: "自動化が取り込む実行物は不変な識別子で固定し、権限は最小に絞り、不正入力では成果物を作る前に失敗する"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  Executable content that this repository's automation pulls in from outside SHALL be
  pinned by an immutable identifier rather than by a mutable name, so that what runs cannot
  change without a change to this repository. The privileges granted to each automated
  workflow SHALL be narrowed to what that workflow needs. Automation that takes an input
  SHALL validate it and SHALL fail before producing any artefact when the input is invalid,
  rather than producing an artefact derived from it.
specification_ja: |
  このリポジトリの自動化が外部から取り込む実行物は、可変な名前ではなく不変な識別子で
  固定されなければならない（このリポジトリへの変更なしに実行される内容が変わらないように
  するため）。自動化された各 workflow へ与える権限は、その workflow が必要とする範囲へ
  絞られなければならない。入力を取る自動化はその入力を検証しなければならず、入力が不正な
  場合は、それに由来する成果物を作るのではなく、成果物を作る前に失敗しなければならない。
---
# QA-d028e302: 自動化が取り込む実行物は不変な識別子で固定し、権限は最小に絞り、不正入力では成果物を作る前に失敗する

## 仕様

CI と自動化は、このリポジトリの変更なしに振る舞いが変わり得る面を 3 つ持つ。外部から取り込む
実行物・与えた権限・受け取る入力である。3 つとも、事故が起きたときに影響が及ぶ先が
「マージ前の検証」と「リリースされる成果物」なので、規範として固定する。

**可変な名前での参照を許さない**のが要点になる。名前で参照した実行物は、参照側を変えずに
中身を差し替えられる。固定の手段（コミット SHA 等）は基盤の担当だが、「不変であること」は
規範として動かさない。可読性のために可変な版名を注記として併記することは、参照が不変である
限り妨げない。

不正入力での失敗を **成果物を作る前**に置くのも同じ考え方になる。検証を後段に置くと、誤った
入力から作られた成果物（壊れたバージョン文字列を含むタグやリリースなど）が先に外へ出る。
誤ったものを出すより失敗させる方が、事後の調査も回復も容易になる。

**本 item が縛るのはこのリポジトリの CI 実行環境であり、nput が提供する CLI ではない。**
製品側にも依存の固定（REQ-637599dc）と不正入力での早期失敗（REQ-774cef80）の規範があるが、
それらは利用者が触れる振る舞いを縛るもので、系統が異なる。固定の方式・権限の具体値・検証の
実装形は本 item の規範に含めず、`.github/workflows/` と対応する infrastructure item が持つ。

## 出典

`.github/workflows/` の各 workflow が実運用してきた規範（外部 action の SHA ピンと版名注記の
併記・workflow ごとに絞った `permissions`・`bump-version.yml` と `release.yml` のバージョン
入力検証）を、ADR-0042 のリリース自動化の決定とあわせて、Issue #272 で quality item として
立てたもの。
