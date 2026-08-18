---
id: "REQ-2353259f-5878-452a-8e11-3445de69abc2"
type: requirement
name: "--json 指定時は行指向 stdout を出さずエンベロープが stdout を専有する"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
specification: |
  When `--json` is given, the default line-oriented stdout (the enumeration of
  `gitignore`, the plan of `apply --dryrun` / `reset --dryrun`, and the listing of
  `list-generations`) SHALL NOT be emitted, the envelope occupying stdout exclusively.
  The default contract without `--json` SHALL be unchanged.
specification_ja: |
  `--json` 指定時、既定の行指向 stdout（`gitignore` の列挙・`apply --dryrun` /
  `reset --dryrun` のプラン・`list-generations` の一覧）を出してはならない
  （エンベロープが stdout を専有する）。`--json` なしの既定契約は不変でなければならない。
---
# REQ-2353259f: --json 指定時は行指向 stdout を出さずエンベロープが stdout を専有する

## 仕様

**`--json` 時の stdout 専有**: 既定の行指向 stdout（`gitignore` の列挙・`apply --dryrun` /
`reset --dryrun` のプラン・`list-generations` の一覧）は `--json` 指定時には**出さない**
（エンベロープが stdout を専有する）。`--json` なしの既定契約は不変。

`--json` なしのストリーム規律は REQ-fea038de の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」→「niface 準拠の `--json`
出力」のサブ項目「`--json` 時の stdout 専有」。

決定の実体は ADR-0043。
