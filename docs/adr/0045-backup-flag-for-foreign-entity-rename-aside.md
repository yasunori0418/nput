# ADR-0045: `apply --backup[=suffix]` — 配置を塞ぐ記録外実体の rename 退避

- ステータス: 採用
- 日付: 2026-07-12
- 関連: ADR-0002, ADR-0006, ADR-0015, ADR-0017, ADR-0020, ADR-0044, ADR-0046, ADR-0047, `docs/spec.md`, GitHub Issue #169, #172
- 起点: 旧 epic #167 から統合された epic #172 の grilling セッション（2026-07-11・2026-07-12）で確定

## 背景

nput は配置先に記録外の実体（foreign な通常ファイル・ディレクトリ、copy の構造不一致、method 変更 copy→symlink 等）があると、保守的不変条件（→ ADR-0002, ADR-0006, ADR-0015）に従って **conflict で停止し上書きしない**。これはユーザーデータを誤って失わないための正しい既定だが、実運用では「このファイルは既に見て、上書きして問題ないと分かっている」場面が頻発する（dotfiles を nput 管理へ移行する初回・手動で作った設定ファイルを置き換える等）。現状の唯一の回避策は毎回手動 `mv` してから再実行することで、複数 entry が同時に conflict すると 1 件ずつ「確認 → 手動退避 → 再実行」を繰り返す運用コストが高い。

ADR-0047（PreRemove 一般化）は「自己記録の stale」を自動移行する範囲を広げたが、**foreign（記録外）実体は意図的にスコープ外**のままにした（他人のデータを nput が判定なしに動かすのは危険なため）。`--backup` はこのギャップを埋める、**ユーザーが明示 opt-in したときだけ**foreign 実体を退避する脱出ハッチである。

## 決定

### 1. CLI: `apply --backup[=suffix]`（cobra `NoOptDefVal`・`=` 区切り必須）

`--backup` は値なしで指定すると既定 suffix `nput-backup` を使う。`--backup=bak` のように `=` で明示すればその suffix を使う。cobra の `NoOptDefVal`（optional-value flag）で実装するため、**スペース区切り `--backup bak` は次の位置引数として扱われ suffix にならない**（`bak` は `apply` の `<name>` 引数と誤認されるか、位置引数エラーになる）。これは `NoOptDefVal` フラグの一般的な制約であり、ユーザーには spec で明記して周知する。

退避先は常に `<target>` に `.` を付与した `<target>.<suffix>` で固定する（entry ごとの退避先カスタマイズは提供しない）。

### 2. 適用範囲: 配置を塞ぐ記録外実体全般。祖先 symlink conflict は対象外

`--backup` が退避対象にするのは、実体を持つ既存物のうち以下（全て「記録外 = foreign、または nput 管理下でも安全に自動移行できないと ADR-0047 が判断したケース」）:

- symlink モードの通常ファイル / ディレクトリ conflict（`ConflictForeignEntity`）
- symlink モードで実 dir target の migration が失敗するケース（ADR-0047 の「配下に中身のある foreign が 1 つでもあれば dir 全体を conflict」）— **dir 全体を 1 回の rename で退避**する。ADR-0047 D2 の「部分除去はしない」方針をそのまま踏襲し、退避も全か無かで行う
- copy モードの構造不一致（`ConflictCopyStructureMismatch`）
- copy モードで foreign な実ファイルが place-once の skip 対象になるケース（`WarnCopyForeign`）— 通常は skip + warning だが、`--backup` 有効時は退避してから copy を新規配置する（この場合 warning は出さない。退避自体が可視化を兼ねるため）
- method 変更 copy→symlink（ADR-0047 D5 が「ユーザー編集済み copy データの保護を優先し自動移行しない」と決めたケース）— `--backup` はこの D5 が明示的に残した脱出ハッチ

**祖先 symlink conflict（`ConflictForeignAncestor` / `ConflictSelfContradictoryAncestor`）は `--backup` の対象外のまま**にする。これは「実体の有無」ではなく「ネスト構造がそもそも成立しない」という構造問題であり、退避しても解決しない（祖先を退避すると配下の別 entry のネスト前提が崩れる可能性があり、単純な 1:1 rename では表現できない）。祖先 symlink 問題の解消手段は entry 定義の見直しであって `--backup` の役割ではない。

### 3. 退避先が既存なら conflict で停止（黙って上書きしない）

`<target>.<suffix>` が既に存在する場合（前回の `--backup` 実行の退避物が残っている等）、新しい `ConflictKind`（`ConflictBackupTargetExists`）で conflict にし、ADR-0044 の巻き戻しで apply 全体を pre-apply 状態へ戻す。前回の退避物を黙って上書き・消去しない — `--backup` はユーザーデータを守るための機能であり、その退避物自体を粗雑に扱っては本末転倒になる。

### 4. `--backup` の rename も undo ジャーナル対象（ADR-0044 に統合）

`--backup` の rename 退避は Plan の新しいステージ `Backup`（実行順序: PreRemove → **Backup** → Place / Copies → 除去）として engine が実行し、ADR-0044 の undo ジャーナルにそのまま乗る。apply が後続の段で失敗すれば、退避した実体は自動的に元の場所へ rename back される。

ただし **commit 成功後の扱いは `--recopy` の rename 退避と非対称**にする: `--recopy` の退避物（`asidePath`）は成功時に削除される一時的な overwrite debris だが、`--backup` の退避物は**成功時も削除せず永続的に残す**（§5 参照）。undo journal の内部実装では、この非対称を表現するために `--recopy` と異なる undo kind（`undoRestoreBackup`）を割り当て、`discardJournal`（commit 成功後のクリーンアップ）がこの kind だけをスイープ対象から除外する。

世代スキップ経路（project mode の drift 修復・`repairDrift`）にも `--backup` は適用する。PreRemove と異なり、Backup が発火する条件（target に記録外の実体がある）は manifest / derivation の変化を前提としない（他ツールが shell 再入の間に target へファイルを置く、といった FS 状態変化だけで起こり得る）ため、「derivation 不変なら PreRemove は構造的に空」という ADR-0047 の不変条件を Backup には適用できない。

### 5. `reset` は退避物を復元しない。退避物はユーザー所有物として残置する

`--backup` で退避されたファイルは、以後 nput の管理対象に一切ならない。`nput reset` はそれを検出も復元もしない（reset レポートにも現れない）。復元が必要な場合はユーザーが手動 `mv` する。これは「nput が動かした事実は必ず可視化するが、動かした後の所有権はユーザーに戻す」という単純な線引きで、`--recopy` の rename 退避（apply 内で完結する一時操作）とは性質が異なる。

### 6. 出力規律: 退避発動は warning 級・常時 stderr。`--dryrun --backup` は非 conflict

退避が発生した事実は「ユーザー所有物を動かした」という重い意味を持つため、**既定 silent（→ ADR-0031）の対象にせず、常に stderr へ出す**（`-v` 無しでも出る）。`apply --dryrun --backup` は、`--backup` 無しなら conflict（exit 2）になる箇所を「backup + 配置予定」という非 conflict のプランに変える（exit 0）。退避先が既存の場合は `--dryrun --backup` でも conflict のままである（§3 は dryrun でも計算されるため）。

### 7. JSON: `backedUp` を niface 枠内へ追加

`--json`（→ ADR-0043）の結果ペイロードに、`removed` / `pruned` 等と並ぶ形で `backedUp`（退避した target の配列）を追加する。詳細なフィールド設計は epic #126 の niface 詳細化と合わせて詰める（本 ADR は「追加する」という方針のみ確定し、実装は #126 との seam 調整に委ねる）。

### 8. HM モジュール: `nput.backup.enable` + `nput.backup.suffix`

`nput.backup`（submodule。`enable :: bool`・`suffix :: str`、既定 `nput-backup`）を全モジュール共通オプションに追加する。`enable = true` のとき、activation の `nput apply --manifest <link-farm>` 起動へ `--backup=<suffix>` を付加する。これは**起動配線レイヤーのみの変更**であり、manifest v1 契約（`lib/types.nix` の `entriesType` / `mkManifest` の出力）には一切触れない。`nput.entries` の各 entry に `backup` 相当のオプションは追加しない（`--backup` は apply 全体の modifier であり、entry 単位の粒度は提供しない — CLI と同じ設計）。

## 根拠

- **祖先 symlink conflict を対象外にする**のは、ADR-0015/0046/0047 が「実体を持つ記録外物」と「ネスト構造の破綻」を明確に区別してきた設計を崩さないため。`--backup` は前者（rename で解決できる問題）にのみ効く道具であり、後者に無理に適用すると「動いたように見えて実は別の問題を隠す」footgun になる。
- **dir target の migration 失敗を全体 1 回で退避する**のは、ADR-0047 D2 の「部分除去はしない」という一貫性を保つため。dir 内の一部だけ退避すると、退避後の残骸と新配置後の状態が入り混じり、ユーザーが何が起きたか把握しにくくなる。
- **退避先が既存なら conflict にする**のは、`--backup` が守ろうとしているのがまさに「ユーザーの既存データ」であり、その退避物自体を上書きしては目的と矛盾するため。冪等な繰り返し実行より安全側に倒す。
- **成功時に退避物を残置する（`--recopy` と非対称にする）**のは、`--recopy` の退避が「apply 内で完結する一時的な overwrite 手順」であるのに対し、`--backup` の退避は「ユーザーへの成果物の引き渡し」だから。`--recopy` と同じ扱いで自動削除すると、`--backup` の存在意義（データを失わず確認できる）が消える。
- **世代スキップ経路でも Backup を実行する**のは、target への foreign 実体の出現が config 内容と無関係な FS イベントであり、drift 修復のたびに `--backup` が無効化されると "shell を再入したら安全機構が働かなかった" という驚きを生むため。
- **HM の `nput.backup` を manifest 契約から独立させる**のは、`--backup` が「配置ロジックの一部」ではなく「起動時の modifier」だから。ADR-0026（`--manifest` seam）の設計と同じく、モジュールは engine の CLI 引数を組み立てるだけで、engine 自体の判定ロジックには関与しない。

## 影響

- **`internal/planner/planner.go`**: `Compute` に `Options{Backup bool; Suffix string}` を追加。`Plan` に `Backup []BackupAction` を追加。既存の conflict / skip 発生源（`ConflictForeignEntity` の一部・実 dir migration 失敗・`ConflictCopyStructureMismatch`・`WarnCopyForeign`・method 変更 copy→symlink）を、`Options.Backup` が有効なとき conflict/skip の代わりに `BackupAction` へ振り替える共通ヘルパー（`appendBackupOrConflict` / `appendBackup`）を追加。新しい `ConflictKind`（`ConflictBackupTargetExists`）を追加。祖先 symlink 分岐（`ancestorSymlink` 由来）には手を入れない。
- **`internal/engine/backup.go`**（新設）: `applier.backup(actions []planner.BackupAction) error` — PreRemove の後・Place/materializeCopies の前に実行し、rename 退避を行う。TOCTOU 再検証（rename 直前に退避先の非存在を再確認）を行う。
- **`internal/engine/undo.go`**: 新しい undo kind `undoRestoreBackup` を追加（`undoRestoreRename` と同じ逆操作だが、`discardJournal` のクリーンアップ対象から除外する点が異なる）。
- **`internal/engine/engine.go`**: `Apply` の実行順序に Backup ステージを追加（PreRemove → Backup → Place → Copies → 除去）。`Options` に `Backup bool` / `BackupSuffix string` を追加。`Result` に `BackedUp []string` を追加。`dryRun` にも同じ分類を反映。`repairDrift`（世代スキップ）にも Backup 実行を追加。`conflictGuidance` に `ConflictBackupTargetExists` のガイダンスを追加。
- **`cmd/nput/apply.go`**: `--backup` フラグ（`StringVar` + `NoOptDefVal = "nput-backup"`）を追加し、`cmd.Flags().Changed("backup")` で有効判定する。`printApplyPlan` / `reportResult` に `backup` / `backedUp` 行を追加。
- **`modules/common.nix`**: `nput.backup`（submodule: `enable` / `suffix`）を追加。
- **`modules/home-manager.nix`**: activation スクリプトへ `--backup=<suffix>` を条件付きで付加する配線を追加。
- **`docs/spec.md`**: CLI リファレンス・グローバルフラグ・配置動作仕様（symlink/copy モード）・途中失敗時の巻き戻し・エラー仕様表・conflict の全件報告・モジュールオプション仕様を更新。
- **cross-epic**: 本 ADR は #168（ADR-0044）に完全直列 stack。#126（`--json` epic）とは `backedUp` フィールドの詳細設計で seam 調整が必要（本 ADR は方針のみ確定）。

## 棄却した代替案

- **entry 単位の `backup` オプション（`nput.entries.<key>.backup`）**: 個々の entry ごとに退避可否を制御できて柔軟に見えるが、実運用で「この apply 実行だけ安全側に倒したい」というのはほぼ全 entry 一括の要求であり、entry 単位の粒度は使われない複雑さを追加するだけになる。CLI の `--backup` は apply 全体の modifier として設計されており、HM モジュールもそれをそのまま反映する方が一貫する。
- **祖先 symlink conflict も `--backup` の対象に含める**: 祖先を退避すると配下の複数 entry のネスト前提が同時に崩れる可能性があり、単純な 1 target ↔ 1 backup destination のモデルでは表現しきれない。将来必要になれば別 ADR で設計する。
- **退避先が既存でも上書きする（`--force` 的な二段階オプトイン）**: 「安全のための機能」自身が別のデータ損失経路になるのは本末転倒。二段階オプトインは UX の複雑化に対して得られる利便性が小さい。
- **成功時に退避物を自動削除する（`--recopy` と同じ扱い）**: `--backup` の目的が「安全に確認してから消せるようにする」ことである以上、apply 成功と同時に消えてしまっては目的を果たさない。
- **`.` ではなく別の区切り文字、または dotfile 形式（`.<target>.bak`）にする**: `<target>.<suffix>` は既存の `--recopy` の `asidePath`（`<target>.nput-recopy-aside`）と同じ命名規則で一貫性があり、ユーザーが `ls` で見たときに一目で分かる。dotfile 形式は隠しファイル化してしまい、退避の可視性という本 ADR の目的（§6）と逆行する。
