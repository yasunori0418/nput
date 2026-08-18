---
id: "REQ-6a950d6d-c191-4235-a1b4-73ffc7c2bb38"
type: requirement
name: "reset --dryrun は副作用ゼロで削除対象を表示して終了する"
derives_from:
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  `nput reset <name> --dryrun` SHALL display the deletion targets (symlinks / copy
  targets) with zero side effects and then exit, performing neither filesystem deletion,
  nor the confirmation prompt, nor the flock, symmetrically with `apply --dryrun`. Its
  exit code SHALL be 0 regardless of whether there are targets to delete.
specification_ja: |
  `nput reset <name> --dryrun` は副作用ゼロで削除対象（symlink / copy target）を表示して
  終了しなければならない（FS 削除・confirm・flock のいずれも行わない・`apply --dryrun` と
  対称）。終了コードは削除対象の有無に依らず 0 でなければならない。
---
# REQ-6a950d6d-c191-4235-a1b4-73ffc7c2bb38: reset --dryrun は副作用ゼロで削除対象を表示して終了する

## 仕様

```bash
nput reset <name> --dryrun     # 副作用ゼロで削除対象（symlink / copy target）を表示して exit（confirm/flock なし）
```

`reset <name> --dryrun` は**副作用ゼロ**で削除対象（symlink / copy target）を表示して
exit する（FS 削除・confirm・flock いずれも行わない・`apply --dryrun` と対称）。
終了コードは削除対象の有無に依らず 0。

`apply --dryrun` 側の規範は REQ-02a33511-0941-4813-b289-a05eb8e9aa57 の担当（conflict 検出時に非ゼロ終了する点が
異なる）。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `reset <name> --dryrun` の箇条書き。

決定の実体は ADR-0021「reset の `--dryrun` 対応」。
