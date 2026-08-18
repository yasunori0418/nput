---
id: "REQ-5dd5a4e9-6162-4fa5-b295-66844f5a4f3b"
type: requirement
name: "apply --backup は配置を塞ぐ記録外実体を rename 退避してから配置する"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  `nput apply <name> --backup[=<suffix>]` SHALL rename an existing *unrecorded* entity
  that blocks placement — such as a foreign regular file / directory, a copy structure
  mismatch, a foreign real file under a copy, or a method change from copy to symlink, the
  extension of the subjects being settled by REQ-9b0046e0 — aside to
  `<target>.<suffix>` before placing, as an escape hatch out of a conflict. Omitting the
  value SHALL mean the suffix `nput-backup`. Because of the cobra `NoOptDefVal`
  constraint, specifying a suffix SHALL require the `=` form (`--backup=bak`); the
  space-separated form SHALL NOT be interpreted as a suffix. An ancestor symlink conflict
  SHALL remain out of scope, being a structural problem that renaming aside does not
  resolve. A rename aside SHALL always be reported on stderr at warning level. When the
  destination `<target>.<suffix>` already exists, the command SHALL stop with a conflict
  and SHALL NOT silently overwrite it. A rename aside SHALL also be subject to the undo
  journal on mid-run failure. `nput reset` SHALL NOT restore the entities set aside.
specification_ja: |
  `nput apply <name> --backup[=<suffix>]` は、配置を塞ぐ既存の記録外実体（例: foreign な
  通常ファイル / ディレクトリ・copy 構造不一致・copy foreign 実ファイル・method 変更
  copy→symlink。対象の外延は REQ-9b0046e0 が定める）を `<target>.<suffix>` へ rename 退避して
  から配置しなければならない（conflict の脱出ハッチ）。値なしは suffix `nput-backup` を
  意味しなければならない。cobra `NoOptDefVal` の制約により suffix 指定は `=` 区切り必須と
  しなければならず（`--backup=bak`）、スペース区切りは suffix として扱ってはならない。
  祖先 symlink conflict は対象外のままでなければならない
  （構造問題であり退避では解消しないため）。退避の発動は warning 級で常時 stderr に
  出さなければならない。退避先 `<target>.<suffix>` が既に存在するときは conflict で停止
  しなければならず、黙って上書きしてはならない。退避も途中失敗時の undo ジャーナルの
  対象としなければならない。
  `nput reset` は退避物を復元してはならない。
---
# REQ-5dd5a4e9: apply --backup は配置を塞ぐ記録外実体を rename 退避してから配置する

## 仕様

```bash
nput apply <name> --backup           # 既存の記録外実体を <target>.nput-backup へ rename 退避してから配置（既定 suffix）
nput apply <name> --backup=<suffix>  # 退避 suffix を明示（"=" 区切り必須・スペース区切り不可）
```

`apply <name> --backup[=<suffix>]` は、配置を塞ぐ既存の**記録外**実体（foreign な通常
ファイル / ディレクトリ・copy 構造不一致・copy foreign 実ファイル・method 変更
copy→symlink）を `<target>.<suffix>` へ rename 退避してから配置する、conflict の脱出
ハッチ。値なし = suffix `nput-backup`。suffix 指定は cobra `NoOptDefVal` の制約で
**`=` 区切り必須**（`--backup=bak`。スペース区切り `--backup bak` は次の位置引数として
扱われ suffix にならない）。**祖先 symlink conflict は対象外のまま**（構造問題であり
退避では解消しない）。退避発動は warning 級で**常時 stderr** に出す。退避先
`<target>.<suffix>` が既に存在する（前回の退避物が残っている）ときは conflict で停止し、
黙って上書きしない。退避も途中失敗時の undo ジャーナル対象。**`reset` は退避物を復元
しない**（ユーザー所有物として残置。復元は手動 `mv`）。

> **上は原文の写しで、規範は frontmatter が正**。undo ジャーナルそのものの規範
> （→ ADR-0044）は REQ-5e75aabc の担当で、本 item は「退避もその対象である」ことだけを
> 規定する。退避が配置手順のどの段に入るか（PreRemove の後・配置の前）と、ドリフト修復
> 経路でも実施されることは REQ-9b0046e0 の担当。
>
> **退避対象の外延は REQ-9b0046e0 の担当**。上の写しが挙げる 4 種は原文の「CLI 仕様」節の
> 逐語で、原文は「配置動作仕様」§0.7 でこれに「実 dir migration 失敗」を加えた 5 種を挙げる。
> REQ-9b0046e0 が種別の列挙ではなく判定各段の結論（「エラーで停止」または「copy foreign
> スキップ」）で括ることで広い方を規範としているため、本 item の規範文では 4 種を `例:` と
> 明示して外延の確定から降ろした（写しは原文どおりのまま）。
>
> 一方、**祖先 symlink conflict が対象外であることは本 item の規範**として持つ。退避しない
> 対象の指定は退避契約の一部であり、その理由（構造問題であり退避では解消しない）と同じ
> item に置くのが所在として自然なため。REQ-9b0046e0 の規範文はこの除外を本 item へ委譲する
> 形で括りを述べている。

`--dryrun` / `--all` との合成は REQ-02a33511 / REQ-687e225f の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `apply <name> --backup` の箇条書きと、
グローバルフラグ表の `--backup[=<suffix>]`。

決定の実体は ADR-0045「`apply --backup[=suffix]` — 配置を塞ぐ記録外実体の rename 退避」。
