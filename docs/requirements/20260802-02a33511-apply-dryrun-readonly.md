---
id: "REQ-02a33511-0941-4813-b289-a05eb8e9aa57"
type: requirement
name: "apply --dryrun は読み取り専用で conflict 検出時に非ゼロ終了する"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  `nput apply <name> --dryrun` SHALL be read-only, performing no filesystem write, no
  `--set`, and no flock, and SHALL display place / replace / remove / conflict / no-op
  with zero side effects. When a conflict is present it SHALL exit non-zero, so that it
  can be used as a pre-check gate in CI. `--dryrun --backup` SHALL be combinable: what
  would be a conflict (exit 2) without `--backup` SHALL become a non-conflict plan of
  "back up and then place" (exit 0), except that when the destination of the rename aside
  already exists it SHALL remain a conflict even under `--backup`.
specification_ja: |
  `nput apply <name> --dryrun` は読み取り専用でなければならず、FS 書込・`--set`・flock の
  いずれも行わない。副作用ゼロで place / replace / remove / conflict / no-op を表示する。
  conflict があれば非ゼロ終了しなければならない（CI の事前 gate に使えるように）。
  `--dryrun --backup` は組み合わせ可能とし、`--backup` 無しなら conflict（exit 2）になる
  箇所が「backup + 配置予定」の非 conflict プラン（exit 0）へ変わる。ただし退避先が
  既存の場合は `--backup` 下でも conflict のままとする。
---
# REQ-02a33511: apply --dryrun は読み取り専用で conflict 検出時に非ゼロ終了する

## 仕様

```bash
nput apply <name> --dryrun     # dry-run。副作用ゼロで place/replace/remove/conflict/no-op を表示
```

`apply <name> --dryrun` は FS 書込・`--set`・flock いずれも行わない読み取り専用。
`conflict` があれば非ゼロ終了（CI の事前 gate に使える）。**`--dryrun --backup` は
組み合わせ可**で、`--backup` 無しなら conflict（exit 2）になる箇所が「backup + 配置予定」の
非 conflict プランへ変わる（exit 0）。退避先が既存の場合は `--backup` 下でも conflict の
まま。

conflict に対応する終了コード 2 そのものは REQ-2c5a10d8、`--dryrun` が root を解決しつつ
flock も pending gcroot も取らない点は REQ-7a71a049 の担当（通常 apply が out-link で
gcroot を張る段そのものは REQ-60c6b7ea）。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `apply <name> --dryrun` の箇条書き。

決定の実体は ADR-0006（`--dryrun` の conflict で非ゼロ終了・CI gate）と、
`--dryrun --backup` の組み合わせを定めた ADR-0045。
