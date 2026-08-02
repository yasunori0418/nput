---
id: "REQ-2381d93a-732e-4437-910b-fac14d398aa0"
type: requirement
name: "エンベロープの niface 適合を Go テストと E2E の両方で検証する"
specification: |
  A Go test SHALL verify the emitted document against the niface
  `conformance.NewDefaultChecker()` (the embedded canonical schema including format
  assertions, plus the out-of-schema lint MUSTs) for every shape: success, with and
  without a subject, a pre-stage error, a subject error, a conflict, and the multiple
  subjects of `--all` (partial failure, zero results, dryrun conflict). The E2E SHALL
  verify the real output of the scenarios with the `niface-validate` CLI provided by the
  niface input of the dev flake (omitting `-schema` selects the embedded canonical
  schema), and SHALL cross-check the item ids against the `id-vectors.json` testdata of
  the same input.
specification_ja: |
  Go テストは、emit した文書を niface `conformance.NewDefaultChecker()`（embed 正本
  schema〔format assertion 込み〕+ schema 外 lint MUST）で全形状について検証しなければ
  ならない（成功 / subject あり・なし / 前段エラー / 主体エラー / conflict / `--all` の
  複数 subject〔部分失敗・0 件・dryrun conflict〕）。E2E は dev flake の niface input が
  提供する `niface-validate` CLI（`-schema` 省略 = embed 正本）でシナリオの実出力を検証し、
  item id は同 input の `id-vectors.json` testdata と突き合わせなければならない。
---
# REQ-2381d93a: エンベロープの niface 適合を Go テストと E2E の両方で検証する

## 仕様

**適合検証**: Go テストが niface `conformance.NewDefaultChecker()`（embed 正本 schema
〔format assertion 込み〕+ schema 外 lint MUST）で emit 文書を全形状（成功 / subject
あり・なし / 前段エラー / 主体エラー / conflict / `--all` の複数 subject〔部分失敗・
0 件・dryrun conflict〕）について検証する。E2E は dev flake の niface input が提供する
`niface-validate` CLI（`-schema` 省略 = embed 正本）でシナリオ 01–07 の実出力を検証し、
item id は同 input の `testdata/v1/id-vectors.json`（`NIFACE_ID_VECTORS`）と突き合わせる。

> **上は原文の写しで、規範は frontmatter が正**。原文の「シナリオ 01–07」という
> 具体の本数は E2E 実装の現況であり、規範文では「シナリオの実出力を検証する」に留めた。

item id の導出規則そのものは REQ-57137302 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」→「niface 準拠の `--json`
出力」のサブ項目「適合検証」。

決定の実体は ADR-0043。

> **`derives_from` を持たないのは意図的**（→ Issue #211）。エンベロープを niface 規約準拠に
> すること自体は REQ-a5053191 が担い、そちらは use_case へ紐づく。本 item はその適合を
> どう検証するかの要求であり、ユーザーの使われ方からは導かれない。REQ-6419e4b0・
> REQ-690f2730・REQ-901993e9 と同類で、いずれも use_case を持たない orphan として残す。
