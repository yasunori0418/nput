# nput 仕様書

nput が「何を満たすべきか」の全体像と、個別仕様（requirement item）・品質方針
（quality item）・テスト計画（test_plan item）への索引。

規範的な仕様は **すべて `docs/requirements/` の item が持ち**、開発プロセスと規約に関する
方針は `docs/quality/` の item が、何をどこまでテストするかは `docs/test-plan/` の item が
持つ。本文書は通読の入口として全体像だけを述べ、詳細は item へのリンクで示す
（README → 本文書 → item の 3 層構造）。

> **この文書の書き方（規約）**
>
> - 散文は **見出し 1 つ（h2 / h3 の最も内側）につき 10 行以内**。表・リンク列挙・コード
>   ブロックはこの制限に含めない
> - **詳細は必ず item へリンクし、本文に書き下さない**。仕様の断片をここへ写すと item と
>   二重管理になり、改訂時に片方だけ古くなる
> - requirement / quality / test_plan item を足したら本文書のリンク集にも足す。逆に本文書へ
>   仕様を書き足さない
> - item を横断して検索・追跡するには `sara query`（→ `docs/agents/domain.md`）を使う
>
> **item の `## 出典` は概要文書の現行章立てを指さない。** 出典が記録するのは分割時点
> （epic #203）の原文の記述箇所であり、節見出しに限らず表の行・blockquote・箇条書きの項も
> 指す。縮退で章立てを再編したため、出典の節名の多くは現行の見出しと一致しない。item から
> 現行仕様へ辿るのは出典ではなく relation（`sara query`）の役割。
>
> 原文を読むときは、**出典が名指しする文書**（`docs/spec.md` / `docs/design.md` /
> `docs/concept.md` のいずれか。以下 `<doc>`）の縮退コミットの親を見る。3 文書とも同じ形の
> subject（`docs(<scope>): <doc> を概要 + item リンク集へ縮退する`）を持つ。
>
> ```bash
> git log --oneline -- <doc>        # 「概要 + item リンク集へ縮退する」の <ref> を探す
> git show <ref>^:<doc>             # その親が縮退前の原文
> ```

---

## アーキテクチャ概要

nput は **CLI とエンジンの 2 層**で構成する（→ ADR-0006, ADR-0007）。CLI は entrypoint
（`flake.nix` / `shell.nix` / `default.nix`）を発見して `nix build` / `nix eval` を回し、得た
manifest をエンジンへ渡す。エンジンは `manifest.json` を唯一の入力とし、ネイティブ FS 操作で
配置・stale 除去・profile swap を行う。層の境界は `manifest.json` だけ。

- [REQ-f4d7d4ab](requirements/20260802-f4d7d4ab-two-layer-architecture.md) — nput は CLI とエンジンの 2 層で構成する
- [REQ-1767b250](requirements/20260802-1767b250-config-written-in-nix.md) — config は Nix で書き nix build で評価する
- [REQ-6c4e174a](requirements/20260802-6c4e174a-engine-external-command-constraint.md) — engine が叩く外部コマンドは nix と git のみに限る

---

## lib API

`lib`（`mkManifest` / マーカー群）は nixpkgs.lib のみに依存する純データ生成器で、配置ロジックは
持たない（→ ADR-0006）。入力検査は `mkManifest` 自身が `evalModules` + `normalizeManifest` で
行う単一ゲートに集約する。

- [REQ-d85f0cef](requirements/20260802-d85f0cef-lib-pure-data-generator.md) — lib は nixpkgs.lib のみに依存する純データ生成器である

### `lib.mkManifest`

- [REQ-2b0c2bb8](requirements/20260802-2b0c2bb8-mkmanifest-pure-function.md) — mkManifest は配置データを生成する純粋関数である
- [REQ-97c1e088](requirements/20260802-97c1e088-mkmanifest-arguments.md) — mkManifest の引数は pkgs / entries / root の 3 つとする
- [REQ-60e6b49c](requirements/20260802-60e6b49c-mkmanifest-return-value.md) — mkManifest は manifest.json と symlink farm を含む store オブジェクトを返す
- [REQ-2f9205ee](requirements/20260802-2f9205ee-mkmanifest-passthru-rootkind.md) — mkManifest の返り値は passthru で root kind を露出する
- [REQ-cb77ea05](requirements/20260802-cb77ea05-entry-identity-by-target-key.md) — entry の識別子は属性キー = target とし一意性は Nix が担保する
- [REQ-4ec3accc](requirements/20260802-4ec3accc-root-explicit-required.md) — root は明示必須で暗黙デフォルトを持たない
- [REQ-37b56673](requirements/20260802-37b56673-root-marker-kinds.md) — root は 3 マーカーと絶対パス文字列の union を取る

### マーカー構築子

- [REQ-eb363122](requirements/20260802-eb363122-mkoutofstoresymlink-marker.md) — mkOutOfStoreSymlink は out-of-store symlink を表すマーカーを返す
- [REQ-3f541d39](requirements/20260802-3f541d39-root-markers-runtime-resolution.md) — root マーカーは kind を運ぶ入れ物でパス解決は engine が行う

### 入力検査（`evalModules` + `normalizeManifest`）

- [REQ-d1b5b3f5](requirements/20260802-d1b5b3f5-mkmanifest-input-validation.md) — mkManifest 自身が evalModules で入力を検査する単一ゲートになる
- [REQ-b232ec98](requirements/20260802-b232ec98-normalize-manifest-defaults.md) — normalizeManifest が検査・デフォルト適用・marker 変換を行い mkManifest が derivation を組む
- [REQ-6911eab6](requirements/20260802-6911eab6-path-safety-validation.md) — target / subpath のパス安全性を評価時に検査する
- [REQ-5c6b07da](requirements/20260802-5c6b07da-target-collision-eval-detection.md) — target 衝突の検出経路を同一 manifest 内と cross-config で分ける
- [REQ-16faf428](requirements/20260802-16faf428-normalize-manifest-cross-field-checks.md) — 意図が矛盾する組み合わせをクロスフィールドチェックで評価時に拒否する
- [REQ-1dcc9a33](requirements/20260802-1dcc9a33-marker-tag-discrimination.md) — marker は判別タグで識別し manifest.json には漏らさない

---

## CLI 仕様（一次 UX）

`nput` CLI は PATH に常駐する一次 UX（→ ADR-0007）。entrypoint を発見し、rootKind を先取り
eval して root を解決し、flock を取ってから `nix build` をロック内で回してエンジンを駆動する。

- [REQ-14f0aec9](requirements/20260802-14f0aec9-cli-primary-ux-installation.md) — nput CLI は PATH 常駐の一次 UX で、project mode は devShell 同梱を canonical とする
- [REQ-f9920c87](requirements/20260802-f9920c87-nix-experimental-features-prerequisite.md) — nix experimental-features は前提条件とし、CLI は自動付与せず案内エラーで停止する

### entrypoint の発見・アドレッシング

- [REQ-1cc080f6](requirements/20260802-1cc080f6-entrypoint-discovery-order.md) — entrypoint は CWD で flake.nix → shell.nix → default.nix の順に探し -f で上書きする
- [REQ-496b1a07](requirements/20260802-496b1a07-named-manifest-addressing.md) — entrypoint は nput.\<name\> に named manifest を公開し CLI は形ごとの attr path で build する
- [REQ-205d744d](requirements/20260802-205d744d-default-config-name-and-namespace.md) — config 名 default を慣例の解決先とし専用 nput 名前空間で packages を汚さない
- [REQ-c50df875](requirements/20260802-c50df875-flake-parts-module-path.md) — flake-parts 経路は直書きと同一の derivation を生み CLI のアドレッシングを変えない
- [REQ-c890ce4a](requirements/20260802-c890ce4a-legacy-entrypoint-passthru-canonical.md) — legacy entrypoint は mkShell passthru 形を canonical とし CLI の attr path を分岐させない
- [REQ-da253cab](requirements/20260802-da253cab-legacy-src-not-auto-stored.md) — legacy entrypoint では相対 path の src が自動で store 化されない

### サブコマンド体系

- [REQ-c2d44626](requirements/20260802-c2d44626-apply-config-selection.md) — apply の config 選択は name 省略で default・明示で単一・--all で全件
- [REQ-4cbd9a0d](requirements/20260802-4cbd9a0d-apply-all-lexical-order-continue.md) — apply --all は辞書順に適用し部分失敗でも続行して最後に集約する
- [REQ-d95b814f](requirements/20260802-d95b814f-apply-all-root-mode-filter.md) — --all は root モードフィルタで対象 config を絞れる
- [REQ-687e225f](requirements/20260802-687e225f-apply-modifier-flag-composition.md) — apply 修飾フラグは --all と合成できる
- [REQ-02a33511](requirements/20260802-02a33511-apply-dryrun-readonly.md) — apply --dryrun は読み取り専用で conflict 検出時に非ゼロ終了する
- [REQ-7cc32a2b](requirements/20260802-7cc32a2b-apply-recopy.md) — apply --recopy は config 内の全 copy target を src から無条件に上書き再コピーする
- [REQ-5dd5a4e9](requirements/20260802-5dd5a4e9-apply-backup-rename-aside.md) — apply --backup は配置を塞ぐ記録外実体を rename 退避してから配置する
- [REQ-dec58330](requirements/20260802-dec58330-apply-manifest-direct.md) — apply --manifest はビルド済み link-farm を engine へ直接適用する
- [REQ-31f2882e](requirements/20260802-31f2882e-reset-fs-only-teardown.md) — reset は profile を触らない FS-only teardown で配置物を無い状態へ戻す
- [REQ-a8edc58f](requirements/20260802-a8edc58f-reset-named-only-and-flock.md) — reset は名指し必須で profileDir 単位の blocking flock を取る
- [REQ-6a950d6d](requirements/20260802-6a950d6d-reset-dryrun-no-side-effect.md) — reset --dryrun は副作用ゼロで削除対象を表示して終了する
- [REQ-05abce3e](requirements/20260802-05abce3e-rollback-list-generations-home-only.md) — rollback と list-generations は home mode 限定にする
- [REQ-89c7baf9](requirements/20260802-89c7baf9-rollback-named-only.md) — rollback は名指し必須で --all に対応しない
- [REQ-a480c183](requirements/20260802-a480c183-gitignore-anchored-output.md) — gitignore は配置 target を stdout へ列挙するだけでファイルを書き込まない
- [REQ-eaa8c0df](requirements/20260802-eaa8c0df-gitignore-project-mode-only.md) — gitignore は project mode 限定で非 project config を指定したらエラーで停止する
- [REQ-60787ed2](requirements/20260802-60787ed2-gitignore-includes-copy-targets.md) — gitignore は method を区別せず copy target も含めて全 target を列挙する
- [REQ-1f128917](requirements/20260802-1f128917-gitignore-all-dedup.md) — gitignore --all は projectRoot の全 config の target をソート + 重複除去して出力する
- [REQ-6be1cbf1](requirements/20260802-6be1cbf1-init-template-wrapper.md) — nput init は nix flake init -t への透明なラッパーとしファイルを生成しない
- [REQ-cbd61281](requirements/20260802-cbd61281-init-fixed-flake-ref.md) — init のテンプレート参照はバイナリにハードコードした固定 flake ref とする
- [REQ-196ddabf](requirements/20260802-196ddabf-init-template-contents.md) — template は動く example を 1 config だけ置きバリエーションはコメントで示す
- [REQ-61c05e09](requirements/20260802-61c05e09-root-override-flag.md) — --root は全モード共通で解決 root を明示上書きする
- [REQ-1c1526b1](requirements/20260802-1c1526b1-flock-acquisition-mode.md) — flock の取得は既定 blocking とし --no-wait のときだけ try-lock でスキップする
- [REQ-9ed6b500](requirements/20260802-9ed6b500-version-flag.md) — --version は埋め込みバージョンを cobra 既定書式で表示して終了し短縮形を持たない
- [REQ-4fc98fa6](requirements/20260802-4fc98fa6-no-only-flag.md) — 一部 entry だけを適用する --only は提供しない
- [REQ-4ffda99a](requirements/20260802-4ffda99a-nix-command-transparency-and-delegation.md) — 内部実行する nix コマンドを開示し世代の切替と GC は標準の nix コマンドへ委譲する

### 出力ストリームと終了コード

- [REQ-fea038de](requirements/20260802-fea038de-stream-discipline.md) — stdout は機械可読出力を専有しレポートと warning は stderr へ出す
- [REQ-8ef34101](requirements/20260802-8ef34101-silent-on-success.md) — 成功時はデフォルト沈黙とし warning と error は常時 stderr へ出す
- [REQ-0a123b89](requirements/20260802-0a123b89-verbose-and-debug-separation.md) — 冗長度は -v、デバッグは --debug に分離し --json と直交させる
- [REQ-2c5a10d8](requirements/20260802-2c5a10d8-exit-codes.md) — 終了コードは 0 = 成功 / 1 = 一般エラー / 2 = dryrun の conflict とする
- [REQ-b7bb09d6](requirements/20260802-b7bb09d6-apply-all-dryrun-exit-code-priority.md) — apply --all --dryrun の終了コードは error を conflict より優先する

#### niface 準拠の `--json`（第 2 契約・→ ADR-0043）

- [REQ-a5053191](requirements/20260802-a5053191-json-niface-envelope.md) — --json は niface 規約準拠のエンベロープを出す第 2 契約とする
- [REQ-2353259f](requirements/20260802-2353259f-json-stdout-exclusive.md) — --json 指定時は行指向 stdout を出さずエンベロープが stdout を専有する
- [REQ-5c2e64c3](requirements/20260802-5c2e64c3-json-emit-timing.md) — エンベロープはコマンド完了時に 1 回だけ出し成立条件を満たさない実行では出さない
- [REQ-2ea19863](requirements/20260802-2ea19863-json-change-payload.md) — 変更系の JSON ペイロードは engine 結果からフルインベントリと実差分を導く
- [REQ-fa181aa6](requirements/20260802-fa181aa6-json-readonly-payload.md) — 読み取り系の JSON ペイロードは dryRun パリティと info インベントリで表す
- [REQ-059eb4d5](requirements/20260802-059eb4d5-json-all-subject-results.md) — --all は config ごとの SubjectResult を単一実行と同一形状で積む
- [REQ-9341fa5d](requirements/20260802-9341fa5d-json-error-layering.md) — エンベロープのエラーは主体の有無で層を分けコードを分類する
- [REQ-57137302](requirements/20260802-57137302-json-item-id-derivation.md) — item id は identity の JCS を SHA-256 した小文字 hex とする
- [REQ-2a613337](requirements/20260802-2a613337-json-reset-requires-yes.md) — reset --json は --yes を必須とし無ければ fail fast する

### 実行フロー

- [REQ-60c6b7ea](requirements/20260802-60c6b7ea-exec-flow-eval-lock-build.md) — 実行フローの順序は eval 先行 → flock → build とし build をロック内に閉じる
- [REQ-9c111c32](requirements/20260802-9c111c32-non-build-commands-eval-first.md) — 非 build コマンドも eval 先行を共通前段に持つ
- [REQ-535b811d](requirements/20260802-535b811d-apply-all-batched-eval.md) — apply --all は rootKind を 1 回の一括 eval で取る
- [REQ-7a71a049](requirements/20260802-7a71a049-dryrun-no-lock-no-gcroot.md) — --dryrun は root を解決するが flock も pending gcroot も取らない
- [REQ-840b3641](requirements/20260802-840b3641-pending-gcroot-bounded.md) — 失敗時に残る pending gcroot は config あたり最大 1 個に有界とし回収処理を持たない

### 再現性スタンス

- [REQ-67095391](requirements/20260802-67095391-reproducibility-stance.md) — flake は pure eval で flake.lock が固定し legacy は impure を許容しユーザー責任とする
- [REQ-d0aef5af](requirements/20260802-d0aef5af-flake-check-unknown-output.md) — nput カスタム output は nix flake check の unknown 警告を許容し主検証は nix build で行う

---

## entries スキーマ仕様

`entries` は **属性キー = target の attrset**で、キーが識別子になる（→ ADR-0014）。

- [REQ-a33a11e3](requirements/20260802-a33a11e3-entry-submodule-fields.md) — entry submodule のフィールドは src / subpath / target / method の 4 つとする
- [REQ-3e446ad9](requirements/20260802-3e446ad9-entry-submodule-strict.md) — entry submodule は strict とし未知キーと旧名を評価時エラーにする
- [REQ-99ca5381](requirements/20260802-99ca5381-src-kinds-store-default.md) — src は path / set / marker の 3 種を取り store link を既定として out-of-store は marker で opt-in する
- [REQ-27b75fe6](requirements/20260802-27b75fe6-subpath-omission-whole-repo.md) — subpath は src 内の相対パスとし、リポジトリ全体は省略で表して専用トークンを設けない
- [REQ-77689c68](requirements/20260802-77689c68-method-src-matrix.md) — method は配置方法を選び symlink は世代管理下・copy は世代管理外になる

---

## manifest.json スキーマ（v1・Nix↔Go 契約）

`manifest.json` は Nix とエンジンの唯一の安定契約（→ ADR-0010, ADR-0013）。

- [REQ-79ce0a09](requirements/20260802-79ce0a09-manifest-single-stable-contract.md) — manifest.json が Nix と engine の唯一の安定契約であり schemaVersion は 1 に固定する
- [REQ-250d936c](requirements/20260802-250d936c-manifest-v1-only-no-migration.md) — MVP は manifest v1 のみを発行・受理しマイグレーション機構を持たない
- [REQ-dedd2c28](requirements/20260802-dedd2c28-manifest-toplevel-fields.md) — manifest.json のトップレベルは schemaVersion / root / entries の 3 フィールドとする
- [REQ-dd10d820](requirements/20260802-dd10d820-manifest-root-object.md) — manifest.json の root は rootKind を持ち fixed のときだけ絶対パスを併記する
- [REQ-0b0cd1e3](requirements/20260802-0b0cd1e3-manifest-entries-array.md) — manifest.json の entries は attrset を配列へ正規化し 5 フィールドを記録する
- [REQ-b12fc3c0](requirements/20260802-b12fc3c0-farm-gc-anchor-only.md) — symlink farm は GC アンカー専用でアンカーは store-backed な symlink entry に限る
- [REQ-62eda895](requirements/20260802-62eda895-farm-anchor-name-hash.md) — symlink farm の GC アンカー名は target のハッシュとする

---

## 配置動作仕様

symlink は配置前除去 → 配置 → stale 除去の順で進み、途中失敗はインメモリ undo ジャーナルで
巻き戻す（→ ADR-0044, ADR-0046, ADR-0047）。copy は place-once でユーザー管理に委ねる。

### symlink モード

- [REQ-622787dc](requirements/20260802-622787dc-symlink-placement-procedure.md) — symlink 配置は親 dir を作り配置元/subpath を指すリンクを張り、foreign symlink は警告して後勝ちする
- [REQ-61856da1](requirements/20260802-61856da1-symlink-replace-unlink-symlink.md) — 既存 symlink の張替えは unlink + symlink の 2 操作で行い冪等な再実行で収束させる
- [REQ-053cfed2](requirements/20260802-053cfed2-conflict-on-real-file-or-dir.md) — target に通常ファイル・ディレクトリが在れば上書きせずエラーで停止する
- [REQ-c9ab91c1](requirements/20260802-c9ab91c1-ancestor-symlink-nest-migration.md) — 祖先 symlink は自己記録 stale のみ配置前除去し、それ以外はエラーで停止する
- [REQ-7cee95dd](requirements/20260802-7cee95dd-real-dir-target-migration.md) — 実 dir の target は全 leaf が除去可能なときだけ全体を配置前除去して symlink 化する
- [REQ-2b48620a](requirements/20260802-2b48620a-method-change-symlink-to-copy-migration.md) — method 変更は symlink→copy のみ配置前除去で移行し、copy→symlink は移行しない
- [REQ-9b0046e0](requirements/20260802-9b0046e0-backup-stage-position.md) — backup 退避は配置前除去の後・配置の前に置き、drift 修復経路でも同じく実施する

### 途中失敗時の巻き戻し

- [REQ-5e75aabc](requirements/20260802-5e75aabc-undo-journal-rollback.md) — 途中失敗した apply / rollback はインメモリ undo ジャーナルで全 FS 変更を巻き戻す
- [REQ-9fca28c9](requirements/20260802-9fca28c9-undo-rollback-best-effort.md) — 巻き戻し自体の失敗は best-effort で続行し、全件を stderr へ報告して停止する

### copy モード・out-of-store・recopy / reset

- [REQ-d2277c7a](requirements/20260802-d2277c7a-copy-place-once.md) — copy は target 不在のときだけマテリアライズする place-once で世代管理の対象外とする
- [REQ-07c3b735](requirements/20260802-07c3b735-copy-foreign-skip-warning.md) — copy が foreign 実ファイルを skip したときは warning で可視化する
- [REQ-84e3c717](requirements/20260802-84e3c717-copy-mode-owner-write.md) — copy は元の mode を保存しつつ owner-write を付与する
- [REQ-0bd55dfc](requirements/20260802-0bd55dfc-copy-preserves-inner-symlinks.md) — copy は src ツリー内の symlink を deref せず symlink のまま複製する
- [REQ-a8a923ad](requirements/20260802-a8a923ad-out-of-store-link-mapping-only.md) — out-of-store symlink は marker の絶対パスを指し、版管理はリンク先マッピングのみとする
- [REQ-b4e4b65d](requirements/20260802-b4e4b65d-recopy-rename-aside.md) — recopy の上書きは削除ではなく同一親内への rename 退避で行う
- [REQ-31dae599](requirements/20260802-31dae599-reset-confirm-non-tty.md) — reset の確認プロンプトは stdin が TTY のときだけ出し、非 TTY で同意が無ければ即エラー停止する

---

## 世代管理仕様

世代は link farm derivation を nput 自前の profile へコミットして積む（→ ADR-0002, ADR-0025）。
stale 除去は前世代 manifest の記録通りを指す symlink だけに限る保守的な操作。

- [REQ-1be4d678](requirements/20260802-1be4d678-generation-mechanism-link-farm-manifest.md) — 世代は link farm derivation を nput 自前 profile へコミットして積み、前世代 manifest から stale を除去する
- [REQ-2aa3abbc](requirements/20260802-2aa3abbc-profile-on-disk-layout.md) — profileDir は config 専用ディレクトリとし、profile リンク・世代・pending out-link をその中に並べる
- [REQ-16aef46b](requirements/20260802-16aef46b-stale-removal-invariants.md) — stale 除去は前世代の記録通りを指す symlink のみに限り、copy は消さず orphan を警告する
- [REQ-8409db86](requirements/20260802-8409db86-empty-parent-dir-pruning.md) — target 除去後は空の親ディレクトリチェーンを root 境界まで保守的に剪定する
- [REQ-706de717](requirements/20260802-706de717-generation-gc.md) — 世代操作は nix-env --profile 系で統一し、GC root の間引きと store 解放を分けて行う
- [REQ-0e341430](requirements/20260802-0e341430-rollback-refit-then-pointer.md) — rollback は FS を先に収束させてから profile ポインタを最後に移す
- [REQ-844ee375](requirements/20260802-844ee375-module-mode-profile-internal.md) — module 時は rollback を host へ一本化し、nput profile は前進のみで追従する
- [REQ-46fccb80](requirements/20260802-46fccb80-project-mode-generation-skip.md) — project mode は世代を非公開にし、derivation 同一なら世代を積まず lstat ドリフト修復だけ行う
- [REQ-d41b1d0a](requirements/20260802-d41b1d0a-project-mode-orphan-profile.md) — 孤児 profile は backref で逆引き可能なまま放置許容とし、MVP では cleanup コマンドを持たない
- [REQ-fc1118b1](requirements/20260802-fc1118b1-cross-config-target-oscillation.md) — 同一 target を複数 config で狙うことによる振動はユーザー責任とし warning で可視化するに留める

---

## root の解決

root は評価時にパスへ展開せず、マーカーが運ぶ kind をエンジンが実行時に解決する
（→ ADR-0005, ADR-0007）。

- [REQ-8d965ca2](requirements/20260802-8d965ca2-home-mode-root-resolution.md) — home mode の root は層ごとに定まった供給元から解決する
- [REQ-9cb26ffd](requirements/20260802-9cb26ffd-project-mode-root-resolution.md) — project mode の root は git toplevel から解決し、config 相対も CWD 相対も採らない
- [REQ-e79178f5](requirements/20260802-e79178f5-project-mode-ephemeral.md) — project mode の配置物は ephemeral とし、activation は git 状態に干渉しない
- [REQ-d5a2e289](requirements/20260802-d5a2e289-profiledir-keying.md) — profileDir は home のみ name 直キーとし、fixed root と --root 上書きは roothash でキーする
- [REQ-81249072](requirements/20260802-81249072-out-of-store-eval-time-path.md) — out-of-store のローカルパスは評価時に確定し、target の root 解決には影響しない

---

## モジュールオプション仕様・モジュール別動作仕様

全モジュール（HM / NixOS / nix-darwin）と devShell は、root と activation タイミングを供給して
エンジンを起動する薄い配線に徹する（→ ADR-0003）。ネイティブ機構へは翻訳しない。

- [REQ-fc1c7ce6](requirements/20260802-fc1c7ce6-module-common-options.md) — 全モジュールは共通オプションの同一集合を公開し、entries は configs 経由・root はモジュールが pin する
- [REQ-c6891aeb](requirements/20260802-c6891aeb-hm-named-configs-profiles.md) — HM モジュール経由でも名前つき config ごとに役割分離した独立 profile を取れる
- [REQ-e1e1114b](requirements/20260802-e1e1114b-backup-wiring-layer.md) — nput.backup は engine 起動の配線レイヤーのオプションで manifest には影響しない
- [REQ-c2654ca5](requirements/20260802-c2654ca5-module-user-option.md) — NixOS / nix-darwin モジュールは配置先ユーザーを特定する user オプションを必須で取る
- [REQ-c1b3ca5f](requirements/20260802-c1b3ca5f-modules-are-engine-wiring.md) — 全モジュールと devShell は engine をキックするだけの配線とし、ネイティブ機構へ翻訳しない
- [REQ-8085f194](requirements/20260802-8085f194-hm-activation-contract.md) — home-manager モジュールの engine kick 1 回は activation からビルド済み link-farm を渡し、失敗で switch を止める
- [REQ-c847d1af](requirements/20260803-c847d1af-hm-activation-per-config-kick.md) — HM の activation は configs を辞書順に走査して profile ごとに engine を起動し、部分失敗を最後に集約する
- [REQ-5923ac79](requirements/20260803-5923ac79-hm-configs-target-collision-assertion.md) — 単一 HM config 内の configs 間 target 衝突は eval 時 assertion で停止する
- [REQ-a0bdf6db](requirements/20260802-a0bdf6db-devshell-wiring.md) — devShell は shellHook から engine を起動する配線で、シェル入室のたびに project mode で配置する

---

## エラー仕様

設定の誤りは評価時に、実体の不整合はエンジン実行時に検出する層分けを守る。要求された操作が
成立しないときは暗黙のフォールバックを採らず、エラーで停止する。

- [REQ-c5dfcae6](requirements/20260802-c5dfcae6-error-detection-layering.md) — 設定の誤りは評価時に、実体の不整合は engine 実行時に検出する層分けを守る
- [REQ-774cef80](requirements/20260802-774cef80-cli-stop-conditions.md) — 要求された操作が成立しないときは CLI がエラーで停止し、暗黙のフォールバックを採らない
- [REQ-9dc7dac7](requirements/20260802-9dc7dac7-src-subpath-existence-checks.md) — 配置元の実在は判定できる層で検査し、いずれの層でも停止する
- [REQ-6506bc82](requirements/20260802-6506bc82-project-mode-root-resolution-failure.md) — project mode で git から root を解決できないときは engine 実行時に停止する
- [REQ-fc64de4c](requirements/20260802-fc64de4c-empty-entries-full-clear.md) — 空の entries は正当な全クリアとして扱い、エラーにも警告にもしない
- [REQ-95e97d01](requirements/20260802-95e97d01-conflict-full-report.md) — conflict で停止するときは全件を対処ガイダンス付きで列挙してから 1 本の集約エラーを返す

---

## 依存関係

- [REQ-b74a118a](requirements/20260802-b74a118a-engine-stdlib-only.md) — engine は第三者依存ゼロの stdlib-only とし内部層に閉じる
- [REQ-2bd0d35f](requirements/20260803-2bd0d35f-modules-common-nixpkgs-lib-only.md) — modules/common.nix は nixpkgs.lib のみに依存する
- [REQ-637599dc](requirements/20260802-637599dc-cli-dependency-policy.md) — CLI が持ち込む依存は許可した第三者ライブラリと pin した Go に限り、いずれも固定する

---

## 品質

quality は開発プロセス・規約・ガバナンスに関する方針を持つ。requirement とは別系統で、
use_case を経由せず solution 直下に接続する。方針を実現する仕組みは `docs/infrastructure/` の
item が持ち、`satisfies` で quality へ接続する。

- [QA-030f926b](quality/20260809-030f926b-adr-decisions-recorded-and-connected.md) — 設計判断は ADR に記録し、改訂の書き戻しと item への接続を同じ変更の中で完了させる
- [QA-0949183b](quality/20260808-0949183b-release-driven-by-source-of-truth.md) — リリースはバージョンの一次情報の変更で駆動し、手作業の工程を挟まない
- [QA-0d42104c](quality/20260808-0d42104c-reference-docs-generated-from-source.md) — リファレンスドキュメントは記述対象のソースから生成し、生成物を持たない
- [QA-4a623664](quality/20260809-4a623664-glossary-fixes-canonical-terms.md) — 用語の正名は glossary が固定し、執筆はそれに従う
- [QA-58522afb](quality/20260808-58522afb-prebuilt-artifact-availability.md) — 最新 main のビルド成果物を再ビルドなしに消費できる状態を保つ
- [QA-5ecd74ba](quality/20260809-5ecd74ba-public-docs-english-canonical.md) — 公開ドキュメントは英語を canonical とし、日本語版を対で保守する
- [QA-67715bb3](quality/20260809-67715bb3-normative-items-placement-and-authoring.md) — 規範は 1 ファイル 1 item のグラフが持ち、概要文書は索引に縮退させる
- [QA-6bf957d9](quality/20260808-6bf957d9-document-graph-mechanically-verified.md) — ドキュメントグラフのトレーサビリティは機械的に検証する
- [QA-87b7776a](quality/20260809-87b7776a-defects-in-tracker-graph-holds-norms.md) — 欠陥はトラッカーが持ちドキュメントグラフは規範のみを持つ。起票は分類語彙を経由する
- [QA-8c6767e4](quality/20260809-8c6767e4-measurement-reports-without-gating.md) — 傾向の計測は報告に留め、マージのゲートにしない
- [QA-9b5ccfce](quality/20260809-9b5ccfce-dod-single-machine-parsable-document.md) — 完成の定義は単一の機械パース可能な文書で持ち、項目数を上限で縛る
- [QA-a5f7f088](quality/20260808-a5f7f088-cross-platform-verification-before-merge.md) — マージ前の自動検証を必須にし、プラットフォーム差が効く層は全プラットフォームで通す
- [QA-a92341b9](quality/20260809-a92341b9-formatting-and-static-analysis-automated.md) — コード整形と静的解析を自動検証に載せ、同じ判定を手元でも得られるようにする
- [QA-d028e302](quality/20260809-d028e302-automation-supply-chain-safety.md) — 自動化が取り込む実行物は不変な識別子で固定し、権限は最小に絞り、不正入力では成果物を作る前に失敗する

---

## テスト計画

test_plan は requirement とは別系統で、use_case を経由せず solution 直下に接続する。

- [TP-0734996e](test-plan/20260809-0734996e-hm-module-eval-assert.md) — home-manager モジュールの配線は build sandbox 内の評価アサートで検証する
- [TP-229b69c0](test-plan/20260808-229b69c0-e2e-harness-scope.md) — 非 NixOS で動く主張を実 nix の一気通貫 E2E で検証する
- [TP-36e90d5d](test-plan/20260809-36e90d5d-nix-unit-namaka-split.md) — 評価テストは nix-unit で不変条件を、namaka で manifest 全体のスナップショットを見る
- [TP-403c55c7](test-plan/20260808-403c55c7-lib-internal-test-seam.md) — lib.\_\_internal は private helper のテスト seam として公開する
- [TP-b7f1dc79](test-plan/20260808-b7f1dc79-nixos-vm-test-future.md) — NixOS / nix-darwin モジュール経路の実 activate は E2E ハーネスの対象外とする
- [TP-d3000054](test-plan/20260808-d3000054-json-conformance-verification.md) — エンベロープの niface 適合を Go テストと E2E の両方で検証する
- [TP-d3d06fe4](test-plan/20260809-d3d06fe4-eval-test-double.md) — 評価テストの store-backed な入力は固定 outPath を持つ fake flake-input で与える
- [TP-d7da4065](test-plan/20260809-d7da4065-sara-id-contract-test.md) — sara-id はモデル定義との一致を含めて契約テストで検証する
- [TP-deb05610](test-plan/20260809-deb05610-fault-injection-atomicity.md) — 原子性は実 FS の条件で故障を誘発して不変条件ごとに検証する
- [TP-e7c25263](test-plan/20260809-e7c25263-go-test-layering.md) — エンジンとコマンド層の Go テストは nix を介さない実 FS 統合テストを主戦力とする

---

## 設定の書き方（本文書の対象外）

実際に動く設定例は本文書では扱わない。`templates/`（`nput init` が展開する実物）と README の
コード例を参照する。本文書が扱うのは「何を満たすべきか」であり、書き方の例ではない。

---

## 関連文書

- `README.md` / `README.ja.md` — 3 層構造の最上段（導入と使い方）
- `docs/concept.md` — コンセプト（solution / use_case への索引）
- `docs/design.md` — 設計（design item への索引）
- `docs/infrastructure/` — 品質方針を実現する技術基盤（CI / リリース / 配信）
- `docs/adr/` — 意思決定の記録。requirement / design / quality / test_plan などを `justifies` で裏づける
- `docs/model.yaml` — sara の型定義（item の型・関係・ID 形式）
