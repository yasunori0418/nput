---
id: "REQ-059eb4d5-63fb-4f8e-b705-11b5e2ed4ae5"
type: requirement
name: "--all は config ごとの SubjectResult を単一実行と同一形状で積む"
derives_from:
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
specification: |
  For `apply --all` / `list-generations --all` / `gitignore --all`, one `SubjectResult`
  SHALL be pushed onto `results[]` per selected config. Its shape SHALL be exactly
  identical to that of a single-config execution, and SHALL NOT carry any field that
  discriminates `--all`, a single execution being the N=1 special case rather than a
  different document. `specVersion` / `tool` / `command` / `dryRun` / `startedAt` /
  `finishedAt` SHALL be placed once at the top level, so that an extracted `results[i]`
  SHALL NOT be a valid document on its own. The mapping rules for the contents of each
  `results[i]` SHALL be the same as for a single config, with no `--all`-specific mapping.

  `results[]` SHALL be ordered lexically after applying the per-command selection rule.
  Since each config is independently atomic, the order SHALL NOT affect the outcome, and a
  consumer SHALL look results up by `subject.name`; the resolution of `changes[].itemId`
  SHALL be closed within the same `SubjectResult`.

  The aggregate `status` SHALL be `error` when even one subject is `error`, and `success`
  when all succeeded. Even on partial failure, the `SubjectResult` of every successful
  config SHALL remain. Zero targets SHALL give `results: []` with `status:"success"`, the
  shape being identical for N=0 and the keys always present.

  A failure of a config that is independent of an item (build / eval / lock) SHALL go to
  that `results[i].errors[]`, the layering rule itself being that of a single-config
  execution and not restated here. Specific to `--all`: the aggregate error SHALL NOT be
  piled onto the top level, being a restatement of failures the subjects already carry.

  The try-lock skip (another apply in progress) SHALL be treated as `status:"success"` for
  that subject, symmetrically with a named apply returning exit 0, a skip not being a
  failure.

  `apply --all --dryrun` SHALL go through the same payload builder as a real apply, so its
  structural parity is that of the single `--dryrun`. A config with a conflict has the
  entry as `item.status:"failed"` with `error.code:"E_NPUT_COLLISION"`, so that subject
  SHALL be `status:"error"` and therefore the aggregate SHALL also be `status:"error"`.
  `status` SHALL strictly observe only "non-zero iff error", the priority among the exit
  codes themselves being stated elsewhere and not restated here.

  `gitignore --all` SHALL NOT deduplicate across configs in JSON: each
  `results[i].result.info.paths` SHALL hold only the targets of that config itself, and a
  path shared by several configs SHALL appear in both `SubjectResult`s, since attributing
  it to one of them would misstate which config declares it. A consumer that needs the
  union SHALL union and deduplicate across `results`. The asymmetry against the default
  text output, which aggregates across configs, SHALL be kept intentionally; what that
  text output does is stated elsewhere and not restated here.

  `list-generations --all` SHALL hold `result.info.generations` per home mode config found
  by scanning the profile directories, with `items: []`, identically to a single
  execution. A read failure during enumeration SHALL abort there as before, the failing
  config's subject SHALL carry that failure, and the results of the configs already
  enumerated SHALL remain.

  The read-only `--all` commands SHALL NOT place the configs after the abort into
  `results[]`, since both abort enumeration at the first failure, whereas `apply --all`
  continues through partial failure so that every selected config always appears in
  `results[]`. A consumer therefore SHALL NOT assume that the set of subjects in
  `results[]` matches the set of target configs; they match on success.
specification_ja: |
  `apply --all` / `list-generations --all` / `gitignore --all` は、選択した config ごとに
  `SubjectResult` を 1 つ `results[]` へ積まなければならない。形状は単一 config 実行と
  完全に同一とし、`--all` を判別するフィールドを持ってはならない（単一実行は N=1 の
  特殊形であって別文書ではない）。`specVersion` / `tool` / `command` / `dryRun` /
  `startedAt` / `finishedAt` はトップに 1 度だけ置き、`results[i]` を切り出しても単独で
  valid な文書にはならないようにしなければならない。各 `results[i]` の中身の写像規則は
  単一 config と同一でなければならず、`--all` 固有の写像を持ってはならない。

  `results[]` はコマンドごとの選択規則を適用したあとの辞書順で並べなければならない。各
  config は独立 atomic なので順序は結果に影響せず、消費側は `subject.name` で引かなければ
  ならず、`changes[].itemId` の解決は同一 `SubjectResult` 内に閉じなければならない。

  集約 `status` は 1 主体でも `error` なら `error`、全て成功なら `success` としなければ
  ならない。部分失敗でも成功した config の `SubjectResult` は全て残さなければならない。
  対象 0 件は `results: []` + `status:"success"` としなければならず、N=0 でも同一形状・
  キーは常に存在させなければならない。

  config 単位の item 非依存な失敗（build / eval / lock 等）は該当 `results[i].errors[]` へ
  載せなければならない。層の振り分け規則そのものは単一 config 実行と同一で、本 item では
  規定しない。
  `--all` 固有の規範として、集約エラーは既に各 subject が持つ失敗の言い換えなのでトップへ
  重ねてはならない。

  try-lock skip（他の apply が進行中）は、名指し apply が exit 0 を返すのと対称に、その
  subject を `status:"success"` として扱わなければならない（skip は失敗ではない）。

  `apply --all --dryrun` は本 apply と同一の payload builder を通さなければならず、構造
  parity は単一 `--dryrun` と同じとする。conflict のある config は該当 entry が
  `item.status:"failed"` + `error.code:"E_NPUT_COLLISION"` となり、その subject は
  `status:"error"`、したがって集約も `status:"error"` としなければならない。`status` は
  「非 0 ⇔ error」だけを厳守しなければならない。終了コード同士の優先度は別 item の担当で、
  本 item では規定しない。

  `gitignore --all` は JSON で cross-config dedup をしてはならない。各
  `results[i].result.info.paths` はその config 自身の target のみを持たなければならず、
  複数 config が共有する path は双方の `SubjectResult` に現れなければならない（どちらか
  一方へ帰属させるのは「どの config が宣言しているか」の偽りであるため）。union が必要な
  消費側が `results` を跨いで union + dedup する。config 横断で集約するテキスト既定出力との
  非対称は意図的に保たなければならない。そのテキスト出力が何をするかは別 item の担当で、
  本 item では規定しない。

  `list-generations --all` は profile ディレクトリ走査で見つかった home mode config ごとに
  `result.info.generations` を持たなければならない（`items: []`・単一実行と同一）。列挙途中の
  読み取り失敗は従来どおりそこで打ち切らなければならず、失敗した config の subject がその
  失敗を持ち、既に列挙済みの config の結果は残さなければならない。

  読み取り系 `--all` は打ち切り以降の config を `results[]` に載せてはならない
  （両者は最初の失敗で列挙を中断するため）。`apply --all` は部分失敗でも全 config を
  続行するので、選択された全 config が必ず `results[]` に載る。したがって消費側は
  `results[]` の subject 集合が対象 config 集合と一致することを前提にしてはならない
  （成功時は一致する）。
---
# REQ-059eb4d5: --all は config ごとの SubjectResult を単一実行と同一形状で積む

## 仕様

**`--all` の複数 `SubjectResult`**（`apply --all` / `list-generations --all` /
`gitignore --all`）: 選択した config ごとに `SubjectResult` を 1 つ `results[]` へ積む。
**形状は単一 config 実行と完全に同一**で、`--all` を判別するフィールドは持たない
（単一実行は N=1 の特殊形であって別文書ではない）。`specVersion` / `tool` / `command` /
`dryRun` / `startedAt` / `finishedAt` はトップに 1 度だけ置き、`results[i]` を切り出しても
単独で valid な文書にはならない。各 `results[i]` の中身（items / changes / generation /
warnings / info）の写像規則は単一 config と同一で、`--all` 固有の写像は無い。

- **順序**: `results[]` はコマンドごとの選択規則（`apply --all` は `--project-root` 等の
  フィルタ適用後・`gitignore --all` は projectRoot config のみ・`list-generations --all` は
  profile ディレクトリ走査で見つかった home mode config）を適用したあとの辞書順で並ぶ。
  ただし各 config は独立 atomic なので順序は結果に影響せず、消費側は `subject.name` で
  引くこと（`changes[].itemId` の解決は**同一 `SubjectResult` 内**に閉じる・result 跨ぎ
  参照は niface lint MUST 違反）。
- **集約 `status`**: **1 主体でも `error` なら `error`**、全て成功なら `success`。
  **部分失敗でも成功した config の `SubjectResult` は全て残る**（失敗した config だけが
  `status:"error"` になる）。対象 0 件（フィルタにマッチせず / profile 未作成 /
  projectRoot config 無し）は `results: []` + `status:"success"`（N=0 でも同一形状・
  キーは常に存在する）。
- **エラーの層**: config 単位の item 非依存な失敗（build / eval / lock 等）は**該当
  `results[i].errors[]`**、item 起因の失敗（entry 失敗 / conflict）は `item.error` に
  埋めて `results[i].errors[]` へ重複させない。**トップ `errors[]` に載るのは主体列挙前の
  失敗のみ**（entrypoint 発見・一括 eval・RunE 到達後の引数拒否〔`apply cannot combine
  <name> with --all` 等〕など、subject が 1 つも確定していない段階の失敗。cobra の `Args:`
  検証やフラグ解析の失敗はそもそもエンベロープを出さない）。`--all` の集約エラー
  （`N config(s) failed`）は既に各 subject が持つ失敗の言い換えなのでトップへ重ねない。
- **try-lock skip**: `ErrSkipped`（他の apply が進行中）は名指し apply が exit 0 を返すのと
  対称に、その subject を `status:"success"` として扱う（skip は失敗ではない）。
- **`apply --all --dryrun`**: 本 apply と同一の payload builder を通すため構造 parity は
  単一 `--dryrun` と同じ。conflict のある config は該当 entry が `item.status:"failed"` +
  `error.code:"E_NPUT_COLLISION"`（item 起因）で、その **subject は `status:"error"`**、
  したがって**集約も `status:"error"`**。終了コードは `error(1) > conflict(2) > 0` の
  優先度で不変であり、`status` は「非 0 ⇔ error」だけを厳守する（1 と 2 の内訳は exit code が
  語る）。単一 `apply --dryrun` の conflict と完全対称。
- **`gitignore --all` は JSON で cross-config dedup をしない**: 各
  `results[i].result.info.paths` はその config 自身の target のみを持ち、複数 config が
  共有する path は**双方の `SubjectResult` に現れる**（どちらか一方へ帰属させるのは
  「どの config が宣言しているか」の偽り）。union が必要な消費側が `results` を跨いで
  union + dedup する。**テキスト既定出力は従来どおり dedup + sort**（repo の `.gitignore` は
  1 つ）で、テキスト集約 / JSON per-config の非対称を意図的に保つ。
- **`list-generations --all`**: profile ディレクトリ走査で見つかった home mode config ごとに
  `result.info.generations` を持つ（`items: []`・単一実行と同一）。列挙途中の読み取り失敗は
  従来どおりそこで打ち切り、失敗した config の subject がその失敗を持ち、既に列挙済みの
  config の結果は残る。
- **読み取り系 `--all` は打ち切り以降の config を `results[]` に載せない**
  （`list-generations --all` / `gitignore --all`）。両者は最初の失敗で列挙を中断するため、
  `results[]` は「失敗した config まで」で終わり、未到達の config は要素として現れない。
  **`apply --all` は部分失敗でも全 config を続行する**（各 config が独立 atomic なため）ので、
  選択された全 config が必ず `results[]` に載る。この非対称は「読み取り系は打ち切り /
  変更系は続行」という既存の実行方針の反映であり、消費側は **`results[]` の subject 集合が
  対象 config 集合と一致することを前提にしてはならない**（成功時は一致する）。

> **上は原文の写しで、規範は frontmatter が正**。「エラーの層」項のうち、item 起因の
> 失敗を `item.error` に埋めて `results[i].errors[]` へ重複させないこと・トップ
> `errors[]` に載るのが主体列挙前の失敗のみであることは、単一 config 実行と共通の
> 振り分け規則であり REQ-9341fa5d の規範。本 item が規範化するのは `--all` 固有の差分
> （集約エラーをトップへ重ねない）に限る。
>
> 同様に、次の 2 点も本 item の規範ではない。本 item が規範化するのは、それらを前提と
> した `--all` の JSON 側の帰結（`status` が「非 0 ⇔ error」だけを厳守すること・
> JSON では cross-config dedup をしないこと）に限る。
>
> - 終了コード同士の優先度（error(1) > conflict(2) > 0）→ REQ-b7bb09d6
> - `gitignore --all` のテキスト既定出力が dedup + sort すること → REQ-1f128917

`--all` のテキスト側の挙動（辞書順・部分失敗でも続行）は REQ-4cbd9a0d、エラー層の
振り分け規則は REQ-9341fa5d、終了コードの優先度は REQ-b7bb09d6、`gitignore --all` の
テキスト集約は REQ-1f128917 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」→「niface 準拠の `--json`
出力」のサブ項目「`--all` の複数 `SubjectResult`（#164）」とその配下の全項目。

決定の実体は ADR-0043 §7（`--all` の複数 `SubjectResult`）・§6（conflict 1 件でも
`status:error`）。
