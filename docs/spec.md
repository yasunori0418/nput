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

- [REQ-f4d7d4ab-fbdb-48c6-b29f-08dd88e72645](requirements/20260802-f4d7d4ab-fbdb-48c6-b29f-08dd88e72645-two-layer-architecture.md) — nput は CLI とエンジンの 2 層で構成する
- [REQ-1767b250-b475-4276-a551-20dc79e75a30](requirements/20260802-1767b250-b475-4276-a551-20dc79e75a30-config-written-in-nix.md) — config は Nix で書き nix build で評価する
- [REQ-6c4e174a-4d16-477a-96ff-17cb4eb5b564](requirements/20260802-6c4e174a-4d16-477a-96ff-17cb4eb5b564-engine-external-command-constraint.md) — engine が叩く外部コマンドは nix と git のみに限る

---

## lib API

`lib`（`mkManifest` / マーカー群）は nixpkgs.lib のみに依存する純データ生成器で、配置ロジックは
持たない（→ ADR-0006）。入力検査は `mkManifest` 自身が `evalModules` + `normalizeManifest` で
行う単一ゲートに集約する。

- [REQ-d85f0cef-0f1e-4897-a841-41b61a8dae51](requirements/20260802-d85f0cef-0f1e-4897-a841-41b61a8dae51-lib-pure-data-generator.md) — lib は nixpkgs.lib のみに依存する純データ生成器である

### `lib.mkManifest`

- [REQ-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4](requirements/20260802-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4-mkmanifest-pure-function.md) — mkManifest は配置データを生成する純粋関数である
- [REQ-97c1e088-a17e-46d9-a9a1-83d1757d0f7d](requirements/20260802-97c1e088-a17e-46d9-a9a1-83d1757d0f7d-mkmanifest-arguments.md) — mkManifest の引数は pkgs / entries / root の 3 つとする
- [REQ-60e6b49c-9ba1-4552-a0ec-d340421ec281](requirements/20260802-60e6b49c-9ba1-4552-a0ec-d340421ec281-mkmanifest-return-value.md) — mkManifest は manifest.json と symlink farm を含む store オブジェクトを返す
- [REQ-2f9205ee-cec5-4072-ac3e-890caae79904](requirements/20260802-2f9205ee-cec5-4072-ac3e-890caae79904-mkmanifest-passthru-rootkind.md) — mkManifest の返り値は passthru で root kind を露出する
- [REQ-cb77ea05-bab8-4ccf-b09e-d23d8f71cdc7](requirements/20260802-cb77ea05-bab8-4ccf-b09e-d23d8f71cdc7-entry-identity-by-target-key.md) — entry の識別子は属性キー = target とし一意性は Nix が担保する
- [REQ-4ec3accc-8bb6-461f-9024-dcf0027849e4](requirements/20260802-4ec3accc-8bb6-461f-9024-dcf0027849e4-root-explicit-required.md) — root は明示必須で暗黙デフォルトを持たない
- [REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66](requirements/20260802-37b56673-6e40-4a1b-a2a7-5d3c084e3e66-root-marker-kinds.md) — root は 3 マーカーと絶対パス文字列の union を取る

### マーカー構築子

- [REQ-eb363122-385a-499c-a074-c95efb949d07](requirements/20260802-eb363122-385a-499c-a074-c95efb949d07-mkoutofstoresymlink-marker.md) — mkOutOfStoreSymlink は out-of-store symlink を表すマーカーを返す
- [REQ-3f541d39-da41-4ef8-858b-707f54cf6a29](requirements/20260802-3f541d39-da41-4ef8-858b-707f54cf6a29-root-markers-runtime-resolution.md) — root マーカーは kind を運ぶ入れ物でパス解決は engine が行う

### 入力検査（`evalModules` + `normalizeManifest`）

- [REQ-d1b5b3f5-10a0-400d-9f03-ba00c63d1c34](requirements/20260802-d1b5b3f5-10a0-400d-9f03-ba00c63d1c34-mkmanifest-input-validation.md) — mkManifest 自身が evalModules で入力を検査する単一ゲートになる
- [REQ-b232ec98-af3b-41f3-a050-29d417322002](requirements/20260802-b232ec98-af3b-41f3-a050-29d417322002-normalize-manifest-defaults.md) — normalizeManifest が検査・デフォルト適用・marker 変換を行い mkManifest が derivation を組む
- [REQ-6911eab6-12b4-457c-9db4-d7430a9e9b3f](requirements/20260802-6911eab6-12b4-457c-9db4-d7430a9e9b3f-path-safety-validation.md) — target / subpath のパス安全性を評価時に検査する
- [REQ-5c6b07da-3d06-414d-8770-4f438234b322](requirements/20260802-5c6b07da-3d06-414d-8770-4f438234b322-target-collision-eval-detection.md) — target 衝突の検出経路を同一 manifest 内と cross-config で分ける
- [REQ-16faf428-77f3-492f-b858-222c5274cbf7](requirements/20260802-16faf428-77f3-492f-b858-222c5274cbf7-normalize-manifest-cross-field-checks.md) — 意図が矛盾する組み合わせをクロスフィールドチェックで評価時に拒否する
- [REQ-1dcc9a33-b0f2-43e0-8310-fc4b19e68fe7](requirements/20260802-1dcc9a33-b0f2-43e0-8310-fc4b19e68fe7-marker-tag-discrimination.md) — marker は判別タグで識別し manifest.json には漏らさない

---

## CLI 仕様（一次 UX）

`nput` CLI は PATH に常駐する一次 UX（→ ADR-0007）。entrypoint を発見し、rootKind を先取り
eval して root を解決し、flock を取ってから `nix build` をロック内で回してエンジンを駆動する。

- [REQ-14f0aec9-abae-4621-82f3-40536a1ad904](requirements/20260802-14f0aec9-abae-4621-82f3-40536a1ad904-cli-primary-ux-installation.md) — nput CLI は PATH 常駐の一次 UX で、project mode は devShell 同梱を canonical とする
- [REQ-f9920c87-8551-4aa3-bf03-26fdf4191ed6](requirements/20260802-f9920c87-8551-4aa3-bf03-26fdf4191ed6-nix-experimental-features-prerequisite.md) — nix experimental-features は前提条件とし、CLI は自動付与せず案内エラーで停止する

### entrypoint の発見・アドレッシング

- [REQ-1cc080f6-ae91-4c1f-973e-b7054cfc0198](requirements/20260802-1cc080f6-ae91-4c1f-973e-b7054cfc0198-entrypoint-discovery-order.md) — entrypoint は CWD で flake.nix → shell.nix → default.nix の順に探し -f で上書きする
- [REQ-496b1a07-5b74-416b-9e5f-3952b4c03737](requirements/20260802-496b1a07-5b74-416b-9e5f-3952b4c03737-named-manifest-addressing.md) — entrypoint は nput.\<name\> に named manifest を公開し CLI は形ごとの attr path で build する
- [REQ-205d744d-5a53-4511-bc09-892ba01d4e6f](requirements/20260802-205d744d-5a53-4511-bc09-892ba01d4e6f-default-config-name-and-namespace.md) — config 名 default を慣例の解決先とし専用 nput 名前空間で packages を汚さない
- [REQ-c50df875-2cb0-4e72-8a21-858359a11cae](requirements/20260802-c50df875-2cb0-4e72-8a21-858359a11cae-flake-parts-module-path.md) — flake-parts 経路は直書きと同一の derivation を生み CLI のアドレッシングを変えない
- [REQ-c890ce4a-6528-4ab3-ac86-23d7aebff7da](requirements/20260802-c890ce4a-6528-4ab3-ac86-23d7aebff7da-legacy-entrypoint-passthru-canonical.md) — legacy entrypoint は mkShell passthru 形を canonical とし CLI の attr path を分岐させない
- [REQ-da253cab-34d4-4d6e-96f0-de99e012b376](requirements/20260802-da253cab-34d4-4d6e-96f0-de99e012b376-legacy-src-not-auto-stored.md) — legacy entrypoint では相対 path の src が自動で store 化されない

### サブコマンド体系

- [REQ-c2d44626-d8f4-446a-a80a-319a500129b4](requirements/20260802-c2d44626-d8f4-446a-a80a-319a500129b4-apply-config-selection.md) — apply の config 選択は name 省略で default・明示で単一・--all で全件
- [REQ-4cbd9a0d-9f94-4747-8881-56020dc6d5af](requirements/20260802-4cbd9a0d-9f94-4747-8881-56020dc6d5af-apply-all-lexical-order-continue.md) — apply --all は辞書順に適用し部分失敗でも続行して最後に集約する
- [REQ-d95b814f-aa7a-470e-9320-c14f9c14da7b](requirements/20260802-d95b814f-aa7a-470e-9320-c14f9c14da7b-apply-all-root-mode-filter.md) — --all は root モードフィルタで対象 config を絞れる
- [REQ-687e225f-5046-46db-88fb-f9e527a1e97a](requirements/20260802-687e225f-5046-46db-88fb-f9e527a1e97a-apply-modifier-flag-composition.md) — apply 修飾フラグは --all と合成できる
- [REQ-02a33511-0941-4813-b289-a05eb8e9aa57](requirements/20260802-02a33511-0941-4813-b289-a05eb8e9aa57-apply-dryrun-readonly.md) — apply --dryrun は読み取り専用で conflict 検出時に非ゼロ終了する
- [REQ-7cc32a2b-eee4-4a29-8dc1-a1dc23e7a065](requirements/20260802-7cc32a2b-eee4-4a29-8dc1-a1dc23e7a065-apply-recopy.md) — apply --recopy は config 内の全 copy target を src から無条件に上書き再コピーする
- [REQ-5dd5a4e9-6162-4fa5-b295-66844f5a4f3b](requirements/20260802-5dd5a4e9-6162-4fa5-b295-66844f5a4f3b-apply-backup-rename-aside.md) — apply --backup は配置を塞ぐ記録外実体を rename 退避してから配置する
- [REQ-dec58330-6dad-47f7-8f56-2402764a89c7](requirements/20260802-dec58330-6dad-47f7-8f56-2402764a89c7-apply-manifest-direct.md) — apply --manifest はビルド済み link-farm を engine へ直接適用する
- [REQ-31f2882e-d2e3-4e3b-b783-feb627d73ac6](requirements/20260802-31f2882e-d2e3-4e3b-b783-feb627d73ac6-reset-fs-only-teardown.md) — reset は profile を触らない FS-only teardown で配置物を無い状態へ戻す
- [REQ-a8edc58f-4adc-4637-b888-ab8ccc7e73e4](requirements/20260802-a8edc58f-4adc-4637-b888-ab8ccc7e73e4-reset-named-only-and-flock.md) — reset は名指し必須で profileDir 単位の blocking flock を取る
- [REQ-6a950d6d-c191-4235-a1b4-73ffc7c2bb38](requirements/20260802-6a950d6d-c191-4235-a1b4-73ffc7c2bb38-reset-dryrun-no-side-effect.md) — reset --dryrun は副作用ゼロで削除対象を表示して終了する
- [REQ-05abce3e-9797-432b-b93f-37c55d09afde](requirements/20260802-05abce3e-9797-432b-b93f-37c55d09afde-rollback-list-generations-home-only.md) — rollback と list-generations は home mode 限定にする
- [REQ-89c7baf9-9be0-417b-bd2d-b2e4edabe796](requirements/20260802-89c7baf9-9be0-417b-bd2d-b2e4edabe796-rollback-named-only.md) — rollback は名指し必須で --all に対応しない
- [REQ-a480c183-40ce-4201-93b5-65a7a59c1b9e](requirements/20260802-a480c183-40ce-4201-93b5-65a7a59c1b9e-gitignore-anchored-output.md) — gitignore は配置 target を stdout へ列挙するだけでファイルを書き込まない
- [REQ-eaa8c0df-af44-4f52-9603-cd2bc22a67e9](requirements/20260802-eaa8c0df-af44-4f52-9603-cd2bc22a67e9-gitignore-project-mode-only.md) — gitignore は project mode 限定で非 project config を指定したらエラーで停止する
- [REQ-60787ed2-4176-4bdd-800f-1600c0315551](requirements/20260802-60787ed2-4176-4bdd-800f-1600c0315551-gitignore-includes-copy-targets.md) — gitignore は method を区別せず copy target も含めて全 target を列挙する
- [REQ-1f128917-4424-4e37-8a88-e0bb23a09da7](requirements/20260802-1f128917-4424-4e37-8a88-e0bb23a09da7-gitignore-all-dedup.md) — gitignore --all は projectRoot の全 config の target をソート + 重複除去して出力する
- [REQ-6be1cbf1-6c6e-498b-8acb-7f4b80037169](requirements/20260802-6be1cbf1-6c6e-498b-8acb-7f4b80037169-init-template-wrapper.md) — nput init は nix flake init -t への透明なラッパーとしファイルを生成しない
- [REQ-cbd61281-64b0-4487-a4b7-ce76e70dc4f9](requirements/20260802-cbd61281-64b0-4487-a4b7-ce76e70dc4f9-init-fixed-flake-ref.md) — init のテンプレート参照はバイナリにハードコードした固定 flake ref とする
- [REQ-196ddabf-6569-4303-942e-050872972501](requirements/20260802-196ddabf-6569-4303-942e-050872972501-init-template-contents.md) — template は動く example を 1 config だけ置きバリエーションはコメントで示す
- [REQ-61c05e09-0bde-4f74-9a96-03185f9df606](requirements/20260802-61c05e09-0bde-4f74-9a96-03185f9df606-root-override-flag.md) — --root は全モード共通で解決 root を明示上書きする
- [REQ-1c1526b1-59e3-4264-bb7c-65a10a4aa461](requirements/20260802-1c1526b1-59e3-4264-bb7c-65a10a4aa461-flock-acquisition-mode.md) — flock の取得は既定 blocking とし --no-wait のときだけ try-lock でスキップする
- [REQ-9ed6b500-a11f-414e-a763-adb47c89f7d4](requirements/20260802-9ed6b500-a11f-414e-a763-adb47c89f7d4-version-flag.md) — --version は埋め込みバージョンを cobra 既定書式で表示して終了し短縮形を持たない
- [REQ-4fc98fa6-fd5f-4f25-8b10-60755bf49bd2](requirements/20260802-4fc98fa6-fd5f-4f25-8b10-60755bf49bd2-no-only-flag.md) — 一部 entry だけを適用する --only は提供しない
- [REQ-4ffda99a-7062-4c00-915f-70b525cb215b](requirements/20260802-4ffda99a-7062-4c00-915f-70b525cb215b-nix-command-transparency-and-delegation.md) — 内部実行する nix コマンドを開示し世代の切替と GC は標準の nix コマンドへ委譲する

### 出力ストリームと終了コード

- [REQ-fea038de-55eb-45ac-87fc-ec3a7287592a](requirements/20260802-fea038de-55eb-45ac-87fc-ec3a7287592a-stream-discipline.md) — stdout は機械可読出力を専有しレポートと warning は stderr へ出す
- [REQ-8ef34101-8150-4124-92d5-94fabe6b5d90](requirements/20260802-8ef34101-8150-4124-92d5-94fabe6b5d90-silent-on-success.md) — 成功時はデフォルト沈黙とし warning と error は常時 stderr へ出す
- [REQ-0a123b89-0399-4f76-b988-56a5f7e0becf](requirements/20260802-0a123b89-0399-4f76-b988-56a5f7e0becf-verbose-and-debug-separation.md) — 冗長度は -v、デバッグは --debug に分離し --json と直交させる
- [REQ-2c5a10d8-112b-4f96-947a-aba7164779c4](requirements/20260802-2c5a10d8-112b-4f96-947a-aba7164779c4-exit-codes.md) — 終了コードは 0 = 成功 / 1 = 一般エラー / 2 = dryrun の conflict とする
- [REQ-b7bb09d6-74c4-44d6-905f-cb5e8383ea32](requirements/20260802-b7bb09d6-74c4-44d6-905f-cb5e8383ea32-apply-all-dryrun-exit-code-priority.md) — apply --all --dryrun の終了コードは error を conflict より優先する

#### niface 準拠の `--json`（第 2 契約・→ ADR-0043）

- [REQ-a5053191-1c6a-449b-9c5e-5ff49dc5aead](requirements/20260802-a5053191-1c6a-449b-9c5e-5ff49dc5aead-json-niface-envelope.md) — --json は niface 規約準拠のエンベロープを出す第 2 契約とする
- [REQ-2353259f-5878-452a-8e11-3445de69abc2](requirements/20260802-2353259f-5878-452a-8e11-3445de69abc2-json-stdout-exclusive.md) — --json 指定時は行指向 stdout を出さずエンベロープが stdout を専有する
- [REQ-5c2e64c3-09a7-4ae8-b60c-4f1ccabce4fd](requirements/20260802-5c2e64c3-09a7-4ae8-b60c-4f1ccabce4fd-json-emit-timing.md) — エンベロープはコマンド完了時に 1 回だけ出し成立条件を満たさない実行では出さない
- [REQ-2ea19863-eaa2-466b-b1ed-3f56f6417c62](requirements/20260802-2ea19863-eaa2-466b-b1ed-3f56f6417c62-json-change-payload.md) — 変更系の JSON ペイロードは engine 結果からフルインベントリと実差分を導く
- [REQ-fa181aa6-29a2-48c3-ae07-cc1b9a3b0303](requirements/20260802-fa181aa6-29a2-48c3-ae07-cc1b9a3b0303-json-readonly-payload.md) — 読み取り系の JSON ペイロードは dryRun パリティと info インベントリで表す
- [REQ-059eb4d5-63fb-4f8e-b705-11b5e2ed4ae5](requirements/20260802-059eb4d5-63fb-4f8e-b705-11b5e2ed4ae5-json-all-subject-results.md) — --all は config ごとの SubjectResult を単一実行と同一形状で積む
- [REQ-9341fa5d-836e-4023-af53-cc7d273438d1](requirements/20260802-9341fa5d-836e-4023-af53-cc7d273438d1-json-error-layering.md) — エンベロープのエラーは主体の有無で層を分けコードを分類する
- [REQ-57137302-de29-4f71-a565-034cd5de080b](requirements/20260802-57137302-de29-4f71-a565-034cd5de080b-json-item-id-derivation.md) — item id は identity の JCS を SHA-256 した小文字 hex とする
- [REQ-2a613337-7646-4ced-8807-e43bca18acf3](requirements/20260802-2a613337-7646-4ced-8807-e43bca18acf3-json-reset-requires-yes.md) — reset --json は --yes を必須とし無ければ fail fast する

### 実行フロー

- [REQ-60c6b7ea-e936-4ce8-bd75-ad35e9c693b9](requirements/20260802-60c6b7ea-e936-4ce8-bd75-ad35e9c693b9-exec-flow-eval-lock-build.md) — 実行フローの順序は eval 先行 → flock → build とし build をロック内に閉じる
- [REQ-9c111c32-8f6c-4eda-859c-dae02c0624fc](requirements/20260802-9c111c32-8f6c-4eda-859c-dae02c0624fc-non-build-commands-eval-first.md) — 非 build コマンドも eval 先行を共通前段に持つ
- [REQ-535b811d-dfc5-4eac-92db-737e70eb5415](requirements/20260802-535b811d-dfc5-4eac-92db-737e70eb5415-apply-all-batched-eval.md) — apply --all は rootKind を 1 回の一括 eval で取る
- [REQ-7a71a049-5876-4cfc-a65e-44e9a0349856](requirements/20260802-7a71a049-5876-4cfc-a65e-44e9a0349856-dryrun-no-lock-no-gcroot.md) — --dryrun は root を解決するが flock も pending gcroot も取らない
- [REQ-840b3641-6e76-46da-82e9-680cabd65abe](requirements/20260802-840b3641-6e76-46da-82e9-680cabd65abe-pending-gcroot-bounded.md) — 失敗時に残る pending gcroot は config あたり最大 1 個に有界とし回収処理を持たない

### 再現性スタンス

- [REQ-67095391-eab2-45d2-b75b-b428d481bcc2](requirements/20260802-67095391-eab2-45d2-b75b-b428d481bcc2-reproducibility-stance.md) — flake は pure eval で flake.lock が固定し legacy は impure を許容しユーザー責任とする
- [REQ-d0aef5af-e922-400b-b250-ca38719c480b](requirements/20260802-d0aef5af-e922-400b-b250-ca38719c480b-flake-check-unknown-output.md) — nput カスタム output は nix flake check の unknown 警告を許容し主検証は nix build で行う

---

## entries スキーマ仕様

`entries` は **属性キー = target の attrset**で、キーが識別子になる（→ ADR-0014）。

- [REQ-a33a11e3-830d-4142-88ed-4c1fc35e7f74](requirements/20260802-a33a11e3-830d-4142-88ed-4c1fc35e7f74-entry-submodule-fields.md) — entry submodule のフィールドは src / subpath / target / method の 4 つとする
- [REQ-3e446ad9-a6f4-4229-b5c5-184754b0ef51](requirements/20260802-3e446ad9-a6f4-4229-b5c5-184754b0ef51-entry-submodule-strict.md) — entry submodule は strict とし未知キーと旧名を評価時エラーにする
- [REQ-99ca5381-6c53-426c-b145-7b4297c53868](requirements/20260802-99ca5381-6c53-426c-b145-7b4297c53868-src-kinds-store-default.md) — src は path / set / marker の 3 種を取り store link を既定として out-of-store は marker で opt-in する
- [REQ-27b75fe6-6c36-44a8-8cd3-5cc98043022a](requirements/20260802-27b75fe6-6c36-44a8-8cd3-5cc98043022a-subpath-omission-whole-repo.md) — subpath は src 内の相対パスとし、リポジトリ全体は省略で表して専用トークンを設けない
- [REQ-77689c68-953c-4cbb-ab31-1ac1e4f5f2fe](requirements/20260802-77689c68-953c-4cbb-ab31-1ac1e4f5f2fe-method-src-matrix.md) — method は配置方法を選び symlink は世代管理下・copy は世代管理外になる

---

## manifest.json スキーマ（v1・Nix↔Go 契約）

`manifest.json` は Nix とエンジンの唯一の安定契約（→ ADR-0010, ADR-0013）。

- [REQ-79ce0a09-f9bd-4e61-ba7f-45fb5643137b](requirements/20260802-79ce0a09-f9bd-4e61-ba7f-45fb5643137b-manifest-single-stable-contract.md) — manifest.json が Nix と engine の唯一の安定契約であり schemaVersion は 1 に固定する
- [REQ-250d936c-1df0-491d-a7af-708f38b61f33](requirements/20260802-250d936c-1df0-491d-a7af-708f38b61f33-manifest-v1-only-no-migration.md) — MVP は manifest v1 のみを発行・受理しマイグレーション機構を持たない
- [REQ-dedd2c28-bba3-4ecf-80c9-8c77347e8e1f](requirements/20260802-dedd2c28-bba3-4ecf-80c9-8c77347e8e1f-manifest-toplevel-fields.md) — manifest.json のトップレベルは schemaVersion / root / entries の 3 フィールドとする
- [REQ-dd10d820-e453-4099-a47a-ffb9a7de02fb](requirements/20260802-dd10d820-e453-4099-a47a-ffb9a7de02fb-manifest-root-object.md) — manifest.json の root は rootKind を持ち fixed のときだけ絶対パスを併記する
- [REQ-0b0cd1e3-bfeb-45c1-978d-e2e11c568336](requirements/20260802-0b0cd1e3-bfeb-45c1-978d-e2e11c568336-manifest-entries-array.md) — manifest.json の entries は attrset を配列へ正規化し 5 フィールドを記録する
- [REQ-b12fc3c0-d7fe-4003-922c-f3ac0d969b66](requirements/20260802-b12fc3c0-d7fe-4003-922c-f3ac0d969b66-farm-gc-anchor-only.md) — symlink farm は GC アンカー専用でアンカーは store-backed な symlink entry に限る
- [REQ-62eda895-efd4-4eaf-a58b-600e8637da75](requirements/20260802-62eda895-efd4-4eaf-a58b-600e8637da75-farm-anchor-name-hash.md) — symlink farm の GC アンカー名は target のハッシュとする

---

## 配置動作仕様

symlink は配置前除去 → 配置 → stale 除去の順で進み、途中失敗はインメモリ undo ジャーナルで
巻き戻す（→ ADR-0044, ADR-0046, ADR-0047）。copy は place-once でユーザー管理に委ねる。

### symlink モード

- [REQ-622787dc-4512-4ce9-9c7d-7b32bbb70557](requirements/20260802-622787dc-4512-4ce9-9c7d-7b32bbb70557-symlink-placement-procedure.md) — symlink 配置は親 dir を作り配置元/subpath を指すリンクを張り、foreign symlink は警告して後勝ちする
- [REQ-61856da1-8883-401e-ad57-9f326b96d400](requirements/20260802-61856da1-8883-401e-ad57-9f326b96d400-symlink-replace-unlink-symlink.md) — 既存 symlink の張替えは unlink + symlink の 2 操作で行い冪等な再実行で収束させる
- [REQ-053cfed2-265a-4997-a27b-97b0bed10d8a](requirements/20260802-053cfed2-265a-4997-a27b-97b0bed10d8a-conflict-on-real-file-or-dir.md) — target に通常ファイル・ディレクトリが在れば上書きせずエラーで停止する
- [REQ-c9ab91c1-f778-4f87-a2ea-c66d6b3c2575](requirements/20260802-c9ab91c1-f778-4f87-a2ea-c66d6b3c2575-ancestor-symlink-nest-migration.md) — 祖先 symlink は自己記録 stale のみ配置前除去し、それ以外はエラーで停止する
- [REQ-7cee95dd-bc5a-4e86-bebc-6080ef78fe26](requirements/20260802-7cee95dd-bc5a-4e86-bebc-6080ef78fe26-real-dir-target-migration.md) — 実 dir の target は全 leaf が除去可能なときだけ全体を配置前除去して symlink 化する
- [REQ-2b48620a-abaa-43df-a106-954bbba3de56](requirements/20260802-2b48620a-abaa-43df-a106-954bbba3de56-method-change-symlink-to-copy-migration.md) — method 変更は symlink→copy のみ配置前除去で移行し、copy→symlink は移行しない
- [REQ-9b0046e0-8ddc-4c0b-940e-3fe6f36d0e98](requirements/20260802-9b0046e0-8ddc-4c0b-940e-3fe6f36d0e98-backup-stage-position.md) — backup 退避は配置前除去の後・配置の前に置き、drift 修復経路でも同じく実施する

### 途中失敗時の巻き戻し

- [REQ-5e75aabc-0e8f-4a6c-92bd-a712dc68a940](requirements/20260802-5e75aabc-0e8f-4a6c-92bd-a712dc68a940-undo-journal-rollback.md) — 途中失敗した apply / rollback はインメモリ undo ジャーナルで全 FS 変更を巻き戻す
- [REQ-9fca28c9-d3b1-4ad7-8f24-13b2ec7aeab2](requirements/20260802-9fca28c9-d3b1-4ad7-8f24-13b2ec7aeab2-undo-rollback-best-effort.md) — 巻き戻し自体の失敗は best-effort で続行し、全件を stderr へ報告して停止する

### copy モード・out-of-store・recopy / reset

- [REQ-d2277c7a-7992-49af-a9dc-4cc73843a6f9](requirements/20260802-d2277c7a-7992-49af-a9dc-4cc73843a6f9-copy-place-once.md) — copy は target 不在のときだけマテリアライズする place-once で世代管理の対象外とする
- [REQ-07c3b735-3744-4778-a640-8c6fb66f4aa7](requirements/20260802-07c3b735-3744-4778-a640-8c6fb66f4aa7-copy-foreign-skip-warning.md) — copy が foreign 実ファイルを skip したときは warning で可視化する
- [REQ-84e3c717-adf5-4ff3-b0db-d039b82ef19c](requirements/20260802-84e3c717-adf5-4ff3-b0db-d039b82ef19c-copy-mode-owner-write.md) — copy は元の mode を保存しつつ owner-write を付与する
- [REQ-0bd55dfc-b28c-4ad2-ad50-bb1f76b2246c](requirements/20260802-0bd55dfc-b28c-4ad2-ad50-bb1f76b2246c-copy-preserves-inner-symlinks.md) — copy は src ツリー内の symlink を deref せず symlink のまま複製する
- [REQ-a8a923ad-07fb-4582-b90a-07a6e0c41baa](requirements/20260802-a8a923ad-07fb-4582-b90a-07a6e0c41baa-out-of-store-link-mapping-only.md) — out-of-store symlink は marker の絶対パスを指し、版管理はリンク先マッピングのみとする
- [REQ-b4e4b65d-6e35-40c3-a00e-20c14043df6f](requirements/20260802-b4e4b65d-6e35-40c3-a00e-20c14043df6f-recopy-rename-aside.md) — recopy の上書きは削除ではなく同一親内への rename 退避で行う
- [REQ-31dae599-f3a3-4bbe-b367-c955535265da](requirements/20260802-31dae599-f3a3-4bbe-b367-c955535265da-reset-confirm-non-tty.md) — reset の確認プロンプトは stdin が TTY のときだけ出し、非 TTY で同意が無ければ即エラー停止する

---

## 世代管理仕様

世代は link farm derivation を nput 自前の profile へコミットして積む（→ ADR-0002, ADR-0025）。
stale 除去は前世代 manifest の記録通りを指す symlink だけに限る保守的な操作。

- [REQ-1be4d678-959c-44d7-a346-44bfd95af56e](requirements/20260802-1be4d678-959c-44d7-a346-44bfd95af56e-generation-mechanism-link-farm-manifest.md) — 世代は link farm derivation を nput 自前 profile へコミットして積み、前世代 manifest から stale を除去する
- [REQ-2aa3abbc-90b2-486e-92de-d785554bdeb3](requirements/20260802-2aa3abbc-90b2-486e-92de-d785554bdeb3-profile-on-disk-layout.md) — profileDir は config 専用ディレクトリとし、profile リンク・世代・pending out-link をその中に並べる
- [REQ-16aef46b-7bb8-4ca1-b962-e9f3ed1fd1d2](requirements/20260802-16aef46b-7bb8-4ca1-b962-e9f3ed1fd1d2-stale-removal-invariants.md) — stale 除去は前世代の記録通りを指す symlink のみに限り、copy は消さず orphan を警告する
- [REQ-8409db86-a1ba-4053-86dc-588985cc1ca7](requirements/20260802-8409db86-a1ba-4053-86dc-588985cc1ca7-empty-parent-dir-pruning.md) — target 除去後は空の親ディレクトリチェーンを root 境界まで保守的に剪定する
- [REQ-706de717-4e47-471a-a1c0-448635be159c](requirements/20260802-706de717-4e47-471a-a1c0-448635be159c-generation-gc.md) — 世代操作は nix-env --profile 系で統一し、GC root の間引きと store 解放を分けて行う
- [REQ-0e341430-17f0-498b-9439-65491652163a](requirements/20260802-0e341430-17f0-498b-9439-65491652163a-rollback-refit-then-pointer.md) — rollback は FS を先に収束させてから profile ポインタを最後に移す
- [REQ-844ee375-919f-4341-81e1-a5f89fd32840](requirements/20260802-844ee375-919f-4341-81e1-a5f89fd32840-module-mode-profile-internal.md) — module 時は rollback を host へ一本化し、nput profile は前進のみで追従する
- [REQ-46fccb80-4bae-4d37-bc19-dded88e9a9c0](requirements/20260802-46fccb80-4bae-4d37-bc19-dded88e9a9c0-project-mode-generation-skip.md) — project mode は世代を非公開にし、derivation 同一なら世代を積まず lstat ドリフト修復だけ行う
- [REQ-d41b1d0a-c6d5-41cc-93f9-e5cc7f152da4](requirements/20260802-d41b1d0a-c6d5-41cc-93f9-e5cc7f152da4-project-mode-orphan-profile.md) — 孤児 profile は backref で逆引き可能なまま放置許容とし、MVP では cleanup コマンドを持たない
- [REQ-fc1118b1-b0e8-4ddf-80f6-c70956651693](requirements/20260802-fc1118b1-b0e8-4ddf-80f6-c70956651693-cross-config-target-oscillation.md) — 同一 target を複数 config で狙うことによる振動はユーザー責任とし warning で可視化するに留める

---

## root の解決

root は評価時にパスへ展開せず、マーカーが運ぶ kind をエンジンが実行時に解決する
（→ ADR-0005, ADR-0007）。

- [REQ-8d965ca2-f8fd-44a4-87f3-94e850e9f85b](requirements/20260802-8d965ca2-f8fd-44a4-87f3-94e850e9f85b-home-mode-root-resolution.md) — home mode の root は層ごとに定まった供給元から解決する
- [REQ-9cb26ffd-071e-4c68-a6fc-faac6373b75e](requirements/20260802-9cb26ffd-071e-4c68-a6fc-faac6373b75e-project-mode-root-resolution.md) — project mode の root は git toplevel から解決し、config 相対も CWD 相対も採らない
- [REQ-e79178f5-5865-4444-a05d-3ab06f33cd6d](requirements/20260802-e79178f5-5865-4444-a05d-3ab06f33cd6d-project-mode-ephemeral.md) — project mode の配置物は ephemeral とし、activation は git 状態に干渉しない
- [REQ-d5a2e289-40bc-45a9-9d44-21b8dc561b81](requirements/20260802-d5a2e289-40bc-45a9-9d44-21b8dc561b81-profiledir-keying.md) — profileDir は home のみ name 直キーとし、fixed root と --root 上書きは roothash でキーする
- [REQ-81249072-56b8-42f5-807b-ea623c1efe99](requirements/20260802-81249072-56b8-42f5-807b-ea623c1efe99-out-of-store-eval-time-path.md) — out-of-store のローカルパスは評価時に確定し、target の root 解決には影響しない

---

## モジュールオプション仕様・モジュール別動作仕様

全モジュール（HM / NixOS / nix-darwin）と devShell は、root と activation タイミングを供給して
エンジンを起動する薄い配線に徹する（→ ADR-0003）。ネイティブ機構へは翻訳しない。

- [REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10](requirements/20260802-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10-module-common-options.md) — 全モジュールは共通オプションの同一集合を公開し、entries は configs 経由・root はモジュールが pin する
- [REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a](requirements/20260802-c6891aeb-13c0-4ae7-9ad1-5c343735266a-hm-named-configs-profiles.md) — HM モジュール経由でも名前つき config ごとに役割分離した独立 profile を取れる
- [REQ-e1e1114b-ba07-4d57-8e04-6e30e39a5da3](requirements/20260802-e1e1114b-ba07-4d57-8e04-6e30e39a5da3-backup-wiring-layer.md) — nput.backup は engine 起動の配線レイヤーのオプションで manifest には影響しない
- [REQ-c2654ca5-62c2-4e4b-ad67-ffc5468f429b](requirements/20260802-c2654ca5-62c2-4e4b-ad67-ffc5468f429b-module-user-option.md) — NixOS / nix-darwin モジュールは配置先ユーザーを特定する user オプションを必須で取る
- [REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9](requirements/20260802-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9-modules-are-engine-wiring.md) — 全モジュールと devShell は engine をキックするだけの配線とし、ネイティブ機構へ翻訳しない
- [REQ-8085f194-c903-4ecb-abd8-c719fe7b3292](requirements/20260802-8085f194-c903-4ecb-abd8-c719fe7b3292-hm-activation-contract.md) — home-manager モジュールの engine kick 1 回は activation からビルド済み link-farm を渡し、失敗で switch を止める
- [REQ-c847d1af-a437-46bb-bd64-42083810d034](requirements/20260803-c847d1af-a437-46bb-bd64-42083810d034-hm-activation-per-config-kick.md) — HM の activation は configs を辞書順に走査して profile ごとに engine を起動し、部分失敗を最後に集約する
- [REQ-5923ac79-4a2d-43cd-b56c-2f1000c01b44](requirements/20260803-5923ac79-4a2d-43cd-b56c-2f1000c01b44-hm-configs-target-collision-assertion.md) — 単一 HM config 内の configs 間 target 衝突は eval 時 assertion で停止する
- [REQ-a0bdf6db-6c0c-476c-916a-61ee4e4510d9](requirements/20260802-a0bdf6db-6c0c-476c-916a-61ee4e4510d9-devshell-wiring.md) — devShell は shellHook から engine を起動する配線で、シェル入室のたびに project mode で配置する

---

## エラー仕様

設定の誤りは評価時に、実体の不整合はエンジン実行時に検出する層分けを守る。要求された操作が
成立しないときは暗黙のフォールバックを採らず、エラーで停止する。

- [REQ-c5dfcae6-6094-4850-99e5-bf14530bc60a](requirements/20260802-c5dfcae6-6094-4850-99e5-bf14530bc60a-error-detection-layering.md) — 設定の誤りは評価時に、実体の不整合は engine 実行時に検出する層分けを守る
- [REQ-774cef80-2872-4ea1-937b-a0fbabc305a9](requirements/20260802-774cef80-2872-4ea1-937b-a0fbabc305a9-cli-stop-conditions.md) — 要求された操作が成立しないときは CLI がエラーで停止し、暗黙のフォールバックを採らない
- [REQ-9dc7dac7-4e9e-494a-a17b-73853c119653](requirements/20260802-9dc7dac7-4e9e-494a-a17b-73853c119653-src-subpath-existence-checks.md) — 配置元の実在は判定できる層で検査し、いずれの層でも停止する
- [REQ-6506bc82-d1e1-4dbf-8c57-d5d1babf218a](requirements/20260802-6506bc82-d1e1-4dbf-8c57-d5d1babf218a-project-mode-root-resolution-failure.md) — project mode で git から root を解決できないときは engine 実行時に停止する
- [REQ-fc64de4c-c82b-419c-8706-07d8d97daa37](requirements/20260802-fc64de4c-c82b-419c-8706-07d8d97daa37-empty-entries-full-clear.md) — 空の entries は正当な全クリアとして扱い、エラーにも警告にもしない
- [REQ-95e97d01-5c34-44b3-bc85-9ca53472bc3d](requirements/20260802-95e97d01-5c34-44b3-bc85-9ca53472bc3d-conflict-full-report.md) — conflict で停止するときは全件を対処ガイダンス付きで列挙してから 1 本の集約エラーを返す

---

## 依存関係

- [REQ-b74a118a-1272-44eb-944c-7725163211c6](requirements/20260802-b74a118a-1272-44eb-944c-7725163211c6-engine-stdlib-only.md) — engine は第三者依存ゼロの stdlib-only とし内部層に閉じる
- [REQ-2bd0d35f-dc29-4771-9cfe-6998247afa0f](requirements/20260803-2bd0d35f-dc29-4771-9cfe-6998247afa0f-modules-common-nixpkgs-lib-only.md) — modules/common.nix は nixpkgs.lib のみに依存する
- [REQ-637599dc-a1ec-4af5-9e97-e882c7df56d0](requirements/20260802-637599dc-a1ec-4af5-9e97-e882c7df56d0-cli-dependency-policy.md) — CLI が持ち込む依存は許可した第三者ライブラリと pin した Go に限り、いずれも固定する

---

## 品質

quality は開発プロセス・規約・ガバナンスに関する方針を持つ。requirement とは別系統で、
use_case を経由せず solution 直下に接続する。方針を実現する仕組みは `docs/infrastructure/` の
item が持ち、`satisfies` で quality へ接続する。

- [QA-030f926b-5ae7-4543-b8cb-e66aae5e2b5e](quality/20260809-030f926b-5ae7-4543-b8cb-e66aae5e2b5e-adr-decisions-recorded-and-connected.md) — 設計判断は ADR に記録し、改訂の書き戻しと item への接続を同じ変更の中で完了させる
- [QA-0949183b-7ef0-4cae-b88f-3ad361576b63](quality/20260808-0949183b-7ef0-4cae-b88f-3ad361576b63-release-driven-by-source-of-truth.md) — リリースはバージョンの一次情報の変更で駆動し、手作業の工程を挟まない
- [QA-0d42104c-a3c2-4bd1-9cad-a42d9e5a56a1](quality/20260808-0d42104c-a3c2-4bd1-9cad-a42d9e5a56a1-reference-docs-generated-from-source.md) — リファレンスドキュメントは記述対象のソースから生成し、生成物を持たない
- [QA-4a623664-650d-4a08-800f-691f4ea6ff91](quality/20260809-4a623664-650d-4a08-800f-691f4ea6ff91-glossary-fixes-canonical-terms.md) — 用語の正名は glossary が固定し、執筆はそれに従う
- [QA-58522afb-31d5-4a1f-a7df-0858efa9e44b](quality/20260808-58522afb-31d5-4a1f-a7df-0858efa9e44b-prebuilt-artifact-availability.md) — 最新 main のビルド成果物を再ビルドなしに消費できる状態を保つ
- [QA-5ecd74ba-889a-4e06-b32b-e67f10a45051](quality/20260809-5ecd74ba-889a-4e06-b32b-e67f10a45051-public-docs-english-canonical.md) — 公開ドキュメントは英語を canonical とし、日本語版を対で保守する
- [QA-67715bb3-1162-4ccf-8441-2f44257a57da](quality/20260809-67715bb3-1162-4ccf-8441-2f44257a57da-normative-items-placement-and-authoring.md) — 規範は 1 ファイル 1 item のグラフが持ち、概要文書は索引に縮退させる
- [QA-6bf957d9-17d9-4660-92b7-ebd6eeb71a8c](quality/20260808-6bf957d9-17d9-4660-92b7-ebd6eeb71a8c-document-graph-mechanically-verified.md) — ドキュメントグラフのトレーサビリティは機械的に検証する
- [QA-87b7776a-9ece-42ac-9a56-78daacb42217](quality/20260809-87b7776a-9ece-42ac-9a56-78daacb42217-defects-in-tracker-graph-holds-norms.md) — 欠陥はトラッカーが持ちドキュメントグラフは規範のみを持つ。起票は分類語彙を経由する
- [QA-8c6767e4-dddb-48fb-b010-d363a936e746](quality/20260809-8c6767e4-dddb-48fb-b010-d363a936e746-measurement-reports-without-gating.md) — 傾向の計測は報告に留め、マージのゲートにしない
- [QA-9b5ccfce-25c2-4a90-977f-711ca489e9f4](quality/20260809-9b5ccfce-25c2-4a90-977f-711ca489e9f4-dod-single-machine-parsable-document.md) — 完成の定義は単一の機械パース可能な文書で持ち、項目数を上限で縛る
- [QA-a5f7f088-a459-4bb2-9674-82b1a4a52053](quality/20260808-a5f7f088-a459-4bb2-9674-82b1a4a52053-cross-platform-verification-before-merge.md) — マージ前の自動検証を必須にし、プラットフォーム差が効く層は全プラットフォームで通す
- [QA-a92341b9-c873-406e-8b70-a64f56d8a7d6](quality/20260809-a92341b9-c873-406e-8b70-a64f56d8a7d6-formatting-and-static-analysis-automated.md) — コード整形と静的解析を自動検証に載せ、同じ判定を手元でも得られるようにする
- [QA-d028e302-8262-428c-9030-98d46b4b0cd3](quality/20260809-d028e302-8262-428c-9030-98d46b4b0cd3-automation-supply-chain-safety.md) — 自動化が取り込む実行物は不変な識別子で固定し、権限は最小に絞り、不正入力では成果物を作る前に失敗する

---

## テスト計画

test_plan は requirement とは別系統で、use_case を経由せず solution 直下に接続する。

- [TP-0734996e-aea9-4229-8075-89a64bdf9f79](test-plan/20260809-0734996e-aea9-4229-8075-89a64bdf9f79-hm-module-eval-assert.md) — home-manager モジュールの配線は build sandbox 内の評価アサートで検証する
- [TP-229b69c0-cf5e-4fb6-a353-27e5064d93e9](test-plan/20260808-229b69c0-cf5e-4fb6-a353-27e5064d93e9-e2e-harness-scope.md) — 非 NixOS で動く主張を実 nix の一気通貫 E2E で検証する
- [TP-36e90d5d-4524-4294-bc72-ee263bb02782](test-plan/20260809-36e90d5d-4524-4294-bc72-ee263bb02782-nix-unit-namaka-split.md) — 評価テストは nix-unit で不変条件を、namaka で manifest 全体のスナップショットを見る
- [TP-403c55c7-d996-4951-8e6b-c3a7dddd387c](test-plan/20260808-403c55c7-d996-4951-8e6b-c3a7dddd387c-lib-internal-test-seam.md) — lib.\_\_internal は private helper のテスト seam として公開する
- [TP-b7f1dc79-0222-4b6e-9e91-0545046e34f2](test-plan/20260808-b7f1dc79-0222-4b6e-9e91-0545046e34f2-nixos-vm-test-future.md) — NixOS / nix-darwin モジュール経路の実 activate は E2E ハーネスの対象外とする
- [TP-d3000054-42d9-4bac-912a-dd3abc38d3e9](test-plan/20260808-d3000054-42d9-4bac-912a-dd3abc38d3e9-json-conformance-verification.md) — エンベロープの niface 適合を Go テストと E2E の両方で検証する
- [TP-d3d06fe4-6940-4df8-b111-bb4096d5444f](test-plan/20260809-d3d06fe4-6940-4df8-b111-bb4096d5444f-eval-test-double.md) — 評価テストの store-backed な入力は固定 outPath を持つ fake flake-input で与える
- [TP-deb05610-44bc-4962-8939-952392e5fbd0](test-plan/20260809-deb05610-44bc-4962-8939-952392e5fbd0-fault-injection-atomicity.md) — 原子性は実 FS の条件で故障を誘発して不変条件ごとに検証する
- [TP-e7c25263-6d2d-4a37-8275-26906889d912](test-plan/20260809-e7c25263-6d2d-4a37-8275-26906889d912-go-test-layering.md) — エンジンとコマンド層の Go テストは nix を介さない実 FS 統合テストを主戦力とする

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
