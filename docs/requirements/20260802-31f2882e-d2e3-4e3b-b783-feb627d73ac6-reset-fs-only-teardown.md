---
id: "REQ-31f2882e-d2e3-4e3b-b783-feb627d73ac6"
type: requirement
name: "reset は profile を触らない FS-only teardown で配置物を無い状態へ戻す"
derives_from:
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  `nput reset <name> [target...]` SHALL be a teardown that returns placed objects to the
  state of not being present; omitting `target` SHALL select every entry, and specifying
  it SHALL select only those entries. Symlinks SHALL be removed under the same
  conservative invariants as stale removal (only those managed by nput and matching the
  record; foreign ones SHALL be left with a warning), and copy targets SHALL also be
  deleted, `reset` being the only explicit means of deleting a copy. After removing either
  a symlink or a copy, empty-parent-directory pruning SHALL be applied. Because of the
  risk of data loss, the command SHALL request consent, either through a confirmation
  prompt or via `-y` / `--yes` (the latter being for scripts and CI), and SHALL report the
  deleted targets. `reset` SHALL be an FS-only
  teardown that leaves the profile and generations untouched; as long as the config still
  holds the entry, the next apply SHALL place it again. It SHALL be usable in both home
  and project mode.
specification_ja: |
  `nput reset <name> [target...]` は配置物を無い状態へ戻す teardown でなければならない。
  target 省略で全 entry、指定でその entry のみを対象としなければならない。symlink は stale
  除去と同じ保守的不変条件（nput 管理・記録通りのみ・foreign は warning で残す）で除去し、
  copy target も削除しなければならない（copy を消す唯一の明示手段）。symlink・copy いずれの
  除去後も空親ディレクトリ剪定を適用しなければならない。データ損失リスクのため確認プロンプトを
  出すか `-y` / `--yes`（スクリプト / CI 用）で同意を要求し、削除 target をレポート表示
  しなければならない。`reset` は profile / 世代を触らない FS-only teardown でなければならず、
  config が entry を残す限り次の apply がこれを再配置しなければならない。home / project の
  両モードで使用できなければならない。
---
# REQ-31f2882e-d2e3-4e3b-b783-feb627d73ac6: reset は profile を触らない FS-only teardown で配置物を無い状態へ戻す

## 仕様

```bash
nput reset <name> [target...]  # 配置物を無い状態へ戻す。target 省略で全 entry、指定でその entry のみ
```

`reset <name> [target...]` は配置物を**無い状態へ戻す** teardown。symlink は stale 除去と
同じ**保守的不変条件**（nput 管理・記録通りのみ・foreign は warning で残す）で除去し、
**copy target も削除**する（copy を消す唯一の明示手段）。symlink・copy いずれの除去後も
空親ディレクトリ剪定を適用する。データ損失リスクのため**確認プロンプト**を出すか
`--yes` で同意を要求し、削除 target をレポート表示する。**profile / 世代は触らない
FS-only teardown** で、config が entry を残す限り次の apply で再配置される（transient・
project mode は lstat 検査で復帰）。恒久除去は config から entry を消して apply、
profile 完全除去は `nix-env --profile <profileDir>/profile --delete-generations`。
home / project 両モード可。

> **上は原文の写しで、規範は frontmatter が正**。原文が参照する次の規範は本 item の
> 担当ではない。
>
> - 保守的不変条件の中身 → REQ-16aef46b-7bb8-4ca1-b962-e9f3ed1fd1d2、空親ディレクトリ剪定の規則 → REQ-8409db86-a1ba-4053-86dc-588985cc1ca7。
>   本 item はそれらを reset にも適用することだけを規定する
> - project mode の lstat 検査による復帰（→ ADR-0017）→ REQ-46fccb80-4bae-4d37-bc19-dded88e9a9c0
> - 確認プロンプトを stdin が TTY のときだけ出し、非 TTY で同意が無ければ即エラー停止
>   すること（→ ADR-0025）→ REQ-31dae599-f3a3-4bbe-b367-c955535265da
>
> 逆に、上の写しには現れないが規範文が持つものが 1 つある。同意を取る手段の短縮形
> `-y` で、原文はこの箇条書きでは `--yes` としか書かない。短縮形はグローバルフラグ表
> （`-y, --yes`）と ADR-0020 §3 の見出し（`--yes` / `-y` グローバルフラグ）が定めており、
> `-f` / `--file`・`-v` / `--verbose` が短縮形込みで規範化されているのに揃えて、
> 規範文では `-y` / `--yes` と書いた。

名指し必須と flock は REQ-a8edc58f-4adc-4637-b888-ab8ccc7e73e4、`--dryrun` は REQ-6a950d6d-c191-4235-a1b4-73ffc7c2bb38 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `reset <name> [target...]` の箇条書きと、
同節グローバルフラグ表の `-y, --yes`。

決定の実体は ADR-0020「配置物のリセット（`nput reset`）を追加する」で、`-y` / `--yes` に
よる同意もここで決まっている。
