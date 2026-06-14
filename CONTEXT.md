# nput

フェッチ済み git リポジトリを、ユーザー環境の任意パスへ symlink / copy で配置する Nix ライブラリ・モジュール群の用語集。設定生成は行わない。

ここは glossary であり仕様書ではない。実装詳細は `docs/spec.md`、設計判断は `docs/adr/` に置く。2026-06-07 の方針転換（ADR-0001〜0004）で意味が変わった語を中心に、正名と避けるべき同義語を固定する。

## Language

### 配置の中心抽象

**配置プリミティブ (placement primitive)**:
nput のコア。「nix store のパスを root 相対の target に配置する」純粋関数。モジュール抽象で隠さず、ユーザーが合成して使う（→ ADR-0004）。
_Avoid_: 「配置フレームワーク」「設定管理」（設定は生成しない）

**engine (nput エンジン)**:
配置（**ネイティブ FS 操作**）と stale 除去を一手に所有する**配置コア**。`manifest.json` を入力に取り `nix`（profile）/ `git`（toplevel）のみ叩く Go **ライブラリ**として実装し、**nput CLI** が import して駆動する（→ ADR-0003, ADR-0006, ADR-0007）。config ごとに bash を生成しない。「ライブラリ」は **`internal/` のバイナリ内層分離**であって公開 import 可能な再利用モジュールではない（安定面は `manifest.json` 契約に閉じる）。**stdlib-only 厳守**（`syscall.Flock` / `filepath.WalkDir` ベースのコピー / `encoding/json`）。CLI が link-farm を `nix build --out-link <profileDir>/.pending-<name>` で取得し、配置〜`nix-env --set` の **GC 窓**を indirect gcroot で塞ぐ（→ ADR-0011）。実行順は **eval 先行 → flock → build**で、`mkManifest` が passthru する `rootKind` を安価 eval で先取りして profileDir を確定し、flock を取ってから build を**ロック内**で行う（profileDir 未確定の循環と out-link 競合を同時に解消・→ ADR-0023）。並行実行は**解決後 profileDir 単位**の flock で直列化する（明示 apply は blocking wait / shellHook は try-lock skip・同一 profileDir 衝突はユーザー責任で後勝ち・→ ADR-0013）。配置前に target の祖先 component を lstat walk し、symlink ならネスト不可で error 停止する。target に foreign symlink（自身の前世代 manifest に記録の無い・別 config / 別ツール / 手動）があれば warning を出して後勝ちで置換する（→ ADR-0015）。
_Avoid_: 「config ごとに生成される bash スクリプト」「各層がネイティブ機構へ翻訳する」「層ごとの配置ロジック」「CLI と一体の平らな単一実装」「engine を公開 Go モジュールとして外部から import する」（エンジンは `manifest.json` in 契約を保つ `internal/` 層）

**nput CLI**:
ユーザーが叩く一次 UX。PATH に常駐する `packages.nput`。**entrypoint** を発見（CWD 既定 / `-f` で上書き）し、内部で `nix build` / `eval` を回して named manifest を得て **engine** に配置させる。`apply [<name>]`（省略時は `nput.default`）/ `apply --all` / `rollback`（home mode 限定）/ `list-generations`（home mode 限定）/ `gitignore`（project mode 限定・→ ADR-0023）/ `init` のサブコマンドを持つ。内部で叩く nix コマンドは `--help` で開示する（→ ADR-0007）。
_Avoid_: config ごとの `nix run .#x` ラッパーを一次 UX と説明すること（per-config ラッパー `mkActivationScript` は廃止 → ADR-0007）

**entrypoint**:
nput CLI が読む nix の config ファイル。`flake.nix` / `shell.nix` / `default.nix` のいずれか。**`nput.<name>`** に named manifest（`mkManifest` の結果）を公開する。config は依然 Nix で書き `nix build` で評価される（→ ADR-0007）。
_Avoid_: 「nput が CWD から config 内容そのものを発見する」と説明すること（発見するのは entrypoint *ファイル*。config は Nix 評価で確定する）

**module (モジュール / 配線)**:
standalone・home-manager・将来の NixOS・devShell `shellHook` といった統合層。エンジンを起動する**配線**に徹し、自身では配置しない。`home.file` / `systemd.tmpfiles` へは翻訳しない（→ ADR-0003, ADR-0005）。
_Avoid_: 「モジュールがファイルを配置する」「モジュールがネイティブ機構へ変換する」

### 配置の入出力

**entry / entries**:
配置定義。`entries` は **属性キー = target の attrset**で、各値が entry（`{ src; subpath?; target?; method?; }`）。**属性キーが識別子**で、`target` はキーから既定値を取る（home-manager `home.file` 同型・→ ADR-0014）。識別子の一意性は Nix の attrset キー重複不可で native に担保され、手動 `name` フィールドは持たない。
_Avoid_: 「ファイルエントリ」（配置元はディレクトリのこともある）、entry に `name` フィールドを持たせること（識別子は属性キー = target・→ ADR-0014）、entries を `{ name; … }` のリストと呼ぶこと（旧形）

**src**:
**entry** の配置元（どの store パス / リポジトリか）。デフォルトは Nix ストアへの **store link**。out-of-store は明示マーカーで opt-in する（→ ADR-0001）。**subpath**（その中のどのパスか）と直交する別概念で、`src` を `source` の短縮形と読まない（→ ADR-0008）。
_Avoid_: **subpath** と混同すること、`source` と呼ぶこと

**subpath**:
**entry** で **src** 内のどのパスを取り出すかを表す相対パス。file / dir 両対応。デフォルト `"."`。リポジトリ全体は **subpath を省略**するのが canonical（`subpath = "."` は明示形）。評価時に確定する subpath 選択であり、糖衣 marker（`wholeRepo` 等）は足さない（marker は実行時解決の種別を運ぶ入れ物・→ ADR-0007, ADR-0008）。`listFilesInSrc` の引数も同名（旧 `dir` を統一）。
_Avoid_: 旧名 `source` / `dir` を使うこと、**src** と混同すること、「`"."` 以外で全体を表す専用トークン/marker が要る」と考えること（省略で表せる）

**target**:
**entry** の配置先パス。**root** からの相対パスで指定する。**entries の属性キーが target の既定値**であり、entry の identity（stale 除去の diff キー・一意性）でもある（→ ADR-0014）。

**root**:
配置先の基準パス。公開 API の `root` 引数で**明示必須**に選ぶ（**暗黙デフォルトは持たない**）。型は `string（絶対パス・評価時固定）| marker（実行時解決）` の union。マーカーは **projectRoot** / **homeRoot** / **systemRoot** の 3 つ（→ ADR-0004, ADR-0005）。
_Avoid_: 「`$HOME` 固定」「既定は home mode」「root 省略時の既定」と説明すること（暗黙デフォルトは廃止し、root は明示必須になった）。マーカーを「パス文字列を返す糖衣」と説明すること（実行時解決の種別であって評価時にパスへ展開しない）

### 配置モード

**home mode**:
**root** = `$HOME` の配置モード。`homeRoot` マーカーで明示選択する（旧: `root` 省略時の既定。暗黙デフォルトは廃止）。standalone も home-manager 等の module もこのモード。世代を毎回積み、`--rollback` をユーザーに公開する（→ ADR-0002）。
_Avoid_: 「standalone 専用」「`root` 省略時の既定」と説明すること

**project mode**:
**root** = プロジェクトルートの配置モード。`projectRoot` マーカーで選ぶ。配置物は **ephemeral placement**（コミット対象外）で、世代・rollback はユーザーに公開しない（→ ADR-0005）。
_Avoid_: 「CWD 相対」「設定ファイル相対」と説明すること（root は git toplevel で解決）

**projectRoot**:
**project mode** を選ぶ root マーカー。実行時に git toplevel を **root** に解決する（`--root` で上書き可）。`homeRoot` / `systemRoot` と並ぶ root マーカーの一つ。`mkOutOfStoreSymlink` と同じ「マーカーを渡して挙動を opt-in する」パターン（→ ADR-0005）。
_Avoid_: 設定ファイルの場所を指すと解釈すること（Nix では store path 化して成立しない）

**homeRoot**:
**home mode** を選ぶ root マーカー。実行時に `$HOME` を **root** に解決する。旧来「`root` 省略時の暗黙デフォルト」だった `$HOME` を明示マーカーへ昇格したもの（→ ADR-0004 改訂）。

**systemRoot**:
**system mode**（root = `/`）を選ぶ root マーカー。distro 構想（root = `/`）の配置に使う。ADR-0004 が「将来の絶対パス文字列 seam」としていたものを正式なマーカーへ昇格（→ ADR-0004）。

**ephemeral placement**:
**project mode** の配置物の性質。クローンごとに再生成される前提で、プロジェクトにコミットされない。ゆえに activation は git 状態に干渉しない（→ ADR-0005）。
_Avoid_: 「vendoring」「成果物をコミットする配置」と混同すること

### 配置の種別

**store link**:
コア・デフォルトの配置。配置先が Nix ストアパスである symlink。再現性を担保する既定経路（→ ADR-0001）。「統一」とは「デフォルト/コアをストアにし、out-of-store を明示的な例外に降格する」こと。
_Avoid_: out-of-store symlink と混同すること、「コピー」と呼ぶこと

**out-of-store symlink**:
ローカル絶対パスへのライブ symlink。`nput.lib.mkOutOfStoreSymlink "/abs/path"` でのみ opt-in する明示的退避路（開発中 dotfiles のライブ編集用）。第一級機能ではない（→ ADR-0001）。
_Avoid_: デフォルト挙動として扱うこと、`src` の型による暗黙分岐で生むこと

### 状態管理

**generation (世代)**:
ロールバック単位。nput 自身の nix profile（`nix-env --profile <dir>` 式）に乗せて管理する（→ ADR-0002）。コミット（`--set`）・rollback（`--rollback`）・任意世代切替（`--switch-generation`）・一覧（`--list-generations`）・間引き（`--delete-generations`）は**全て `nix-env --profile <dir>` 系で統一**し、store GC のみ `nix-collect-garbage`（→ ADR-0015）。`rollback` は profile dir ≠ 配置先のため再配置が必須で、stale 除去は baseline=離れる世代・ポインタ移動は最後（→ ADR-0015）。
_Avoid_: 「stateless スクリプト」前提の語り（初期方針からは覆っている）、`nix profile`（新CLI: `list` / `wipe-history` / `rollback`）で管理すること（profile-manifest を要求し `nix-env --set` 製 profile では動かない・→ ADR-0015）

**store マニフェスト (store manifest)**:
「nput が配置した」記録を持つ世代由来のデータ。実体は link-farm derivation 内の **`manifest.json`**（`schemaVersion` 付き）で、Nix（`lib.mkManifest`）が生成し Go エンジンが読む **Nix↔Go の契約**。GC 参照は併存する symlink farm が明示的に張る。エンジンの保守的 stale 除去（記録通りを指す nput 管理 symlink だけ削除し、ユーザーの実ファイルには触れない）の不変条件を支える（→ ADR-0002, ADR-0003, ADR-0006）。

## Flagged ambiguities

- **「symlink」単独では曖昧**。必ず **store link** か **out-of-store symlink** のどちらかに寄せる。`entries` の `method = "symlink"`（旧名 `mode`・→ ADR-0015）+ out-of-store marker の `src` で out-of-store を指すが、デフォルトの store link も symlink で実現される。
- **「統一」は「廃止」ではない**。store link 統一は out-of-store を消すことではなく、デフォルトから降格して明示関数に隔離すること。
- **project mode の配置物は untracked が前提**。**ephemeral placement** を「コミットする vendoring」と取り違えない。`.gitignore` への列挙は専用コマンドの stdout 出力から得る（activation は `.gitignore` に触れない）。
- **`src` と `subpath` は別概念**。`src` = どの物（store パス / リポジトリ）、`subpath` = その中のどのパス。名前が似ているが直交する。旧名 `source`（= 現 `subpath`）は使わない（→ ADR-0008）。
- **「standalone」は配置モードではなく起動形態**。standalone（= CLI を直接叩く起動形態）と配置モード（home / project / system）は直交する。standalone から project mode も home mode も使える。「standalone = home mode」と短絡しない（配置モードは `root` マーカーが決める）。
- **project mode の `nput` は devShell 同梱が canonical**。`templates/project` の devShell `packages` に pin 版 `nput` を入れ、CLI と `nput.lib`（manifest `schemaVersion`）を同一 flake 入力で一致させる。グローバル install は standalone（home mode）の利便（→ ADR-0015）。
- **entry の配置種別フィールドは `method`**（旧名 `mode`）。unix file mode との誤読を避ける改名（→ ADR-0015）。

## 会話例

> **Dev**: この dotfiles、編集しながら即反映したいんだけど src にローカルパス渡せばいい？
>
> **Maintainer**: 文字列で渡す暗黙分岐は廃止した。それは **out-of-store symlink** で、`mkOutOfStoreSymlink "/abs/path"` を `src` に渡して明示的に opt-in する。デフォルトは **store link**——ストアへの symlink で再現性がある方。
>
> **Dev**: 配置自体は home-manager のときは `home.file` に変換されるの？
>
> **Maintainer**: しない。全層で nput **エンジン**が `ln` で配置する。home-manager **モジュール**はエンジンを起動する**配線**でしかない。だから振る舞いは層を跨いで同じ。
>
> **Dev**: 前の **世代** に戻したら、消える symlink がユーザーの実ファイルを巻き込んだりは？
>
> **Maintainer**: しない。**store マニフェスト**が「nput が置いた」と記録した symlink だけをエンジンが消す。**target** に元からある実ファイルには触れない。
