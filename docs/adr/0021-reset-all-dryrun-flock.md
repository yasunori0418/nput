# ADR-0021: reset の `--all` 非対応・`--dryrun` 対応・flock / recopy 合成を確定する

- ステータス: 採用
- 日付: 2026-06-14
- 関連: ADR-0006, ADR-0013, ADR-0017, ADR-0018, ADR-0020, `docs/spec.md`, `docs/design.md`
- 改訂対象: ADR-0020 の `reset` / `--recopy` を細目で補完（新規決定の追加であり反転なし）
- 起点: ADR-0020 で `reset` / `apply --recopy` を追加したことで開いた二次的な細目（`--all` 対応・`--dryrun`・flock・合成）

## 背景

ADR-0020 で `nput reset <name> [target...]`（配置物の teardown）と `apply --recopy`（copy の上書き再コピー）を
追加したが、次の二次的細目が未定義だった。

1. **reset の `--all` 対応有無**。`rollback --all` は破壊的 footgun として却下済み（ADR-0018）。reset は copy も消すぶん更に破壊的。
2. **reset の `--dryrun` 対応有無**。reset は copy データ損失を伴い、`apply` は `--dryrun` を持つ。
3. **reset の flock 取得 / `apply --all --recopy` の合成挙動**。

## 決定

### 1. reset は名指し必須（`--all` 非対応）

- `reset` に `--all`（および root モードフィルタ）を提供しない。**`<name>` 必須**とする。
- 全 config の一斉撤去（copy 含む）は誤操作の被害が大きく、「ユーザーが配置を明示的に握る」思想・`rollback --all` 却下
  （ADR-0018）と一貫させる。複数撤去したいなら名指しを複数回行う。
- per-entry 撤去は従来通り `nput reset <name> [target...]` の `target` 列挙で行う（config 内の粒度は保つ）。

### 2. reset は `--dryrun` 対応

- `nput reset <name> [target...] --dryrun` を**副作用ゼロ**の読み取り専用プレビューとする。削除されるべき symlink / copy target を
  表示して exit し、**FS 削除・confirm・flock いずれも行わない**（`apply --dryrun` と同性質・ADR-0006）。
- CI / スクリプトで「何が消えるか」を非対話で確認でき、copy データ損失前の安全プレビューになる。`apply --dryrun` と対称。
- 終了コード: 削除対象の有無に依らず 0（読み取り専用・破壊予告であってエラーではない）。

### 3. reset は apply と同じ blocking flock を取る / `--recopy` は `--all` と合成可

- **reset は解決後 `profileDir` 単位の flock を blocking（`LOCK_EX`）で取得**する（明示 apply と同じ・ADR-0013）。reset は FS を
  変えるため、同一 config への並行 apply / reset と直列化する。`--dryrun` は読み取り専用なので flock を取らない（ADR-0006）。
- **`apply --all --recopy` は合成可**。`--all`（必要なら `--project-root` 等フィルタ）が選んだ各 config に対し、それぞれ
  `--recopy`（その config の全 copy target を上書き）を適用する。`--recopy` は apply の修飾なので `--all` と直交する。

## 根拠

- **reset 名指し必須**は最も破壊的な一括操作を構造的に禁じる。rollback --all 却下（ADR-0018）と同じ論理で、reset は copy 削除を
  伴うぶん一層 `--all` を避けるべき。
- **reset --dryrun**は破壊操作に対する事前可視化を与える。confirm プロンプトも一覧を出すが、`--dryrun` は非対話 exit で CI / 自動化に
  使え、apply との対称性も保つ。
- **reset の blocking flock**は FS 変更を伴う操作として apply と同じ直列化が必要（並行で apply と競うと配置と撤去が交錯する）。
- **recopy × --all 合成**は `--recopy` が apply 修飾である以上自然。特別扱いせず直交させる。

## 影響

- **`docs/spec.md`**:
  - サブコマンド体系 / reset 節に「名指し必須（`--all` 非対応）」「`--dryrun` 対応（読み取り専用・confirm/flock なし）」を明記。
  - reset 節に「blocking flock を取得（`--dryrun` は取らない）」、`--recopy` 節に「`--all` と合成可」を追記。
  - エラー仕様 / フロー記述で reset の flock・dryrun を反映。
- **`docs/design.md`**: CLI 一覧の reset に `--dryrun` / 名指し必須を反映（任意）。
- **実装フェーズ**: `cmd/nput`（reset の `--all` 拒否・`--dryrun` 分岐・flock 取得、`apply` の `--all`×`--recopy` 合成）。

## 棄却した代替案

- **reset `--all` 対応（`--yes` 必須 / confirm のみ）**: 一括撤去は便利だが破壊的 footgun で rollback --all 却下と非一貫。
- **reset `--dryrun` 非対応（confirm で代用）**: 対話プレビューは得られるが CI / 非対話での事前確認ができず apply と非対称。
- **reset を flock なし**: 並行 apply と交錯し配置 / 撤去が競合し得る。FS 変更操作は直列化すべき。
