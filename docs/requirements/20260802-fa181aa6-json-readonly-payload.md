---
id: "REQ-fa181aa6-29a2-48c3-ae07-cc1b9a3b0303"
type: requirement
name: "読み取り系の JSON ペイロードは dryRun パリティと info インベントリで表す"
specification: |
  `apply --dryrun` SHALL have dryRun parity with apply: it SHALL go through the same
  conversion as apply, so that the schemas agree structurally and only the observed values
  differ. Its `items` SHALL be the full inventory of every entry of the new manifest plus
  the old entries in the stale removal plan, in the same item shape as apply. Because a
  dryrun executes nothing, the skipped state SHALL NOT appear. A conflicting entry SHALL
  be `item.status:"failed"` with `error.code:"E_NPUT_COLLISION"`; the JSON output and exit
  2 SHALL coexist, and the item-caused error SHALL NOT be duplicated into
  `results[i].errors[]`.

  The changes of `apply --dryrun` SHALL be only the planned differences (a new place /
  copy maps to `add`, a replace to `modify`, a removal to `remove`, and an unlink plus
  re-placement of the same target merges into one `modify`). Because a dryrun does not
  perform the re-link, it does not observe the on-disk destination just before it, so a
  planned re-link to the recorded destination SHALL remain a `modify`. The overwrite of
  `--recopy` SHALL NOT appear in the dryrun plan, the planner classifying only place-once
  while recopy is on the materialize path of the engine. Empty-parent-directory pruning
  and the rename aside of `--backup` have no entry identity and SHALL produce neither an
  item nor a change. `generation` SHALL be an observation record with before == after,
  since no switch occurs; when the profile has not been created yet, both `before` and
  `after` SHALL be omitted and only `profile` emitted. The mapping and routing of warnings
  SHALL be shared with apply.

  `list-generations <name>` SHALL emit
  `results[i].result.info.generations = [{number, date, current}]` (`date` being the raw
  string displayed by nix-env) with `items: []`, and SHALL NOT emit the
  `SubjectResult.generation` slot, so as not to encode the generation numbers twice. Even
  for an empty profile it SHALL state `"generations": []` explicitly.

  `gitignore <name>` SHALL emit `results[i].result.info.paths` as anchored targets (with a
  leading `/`) and `items: []`. Even with zero entries it SHALL state `"paths": []`
  explicitly.

  `init <template>` has no subject and SHALL keep `results: []`, carrying `{template, ref}`
  in the top-level `info`. Because the info is fixed before the expansion is executed, it
  SHALL remain alongside the top-level `errors[]` on failure as well.

  The `dryRun` of the read-only commands (list-generations / gitignore / init) SHALL
  always be `false`: `dryRun` SHALL be limited to reflecting the `--dryrun` flag and SHALL
  NOT be used to express the absence of side effects.
specification_ja: |
  `apply --dryrun` は apply と dryRun パリティを持たなければならない。apply と同一の変換を
  通すことでスキーマの一致を構造的に保証し、値（観測結果）だけが異なるようにする。
  `items` は新 manifest の全 entry + stale 除去計画に載った旧 entry のフルインベントリと
  し、item 形は apply と同一とする。dryrun は何も実行しないため skipped 区分は現れない。
  conflict の entry は `item.status:"failed"` + `error.code:"E_NPUT_COLLISION"` とし、
  JSON 出力と exit 2 は両立させ、item 起因エラーを `results[i].errors[]` へ二重化しては
  ならない。

  `apply --dryrun` の changes は予定差分のみとする（place / copy 新規 → `add`・
  replace → `modify`・除去 → `remove`。同一 target の unlink + 再配置は 1 つの `modify` に
  合体）。dryrun は re-link を実行しないため re-link 直前の on-disk dest を観測せず、
  記録どおりの dest への予定 re-link も `modify` として残す。`--recopy` の上書きは dryrun の
  plan に現れない（planner は place-once 分類のみで recopy は engine の materialize 側経路の
  ため）。空親ディレクトリ剪定と `--backup` の rename 退避は entry 識別を持たず item /
  change を生まない。`generation` は観測記録とし、切替が起きないため before = after、
  profile 未作成の初回 plan では `before` / `after` を両省略し `profile` のみとする。
  warnings の写像・振り分けは apply と共通とする。

  `list-generations <name>` は `results[i].result.info.generations = [{number, date,
  current}]`（`date` は nix-env 表示の生文字列）と `items: []` を出し、
  `SubjectResult.generation` スロットは出してはならない（世代番号の二重符号化回避）。
  空 profile でも `"generations": []` を明示する。

  `gitignore <name>` は `results[i].result.info.paths` に anchor 形 target（先頭 `/`）と
  `items: []` を出す。entry 0 件でも `"paths": []` を明示する。

  `init <template>` は主体を持たないため `results: []` のままとし、トップレベル `info` に
  `{template, ref}` を載せる。info は展開実行前に確定するため、失敗時もトップ
  `errors[]` と並んで残さなければならない。

  読み取り系（list-generations / gitignore / init）の `dryRun` は常に `false` とする。
  `dryRun` は `--dryrun` フラグの反映に限定し、副作用の無さを `dryRun` で表現しては
  ならない。
---
# REQ-fa181aa6: 読み取り系の JSON ペイロードは dryRun パリティと info インベントリで表す

## 仕様

- **`apply --dryrun`**: apply と **dryRun パリティ**。実装も同一の変換（変更系の payload
  builder）を通すため、スキーマの一致は構造的に保証され、値（観測結果）だけが異なる。
  `items` = 新 manifest の全 entry + stale 除去計画に載った旧 entry（フルインベントリ・
  item 形は変更系と同一〔`info={target, method, subpath}`〕）。dryrun は何も実行しないため
  skipped 区分は現れない。conflict の entry は `item.status:"failed"` +
  `error.code:"E_NPUT_COLLISION"`（message は planner の理由）で、**JSON 出力と exit 2 は
  両立**し、item 起因エラーは `results[i].errors[]` へ二重化しない。
- **`apply --dryrun` の changes**: 予定差分のみ（place / copy 新規 → `add`・
  replace → `modify`・除去 → `remove`。同一 target の unlink + 再配置〔method 変更〕は
  変更系と同じく 1 つの `modify` に合体）。dryrun は re-link を実行しないため re-link 直前の
  on-disk dest（noop 判定の材料）を観測せず、記録どおりの dest への予定 re-link も
  `modify` として残る（テキスト plan の replace 行と同写像・実 apply の noop 抑止とは
  意図的な非対称）。`--recopy` の上書きは dryrun の plan に現れない（planner は place-once
  分類のみで recopy は engine の materialize 側経路）。reversible は変更系と同じ規則
  （dryrun に現れる change は全て `true`）。空親ディレクトリ剪定（rmdir）と `--backup` の
  rename 退避は entry 識別を持たず item / change を生まない。`generation` = 観測記録
  （切替が起きないため **before = after**・profile 未作成の初回 plan では **before / after を
  両省略**し `profile` のみ）。warnings の写像・振り分けは変更系と共通。
- **`list-generations <name>`**: `results[i].result.info.generations = [{number, date,
  current}]`（`date` は nix-env 表示の生文字列）・`items: []`。`SubjectResult.generation`
  スロットは出さない（世代番号の二重符号化回避）。空 profile でも `"generations": []` を
  明示する。
- **`gitignore <name>`**: `results[i].result.info.paths` = anchor 形 target（先頭 `/`）・
  `items: []`。デフォルトの行指向 stdout は不変（`--json` は opt-in の第 2 契約）。
  entry 0 件でも `"paths": []` を明示する。
- **`init <template>`**: 主体（config）を持たないため `results: []` のまま、トップレベル
  `info` に `{template, ref}`（展開テンプレート名と flake ref）を載せる。info は展開実行前に
  確定するため、失敗時もトップ `errors[]` と並んで「何を展開しようとしたか」が残る。
- **読み取り系の `dryRun` は常に `false`**（list-generations / gitignore / init）。
  `dryRun` は `--dryrun` フラグの反映に限定し、副作用の無さを `dryRun` で表現しない。

変更系ペイロード（item 形・warnings 一覧・部分失敗の写像）は REQ-2ea19863 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」→「niface 準拠の `--json`
出力」のサブ項目「読み取り系ペイロード（#132）」とその配下の全項目。

決定の実体は ADR-0043（読み取り系の info インベントリ・`init` の top-level info）と、
`--recopy` が planner の分類ではなく engine の materialize 経路であることを定めた ADR-0020。
