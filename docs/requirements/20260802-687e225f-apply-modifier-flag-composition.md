---
id: "REQ-687e225f-5046-46db-88fb-f9e527a1e97a"
type: requirement
derives_from:
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
name: "apply 修飾フラグは --all と合成できる"
specification: |
  `apply --all --recopy` SHALL be composable: `--recopy` SHALL be applied to each config
  selected by `--all` (with a filter such as `--project-root` when given), since
  `--recopy` is a modifier of apply and orthogonal to `--all`. `--all --backup[=<suffix>]`
  SHALL likewise be composable, `--backup` also being a modifier of apply.
specification_ja: |
  `apply --all --recopy` は合成可能でなければならない。`--recopy` は apply の修飾であり
  `--all` と直交するため、`--all`（必要なら `--project-root` 等のフィルタ）が選んだ各
  config に `--recopy` を適用する。`--all --backup[=<suffix>]` も同様に合成可能とする
  （`--backup` も apply の修飾であるため）。
---
# REQ-687e225f: apply 修飾フラグは --all と合成できる

## 仕様

`apply --all --recopy` は**合成可**。`--all`（必要なら `--project-root` 等フィルタ）が
選んだ各 config に `--recopy` を適用する（`--recopy` は apply 修飾で `--all` と直交）。
`--all --backup[=<suffix>]` も同様に合成可（`--backup` も apply 修飾）。

各フラグ自体の規範は REQ-7cc32a2b（`--recopy`）・REQ-5dd5a4e9（`--backup`）の担当で、
本 item は `--all` との直交性のみを規定する。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `apply --all --recopy` の箇条書き。

決定の実体は ADR-0021「recopy 合成」と、`--backup` について同じ直交性を定めた ADR-0045。
