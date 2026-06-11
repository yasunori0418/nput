# nput

フェッチ済み git リポジトリを、ユーザー環境の任意パスへ symlink / copy で配置する Nix ライブラリ・モジュール群の用語集。設定生成は行わない。

ここは glossary であり仕様書ではない。実装詳細は `docs/spec.md`、設計判断は `docs/adr/` に置く。2026-06-07 の方針転換（ADR-0001〜0004）で意味が変わった語を中心に、正名と避けるべき同義語を固定する。

## Language

### 配置の中心抽象

**配置プリミティブ (placement primitive)**:
nput のコア。「nix store のパスを root 相対の target に配置する」純粋関数。モジュール抽象で隠さず、ユーザーが合成して使う（→ ADR-0004）。
_Avoid_: 「配置フレームワーク」「設定管理」（設定は生成しない）

**engine (nput エンジン)**:
全層で配置（**ネイティブ FS 操作**）と stale 除去を一手に所有する**単一の固定 Go バイナリ**（`packages.nput`）。`manifest.json` を入力に取り、`nix`（profile）/ `git`（toplevel）のみサブプロセスで叩く。config ごとに bash を生成しない（→ ADR-0003, ADR-0006）。
_Avoid_: 「config ごとに生成される bash スクリプト」「各層がネイティブ機構へ翻訳する」「層ごとの配置ロジック」

**module (モジュール / 配線)**:
standalone・home-manager・将来の NixOS・devShell `shellHook` といった統合層。エンジンを起動する**配線**に徹し、自身では配置しない。`home.file` / `systemd.tmpfiles` へは翻訳しない（→ ADR-0003, ADR-0005）。
_Avoid_: 「モジュールがファイルを配置する」「モジュールがネイティブ機構へ変換する」

### 配置の入出力

**entry / entries**:
配置定義。`{ name; src; target; }` の attrset と、そのリスト。`name` は entries 内で一意。
_Avoid_: 「ファイルエントリ」（配置元はディレクトリのこともある）

**src**:
**entry** の配置元。デフォルトは Nix ストアへの **store link**。out-of-store は明示マーカーで opt-in する（→ ADR-0001）。

**target**:
**entry** の配置先パス。**root** からの相対パスで指定する。

**root**:
配置先の基準パス。公開 API の `root` 引数（マーカー）で選ぶ。既定は **home mode**（`$HOME`）、`projectRoot` で **project mode**。`/` への差し替えは将来の拡張 seam（→ ADR-0004, ADR-0005）。
_Avoid_: 「`$HOME` 固定」と説明すること（root は選択可能になった）

### 配置モード

**home mode**:
**root** = `$HOME` の既定モード。`root` 引数を省略したときの挙動。standalone も home-manager 等の module もこのモード。世代を毎回積み、`--rollback` をユーザーに公開する（→ ADR-0002）。
_Avoid_: 「standalone 専用」と説明すること

**project mode**:
**root** = プロジェクトルートの配置モード。`root = projectRoot` で opt-in。配置物は **ephemeral placement**（コミット対象外）で、世代・rollback はユーザーに公開しない（→ ADR-0005）。
_Avoid_: 「CWD 相対」「設定ファイル相対」と説明すること（root は git toplevel で解決）

**projectRoot**:
**project mode** を選ぶ root マーカー。実行時に git toplevel を **root** に解決する（`--root` で上書き可）。`mkOutOfStoreSymlink` と同じ「マーカーを渡して挙動を opt-in する」パターン（→ ADR-0005）。
_Avoid_: 設定ファイルの場所を指すと解釈すること（Nix では store path 化して成立しない）

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
ロールバック単位。`nix profile` に乗せて管理する（→ ADR-0002）。任意世代への切替・GC は標準の `nix profile` / `nix-collect-garbage` を使う。
_Avoid_: 「stateless スクリプト」前提の語り（初期方針からは覆っている）

**store マニフェスト (store manifest)**:
「nput が配置した」記録を持つ世代由来のデータ。実体は link-farm derivation 内の **`manifest.json`**（`schemaVersion` 付き）で、Nix（`lib.mkManifest`）が生成し Go エンジンが読む **Nix↔Go の契約**。GC 参照は併存する symlink farm が明示的に張る。エンジンの保守的 stale 除去（記録通りを指す nput 管理 symlink だけ削除し、ユーザーの実ファイルには触れない）の不変条件を支える（→ ADR-0002, ADR-0003, ADR-0006）。

## Flagged ambiguities

- **「symlink」単独では曖昧**。必ず **store link** か **out-of-store symlink** のどちらかに寄せる。`entries` の `mode = "symlink"` は marker（out-of-store）を指すが、デフォルトの store link も symlink で実現される。
- **「統一」は「廃止」ではない**。store link 統一は out-of-store を消すことではなく、デフォルトから降格して明示関数に隔離すること。
- **project mode の配置物は untracked が前提**。**ephemeral placement** を「コミットする vendoring」と取り違えない。`.gitignore` への列挙は専用コマンドの stdout 出力から得る（activation は `.gitignore` に触れない）。

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
