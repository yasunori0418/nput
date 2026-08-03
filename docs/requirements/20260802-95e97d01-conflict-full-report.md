---
id: "REQ-95e97d01-5c34-44b3-bc85-9ca53472bc3d"
type: requirement
name: "conflict で停止するときは全件を対処ガイダンス付きで列挙してから 1 本の集約エラーを返す"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
specification: |
  When `apply` or `rollback` stops on a conflict, every conflict collected by the planner
  SHALL be enumerated on stderr before stopping, so that the user is not forced into a
  cycle of fixing one, re-running, and meeting the next. The enumerated set SHALL be the
  same set that `--dryrun` reports. Each line SHALL carry a single line of remediation
  guidance appropriate to the kind of conflict. The error returned on stopping SHALL be a
  single aggregated message carrying the count, and the detail of each individual conflict
  SHALL be borne by the preceding enumeration rather than by that message. This reporting
  SHALL NOT change the exit code, and the machine-readable output SHALL be reconciled with
  the same full set. Which situations constitute a conflict is stated by the items
  responsible for them and SHALL NOT be restated here.
specification_ja: |
  `apply` / `rollback` が conflict で停止するとき、planner が収集した conflict の全件を
  停止前に stderr へ列挙しなければならない（1 件ずつ「修正 → 再実行 → 次の 1 件」を
  強いられないようにするため）。列挙する集合は `--dryrun` が報告する集合と同一で
  なければならない。各行には conflict 種別に応じた 1 行の対処ガイダンスを付けなければ
  ならない。停止時に返すエラーは件数を持つ集約メッセージ 1 本でなければならず、個々の
  conflict の詳細はそのメッセージ本文ではなく直前の列挙が担わなければならない。
  この報告によって終了コードを変えてはならず、機械可読
  出力も同じ全件集合と整合させなければならない。どの状況が conflict になるかは各担当
  item の規範であり、ここでは再掲しない。
---
# REQ-95e97d01: conflict で停止するときは全件を対処ガイダンス付きで列挙してから 1 本の集約エラーを返す

## 仕様

`apply` / `rollback` が conflict で停止するとき、`planner.Compute` が収集した
**`plan.Conflicts` の全件**を stderr へ列挙してから停止する（`--dryrun` の
`Result.Conflicts` と同じ集合。HM の `checkLinkTargets` と対称）。1 件ずつ「修正 →
再実行 → 次の 1 件…」を強いられないよう、1 回の実行で全ての衝突箇所を可視化する。

各行には conflict 種別に応じた 1 行の対処ガイダンスが付く。

| conflict 種別 | ガイダンス |
|---|---|
| foreign 実体（`target` に記録外の通常ファイル・ディレクトリ）| 手動で退避・削除してから再実行（または `apply --backup` で自動退避）|
| foreign 祖先 symlink（記録なし / 記録 dest と不一致 / 前世代なし）| そのリンクを作った主体（別 config / 別ツール / 手動操作）を確認 |
| 自己矛盾 manifest（次世代にも祖先 symlink が残ったまま配下 target を定義）| entry 定義を見直す（祖先と配下のどちらかに一本化）|
| copy 構造不一致（`subpath` の dir/file 種別と既存 `target` の種別が食い違う）| entry 定義を見直す（または `apply --backup` で自動退避）|
| backup 退避先が既存（`apply --backup` 有効時、`<target>.<suffix>` に前回の退避物が残っている）| 手動で退避物を移動・削除してから再実行 |

停止時の `error` は列挙の後に返す**件数付き集約メッセージ 1 本**（例:
`nput: N conflict(s) detected; stopped without placing (see above)`）で、個別 conflict の
詳細は本文ではなく直前の stderr 列挙が担う。終了コードは不変（`apply` 非 dryrun は `1`・
`--dryrun` は `2`）。`--json` の `Conflicts` 配列も同じ全件集合と整合させる。

> **上は原文の写しで、規範は frontmatter が正**。表が挙げる各 conflict 種別が「なぜ
> conflict になるか」は、それぞれ REQ-053cfed2（foreign 実体。実 dir が例外にならない
> 条件は REQ-7cee95dd）・REQ-c9ab91c1（祖先 symlink と自己矛盾）・REQ-d2277c7a（copy の
> 構造不一致）・REQ-5dd5a4e9（backup 退避先の既存）の担当。終了コードの値そのものは
> REQ-2c5a10d8、`--dryrun` が読み取り専用で conflict を返すことは REQ-02a33511、
> `--json` エンベロープ側の形は REQ-a5053191 / REQ-2ea19863、報告先が stderr である
> ことの出力規律は REQ-fea038de / REQ-8ef34101 の担当。この「全件列挙してから 1 本の
> 集約エラー」の形を巻き戻し失敗の報告にも用いることは REQ-9fca28c9 の担当。

## 出典

`docs/spec.md`「エラー仕様」→「conflict の全件報告」節。

決定の実体は grilling 2026-07-12 D6 で、conflict を全件列挙する方針を定めている。この決定を
表明した ADR は無い（ADR-0047 が「#176（conflict 全件報告）はスコープ外のまま」と明記する
とおり、ADR 化されていない）。`--backup` を脱出ハッチとするガイダンスと、退避先が既存の
ときを conflict 種別に加えることは ADR-0045「`apply --backup[=suffix]` — 配置を塞ぐ記録外
実体の rename 退避」による。報告を stderr に置き失敗経路を沈黙の対象にしない出力規律は
ADR-0031「成功時デフォルト沈黙化」が定めるが、同 ADR は conflict に一切言及せず全件列挙を
決めていないため、側面の根拠として `justifies` は張らない（出力規律そのものの帰属は
REQ-fea038de / REQ-8ef34101 が担当する）。同じ ADR-0031 が REQ-9fca28c9 へは `justifies` を
張っているが、あちらは「この報告を成功時沈黙の対象にしてはならない」を自身の規範に含めて
おり、本 item は出力規律を規範に持たず委譲している。この違いによる非対称であって、
張り漏れではない。
