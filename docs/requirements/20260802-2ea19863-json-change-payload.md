---
id: "REQ-2ea19863-eaa2-466b-b1ed-3f56f6417c62"
type: requirement
name: "変更系の JSON ペイロードは engine 結果からフルインベントリと実差分を導く"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
specification: |
  The JSON payload of the changing commands (`apply` / `reset` / `rollback`) SHALL be
  generated from the same engine result as the `-v` report, via a DTO, and SHALL NOT be
  aggregated twice.

  `items` SHALL be a full inventory: for `apply` / `rollback`, every entry of the new
  (for rollback, the target) manifest plus the old entries that appeared in the stale
  removal plan; for `reset`, the selected teardown entries after narrowing by
  `[target...]`. Each item SHALL carry `kind="entry"`, `label` = target and
  `info={target, method, subpath}`, omitting an empty `subpath` and keeping the per-build
  varying `src` out of the item info. An entry with no change SHALL also be `success`, and
  a target appearing in both old and new because of a method change SHALL be represented
  by the new entry.

  `changes` SHALL contain only actual differences and SHALL NOT include no-ops: a new
  place / copy maps to `add`, a replace / recopy to `modify`, and a stale removal or a
  reset removal to `remove`. The `kind` SHALL reflect the transition that actually
  occurred rather than the intended one. An unlink plus re-placement of the same target
  SHALL be merged into a single `modify`. A mechanical re-link to the recorded destination
  (old == new) SHALL NOT be a change, since no state transition occurred.
  `change.info = {old, new}` SHALL carry the symlink destination / copy src, `old` for a
  replace being the actual on-disk destination just before re-linking and `old` for a
  stale removal being the recorded destination, while the `old` of a recopy and the info
  of a copy deletion SHALL be omitted. `reversible` SHALL be `true` for symlink add /
  modify / remove and for a new copy, and `false` for a recopy overwrite and for a copy
  deletion by reset.

  Partial failure SHALL be mapped as follows: a failed entry becomes
  `item.status:"failed"` plus `item.error`; an entry not reached because of an earlier
  failure becomes `"skipped"`, this being the only use of that state; an entry completed
  before the failure stays `"success"` and SHALL include all of its corresponding changes.
  Even for an execution unwound by the undo journal, `changes` SHALL be kept as the record
  of the differences produced up to the point of failure, and the subject warning
  `W_NPUT_UNWOUND` SHALL notify that the differences no longer remain on disk. A failure
  independent of an entry (commit / build / lock) SHALL NOT produce an item and SHALL go
  to `results[0].errors[]`.

  A non-dryrun conflict stop SHALL mark the entry as `item.status:"failed"` with
  `error.code:"E_NPUT_COLLISION"`, and the remaining planned entries SHALL be skipped. The
  aggregate error SHALL NOT be duplicated into `subjectResult.errors[]`, being caused by
  an item.

  `generation = {profile, before?, after?}` SHALL be emitted for `apply` / `rollback`
  only, as observed values at the start / end of execution, omitting what cannot be
  observed. On failure it SHALL be the observation of a pointer that did not move
  (before == after). The From→To of a rollback SHALL be carried by
  `generation.before/after` and SHALL NOT be placed in `result.info`. `reset` SHALL NOT
  emit the generation slot at all, being an FS-only teardown that leaves the profile and
  generations untouched.

  Structured warnings from the planner SHALL be mapped to `W_NPUT_*` and placed in
  `item.warnings` when the target is within the inventory, or in
  `subjectResult.warnings` when it is outside it. The codes SHALL be
  `W_NPUT_FOREIGN_SYMLINK`, `W_NPUT_COPY_FOREIGN`, `W_NPUT_STALE_MISMATCH`,
  `W_NPUT_STALE_NON_SYMLINK`, `W_NPUT_COPY_ORPHAN` and `W_NPUT_UNWOUND`, with
  `detail = {target}`. The human-facing text on stderr SHALL always coexist.

  The changes of `reset` SHALL be only what was actually removed. A rename aside by
  `--backup` and empty-parent-directory pruning are not state transitions of an entry and
  SHALL NOT appear in `changes` / `items`.
specification_ja: |
  変更系コマンド（`apply` / `reset` / `rollback`）の JSON ペイロードは、`-v` レポートと
  同一の engine 結果から DTO 経由で生成しなければならず、二重集計してはならない。

  `items` はフルインベントリでなければならない。`apply` / `rollback` は新（rollback は
  戻り先）manifest の全 entry + stale 除去計画に載った旧 entry、`reset` は選択済み
  teardown entry（`[target...]` 絞り込み後）。各 item は `kind="entry"`・`label` = target・
  `info={target, method, subpath}`（`subpath` 空は省略。ビルド毎に変わる src は item info に
  置かない）を持たなければならない。変更の無い entry も `success` としなければならず、
  method 変更で新旧両方に現れる target は新 entry が item を代表しなければならない。

  `changes` は実差分のみとし no-op を含めてはならない。place / copy 新規 → `add`、
  replace / recopy → `modify`、stale 除去 / reset の除去 → `remove`。`kind` は意図した
  遷移ではなく実際に生じた遷移を反映しなければならない。同一 target の unlink + 再配置は
  1 つの `modify` に合体させなければならない。記録どおりの dest への機械的 re-link
  （old == new）は状態遷移が無いため change にしてはならない。
  `change.info = {old, new}` は symlink 先 / copy src を運ばなければならず、replace の
  `old` は re-link 直前の実 on-disk dest、stale 除去の `old` は記録 dest とする。
  recopy の `old` と copy 削除の info は省略しなければならない。`reversible` は symlink の
  add / modify / remove と copy 新規を `true`、recopy 上書きと reset の copy 削除を
  `false` としなければならない。

  部分失敗は次のように写像しなければならない。失敗した entry → `item.status:"failed"` +
  `item.error`。前段の失敗で未到達だった entry → `"skipped"`（この用途のみ）。失敗までに
  完了した entry → `"success"`（対応する change を全て含まなければならない）。undo
  ジャーナルが巻き戻した実行でも changes は「失敗時点までに生じた差分」の記録として保た
  なければならず、subject 警告 `W_NPUT_UNWOUND` で差分がディスク上に残っていないことを
  通知しなければならない。commit / build / lock など entry 非依存の失敗は item を落として
  はならず、`results[0].errors[]` へ載せなければならない。

  非 dryrun の conflict 停止は該当 entry を `item.status:"failed"` +
  `error.code:"E_NPUT_COLLISION"` としなければならず、残りの計画 entry は skipped と
  しなければならない。集約エラーは item 起因のため `subjectResult.errors[]` へ
  重複させてはならない。

  `generation = {profile, before?, after?}` は `apply` / `rollback` のみが出さなければ
  ならない（実行開始 / 終了時点の観測値・観測できない値は省略）。失敗時は動かなかった
  ポインタの観測（before == after）としなければならない。`rollback` の From→To は
  `generation.before/after` が運ばなければならず、`result.info` には置いてはならない。
  `reset` は generation スロット自体を出してはならない
  （profile / 世代は untouched の FS-only teardown で遷移が存在しないため）。

  planner の構造化 warning は `W_NPUT_*` に写像しなければならず、対象 target が
  インベントリ内なら該当 `item.warnings`、外なら `subjectResult.warnings` へ
  振り分けなければならない。コードは `W_NPUT_FOREIGN_SYMLINK` / `W_NPUT_COPY_FOREIGN` /
  `W_NPUT_STALE_MISMATCH` / `W_NPUT_STALE_NON_SYMLINK` / `W_NPUT_COPY_ORPHAN` /
  `W_NPUT_UNWOUND` としなければならず、`detail = {target}` を付けなければならない。
  stderr の人間向けテキストは常時併存させなければならない。

  `reset` の changes は実際に除去したもののみでなければならない。`--backup` の退避と
  空親ディレクトリ剪定は entry の状態遷移ではないため changes / items に載せてはならない。
---
# REQ-2ea19863: 変更系の JSON ペイロードは engine 結果からフルインベントリと実差分を導く

## 仕様

変更系（`apply` / `reset` / `rollback`・単一 config・非 dryrun）の JSON ペイロードは
`-v` レポートと**同一の engine 結果**（`engine.Result` / `engine.ResetResult`）から DTO
経由で生成する（二重集計しない）。

- **items = フルインベントリ**: `apply` / `rollback` は新（rollback は戻り先）manifest の
  全 entry + **stale 除去計画に載った旧 entry**（前世代記録・除去の完遂と無関係に列挙）、
  `reset` は選択済み teardown entry（`[target...]` 絞り込み後）。各 item は
  `kind="entry"`・`label` = target・`info={target, method, subpath}`（`subpath` 空は省略。
  ビルド毎に変わる src は item info に置かない）。変更の無い entry も `success`。
  method 変更で新旧両方に現れる target は新 entry が item を代表する。
- **changes = 実差分のみ**（noop を含めない）: place / copy 新規 → `add`、
  replace / recopy → `modify`、stale 除去 / reset の除去 → `remove`。`kind` は意図した
  遷移ではなく**実際に生じた遷移**。同一 target の unlink + 再配置（symlink→copy の
  method 変更）は 1 つの `modify` に合体する。記録どおりの dest への機械的 re-link
  （old == new）は状態遷移が無いため change にしない（`-v` の op 表示とは意図的に非対称）。
  `change.info = {old, new}`（symlink 先 / copy src。replace の `old` は re-link 直前の実
  on-disk dest〔foreign 含む〕・stale 除去の `old` は記録 dest。recopy の `old` と copy
  削除の info は旧内容を追跡していないため省略）。**reversible**: symlink の add / modify /
  remove と copy 新規 = `true`、recopy 上書きと reset の copy 削除 = `false`
  （`W_IRREVERSIBLE` は付けず `change.reversible` のみで表現）。
- **部分失敗のマッピング**: 失敗した entry → `item.status:"failed"` + `item.error`
  （コードは上記分類）。前段の失敗で未到達だった entry → `"skipped"`（この用途のみ）。
  失敗までに完了した entry → `"success"` + 対応する change を**全て**含む。undo ジャーナルが
  巻き戻した実行でも changes は「失敗時点までに生じた差分」の記録として保ち、subject 警告
  **`W_NPUT_UNWOUND`** で差分がディスク上に残っていないことを通知する。commit / build /
  lock など entry 非依存の失敗は item を落とさず `results[0].errors[]` へ。
- **conflict**: 非 dryrun の conflict 停止は該当 entry を `item.status:"failed"` +
  `error.code:"E_NPUT_COLLISION"`（message = planner の理由）にし、残りの計画 entry は
  skipped（何も実行されていない）。集約エラー（`N conflict(s) detected`）は item 起因の
  ため `subjectResult.errors[]` へ重複させない。exit 1 / 2 は内部意味のまま。
- **generation**（観測記録）: **`apply` / `rollback` のみ**
  `generation = {profile, before?, after?}` を出す（実行開始 / 終了時点の観測値・
  観測できない値は省略: 初回 apply の `before`、世代リンク形でない profile）。失敗時は
  動かなかったポインタの観測（before == after）。`rollback` の From→To は
  `generation.before/after` が運び、`result.info` には置かない（二重符号化回避）。
  **`reset` は generation スロット自体を出さない**（前世代 manifest を読んで FS を
  除去するだけの FS-only teardown で、profile / 世代は untouched・遷移が存在しない）。
- **warnings**: planner の構造化 warning を W_NPUT_* に写像し、対象 target がインベントリ内
  なら該当 `item.warnings`、外（entry が config を離れた copy orphan 等）なら
  `subjectResult.warnings` へ。コード: `W_NPUT_FOREIGN_SYMLINK`（foreign symlink 上書き）/
  `W_NPUT_COPY_FOREIGN`（place-once の copy skip）/ `W_NPUT_STALE_MISMATCH`（記録不一致で
  残した stale symlink）/ `W_NPUT_STALE_NON_SYMLINK`（symlink でないため残した stale
  target）（後 2 者 = 保守的不変条件による keep・reset の kept-foreign も同じ。当該 item は
  success のまま = 方針による不作為）/ `W_NPUT_COPY_ORPHAN`（copy orphan・subject 級）/
  `W_NPUT_UNWOUND`（上記・subject 級）。`detail = {target}`。stderr の人間向けテキストは
  常時併存。
- **`reset` の changes**: 実際に除去したもののみ（symlink remove = `reversible:true` +
  `info.old` = 記録 dest、copy 削除 = `reversible:false`・info なし）。確認プロンプトで
  中止した実行（`--json` では起き得ない）は差分ゼロ。
- **`--backup` の退避（BackedUp）と空親ディレクトリ剪定（Pruned）は entry の状態遷移では
  ないため changes / items に載らない**（`-v` / stderr のみ）。

> **上は原文の写しで、規範は frontmatter が正**。原文が併記する実装 issue の進捗
> （#131 時点の非対象・`reset --dryrun` は最小形のまま followup・`--all` は #164）は
> 要求ではなく履歴の注記で、`apply --dryrun` のペイロードは REQ-fa181aa6、`--all` は
> REQ-059eb4d5 の担当。undo ジャーナルそのものの規範（→ ADR-0044）は REQ-5e75aabc の担当。

> **注記（規範ではない）— items / changes / 部分失敗 / conflict / generation / warnings を
> 1 item に束ねる理由**: これらは「同一の payload builder が engine 結果から生成する
> 1 つの DTO 契約」を成す規範であり、本 item は「変更系の 1 スキーマ契約」を満たすべき
> こと 1 つと見なす。個別に切ると原文の 1 サブ項目（「#131 変更系ペイロード」）と item の
> 対応が崩れ、出典の追跡性が落ちる。同じ理由で REQ-fa181aa6（読み取り系）・
> REQ-059eb4d5（`--all`）も原文のサブ項目単位で切っている。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」→「niface 準拠の `--json`
出力」のサブ項目「#131 変更系ペイロード」とその配下の全項目。

決定の実体は ADR-0043 §4 / §6 / §8。
