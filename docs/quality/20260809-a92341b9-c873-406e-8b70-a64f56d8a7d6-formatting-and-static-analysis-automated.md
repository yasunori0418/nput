---
id: "QA-a92341b9-c873-406e-8b70-a64f56d8a7d6"
type: quality
name: "コード整形と静的解析を自動検証に載せ、同じ判定を手元でも得られるようにする"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
depends_on:
  - "QA-a5f7f088-a459-4bb2-9674-82b1a4a52053"
specification: |
  Code formatting and static analysis SHALL be among the checks that the pre-merge
  verification runs, so that neither rests on a contributor remembering to run it locally.
  A contributor SHALL be able to run the same check the verification runs, and obtain the
  same verdict from it, without reproducing the pipeline that invokes it.
specification_ja: |
  コード整形と静的解析は、マージ前の自動検証が実行するチェックに含まれなければならない
  （どちらも、貢献者がローカルでの実行を覚えていることに依存しないようにするため）。貢献者は、
  その検証が実行するのと同じチェックを、それを起動するパイプラインを再現することなく実行し、
  同じ判定を得られなければならない。
---
# QA-a92341b9: コード整形と静的解析を自動検証に載せ、同じ判定を手元でも得られるようにする

## 仕様

整形の揺れと静的解析が拾う類の誤りは、レビューで指摘できるが指摘に人の注意を払わせるだけの
価値が無い。マージ前の自動検証の側へ回して、人のレビューを意味判断に集中させる。

QA-a5f7f088 は「マージ前の自動検証を必須にすること」を規範化し、**検証項目の中身は委譲して
いる**。本 item はその委譲を受けて、整形と静的解析を項目に含めることを定める。

**手元で同じ判定を得られること**が要点になる。パイプラインの手順としてしか書かれていない検査は、
手元で走らせる手段がパイプラインの再現になり、CI と手元で結果がずれる余地が残る。指摘された
貢献者が同じ検査を手元で回せなければ、修正は CI の往復に頼ることになる。この性質を満たす
実装形（宣言的なチェック定義など）は基盤の担当で、規範として固定するのは性質の側になる。

対象言語ごとの整形器・解析器の選定、チェックの実装形、どのジョブで走らせるかは本 item の
規範に含めない。ADR-0025 と `flake.nix` のチェック定義、INF-d1230e1f が持つ。

## 出典

ADR-0025 §6（整形は treefmt へ gofmt を追加し、静的解析は go vet + golangci-lint の check
derivation として `nix flake check` に載せる）が置いた方針と、`flake.nix` の checks 定義・
`.github/workflows/test.yml` の flake check ジョブが実運用してきた規範を、Issue #272 で
quality item として立てたもの。
