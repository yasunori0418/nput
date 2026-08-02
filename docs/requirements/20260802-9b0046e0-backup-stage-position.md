---
id: "REQ-9b0046e0-8ddc-4c0b-940e-3fe6f36d0e98"
type: requirement
name: "backup 退避は配置前除去の後・配置の前に置き、drift 修復経路でも同じく実施する"
specification: |
  The rename aside performed under `apply --backup` SHALL be placed as its own stage
  between the pre-placement removal (PreRemove) and the placement itself, and its subjects
  SHALL be exactly the unrecorded entities that the classification stages decided to stop
  with an error or to skip as a copy foreign file, an ancestor symlink conflict being
  excluded from them. On the generation-skip path, where
  drift repair takes place, PreRemove SHALL never be reached because of the invariants
  that govern it, whereas the Backup stage SHALL remain in effect and SHALL rename aside
  and place in the same way as an ordinary apply, because an unrecorded entity may appear
  at a target without any change of manifest or derivation, even between shell re-entries.
specification_ja: |
  `apply --backup` による rename 退避は、配置前除去（PreRemove）の後・配置の前の独立した
  段として置かなければならない。その対象は、判定各段が「エラーで停止」または「copy foreign
  スキップ」と判定した記録外実体とし、祖先 symlink conflict は含まない。ドリフト修復を行う
  世代スキップ経路では、PreRemove は
  その不変条件により到達しないが、Backup 段は対象であり続け、通常 apply と同じく退避 + 配置
  を行わなければならない（target への記録外実体の出現は manifest / derivation の変化を伴わず、
  shell 再入間でも起こり得るため）。
---
# REQ-9b0046e0: backup 退避は配置前除去の後・配置の前に置き、drift 修復経路でも同じく実施する

## 仕様

```
`apply --backup` 有効時、上記 0〜0.6 が「エラーで停止」または「copy foreign スキップ」と判定した
記録外実体（foreign な通常ファイル/ディレクトリ・実 dir migration 失敗・copy 構造不一致・
copy foreign 実ファイル・method 変更 copy→symlink）を、配置前に `<target>.<suffix>` へ rename 退避
（Backup。PreRemove の後・配置の前）してから新規配置する。祖先 symlink conflict は
対象外のまま（構造問題であり退避では解消しない）。退避先が既存なら conflict で停止する。
```

世代スキップ経路の drift 修復（repairDrift）では **PreRemove は不変条件によりこの経路に
到達しない**が、**Backup（`--backup`）はその不変条件の対象外**（target の記録外実体の出現は
manifest / derivation の変化を伴わず、shell 再入間でも起こり得るため）で、drift 修復の一環と
して通常 apply と同じく退避 + 配置される。

> **上は原文の写しで、規範は frontmatter が正**。写しに現れる次の 2 点は本 item の規範では
> なく REQ-5dd5a4e9 の担当で、写しの一部として残しているだけである。
>
> - 退避先 `<target>.<suffix>` が既存なら conflict で停止すること
> - 祖先 symlink conflict が対象外である**理由**（構造問題であり退避では解消しない）。
>   対象外であること自体は、退避対象の外延を定める本 item の規範文にも含めた
>
> 退避そのもののその他の契約（suffix の既定と `=` 区切り・`reset` が復元しないこと）も
> 同じく REQ-5dd5a4e9 の担当で、本 item は退避が配置手順のどの段に入るか・退避対象を
> どう括るか・drift 修復経路でも実施されることを規定する。世代スキップとドリフト修復
> そのものは REQ-46fccb80、undo ジャーナルは REQ-5e75aabc の担当。
>
> **退避対象の外延について**: 原文は 2 箇所で非対称に書く。「CLI 仕様」節（REQ-5dd5a4e9 の
> 出典）は種別を 4 つ（foreign な通常ファイル / ディレクトリ・copy 構造不一致・copy foreign
> 実ファイル・method 変更 copy→symlink）並べるのに対し、「配置動作仕様」§0.7 はそこへ
> 「実 dir migration 失敗」を加えた 5 つを並べる。本 item は種別の列挙ではなく判定各段の
> 結論（「エラーで停止」または「copy foreign スキップ」）で括ることにより、後者の広い方を
> 規範とした。実 dir migration 失敗（REQ-7cee95dd）はこの括りに含まれる。REQ-5dd5a4e9 の
> 規範文が持つ 4 種の列挙は、原文の「CLI 仕様」節に忠実な**例示**であり外延の確定ではない
> （同 item にもその旨を注記した）。

## 出典

`docs/spec.md`「配置動作仕様」→「symlink モード」の手順 0.7 と、「途中失敗時の巻き戻し」節の
箇条書き第 4 項（世代スキップ経路の drift 修復と Backup）。

決定の実体は ADR-0045「`apply --backup[=suffix]` — 配置を塞ぐ記録外実体の rename 退避」で、
PreRemove がドリフト修復経路に到達しない不変条件は ADR-0046 / ADR-0047 が定めている。
