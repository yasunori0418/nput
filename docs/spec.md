# nput 仕様書

## アーキテクチャ概要

nput は **2 層**で構成する（→ ADR-0006, ADR-0007）。

```
[nput CLI]  packages.nput（PATH 常駐・一次 UX）
  ・entrypoint(flake.nix/shell.nix/default.nix)を発見（CWD 既定 / -f 上書き）
  ・内部で nix build/eval を回し named manifest(manifest.json + symlink farm)の store path を得る
  ・engine(ライブラリ)を import して配置・stale 除去・profile swap を駆動
   ↓ manifest.json in
[engine]  Go ライブラリ
  ・manifest.json を入力に取り nix(profile)/git(toplevel)のみ叩く
  ・ネイティブ FS 操作で place/replace/remove、保守的 stale 除去、nix-env --set
```

config は **Nix で書き `nix build` で評価**される。CLI が発見するのは entrypoint *ファイル* であって config 内容ではない（→ ADR-0007）。

---

## lib API

`lib`（`mkManifest` / マーカー群 / `listFilesInSrc`）は nixpkgs.lib のみに依存する純データ生成器。配置ロジックは持たない（→ ADR-0006）。

### `lib.mkManifest`

配置データ（`manifest.json` + symlink farm を含む derivation）を生成する**純粋関数**（→ ADR-0006, ADR-0007）。
entrypoint が `nput.<name>` に公開し、nput CLI が `nix build` してエンジンに渡す。Nix 評価テスト（nix-unit / namaka）の単体対象でもある。

```
mkManifest :: { entries, root } -> derivation
```

**引数**

| 引数 | 型 | デフォルト | 説明 |
|---|---|---|---|
| `entries` | attrset of entry | — | 配置定義の attrset。**属性キー = target パス**が識別子（後述・→ ADR-0014）|
| `root` | string \| marker | **なし（必須）** | 配置先の基準。暗黙デフォルトを持たない（→ ADR-0007）|

> 配置単位名（profile を一意特定する `name`）は **entrypoint の `nput.<name>` 属性キー**が供給する。CLI が選択した `<name>` をエンジンへ渡す。

#### `root` の値

`root` は**明示必須**で、暗黙デフォルトを持たない（→ ADR-0004 改訂, ADR-0007）。型は `string（評価時固定）| marker（実行時解決）` の union。

| `root` の値 | モード | root の解決 |
|---|---|---|
| `nput.lib.projectRoot` | project mode | 実行時に git toplevel（`--root` で上書き可）（→ ADR-0005）|
| `nput.lib.homeRoot` | home mode | 実行時の `$HOME`（standalone / HM 共通）|
| `nput.lib.systemRoot` | system mode | `/`（distro 構想・将来）|
| 絶対パス文字列 | 固定 root | 評価時に確定する絶対パス（任意固定 root の seam）|

- マーカーは「**実行時解決の種別を運ぶ入れ物**」であってパス文字列を返す糖衣ではない。`homeRoot` / `projectRoot` は評価時にパスへ展開できない（`$HOME` / git toplevel は実行環境依存・→ ADR-0005）。`mkManifest` は kind を `manifest.json` に記録し、エンジンが実行時に解決する。
- home mode と project mode は世代の扱いが異なる（後述「世代管理仕様」）。

**返り値**

profile が指す store オブジェクト。`manifest.json`（`schemaVersion` 付き・エンジンの入力契約）と、配置元への明示 symlink farm（GC アンカー）を含む（→ ADR-0006）。
`manifest.json` は各 entry の解決済み配置元（store パス / out-of-store 絶対パス）・`subpath`・`target`・`method`・root の kind（project / home / system / 固定パス）を記録する（全フィールドは後述「manifest.json スキーマ（v1）」・→ ADR-0013）。
副作用は持たない（profile swap・FS 配置は engine の実行時責務）。

> **passthru で root kind を露出する**（→ ADR-0023）: 返り値 derivation は `passthru.rootKind`（`"project"` / `"home"` / `"system"` / `"fixed"`、`fixed` のときは `passthru.root` に絶対パス文字列）を持つ。CLI が **ビルド前に** `nix eval <ep>#nput.<system>.<name>.rootKind` で profileDir を確定し flock → build の順に進む実行フロー（前述「実行フロー」）を成立させるため。`rootKind` は eval 時に確定する（実体パス解決は engine 実行時）。

#### 入力検査（`evalModules` + `normalizeManifest`・→ ADR-0010）

`mkManifest` は内部で `lib.evalModules` を回して `entries` / `root` を検査・正規化する。型をオプションに書くだけだと検査が効くのはモジュール経路（`nput.entries`）のみだが、コアである CLI / entrypoint 経路（`mkManifest` 直呼び）でも検査を効かせるため `mkManifest` 自身が `evalModules` を回す。`lib.types` / `mkOption` / `evalModules` は `nixpkgs.lib` のコアなので「lib は nixpkgs.lib のみ依存」を満たす。

実装は 2 段に分かれる。

- **`normalizeManifest { entries, root } -> attrset`**: `evalModules` 検査・デフォルト適用（`subpath` → `"."` / `method` → `"symlink"` / `target` → 属性キー）+ クロスフィールド `lib.throwIf` チェック（`method = "copy"` かつ out-of-store marker / `root = systemRoot` の未実装拒否・→ ADR-0013）+ **パス安全性検査**（target / subpath が絶対パス（`/` 始まり）ならエラー・`filepath.Clean` 相当で正規化し `..` で root / src の外へ出るものを拒否・→ ADR-0019）+ 内部 marker タグ（`_nputMarker`）→ clean enum（`srcKind` / `rootKind`）変換を行う純データ関数。nix-unit / namaka の単体対象。**識別子の一意性は Nix attrset のキー重複不可で native に担保**し、重複 name の throwIf は持たない（→ ADR-0014）。target の静的な絶対 / `..` 判定は root の実体値（実行時解決）に依らず eval 時に可能（→ ADR-0019）。**別キー A/B が `target` フィールドを同値に明示上書きした衝突は、正規化後 target 文字列の重複として eval 時に `lib.throwIf` で検出・停止する**（engine 実行時ではない・→ ADR-0024）。cross-config（別 profile・別 manifest）の同一 target 衝突は eval では検出不可で、これは engine 実行時の後勝ち + foreign symlink warning（→ ADR-0015）になる（両者は別経路）。
- **`mkManifest = args: derivation`**: `normalizeManifest` の出力を `manifest.json` に書き symlink farm を組む。

entry の型定義（`lib/types.nix` の submodule）は `modules/common.nix` の `attrsOf (submodule …)` と共有する。モジュール経路では host の `evalModules`（`attrsOf`）と `mkManifest` の `evalModules` で二重に検査されるが、純粋・冪等で害はなく、`mkManifest` を「entries が必ず通る単一の検査ゲート」に保つ。

> **marker のタグ方式**: `src` の `set`（derivation）と marker（`mkOutOfStoreSymlink`）はどちらも attrset で構造判別できないため、marker には `_nputMarker` 判別タグを持たせ custom optionType の `check` で判別する。`_nputMarker` は Nix 評価内で完結させ `manifest.json` には漏らさない（Go 契約は `srcKind` / `rootKind` の clean enum・→ ADR-0010）。

**使用例（entrypoint への公開）**

```nix
# flake.nix
outputs.nput.${system}.dotfiles = nput.lib.mkManifest {
  root = nput.lib.projectRoot;
  entries = {
    # 属性キー = target。target フィールドは省略（キーから既定）
    ".local/share/nvim/site/pack/foo/start/foo" = { src = inputs.vim-foo; };
    ".zsh/plugins/autosuggestions" = { src = inputs.zsh-sugg; };
  };
};
```

---

### `lib.mkOutOfStoreSymlink`

ローカルパスへの out-of-store symlink を表すマーカーを返す（→ `docs/adr/0001`）。
`entry.src` に渡すことで、その entry を Nix ストア経由ではなくローカル FS への symlink にする。
ファイル編集が即座に反映されるライブ用途のための明示的退避路である。

```
mkOutOfStoreSymlink :: string -> marker
```

- core lib（nixpkgs のみ依存）では **パスをマーカー attrset に包むだけの純粋関数**。
- 実際の link 生成は engine が行う。プラットフォームのネイティブ機構（HM の `config.lib.file.mkOutOfStoreSymlink` 等）へは委譲しない（→ ADR-0003）。

```nix
src = nput.lib.mkOutOfStoreSymlink "/home/me/dotfiles";
```

> **制約**: 引数は Nix 評価時に確定する絶対パスの文字列。シェルの `$HOME` は使えない。
> ローカルパスをポータブルにしたい場合は flake 内で変数として定義する（後述の「ローカルパスの扱い」参照）。

---

### `lib.projectRoot` / `lib.homeRoot` / `lib.systemRoot`

`root` 引数に渡す **root マーカー**（→ ADR-0004 改訂, ADR-0005, ADR-0007）。`mkOutOfStoreSymlink` と同じ「マーカーを渡して挙動を opt-in する」パターン。

```
projectRoot :: marker   # 実行時に git rev-parse --show-toplevel（--root で上書き可）
homeRoot    :: marker   # 実行時に $HOME
systemRoot  :: marker   # /（distro 構想・将来）
```

- core lib（nixpkgs のみ依存）では **kind を運ぶマーカー attrset を返す純粋関数**。実体パス解決は engine の実行時責務。
- 暗黙デフォルトは無い。`root` を省略すると Nix 評価時にエラー（後述「エラー仕様」）。

---

### `lib.listFilesInSrc`

リポジトリ内の指定ディレクトリのファイル一覧を型情報付きで返す。

```
listFilesInSrc :: { src, subpath? } -> { filename: fileType }
```

**引数**

| 引数 | 型 | デフォルト | 説明 |
|---|---|---|---|
| `src` | path | — | フェッチ済みのストアパス。**path 限定**（out-of-store marker・`set`（derivation）は不可・→ ADR-0023, ADR-0024）|
| `subpath` | string | `"."` | リポジトリ内のディレクトリパス。`readDir` するため**ディレクトリ限定**（非 dir はエラー）。entries の `subpath` と同名（旧称 `dir`・→ ADR-0008）|

> `src` は eval 時に `builtins.readDir` するため **path（store パス）に限る**。`mkOutOfStoreSymlink` の marker は実行時解決の入れ物で eval 時にパスへ展開できないため渡せない（→ ADR-0023）。**`set`（`fetchFromGitHub` の生 derivation）も型で弾く**：未 realise の derivation を readDir すると IFD（import-from-derivation）を誘発し flake pure eval で破綻するため（→ ADR-0024）。entries の `src` が `set` を許容するのは engine 実行時解決だからで、`listFilesInSrc`（eval 時 readDir）との非対称はこの違いによる。典型ユースケース（`flake = false` の flake input = 既 realise の store path）は path として問題なく通る。

**返り値**

`builtins.readDir` の返り値と同形式の attrset。

| 型文字列 | 対象 |
|---|---|
| `"regular"` | 通常ファイル |
| `"directory"` | ディレクトリ |
| `"symlink"` | シンボリックリンク |
| `"unknown"` | 特殊ファイル（デバイスファイル・パイプ等）|

**使用例**

```nix
lib.listFilesInSrc { src = inputs.skills-repo; subpath = "skills"; }
# => { "nix" = "directory"; "python" = "directory"; "README.md" = "regular"; }
```

---

## CLI 仕様（一次 UX・→ ADR-0007）

`nput` CLI（`packages.nput`）は PATH に常駐する一次 UX。

- **standalone（home mode）**: `nix profile install github:yasunori0418/nput` 等でグローバルに導入する。
  グローバル CLI とユーザー flake が pin する `nput.lib`（manifest 生成側）は別入力になり `schemaVersion` が skew し得るが、
  engine が自身の対応版より新しい `schemaVersion` を拒否する（→ ADR-0006）ため skew は明確な error になる（MVP は v1 のみ・→ ADR-0015）。
  skew を避けるには **CLI と flake pin の `nput` を同一 input から揃える**ことを推奨する（→ ADR-0016）。project mode のような devShell 同梱は強制しない（PATH 常駐の利便を優先）。
- **project mode（canonical）**: `templates/project` の devShell に **pin 版 `nput` を同梱**する（`packages = [ nput.packages.${system}.nput ]`）。
  `nix develop` / direnv 入室時に flake.lock で固定した `nput` が PATH に載り、CLI と `nput.lib`（manifest `schemaVersion`）が
  同一 flake 入力から来て一致する。グローバル install 依存だと flake.lock の pin を CLI で破り version skew を招くため、
  project mode では devShell 同梱を canonical とする（→ ADR-0015）。

> **前提条件: nix experimental-features**（→ ADR-0025）: CLI は内部で `nix eval` / `nix build`（新 CLI）を使うため、ユーザー環境で
> `experimental-features = nix-command`（flake entrypoint はさらに `flakes`）が**有効化済みであることを前提**とする。CLI は
> `--extra-experimental-features` を自動付与せず（nix.conf / Determinate Nix / 組織ポリシーの設定を黙って上書きしないため）、未有効で
> nix が機能未有効エラーを返したら**前提条件と有効化方法を案内する分かりやすいエラーで停止**する（生の nix エラーを握り潰さない）。

### entrypoint の発見

CLI は **entrypoint ファイル**（`nput.<name>` に named manifest を公開する nix ファイル）を発見する。

| 方法 | 挙動 |
|---|---|
| 既定（自動探索）| CWD で `flake.nix` → `shell.nix` → `default.nix` の優先順で探す |
| `-f` / `--file <path>` | entrypoint を明示指定（自動探索を上書き）|

### アドレッシング（`nput.<name>`）

entrypoint は `nput.<name>` に named manifest（`mkManifest` の結果）を公開する。1 プロジェクトに複数 config を置ける（役割ごと独立 profile・→ concept.md）。

| entrypoint | 公開形 | CLI が叩く build |
|---|---|---|
| `flake.nix` | `outputs.nput.<system>.<name> = mkManifest { ... }` | `nix build <ep>#nput.<system>.<name>`（CLI が現行 `<system>` を差し込む）|
| `default.nix` / `shell.nix` | `{ nput.<name> = mkManifest { ... }; }`（トップレベル）| `nix-build <ep> -A nput.<name>` |

`<name>` = `default` は flake の `default` 慣例に倣う特別な名前で、`nput apply`（name 省略）が解決先に使う。専用 `nput` 名前空間を使い `packages` を汚さない（manifest が通常パッケージとして `nix flake show` / `nix build` に混ざらない・→ ADR-0007）。

### サブコマンド体系

```bash
nput apply                     # name 省略時は nput.default を適用（flake の default 慣例。無ければエラー）
nput apply <name>              # nput.<name> をビルドし新世代を作って適用
nput apply <name> --dryrun     # dry-run。副作用ゼロで place/replace/remove/conflict/no-op を表示
nput apply <name> --recopy     # 通常 apply に加え config 内の全 copy target を src から無条件上書き再コピー（→ ADR-0020）
nput apply --manifest <link-farm>  # ビルド済み link-farm を直接適用（entrypoint 発見・eval/build なし・host/module activation seam・→ ADR-0026）
nput reset <name> [target...]  # 配置物を無い状態へ戻す。target 省略で全 entry、指定でその entry のみ。名指し必須（--all 非対応）（→ ADR-0020, ADR-0021）
nput reset <name> --dryrun     # 副作用ゼロで削除対象（symlink / copy target）を表示して exit（confirm/flock なし・→ ADR-0021）
nput apply --all               # entrypoint の nput.* を全て辞書順に適用
nput apply --all --project-root # nput.* のうち projectRoot の config のみ適用（--home-root / --system-root も同様・→ ADR-0017）
nput rollback <name>           # 前世代へ戻す（home mode 限定・名指し必須。--all 非対応・→ ADR-0018）
nput list-generations <name>   # 世代一覧を表示（home mode 限定）
nput list-generations --all    # home mode の全 config の世代を一覧（→ ADR-0018）
nput gitignore <name>          # 配置 target を .gitignore 向けに stdout 出力（書き込みなし）
nput gitignore --all           # projectRoot の全 config の target をソート + 重複除去して出力（→ ADR-0018）
nput init <template>           # nix flake init -t <nput>#<template> のラッパー（後述）
```

**グローバルフラグ**

```bash
-f, --file <path>   # entrypoint を明示（自動探索を上書き）
--root <path>       # 解決 root を明示上書き（全モード共通。project は git toplevel を、home は $HOME を使わない・→ ADR-0017）
                    # 明示時は全モードで profileDir を上書き後 root の <roothash> でキー（→ ADR-0023）
--no-wait           # flock 競合時に待たず skip（shellHook 用。既定は明示 apply=blocking wait・→ ADR-0013）
--quiet             # 進捗 / 配置レポートを抑制（warning / error は残す。shellHook 向き・→ ADR-0023）
--verbose           # 内部実行する nix コマンド等の詳細を出力（→ ADR-0023）
--project-root      # --all の修飾。nput.* のうち projectRoot の config のみ適用（→ ADR-0017）
--home-root         # --all の修飾。homeRoot の config のみ適用
--system-root       # --all の修飾。systemRoot の config のみ適用（system mode は未実装のため当面マッチなし・将来 seam）
--recopy            # apply の修飾。config 内の全 copy target を src から無条件上書き再コピー（→ ADR-0020）
--manifest <path>   # apply 専用。ビルド済み link-farm を直接適用（entrypoint 発見・eval/build なし・host/module activation seam・→ ADR-0026）
-y, --yes           # reset の確認プロンプトをスキップ（スクリプト / CI 用・→ ADR-0020）
```

> `--json`（機械可読出力）は **MVP では持たない**（将来送り・→ ADR-0023）。当面はテキスト出力 + 下記のストリーム規律で足りる。

- `apply` の **name 省略時は `nput.default` を適用**する（flake の `default` 慣例に倣う。`default` が未定義ならエラー）。`<name>` を明示すればその config を、`--all` で `nput.*` 全てを適用する。profile は config 単位で atomic（→ ADR-0002）。
- `apply --manifest <link-farm>` は **ビルド済み link-farm を engine へ直接適用**する（host / module activation の seam・→ ADR-0026）。entrypoint 発見・rootKind 先取り eval・`nix build` を**行わず**、引数の link-farm 内 `manifest.json` から engine が rootKind を読む（project / home / fixed の全モード対応）。取得後の挙動（flock → 前世代 diff → 配置 → 保守的 stale 除去 → `nix-env --set` → レポート）は通常 apply と同一で、配置ロジックは二重化しない（engine の `Build` / `LinkFarm` seam に対応・→ ADR-0011, ADR-0003）。引数は **link-farm**（`mkManifest` 出力の store パス = 世代としてコミットされる対象）で、`manifest.json` 単体ファイルは渡さない。`-f` / `--all` は取得元が衝突するため**併用エラー**、位置引数 `name`（profile 選択）とは直交し両立する（省略 = `default`・→ ADR-0024）。HM モジュールの activation が使う主経路（→「モジュール別動作仕様」）。module 経路は CLI と `mkManifest` が同一 flake input 由来のため schemaVersion skew が構造的に起きない（→ ADR-0026, ADR-0006）。
- `apply --all` は **entrypoint の `nput.*` を辞書順（キーソート・決定的）に適用**し、**一部が失敗しても残りを続行**する（各 config は独立 profile で atomic なため）。Nix attrset は定義順を保持せず `builtins.attrNames` が辞書順を返すため適用順は辞書順になるが、各 config が独立 atomic なので順序は結果に影響しない（表示・失敗集約のための決定的順序・→ ADR-0016）。最後に成功 / 失敗を集約表示し、**1 つでも失敗なら非ゼロ終了**する。`--all` 自体は全体 atomic にしない（project mode は rollback 非公開で意味論が崩れるため・→ ADR-0013）。
- `apply --all` に **`--project-root` / `--home-root` / `--system-root`** を付けると、`nput.*` のうち該当 root モードの config **のみ**を適用する（root マーカー名に揃えたフィルタ・opt-in・→ ADR-0017）。素の `--all` は全 config を適用する。home mode と project mode の config が混在する entrypoint で devShell の `shellHook` から `--all` を打つと home mode config も `$HOME` に配置される footgun があるため、devShell は **`nput apply --all --project-root`**（または名指し `nput apply <name>`）を使う。フィルタは `--all` 修飾で、名指し apply では `<name>` が 1 config を pin するため無意味。
- `rollback` / `list-generations` は **home mode 限定**。project mode は世代を内部機構に留めユーザーに公開しない（→ ADR-0005）。
- `rollback` は **名指し必須**（`--all` 非対応）。全 config を一斉に戻すのは破壊的で footgun、途中失敗で状態が不揃いになり得るため（→ ADR-0018）。
- `list-generations --all` は home mode の全 config の世代を一覧（読み取り専用）。`gitignore --all` は projectRoot の全 config の target を **ソート + 重複除去**して出力する（repo の `.gitignore` は 1 つなので一括列挙が自然・→ ADR-0018）。
- `apply <name> --recopy` は通常 apply に加え **config 内の全 copy target を現 `src`/`subpath` から無条件に上書き再コピー**する（→ ADR-0020）。copy は世代外で hash 追跡しないため差分判定はせず無条件。上書きした target をレポート表示し、フラグ自体が opt-in なので確認は出さない。**ローカルの copy 編集は破棄され src 内容に戻る**（= upstream 追従の意図）。symlink 部の世代コミット挙動は不変（copy は世代を増やさない）。
- `reset <name> [target...]` は配置物を**無い状態へ戻す** teardown（→ ADR-0020）。symlink は stale 除去と同じ**保守的不変条件**（nput 管理・記録通りのみ・foreign は warning で残す）で除去し、**copy target も削除**する（copy を消す唯一の明示手段）。データ損失リスクのため**確認プロンプト**を出すか `--yes` で同意を要求し、削除 target をレポート表示する。**profile / 世代は触らない FS-only teardown** で、config が entry を残す限り次の apply で再配置される（transient・project mode は ADR-0017 の lstat 検査で復帰）。恒久除去は config から entry を消して apply、profile 完全除去は `nix-env --profile <profileDir>/profile --delete-generations`。home / project 両モード可。
- `reset` は **名指し必須（`--all` 非対応）**（一斉撤去は破壊的 footgun・`rollback --all` 却下と一貫・→ ADR-0018, ADR-0021）。複数撤去は名指しを複数回。**解決後 `profileDir` 単位の blocking flock を取得**して並行 apply / reset と直列化する（→ ADR-0013, ADR-0021）。profileDir 確定のため **rootKind 先取り eval → root 解決**を build しないコマンドでも先行する（apply と共通の前段・`--root` 時は同じ roothash キー・→ ADR-0024）。
- `reset <name> --dryrun` は**副作用ゼロ**で削除対象（symlink / copy target）を表示して exit する（FS 削除・confirm・flock いずれも行わない・`apply --dryrun` と対称・→ ADR-0021）。終了コードは削除対象の有無に依らず 0。
- `apply --all --recopy` は**合成可**。`--all`（必要なら `--project-root` 等フィルタ）が選んだ各 config に `--recopy` を適用する（`--recopy` は apply 修飾で `--all` と直交・→ ADR-0021）。
- `apply <name> --dryrun` は FS 書込・`--set`・flock いずれも行わない読み取り専用。`conflict` があれば非ゼロ終了（CI の事前 gate に使える）（→ ADR-0006）。
- `gitignore <name>` は配置 target を `.gitignore` 向けに列挙して stdout に出力するだけで、ファイルは書き込まない。更新責務はプロジェクト管理者（→ ADR-0005）。出力は **root 相対 target に先頭 `/` を付けたアンカー形式**（例: `/.claude/skills/nix`）で 1 行 1 件。project mode の root = git toplevel = `.gitignore` 置き場所なので先頭 `/` が正しくアンカーし、別階層の同名パスを誤って無視しない。ディレクトリ / ファイルとも末尾 `/` は付けない（→ ADR-0013）。
- `gitignore` は **project mode 限定**。単体 `gitignore <name>` も project mode の config のみ受理し、**非 project config（home / fixed）を指定したらエラーで停止**する（出力のアンカー形式が git toplevel = `.gitignore` 置き場所を前提とし、home / fixed では意味を成さないため）。`rollback` / `list-generations` が home mode 限定なのと対称（→ ADR-0023）。
- `gitignore` は **`method` を区別せず全 target を列挙**する（copy target も含む・→ ADR-0019）。project mode の copy target も ephemeral 扱いで、各 clone で place-once で再マテリアライズされ**編集は clone local / 使い捨て**（`git clean` で消える）。copy を committed（vendoring）にするのは nput の責務外（手動コミット）で、project mode の ephemeral 原則は崩さない（→ ADR-0019）。
- 透明性: `nput --help` 等で内部実行する nix コマンドを開示し、ユーザーが選択的に手で実行できる（→ ADR-0007）。
- 任意世代への切替・世代の GC は標準の `nix-env` / `nix-collect-garbage` を profile パスに対して使う。
- `--only`（一部 entry だけ適用）は profile 世代の atomic 性と衝突するため提供しない。選択的更新は config（`nput.<name>`）の分割で担保する。

### 出力ストリームと終了コード（→ ADR-0023）

- **ストリーム規律**: **stdout は機械可読出力を専有**する（`gitignore` の列挙・`apply --dryrun` のプラン）。**進捗 / 配置レポート（placed / replaced / removed / skipped）・warning・shellHook の skip 通知はすべて stderr**。これにより `nput gitignore <name> >> .gitignore` や `nput apply <name> --dryrun | ...` が安全にパイプできる。
- **終了コード**:

  | code | 意味 |
  |---|---|
  | `0` | 成功 / no-op / `--no-wait` の try-lock skip（正常スキップ）|
  | `1` | 一般エラー（eval エラー・engine 実行時エラー・`apply --all` の部分失敗）|
  | `2` | `apply --dryrun` で conflict を検出（CI の事前 gate に使える）|

- `--quiet` は進捗 / 配置レポートを抑制し warning / error は残す（shellHook 既定との相性が良い）。`--verbose` は内部実行する nix コマンド等の詳細を出す。`--json` は MVP では持たない（将来送り）。
- **`--quiet` が抑制するのは stderr の進捗 / 配置レポートのみ**で、**stdout 専有の機械可読出力（`apply --dryrun` の plan・`gitignore` の列挙）は `--quiet` 下でも抑制しない**（→ ADR-0024）。stdout 専有原則を貫き、`nput apply <name> --dryrun --quiet | ...` や `nput gitignore <name> --quiet >> .gitignore` のパイプを壊さない。
- **`apply --all --dryrun` の終了コードは「いずれかが error なら 1、error が無く conflict があれば 2、どちらも無ければ 0」**（error(1) 最優先 → conflict(2) → 0・→ ADR-0024）。単純な最大値（2 > 1 で conflict 優先）は採らない（より深刻な eval / engine エラーを CI で隠すため）。非 dryrun の `--all` は conflict 概念が無く 0 / 1 のみ。

### `nput init`（テンプレート展開）

```bash
nput init standalone   # nix flake init -t github:yasunori0418/nput#standalone のラッパー
nput init project      # nix flake init -t github:yasunori0418/nput#project のラッパー（devShell shellHook 配線 + .gitignore ガイド入り）
```

- `nix flake init -t github:yasunori0418/nput#<template>` への**透明なラッパー**。ファイルを作るのは nix の templates 機構であり nput 自身は generate しない（「設定を生成しない」thesis を維持・→ ADR-0007）。
- `nix flake init` の「既存ファイルを上書きしない」保守性を継承する。
- **テンプレート参照はバイナリにハードコードした固定 flake ref `github:yasunori0418/nput`**（→ ADR-0025）。registry 登録に依存せず動く。`init` は新規 bootstrap 用途で、CLI 版と template 版のズレ（常に latest main 参照）は許容する。apply 時の `schemaVersion` 整合は project mode の devShell 同梱 pin（→ ADR-0015）が担う。

#### テンプレートの内容（最小 + 手厚いコメント・→ ADR-0018）

各 template は**動く example を 1 config だけ**置き、バリエーション（`subpath` / `method = "copy"` / `mkOutOfStoreSymlink` / 複数 entry / 動的生成）は**コメントで示す**。starter を小さく保ち、ユーザーが不要分を削除する手間を最小化する。

| template | ファイル | 内容 |
|---|---|---|
| `standalone` | `flake.nix` | `homeRoot` の 1 config 例（`nput.<system>.<name> = mkManifest { root = homeRoot; entries = {...}; }`）+ バリエーションコメント |
| `project` | `flake.nix` | `projectRoot` の 1 config 例 + devShell（`packages = [ nput.packages.${system}.nput ]`・shellHook = 名指し apply）+ 「配置物は ephemeral・`.gitignore` へ `nput gitignore <name>` 出力を追記」コメント |
| `project` | `.gitignore` | 先頭に `# nput: regenerate with 'nput gitignore <name>'` ヘッダコメント付きの雛形 |

- project template の `shellHook` は **`nput apply <name> --no-wait`（名指し）**を既定にする。example が 1 config なので最も明確で混在 footgun（→ ADR-0017）が起きない。「複数 config なら `nput apply --all --project-root --no-wait`」をコメントで示す。
- `.envrc`（direnv）は同梱しない（非利用者に不要ファイルを増やすため。コメント案内に留める・→ ADR-0018）。

### 再現性スタンス（→ ADR-0007）

| entrypoint | eval | 再現性 |
|---|---|---|
| `flake.nix` | pure（root 解決はエンジン実行時なので eval は pure のまま）| flake.lock で固定 |
| `shell.nix` / `default.nix` | impure（NIX_PATH / channels 依存）を許容 | **ユーザー責任**。nput lib を含め nixpkgs を npins / fetchTarball / flake-compat 等で固定することを推奨 |

> **`nix flake check` と `nput` カスタム output**（→ ADR-0015）: consumer の `outputs.nput.<system>.<name>`（`packages` 汚染回避の
> 専用 namespace・ADR-0007）は `nix flake check` で **`warning: unknown flake output 'nput'` を出すが exit 0**（CI を壊さない・想定内・無害）。
> `flake check` は `nput` 直下 attrset の eval 健全性は検査するが、**配下の `.<system>.<name>` derivation は build も eval もしない**
> （誤 build しない）。警告は flake-parts 経由でも消えず、output 名を変えても消えない（`lib` 以外は unknown 警告）。`nput` 成果物の主検証は
> `nix build .#nput.<system>.<name>` で行う。将来 upstream の flake-schemas（PR #8892）がマージされたら `schemas.nput` で消す余地を残す。

### 実行フロー

**順序は「eval 先行 → flock → build」**（→ ADR-0023）。profileDir は root 解決後にしか確定せず（project / `--root` 時は `<roothash>`）、root 解決には `manifest.json` の `rootKind` が要る。これを安価な `nix eval`（root kind のみ）で**ビルド前に**先取りし、profileDir を確定してから flock を取り、**build をロック内**に閉じる。これで profileDir 未確定の循環と、ロック外 build の `.pending` out-link 競合（並行 apply の奪い合い）が同時に解消する。`profileDir` は config 専用ディレクトリで profile リンクは `<profileDir>/profile`・pending out-link は `<profileDir>/.pending`（レイアウトは「世代管理仕様」の「機構」節・→ ADR-0025）。

```
nput apply <name> [-f <ep>] [--root <p>]
  0. entrypoint 発見（-f 上書き）
  1. root kind を先取り eval:
     nix eval <ep>#nput.<system>.<name>.rootKind（legacy は nix eval -f <ep> nput.<name>.rootKind）
     → root 解決（kind: project=git rev-parse / home=$HOME / system=/ / 固定パス、--root 上書き）
     → profileDir 確定（home: <name> / project: <roothash>/<name>。--root 明示時は全モード <roothash>/<name>・→ ADR-0023）
       ※ rootKind は mkManifest の passthru として eval 時に確定（git toplevel / $HOME の実体解決は engine 実行時）
  2. engine を駆動:
     a. flock を取得（キー = 確定 profileDir）。
        明示 apply / rollback は blocking（LOCK_EX・取得まで待ち「他の apply 完了待ち」を表示）。
        shellHook 経路（--no-wait）は try-lock（LOCK_NB）で保持中ならスキップし、stderr に1行通知する
        （例: `nput: another apply in progress, skipped (run \`nput apply\` manually)`・シェル入室はブロックしない・→ ADR-0022）。
        同一 profileDir への同時実行はユーザー責任で衝突時は後勝ち（→ ADR-0013）
     b. ロック内で nix build <ep>#nput.<system>.<name> --out-link <profileDir>/.pending
        （legacy は nix-build <ep> -A nput.<name> --out-link <profileDir>/.pending）
        → os.Readlink で link-farm store path を得る。out-link が indirect gcroot を張り
          配置〜--set の GC 窓を塞ぐ（→ ADR-0011）。build がロック内なので out-link 競合は構造的に起きない（→ ADR-0023）
     c. profileDir の前世代 manifest.json を読む（無ければ初回 = 削除対象ゼロ）
     d. project mode かつ新 link-farm が前世代と同一なら新世代は積まない（世代スキップ）。ただし各 target を lstat 検査し、
        ドリフトした entry だけ再張りする（完全 no-op にしない・→ ADR-0017）
     e. manifest.json を新旧 diff → 新規/張替を配置 → 保守的 stale 除去（ネイティブ FS）
     f. nix-env --profile <profileDir>/profile --set <link-farm>（サブプロセス・コミット点）
     g. --set 成功後に <profileDir>/.pending を削除（世代リンクが gcroot を引き継ぐ・→ ADR-0011）
```

- **非 build コマンド（`reset` / `rollback` / `list-generations`）も eval 先行を共通前段に持つ**（→ ADR-0024）。build はしないが、profileDir 単位の flock / 前世代 manifest 読みのため profileDir 確定（= rootKind 先取り eval → root 解決）が前提になる。`--root` 上書き時は §「root の解決」と同じ roothash キーで profileDir を引く（`--root` を付けた世代を操作するには同じ `--root` が要る）。`reset` はさらに entries 読みのため entrypoint eval も行う。
- **`apply --all` は rootKind を 1 回の一括 eval で取る**（→ ADR-0024）。`nix eval <ep>#nput.<system> --apply 'cs: builtins.mapAttrs (_: c: c.rootKind) cs' --json`（legacy は対応する `-f` 形）で config 名 → rootKind マップを 1 回で取得し、各 profileDir を確定する。`--project-root` 等のフィルタもこの結果で振り分ける。build だけは atomic 性のため config ごと N 回。eval プロセス起動コストを N→1 に固定する。
- `--dryrun` は root kind を eval し root を解決するが（プラン表示のため）、link-farm を build しても配置しない読み取り専用なので、flock も pending gcroot（out-link）も取らない（→ ADR-0011, ADR-0023）。
- `--set`（f）到達前に apply が失敗すると `<profileDir>/.pending` gcroot が残り、ビルド済み未使用 link-farm を掴み続けるが、次回 apply が**同名**（`.pending`）で上書きするため config あたり最大 1 個に有界。回収処理は持たず許容する（→ ADR-0016）。
- **`apply --manifest <link-farm>` は 0〜1（entrypoint 発見・rootKind 先取り eval）と 2b（ロック内 `nix build`）を skip する**（→ ADR-0026）。ビルド済み link-farm を engine へ直接渡し、rootKind は link-farm 内 `manifest.json` から engine が読む。2a（flock 取得）以降〜2g は通常 apply と同一。pending out-link は build しないため張らない。host / module activation（HM 等）が switch 時に使う経路で、`-f` / `--all` とは取得元衝突で併用エラー。

---

## entries スキーマ仕様

`entries` は **attrset で属性キー = target**（→ ADR-0014）。各値が entry submodule。

```nix
entries :: {
  "<target>" = entry;   # 属性キー = root 相対 target パス（識別子）
  ...
}
```

### フィールド定義（entry submodule）

```
entry :: {
  src    : path | set | marker # 必須（type/marker によって挙動が変わる）
  subpath: string              # 省略可、デフォルト: "."（省略 = リポジトリ全体）
  target : string              # 省略可、デフォルト: 属性キー
  method : "symlink"
         | "copy"              # 省略可、デフォルト: "symlink"
}
```

各フィールドは `lib/types.nix` の entry submodule（`lib.types`）として定義され、`mkManifest` の `evalModules` が検査・デフォルト適用する（→ ADR-0010）。submodule は **strict**（未知キー拒否）で、タイポや旧名（`name` / `source` / `dir` / `mode`・→ ADR-0008, ADR-0014, ADR-0015）は評価時エラーになる。識別子（target）の一意性は attrset のキー重複不可で native に担保される（→ ADR-0014）。

#### `src`

- **必須**: yes
- **説明**: 配置元。デフォルトは Nix ストアへの symlink。out-of-store は明示マーカーで opt-in する（→ ADR-0001）。

| `src` の値 | 例 | symlink の指す先 | 用途 |
|---|---|---|---|
| `path` | `inputs.myrepo` | Nix ストア（不変）| 外部リポジトリ（バージョン固定）|
| `path` | `builtins.path { path = /home/...; name = "..."; }` | Nix ストア（ローカルをコピー）| ローカルをストア経由で扱う |
| `set` | `pkgs.fetchFromGitHub { ... }` | Nix ストア（不変）| 外部リポジトリ（バージョン固定）|
| `marker` | `nput.lib.mkOutOfStoreSymlink "/abs/path"` | ローカル FS（ライブ）| 開発中の手元 dotfiles |

```nix
# 外部リポジトリ（store link）
src = inputs.myrepo;
src = pkgs.fetchFromGitHub { owner = "..."; repo = "..."; rev = "..."; hash = "..."; };

# ローカルをストア経由（評価時点の内容をストアにコピー）
src = builtins.path { path = /path/to/dotfiles; name = "dotfiles"; };

# ローカルを out-of-store symlink（ライブ反映）— 明示関数で opt-in
src = nput.lib.mkOutOfStoreSymlink "/path/to/dotfiles";

# 廃止: string を直接渡す暗黙の out-of-store 分岐は提供しない
# src = "/path/to/dotfiles";   # NG
```

#### `subpath`

- **型**: string / **必須**: no / **デフォルト**: `"."`（リポジトリルート全体）
- **説明**: `src` 内のどのパスを取り出すかを表す相対パス。ファイル・ディレクトリどちらも指定可能。
- **リポジトリ全体は `subpath` を省略する**のが canonical（`subpath = "."` は同義の明示形）。「`"."` 以外で全体を表す専用トークン / marker」は設けない。`subpath` は評価時に確定する subpath 選択であり、糖衣 marker は marker パターン（実行時解決の種別を運ぶ入れ物）と相反するため（→ ADR-0007, ADR-0008）。
- 旧称は `source`。`src`（どの物か）との命名衝突を解消するため `subpath`（その中のどのパスか）へ改名した（→ ADR-0008）。

```nix
# 省略 = リポジトリ全体（canonical）
subpath = ".";                  # リポジトリ全体（明示形）
subpath = "skills/nix";         # サブディレクトリのみ取り出す
subpath = "themes/dark.json";   # 単一ファイル
```

#### `target`

- **型**: string / **必須**: no / **デフォルト**: **属性キー**（→ ADR-0014）
- **説明**: 配置先パス。**root**（`mkManifest` の `root` で明示選択した基準）からの相対パスで指定する。
- entries は **attrset で属性キー = target**。キーをそのまま target とするのが canonical（`target` 省略）。キーを論理ラベルにして `target` を明示上書きすることもできる（home-manager の `home.file` と同型）。
- **identity（stale 除去の diff キー・一意性）は解決後 target**。同一 target を 2 つ置くことは attrset のキー重複として表現できず、Nix が一意性を担保する。

#### `method`

- **型**: `"symlink"` | `"copy"` / **必須**: no / **デフォルト**: `"symlink"`
- 旧名は `mode`。unix file mode（`0644` 等）との誤読を避けるため `method`（配置方法）へ改名した（→ ADR-0015）。

| method | `src` の種別 | 動作 | 世代管理 |
|---|---|---|---|
| `"symlink"` | path / set | Nix ストアへの symlink（読み取り専用）| あり（profile）|
| `"symlink"` | marker | ローカルパスへの out-of-store symlink（ライブ）| あり（リンク先のみ）|
| `"copy"` | path / set | place-once コピー（書き込み可・ユーザー管理）| **なし** |
| `"copy"` | marker | **eval 時エラー**（意図矛盾・`lib.throwIf`・→ ADR-0013）| — |

### entries を動的に生成する（target キーの string interpolation）

`entries` は attrset で**キー = target**。キー（target）に名前を補間して動的に組み立てる。list から attrset を作るには `builtins.listToAttrs`（`{ name; value; }` の `name` がキー = target）等を使う（→ ADR-0014）。

#### 基本形：名前を target キーに補間する

配置先ディレクトリ名を `src` 選択と同じ変数から導き、target キーに補間する。

```nix
# nvim プラグインを名前リストから一括生成
let
  plugins = [ "telescope" "treesitter" "cmp" ];
in
nput.lib.mkManifest {
  root = nput.lib.homeRoot;
  entries = builtins.listToAttrs (map (n: {
    name  = ".local/share/nvim/site/pack/plugins/start/${n}";  # キー = target
    value = { src = inputs.${n}; };                            # subpath 省略 = リポジトリ全体
  }) plugins);
}
```

#### ⚠️ アンチパターン：`src` の store パスから target を導く

`baseNameOf src` で target を作るのは**罠**。`src`（flake input / `fetchFromGitHub`）は `/nix/store/<hash>-source` のような **hash 前置の store 名**に解決されるため、`baseNameOf` は `<hash>-source` を返し target には使えない。

```nix
# NG: キー(target) が ".config/abcd1234...-source" のようになる
entries = { ".config/${baseNameOf inputs.foo}" = { src = inputs.foo; }; };
```

target は **`src` の store 名ではなく、ユーザーが制御する変数から導く**（上の基本形）。

#### 応用：`listFilesInSrc` で subdir を列挙して展開する

リポジトリ内の subdir を `listFilesInSrc` で走査し、各 subdir を 1 entry に展開する。target キーと `subpath` の両方を列挙した名前で補間する。

```nix
let
  # claude-skills/skills 配下の dir 名を列挙
  skills = nput.lib.listFilesInSrc { src = inputs.claude-skills; subpath = "skills"; };
  names  = builtins.attrNames (nixpkgs.lib.filterAttrs (_: t: t == "directory") skills);
in
nput.lib.mkManifest {
  root = nput.lib.homeRoot;
  entries = builtins.listToAttrs (map (n: {
    name  = ".claude/skills/${n}";                                # キー = target（配置先）
    value = { src = inputs.claude-skills; subpath = "skills/${n}"; };  # 取り出す側も補間
  }) names);
}
```

---

## manifest.json スキーマ（v1・Nix↔Go 契約・→ ADR-0010, ADR-0013）

`manifest.json` は `lib.mkManifest` が生成し engine が読む**唯一の安定契約**（→ ADR-0011）。`schemaVersion` は **`1`** で固定し、engine は自身の対応版より新しい `schemaVersion` を拒否する（→ ADR-0006）。内部タグ `_nputMarker` は manifest に漏らさず clean enum で写す（→ ADR-0010）。**MVP は v1 のみを発行・受理し、マイグレーション（schema 後方互換）は現時点では考慮しない**（古い版の受理機構を最初から作らない）。**最初のリリース後、フィールド追加で v2 が必要になった時点で**、後方互換ポリシー（engine が古い版の manifest を stale 除去 / rollback のために読めること。アップグレード直後の stale 除去は前世代 manifest を読む）を改めて検討する（→ ADR-0022）。

### トップレベル

| フィールド | 型 | 説明 |
|---|---|---|
| `schemaVersion` | int | 契約バージョン。v1 は `1` |
| `root` | object | 配置先基準の kind（下記）|
| `entries` | array of object | 配置定義（下記）|

### `root`

| フィールド | 型 | 説明 |
|---|---|---|
| `rootKind` | `"project"` \| `"home"` \| `"system"` \| `"fixed"` | root マーカーの種別。engine が実行時に解決 |
| `root` | string | `rootKind = "fixed"` のときのみ存在する絶対パス。それ以外は省略 |

`project` / `home` / `system` は実行時解決（git toplevel / `$HOME` / `/`）のためパスを持たない。`fixed` のみ評価時確定の絶対パスを `root` に持つ。

### `entries[]`

attrset を**配列に正規化**して記録する（Go は配列を読む）。identity は `target`（→ ADR-0014）。

| フィールド | 型 | 説明 |
|---|---|---|
| `srcKind` | `"store"` \| `"outOfStore"` | 配置元の種別 |
| `src` | string | `store`: 解決済み store パス文字列 / `outOfStore`: marker の絶対パス |
| `subpath` | string | src 内の相対パス（デフォルト適用後。省略形も `"."` で記録）|
| `target` | string | root 相対の配置先。**entry の identity**（属性キー由来・stale 除去の diff キー）|
| `method` | `"symlink"` \| `"copy"` | 配置種別（デフォルト適用後。旧名 `mode`・→ ADR-0015）|

> entry の `name` フィールドは廃止した（→ ADR-0014）。symlink farm の GC アンカー名は **`target` のハッシュ（sha256 の短縮 hex・固定長・FS 安全）**を用いる（→ ADR-0016）。`target` をサニタイズ（`/` 除去等）すると別 target が同名に潰れ linkFarm のキー一意制約に反するため。farm は GC アンカー専用でアンカー名は可読である必要がなく、衝突不可能なハッシュで十分（engine が配置に使う値は `manifest.json` の解決済み `src` 文字列・→ ADR-0010）。

### symlink farm との対応

derivation は `manifest.json` と symlink farm を含む。farm は **GC アンカー専用**で、engine が配置に使う値は `manifest.json` の解決済み `src` 文字列（→ ADR-0010）。

- store-backed entry（`srcKind = "store"`）**かつ `method = "symlink"`** は farm に store パスへの symlink アンカーを持ち、profile 世代が GC root として全 store src を掴む。
- out-of-store entry（`srcKind = "outOfStore"`）は store 外を指すため farm アンカーを持たない。
- **`method = "copy"` entry は farm アンカーを持たない**（store src でも・→ ADR-0019）。copy は place-once でマテリアライズ後は store から独立（世代外・→ ADR-0002）なので store src を掴む必要がなく、`nix-collect-garbage` で解放されてよい。`manifest.json` には記録する（orphan 警告・stale 判定のため）。

### 例

```json
{
  "schemaVersion": 1,
  "root": { "rootKind": "project" },
  "entries": [
    {
      "srcKind": "store",
      "src": "/nix/store/abcd1234...-source",
      "subpath": "skills/nix",
      "target": ".claude/skills/nix",
      "method": "symlink"
    },
    {
      "srcKind": "outOfStore",
      "src": "/home/me/dotfiles",
      "subpath": "home/.config/nvim",
      "target": ".config/nvim",
      "method": "symlink"
    }
  ]
}
```

---

## 配置動作仕様

engine が**ネイティブ FS 操作**で行う（`ln` / `rsync` は使わない・→ ADR-0006）。

### symlink モード

```
0. target の各祖先 component を lstat で walk。いずれかが symlink ならエラーで停止（→ ADR-0015）
   （祖先が全体 symlink 配置のとき、その配下にネストできない。store 汚染 / dangling を防ぐ）
1. target の親ディレクトリを作成（mkdir -p 相当。祖先 symlink は 0 で弾き済み）
2. target が既存 symlink のとき:
   - 自身の前世代 manifest が記録した symlink → そのまま置き換える（silent）
   - 記録の無い symlink（foreign = 他 nput profile / 他ツール / 手動）→ warning を出して置き換える（後勝ち・→ ADR-0015）
3. <配置元>/<subpath> を指す symlink を <root>/<target> に作成（os.Symlink）
   - store link:        配置元 = Nix ストアパス
   - out-of-store:      配置元 = marker の絶対パス
```

- 既存 symlink の張替えは **unlink + symlink の 2 操作**で行う（rename ベースの atomic swap は採らない）。間でクラッシュすると target が一時消失しうるが、**冪等な再実行で収束**する（ADR-0006「積まれる世代は常に完全適用済み」と整合・→ ADR-0017）。
- target に通常ファイルまたはディレクトリが存在する場合はエラーで停止（上書きしない）
- subpath がファイル・ディレクトリどちらでも同じ処理
- 別 config（別 profile）が同一 target を狙うのは基本「衝突させない前提」。後勝ちを許容しつつ foreign symlink 上書きは warning で可視化する（→ ADR-0015）

### copy モード（place-once・ユーザー管理）

```
subpath がディレクトリの場合:
  target が不在のとき: <root>/<target> を作成しネイティブ再帰コピー（mode 保存 + owner-write 付与。<src>/<subpath>/ → <root>/<target>/）
  target が存在するとき: 何もしない（ユーザー管理に委ねる）

subpath がファイルの場合:
  target が不在のとき: 親ディレクトリを作成しネイティブコピー（mode 保存 + owner-write 付与。<src>/<subpath> → <root>/<target>）
  target が存在するとき: 何もしない
```

- **foreign 実ファイルの skip は warning で可視化する**: target が存在し**かつ前世代 manifest にこの copy entry が無い**（= nput が置いていない foreign ファイル）のとき、place-once により skip するが **warning を出す**（「target に既存ファイルがあり copy をスキップした」）。symlink の foreign 警告（→ ADR-0015）と対称化し、「nput が中身を置いた」と誤認する masking を防ぐ。上書きはせず apply 全体も止めない（→ ADR-0022）。「自分が置いたか」は前世代 manifest の entry 有無で判別し、内容は判別しない。
- **place-once**: 初回マテリアライズ後、target が在れば触らない。ストア更新の反映は **`apply --recopy`（全 copy target を src から無条件上書き）**、または `reset <name> [target]` で撤去後に再 apply で行う（→ ADR-0020）。
- **mode は保存しつつ owner-write を付与する**（例: `0444 → 0644` / `0555 → 0755`・→ ADR-0016）。store パスは read-only（0444 / 0555）のため、そのまま保存すると編集できない。copy は「store から切り離してユーザーが所有・編集する」用途なので、perm の相対構造（実行ビット・group/other）は保ちつつ所有者が編集できる状態にする。
- **src ツリー内の symlink は symlink のまま複製する**（deref しない。循環・サイズ膨張回避・→ ADR-0016）。ただし store 内への絶対 symlink を複製すると **store 依存（read-only / GC 後 dangling）が残る**点に注意。相対 symlink はそのまま保つ。
- 世代管理の対象外。ロールバックされない。
- `subpath` がディレクトリのとき `target` に通常ファイルが存在する場合、または `subpath` がファイルのとき `target` がディレクトリの場合は、構造の不一致としてエラーで停止。

### out-of-store symlink

- symlink として配置する。指す先は marker の絶対パス（ローカル FS）。
- 世代では「どの絶対パスを指すか」のリンク先マッピングのみ版管理する。指す先の内容は設計上ライブで、永遠にスナップショットしない（→ ADR-0002）。

### recopy（`apply --recopy`）・reset（`nput reset`）（→ ADR-0020）

place-once（copy は触らない）と保守的 stale 除去（copy は消さない）の既定を、ユーザー責任で明示的に破る 2 経路。

**recopy（`apply --recopy`）** — copy target を src 更新へ追従させる。

```
config 内の各 copy entry について:
  target が存在 → 削除してから <src>/<subpath> を再コピー（mode 保存 + owner-write 付与・symlink 複製は通常 copy と同じ）
  target が不在 → 通常の place-once コピー
```

- 全 copy entry を**無条件**に上書き（差分判定なし）。ローカル編集は破棄され src 内容に戻る。上書き target をレポート表示。
- symlink 部の通常 apply（stale 除去 + 世代コミット）は同時に行う。copy は世代外のまま（世代を増やさない）。

**reset（`nput reset <name> [target...]`）** — 配置物を無い状態へ戻す FS-only teardown。

```
対象 entry（target 省略で全 entry・指定でその entry）について:
  method = symlink → 保守的不変条件（nput 管理・記録通りのみ）を満たす symlink を削除。foreign symlink は warning で残す
  method = copy    → target を削除（copy を消す唯一の明示手段。事前存在ファイルを消すリスクは確認で守る）
profile / 世代は触らない（FS のみ）。
```

- データ損失リスク（copy のユーザー編集・事前存在ファイル）のため**確認プロンプト**を出すか `--yes` を要求。削除 target をレポート表示。確認プロンプトは **stdin が TTY のときのみ**出す。**非 TTY（CI / direnv / パイプ）かつ `-y/--yes` 未指定なら、プロンプトを出さず即エラー停止（exit 1）**する（ハングと空入力誤削除を防ぐ・→ ADR-0025）。
- profile / 世代は不変。config が entry を残す限り**次の apply で再配置される**（transient・project mode は ADR-0017 の lstat 検査で復帰）。
- 恒久除去は config から entry を消して apply、profile 完全除去は `nix-env --profile <profileDir>/profile --delete-generations`。home / project 両モード可。
- **名指し必須（`--all` 非対応）**・**blocking flock を取得**（`--dryrun` は読み取り専用で flock を取らない）。`--dryrun` は削除対象を表示して exit（FS 削除・confirm なし・→ ADR-0021）。

---

## 世代管理仕様

→ `docs/adr/0002`

### 機構

- 純粋関数 `lib.mkManifest` が **link farm derivation**（`manifest.json` + 配置元への symlink ツリー）を生成する（→ ADR-0006）。
  「配置したもの」のマニフェストは `manifest.json` として link farm の一部に **store 内に**埋め込む（store 外の可変 JSON は持たない。`manifest.json` は不変）。
- **engine**（Go ライブラリ）が実行時の副作用として（→ ADR-0006, ADR-0007）:
  0. 解決後 `profileDir` 単位の flock を取得（明示 apply は blocking wait / shellHook は try-lock skip・衝突時は後勝ち・→ ADR-0013）。
  1. **前世代の store マニフェスト**（`manifest.json`）と新世代を diff し、消えた entry の **symlink を除去**（stale 除去）
     - 前世代は **全モード共通で nput 自身の profile の前世代**から読む（standalone も module も同一。ホストの oldGenPath には依存しない）
  2. symlink / out-of-store / place-once copy を**ネイティブ FS 操作**で配置（`ln` / `rsync` は使わない。新規・張替を先に、stale 除去を最後に）
  3. 全て成功してから `nix-env --profile <profileDir>/profile --set <link-farm-drv>` で nput の nix profile を更新（コミット点・全モード・→ ADR-0025）
  - project mode は新 link-farm が前世代と同一なら新世代を積まない（世代スキップ・3 の `--set` を省く）。ただし lstat 検査でドリフトした entry だけ再張りする（完全 no-op にしない・→ ADR-0017）。途中失敗は 3 に到達せず前世代を保つ（部分失敗のコミット最後・→ ADR-0006）。

配置・cleanup アルゴリズムは home-manager の `linkGeneration`/`cleanup` を参考に Go で再実装する（`home.file` 自体は再利用しない）。`nix` / `git` 以外はサブプロセスを使わない（配置エンジン層）。

**nput は全モードで自前 profile を持つ**（→ ADR-0002）。standalone では profile をユーザー向け rollback に使う。
module（HM/NixOS/darwin）では profile を**内部機構**（前世代マニフェスト + stale 追跡）に留め、ユーザー向け rollback は
host（`home-manager --rollback` 等）に一本化する。`nput rollback <name>` は home mode 限定。host rollback は旧 config を
再 activate して nput を再 kick することで自動追従する（nput profile は前進のみ＝旧内容の新世代を積む）。

| 機構 | 役割 | 適用層 | 位置（確定）|
|---|---|---|---|
| 世代由来の store マニフェスト | stale 除去のための前回状態（不変・GC-root 済み）| 全層共通 | `manifest.json` として link farm derivation 内に埋め込み（→ ADR-0006）|
| nput の nix profile | 前世代の保持・世代番号・GC root | 全モード（standalone はユーザー向け / module は内部）| `profileDir`（下記レイアウト）。home（`--root` なし）: `<state>/nix/profiles/nput/<name>/` / project・fixed・`--root` 上書き: `<state>/nix/profiles/nput/<roothash>/<name>/` / HM モジュール経由: `<state>/nix/profiles/nput/default/`（MVP は固定名 1 profile・→ ADR-0024）（→ ADR-0005, ADR-0022, ADR-0024, ADR-0025）|

> profile の基底 `<state>` は **`$XDG_STATE_HOME` があればそれ、無ければ `~/.local/state`**（nix 本体の profile 既定と整合・→ ADR-0022）。
> `<roothash>` / backref / flock キーはこの確定パスを基準にする（→ ADR-0005, ADR-0013）。

#### profile のオンディスクレイアウト（→ ADR-0025）

`profileDir` は**各 config 専用のディレクトリ**であり（= flock キー）、その中に profile リンク・世代・build out-link を置く。`profileDir` の**キー**（`<name>` / `<roothash>/<name>`）は ADR-0023 §「root の解決」・ADR-0024 で確定済みで、本レイアウトは「キーで指す先がディレクトリで、profile リンクはその中の `profile`」という物理形を定める。

```
<state>/nix/profiles/nput/<roothash>/.root              # backref（roothash 階層・複数 <name> で共有・→ ADR-0013）
<state>/nix/profiles/nput/<roothash>/<name>/             # ← profileDir（flock キー）
<state>/nix/profiles/nput/<roothash>/<name>/profile        # profile リンク（nix-env --profile <profileDir>/profile の対象）
<state>/nix/profiles/nput/<roothash>/<name>/profile-N-link # 世代（nix-env が profile の兄弟に作成）
<state>/nix/profiles/nput/<roothash>/<name>/.pending       # nix build --out-link（profile を貫通しない兄弟）
# home（--root なし）: <state>/nix/profiles/nput/<name>/{profile, profile-N-link, .pending}
```

- profile 操作は `nix-env --profile <profileDir>/profile ...`、build は `nix build --out-link <profileDir>/.pending`。世代兄弟 `profile-N-link` と `.pending` が profile リンクを貫通せず兄弟として並ぶため、read-only な store パスを貫通する破綻が構造的に起きない。
- **pending out-link は専用ディレクトリ内に 1 個なので名は `.pending`**（ADR-0022/0023 の `.pending-<name>` を改訂。`<name>` 次元はディレクトリ階層が表す）。`--set` 前失敗で残る pending は config あたり最大 1（→ ADR-0016）。
- **flock キー = profileDir（専用ディレクトリ）**。同一 config の apply / reset / rollback を直列化し、同 roothash でも別 `<name>` dir とは独立する。
- **backref `.root` は roothash 階層**（`<name>` dir の親）に置き、複数 `<name>` で共有する（孤児 profile 逆引き seam・→ ADR-0013）。

### stale 除去の対象と安全不変条件

削除は保守的に行う（→ ADR-0002）。前世代マニフェストが「nput が配置した」と記録し、
**かつ現状もその記録通りの先（その世代の store パス／記録された out-of-store パス）を指す symlink** のみ削除する。
通常ファイルや nput 非管理の link には触れない。記録と実体が不一致なら削除せず警告する。初回／記録なしは何も消さない。

| 配置種別 | entry が消えたとき |
|---|---|
| symlink（store / out-of-store）| **除去する**（ただし上記の保守的不変条件を満たすもののみ）|
| copy | **除去しない**（ユーザー所有データ）。ただし orphan を警告で通知する |

### GC とストレージ解放

profile の各世代は GC root。`nix-env --profile <profileDir>/profile --delete-generations <gens>` で旧世代を間引き、
`nix-collect-garbage` で無参照になった store パスを解放する（`<profileDir>` のレイアウトは「機構」節・→ ADR-0025）。
（可変 JSON 方式は GC root を作らず参照中 store パスが GC で消えるため採らない。）

> **世代操作は `nix-env --profile <profileDir>/profile` 系で統一する**（→ ADR-0015, ADR-0025）。コミット（`--set`）・rollback（`--rollback`）・
> 任意世代切替（`--switch-generation`）・一覧（`--list-generations`）・間引き（`--delete-generations`）。store GC のみ `nix-collect-garbage`。
> 新 `nix profile`（`list` / `wipe-history` / `rollback`）は profile 直下に自身の profile-manifest を要求するが、`nix-env --set` 製
> profile は profile 直下が link-farm（nput の `manifest.json` を含む）なので **`nix profile` サブコマンドとは非互換**。`<profileDir>` は config 専用ディレクトリで profile リンクは `<profileDir>/profile`（→ ADR-0025）。

### ロールバック

- **standalone（home mode）**: `nput rollback <name>` で前世代に戻す。nput は profile dir 自体ではなく任意 root に配置するため、
  profile ポインタ移動だけでは FS が変わらず**再配置が必須**。stale 除去の diff は次の基準・順序で行う（→ ADR-0015）:
  1. `baseline` = 現世代 N の manifest（FS の現状）／ `target` = 戻る世代 N-1 の manifest。
  2. `N ∖ N-1` の entry を保守的に stale 除去 → N-1 の entry を place/replace。
  3. **最後に** `nix-env --profile <profileDir>/profile --rollback`（または `--switch-generation N-1`）で profile ポインタを移す。
  ポインタを先に動かすと baseline が N-2 へずれ stale 除去が誤るため、FS 収束を先に・ポインタ移動を最後にする。apply エンジンを
  `(baseline, target)` 差し替えで再利用する。任意世代切替・世代 GC は `nix-env --profile <profileDir>/profile` 系 / `nix-collect-garbage` を使う。
- **module**: `nput rollback` は公開しない。ユーザー向けロールバックは host（`home-manager --rollback` 等）に一本化する。
- **project mode**: rollback は公開しない（ephemeral・→ ADR-0005）。

### モジュール時

nput は自前 profile を**持つ**が、それは前世代マニフェスト + stale 追跡のための内部機構に留める（→ ADR-0002）。
ユーザー向けロールバックは host に一本化し、host rollback は旧 config を再 activate して nput を再 kick することで
FS を自動収束させる（nput profile は前進のみ＝旧内容の新世代を積む）。host から nput へ要求するのは「switch 時の kick」だけで、
ホストの oldGenPath 配管は不要。

### project mode の世代（→ ADR-0005）

- **profile は解決済み root でキーする**（例: `<state>/nix/profiles/nput/<roothash>/<name>`・`<state>` は `$XDG_STATE_HOME` か `~/.local/state`・→ ADR-0022）。同一 entrypoint を複数箇所に
  クローンしても profile が衝突せず、stale 除去が互いのクローンの配置を破壊しない。home mode（1 ユーザー 1 つ）では起きない問題。
  `<roothash>` は **解決後の絶対 root パスの sha256 を短縮した hex**（固定長・FS 安全）。`<roothash>` 階層に **元 root の絶対パスを
  記録した backref ファイル**（例: `.root`）を置き、孤児 profile の逆引きを可能にする（→ ADR-0013）。flock のキーもこの解決後 profileDir。
- **世代はユーザーに公開しない**。profile は stale 除去 + 世代スキップ判定の内部機構に留め、`rollback` / `list-generations` を出さない。
  配置物が ephemeral で rollback の意味が薄く、devShell キック時は戻し先 host 世代も無いため。
- **世代スキップ短絡（必須）**: 新 link farm derivation が前世代と同一なら**新世代を積まない**。
  devShell / direnv 運用では `shellHook` がシェル再入のたびに走るため、毎回新世代を積むと世代が無限増殖する。
  home mode は従来通り「適用のたびに新世代」のまま（世代スキップは project mode 限定）。
- **ただし世代スキップ時も FS 検査だけは軽量に行う**（完全 no-op にしない・→ ADR-0017）。各 entry の target を **lstat で検査**し、
  記録通りでない（foreign tool に書き換えられた・消えた）entry があればその entry **だけ**再張りする（foreign symlink なら warning・→ ADR-0015）。
  「derivation 同一 ⇒ FS 同一」は foreign 書き換えで崩れるため、新世代を積まずに FS だけ収束させる。lstat 比較は安価で `shellHook` 高頻度実行に耐える。
  - **cross-config 同一 target の振動はユーザー責任**（→ ADR-0023）: 別 config A / B が同一 target を狙うと、この lstat 修復が「A が置く → B が foreign 検知して奪う → A が再奪取」と `shellHook` 再入のたびに振動しうる（単発「後勝ち」ではなく能動的オシレーション）。「同一 target を複数 config で狙わない」をユーザー責任とし、foreign symlink warning（→ ADR-0015）で可視化する。検知して止める機構は持たない。
  - **振動中の foreign warning は `shellHook` 高頻度実行で出続けるが、これは設定ミスのシグナルとして正しい**（→ ADR-0024）。warning は `--quiet` 対象外なので `--quiet` でも消えない。抑制 / 集約機構は MVP で持たず、config の同一 target 重複を解消して直す（document-only）。
  この lstat ドリフト修復は **symlink と copy の両方**を対象にする。**copy entry は target が不在のときだけ** place-once で再マテリアライズし、
  **存在するが内容が異なる（ユーザー編集）場合は触らない**（home mode の place-once と振る舞いを一致させる。src 追従は `apply --recopy` 限定・内容ハッシュ比較はしない・→ ADR-0022）。
- **orphan profile**: クローンを削除すると profile が `<state>/nix/profiles/nput/` 下に孤児として残る。store は `nix-collect-garbage` で解放されるが
  profile ディレクトリは残る。放置許容（または手動削除）とし、公開ドキュメントに注記する（→ ADR-0005）。どの root 由来の孤児かは
  `<roothash>` 階層の backref ファイル（`.root`）で逆引きできる（→ ADR-0013）。**cleanup コマンドは MVP では持たない**（実害が小さく `.pending` も config あたり最大1）。backref があるので**将来 `nput prune`（実在しない root を指す孤児系列を逆引きして削除）を実装できる seam** を残す（消費側の要求が出た時点で追加・→ ADR-0024）。

---

## モジュールオプション仕様

### 共通オプション（全モジュール）

```
nput.enable  :: bool          # デフォルト: false
nput.entries :: attrsOf entry # デフォルト: {}（属性キー = target・→ ADR-0014）
```

モジュールは自分の性質で root を pin する（HM → `homeRoot` / devShell → `projectRoot`）ため、モジュール利用者は `root` を再指定しない（→ ADR-0003, ADR-0007）。

> **HM モジュールの profile 粒度（MVP）**（→ ADR-0024, ADR-0025）: `nput.entries` は**単一 attrset = 単一 manifest = 1 profile（固定名 `default`）**で、`<name>` 次元を持たない。**standalone / CLI が entrypoint の `nput.<name>` で複数の独立 profile（役割分離・個別 rollback）を持てるのに対し、HM モジュール経由では役割分離はできない**。役割ごとに分けたいユーザーは standalone CLI 経路を使う。HM の複数 profile 化（`nput.configs.<name>.entries` 等）は将来拡張の seam として残す（HM の低い positioning〔ADR-0007〕に鑑み MVP では背負わない）。

### NixOS / nix-darwin 追加オプション（将来拡張）

```
nput.user :: string       # 必須（配置先ユーザーの特定に使用）
```

home-manager と standalone は `$HOME` を参照するため `user` は不要。

---

## root の解決

target を絶対パスに変換する基準（root）の解決方法。配置の実体は全層で engine が実行する（→ ADR-0003）。
root は `mkManifest` の `root` 引数で**明示必須**に選ぶ（暗黙デフォルトなし・→ ADR-0004 改訂, ADR-0005, ADR-0007）。

### home mode（`root = homeRoot`）

| 層 | root の解決方法 | 備考 |
|---|---|---|
| standalone | `$HOME`（実行時のシェル環境変数）| OS 問わず動作 |
| home-manager | `$HOME`（HM が内部解決・モジュールが `homeRoot` を pin）| OS 問わず動作 |
| NixOS（将来）| `config.users.users.${cfg.user}.home` | `isNormalUser = true` で `/home/<user>` |
| nix-darwin（将来）| `config.users.users.${cfg.user}.home` | デフォルト値なし。明示設定が必須 |

### project mode（`root = projectRoot`）

| 解決方法 | 備考 |
|---|---|
| `git rev-parse --show-toplevel`（既定）| どのサブディレクトリから実行しても同じ root に解決される |
| `--root <path>`（上書き）| git 外で使う場合や別ルートを指す場合に明示 |

- config ファイル相対は採らない（Nix で flake source が store にコピーされ store path 化するため成立しない、→ ADR-0005）。
- CWD（`$PWD`）相対は採らない（実行場所で配置先がズレ冪等性を壊すため、→ ADR-0005）。
- `--root` は **project mode に限らず全モードの解決 root を一律上書き**する（home mode の `$HOME` / fixed root も対象・デバッグ / テスト / 特殊配置の脱出路・→ ADR-0017）。
- **`--root` 明示時は全モードで profileDir を上書き後 root の `<roothash>` でキーする**（→ ADR-0023）。home / fixed mode も `<state>/nix/profiles/nput/<roothash>/<name>` になり、異なるオーバーライド root が独立した世代系列に分離されて silent orphan を防ぐ。`apply` / `reset` / `rollback` / `list-generations` で一貫し、`--root` を付けた世代を操作するには再び同じ `--root` が要る。`--root` なしの通常 home は従来どおり `<name>` キー（1 ユーザー 1 profile）。`<roothash>` 算出・backref（`.root`）は project mode と同一機構（→ ADR-0013）。
- **fixed root mode（`root` に絶対パス文字列・`--root` なし）も常に `<roothash>/<name>` でキーする**（→ ADR-0024）。fixed root は評価時確定の任意絶対パスなので project / `--root` 上書きと同じく root ごとに独立系列へ分離し、別 root の同名 config が世代系列を共有する silent orphan を構造的に防ぐ。`<name>` 直キーは「1 ユーザー 1 profile」が成立する home（`--root` なし）に限る。

  | 状況 | profileDir |
  |---|---|
  | home（`--root` なし）| `<state>/nix/profiles/nput/<name>` |
  | home / fixed（`--root /p`）| `<state>/nix/profiles/nput/<roothash(/p)>/<name>` |
  | fixed（`--root` なし・`root = "/abs"`）| `<state>/nix/profiles/nput/<roothash(/abs)>/<name>` |
  | project（`--root` 有無）| `<state>/nix/profiles/nput/<roothash>/<name>` |
  | HM モジュール経由（MVP）| `<state>/nix/profiles/nput/default`（固定名 1 profile・→ ADR-0024）|

  > 上表の各値は **profileDir（config 専用ディレクトリ）**であり、profile リンクはその中の `<profileDir>/profile`、世代は `profile-N-link`、build out-link は `.pending`、backref `.root` は `<roothash>` 階層に置く（レイアウト詳細は「世代管理仕様」の「機構」節・→ ADR-0025）。
- project mode の配置物は **ephemeral**（コミット対象外）。activation は `.gitignore` に触れず git 状態に干渉しない（→ ADR-0005）。

### system mode（`root = systemRoot`・将来）

- root = `/`。distro 構想の system 配置 seam（→ ADR-0004, ADR-0007）。今回の実装スコープ外。

### ローカルパス（out-of-store）の扱い

`mkOutOfStoreSymlink` の引数は Nix 評価時に確定する絶対パス。`$HOME` は使えない。
`target` の root 解決は実行時に行われるため `target` 側には影響しない。

```
Nix 評価時:  mkOutOfStoreSymlink "/path/to/dotfiles"  →  manifest にハードコード
実行時:      root を engine が解決                       →  target を絶対パス化
```

macOS / Linux でホームの慣例が異なるため、ローカルパスは flake 内で OS 判別して解決するのが推奨。

```nix
let
  username = "<username>";
  homeDir  = if pkgs.stdenv.isDarwin then "/Users/${username}" else "/home/${username}";
  dotfiles = "${homeDir}/dotfiles";
in
nput.lib.mkManifest {
  root = nput.lib.homeRoot;
  entries = {
    ".config/nvim" = { src = nput.lib.mkOutOfStoreSymlink dotfiles; subpath = "home/.config/nvim"; };
  };
}
```

`builtins.getEnv "HOME"`（`--impure` 必要）や flake の `specialArgs` 注入も使えるが、通常は上の OS 判別で十分。

---

## モジュール別動作仕様

基本的な利用は **project mode（プロジェクト内配置）と standalone CLI** を中心に考える（→ ADR-0007）。モジュール対応は、他のモジュールシステムの switch と**一括で動いてほしいユースケース**を拾うためだけに存在する。全モジュール（HM / NixOS / nix-darwin）と devShell は **一律「nput エンジンをキックするだけ」の配線**であり、各層は root と activation タイミングを供給するだけ（→ ADR-0003）。
`systemd.tmpfiles` / `home.file` へは翻訳しない。配置・stale 除去は全層で同一の engine + store マニフェスト。

### standalone（CLI・→ ADR-0007）

`nput apply <name>` をユーザーが明示的に実行する。CLI が entrypoint（CWD 既定 / `-f`）を発見し manifest をビルドしてエンジンを駆動する。
`root = homeRoot` の home mode では nix profile による世代管理を行い、`rollback <name>` / `list-generations <name>` を提供する。

### home-manager モジュール

| method | `src` | 動作 |
|---|---|---|
| `"symlink"` | path / set | engine が store link をネイティブ symlink |
| `"symlink"` | marker | engine が out-of-store symlink をネイティブ作成（HM の mkOutOfStoreSymlink には委譲しない）|
| `"copy"` | path / set | engine が place-once ネイティブコピー |

- `home.activation.nput`（`entryAfter ["writeBoundary"]`）から engine を起動する。配置ロジックは HM に依存しない。root は `homeRoot` を pin。
- engine kick は **`nput apply --manifest <link-farm>`**（→ ADR-0026）。モジュール評価時に `nput.entries` から `mkManifest` で link-farm をビルドし、その store パスを activation script から渡す。activation は `nix eval` / `build` を行わない（entrypoint 経路ではない）。**blocking flock・配置レポート可視・engine error（conflict 等）は非 0 終了で switch を止める**（宣言的 switch・`home.file` の clobber エラーと同型）。pin 版 nput CLI（`packages.nput`）と `mkManifest` が同一 flake input 由来のため schemaVersion skew は構造的に起きない。
- 世代は nput 自前 profile（内部機構・前世代マニフェスト + stale 追跡）に乗る。MVP は固定名 `default` の単一 profile（`<state>/nix/profiles/nput/default`・→ ADR-0024, ADR-0025）。ユーザー向け rollback は host（`home-manager --rollback`）に一本化（`nput rollback` 非公開）。

### NixOS モジュール（将来拡張）

- `system.activationScripts.nput` から engine を起動する**配線**に徹する。`systemd.tmpfiles` へは翻訳しない。
- root は `config.users.users.${cfg.user}.home`（`homeRoot` 相当を host が供給）。nput は自前 profile を内部機構として持ち、ユーザー向け rollback は nixos 世代に一本化。
- 配置・stale 除去は他環境と同一の engine + store マニフェスト。OS の機構（tmpfiles 等）は nput の関心外（→ ADR-0003）。

### nix-darwin モジュール（将来拡張）

- `system.activationScripts.nput` から engine を起動する。
- root は `config.users.users.${cfg.user}.home`（明示設定が必須）。nput は自前 profile を内部機構として持ち、ユーザー向け rollback はホスト世代に一本化。

### devShell（配線・→ ADR-0005, ADR-0007）

`devShells.<name>` の `shellHook` から engine を起動する配線。HM モジュールと同型で、配置ロジックは持たず
root（project mode の git toplevel）と activation タイミング（シェル入室）を供給するだけ。root は `projectRoot` を pin。

```nix
devShells.default = pkgs.mkShell {
  shellHook = ''
    nput apply skills --no-wait
  '';
};
```

- `nix develop` / direnv（`use flake`）でシェルに入った瞬間にキックされ、git toplevel を root にプロジェクト内へ配置する。
- `shellHook` は高頻度で走るため、project mode の**世代スキップ短絡**（変更なしなら no-op）が前提（→「project mode の世代」）。

---

## エラー仕様

評価時エラー（`root` 省略・不正な型・未知キー・copy+marker・systemRoot 未実装・絶対パス / `..` エスケープ）は `mkManifest` の `evalModules` / `lib.throwIf` が検出する（→ ADR-0010, ADR-0013, ADR-0019）。識別子の一意性は entries が attrset のため Nix のキー重複不可で担保され、重複 `name` という評価時エラーは存在しない（→ ADR-0014）。

| 条件 | 動作 |
|---|---|
| `root` を省略（暗黙デフォルトなし）| Nix 評価時にエラー（`evalModules` の "option used but not defined"）|
| `src` に素の文字列を渡す（store パス文字列含む）| Nix 評価時にエラー（`srcType` が文字列を拒否・out-of-store は marker で opt-in・→ ADR-0001）|
| entry に未知キーがある（タイポ・旧名 `source` / `dir`）| Nix 評価時にエラー（submodule が strict・→ ADR-0008, ADR-0010）|
| `target` / `subpath` が絶対パス（`/` 始まり）| Nix 評価時にエラー（target は root 相対・subpath は src 内相対・→ ADR-0019）|
| `target` / `subpath` が `..` で root / src の外へ出る | Nix 評価時にエラー（`filepath.Clean` 相当で正規化し検出・→ ADR-0019）|
| `entries = {}`（空 manifest）| 正当な全クリア。前世代の全 nput symlink を保守的 stale 除去し新世代は空（警告なし・→ ADR-0019）|
| `src` が存在しないストアパス（path / set）| Nix 評価時にエラー |
| `src` が marker でローカルパスが存在しない | engine 実行時にエラーで停止 |
| `subpath` が `src` 内に存在しないパス | engine 実行時にエラーで停止 |
| `target` に通常ファイル・ディレクトリが既存（symlink モード）| エラーで停止（上書きしない）|
| `target` の祖先 component が symlink（全体 symlink 配置の配下にネスト）| engine 実行時にエラーで停止（lstat walk・`--dryrun` は conflict・→ ADR-0015）|
| `target` に foreign symlink が既存（自身の前世代 manifest に記録なし・別 config / 別ツール / 手動）| warning を出して後勝ちで置き換える（→ ADR-0015）|
| `subpath` がディレクトリのとき `target` に通常ファイルが既存（copy モード）| エラーで停止 |
| `subpath` がファイルのとき `target` がディレクトリとして既存（copy モード）| エラーで停止 |
| copy モードで `target` が既存（前世代 manifest に entry あり = nput 配置済み）| place-once により何もしない（上書きしない）。`apply --recopy` 時のみ無条件上書き（→ ADR-0020）|
| copy モードで `target` に foreign 実ファイルが既存（前世代 manifest に entry なし）| place-once で skip しつつ **warning** で可視化（masking 防止・symlink の foreign 警告と対称・apply は止めない・→ ADR-0022）|
| 世代スキップ時に copy `target` が不在（project mode）| lstat ドリフト修復で place-once 再マテリアライズ。内容が異なるだけ（編集済み）の場合は不可触（→ ADR-0022）|
| stale 除去で copy entry が消えた | copy target は削除せず、orphan を警告で通知。明示撤去は `reset`（→ ADR-0020）|
| `reset` で対象 entry の配置物が既に無い | no-op（既に無い状態・エラーにしない・→ ADR-0020）|
| `reset` で target が foreign symlink（nput 非管理）| warning を出して残す（保守的・誤破壊しない・→ ADR-0020）|
| `reset` で copy target を削除（ユーザー編集 / 事前存在の可能性）| 確認プロンプト（`--yes` でスキップ）後に削除しレポート表示（→ ADR-0020）|
| `reset` を非 TTY（CI / パイプ）で `--yes` 無しで実行 | プロンプトを出さず即エラー停止（exit 1・ハングと誤削除を防ぐ・→ ADR-0025）|
| `target` の親ディレクトリが存在しない | `mkdir -p` で自動作成 |
| 同一 target（entries の同一キー）| Nix の attrset 仕様で表現不可（一意性を native 担保・→ ADR-0014）|
| 別キーを `target` 明示上書きして同一 target に衝突（同一 manifest 内）| Nix 評価時にエラー（`normalizeManifest` の `lib.throwIf`・正規化後 target 重複・→ ADR-0024）|
| 別 config（別 profile）が同一 target を狙う（cross-config）| engine 実行時の後勝ち + foreign symlink warning（eval では検出不可・→ ADR-0015）|
| `method = "copy"` かつ `src` が out-of-store marker | Nix 評価時にエラー（`lib.throwIf`・意図矛盾・→ ADR-0013）|
| `root = systemRoot`（system mode は未実装）| Nix 評価時にエラー（未実装・予定・→ ADR-0013）|
| `rollback` で前世代が存在しない | エラーメッセージを出力して停止 |
| `apply` で name 省略かつ `nput.default` が未定義 | CLI がエラーで停止（config 名を要求）|
| `gitignore <name>` で `<name>` が非 project config（home / fixed）| CLI がエラーで停止（`gitignore` は project mode 限定・→ ADR-0023）|
| `listFilesInSrc` の `src` に out-of-store marker を渡す | Nix 評価時にエラー（`src` は path 限定・→ ADR-0023）|
| `listFilesInSrc` の `src` に `set`（derivation）を渡す | Nix 評価時にエラー（`src` は path 限定・IFD 回避・→ ADR-0024）|
| `nput.<name>` が entrypoint に存在しない | CLI がエラーで停止 |
| entrypoint が発見できない（CWD に flake.nix/shell.nix/default.nix なし・`-f` 未指定）| CLI がエラーで停止 |
| `apply --manifest` と `-f`（または将来の `--all`）を併用 | CLI がエラーで停止（取得元の二重指定・→ ADR-0026）|
| nix-darwin（将来）で `users.users.<user>.home` が未設定 | Nix 評価時にエラー |
| project mode で git リポジトリ外かつ `--root` 未指定（git toplevel 解決失敗）| engine 実行時にエラーで停止 |
| project mode で `git` が PATH に無い | engine 実行時にエラーで停止 |
| `nix-command` / `flakes` experimental-features が未有効 | CLI が前提条件と有効化方法を案内して停止（`--extra-experimental-features` 自動付与はしない・→ ADR-0025）|

---

## 依存関係

| コンポーネント | 依存 |
|---|---|
| `lib/`（`mkManifest` / マーカー群 / `listFilesInSrc`）| nixpkgs.lib のみ（純データ生成。`rsync` 不要）。型検査に `lib.types` / `mkOption` / `evalModules`（nixpkgs.lib のコア）を使う（→ ADR-0010）|
| `lib/types.nix`（entry submodule / `srcType` / `rootType` / marker custom type）| nixpkgs.lib のみ（`modules/common.nix` と共有・→ ADR-0010）|
| `lib/out-of-store.nix`（`mkOutOfStoreSymlink` / `projectRoot` / `homeRoot` / `systemRoot` マーカー）| なし（`_nputMarker` タグ付きマーカー構築子・→ ADR-0010）|
| `internal/`（配置エンジン = 内部層分離。公開モジュールではない・→ ADR-0011）| **stdlib-only 厳守**（第三者依存ゼロ）。`syscall.Flock`・`filepath.WalkDir`+`io.Copy`+`os.Chmod`・`encoding/json`・`fmt.Errorf`+`%w`。`manifest.json` を入力に取り runtime に `nix`（profile）/ `git`（toplevel）をサブプロセスで要求（→ ADR-0006, ADR-0011）|
| `cmd/nput`（CLI = `packages.nput`）| 配置エンジンを import。最小依存を許可（**cobra** = サブコマンド / help、**fatih/color** = dryrun 色付け）。entrypoint 発見と `nix`（build / eval）オーケストレーションを担う。`buildGoModule` + **vendorHash 文字列**でビルド。Go は nixpkgs の go に pin し `toolchain` ディレクティブ不使用（→ ADR-0007, ADR-0011）|
| `modules/common.nix` | nixpkgs.lib のみ |
| `modules/home-manager.nix` | home-manager の module system（起動配線のみ）|
| `modules/nixos.nix`（将来）| NixOS の module system（起動配線のみ）|
| `modules/nix-darwin.nix`（将来）| nix-darwin の module system（起動配線のみ）|

lib 層は home-manager / NixOS / nix-darwin に依存しない。配置ロジックは Go エンジン（ライブラリ）が単一の源として所有する（→ ADR-0003, ADR-0006, ADR-0007）。

---

## E2E 検証範囲（非 NixOS・→ ADR-0012）

「非 NixOS でも nix さえあれば動く」主張を実 nix で検証する E2E ハーネス（`tests/e2e/`・bash・詳細は `tests/e2e/README.md`）が、flake entrypoint からの `nix build` / `nix eval` / `nix-env --set` を含む実経路を一気通貫で回す。各シナリオは隔離した一時 `$HOME` / `$XDG_STATE_HOME` 下で動き、偽 src は fixture flake 内の相対パス（eval 時に store へコピー）か out-of-store の live ディレクトリで用意する。

| シナリオ | 検証する仕様 |
|---|---|
| project mode | 一時 git repo で `nput apply <name>` → git toplevel 配下に store symlink 配置（「root の解決」project mode）・再 apply の冪等性 |
| home mode | 仮 `$HOME` で apply → `$HOME` 配下配置 + profile 世代コミット、entry 入替で世代を進め `nput rollback` で前世代の配置へ復帰（「世代管理仕様」・「ロールバック」）|
| stale 除去 | entry を config から削除 → 再 apply で旧 symlink が消える（「stale 除去の対象と安全不変条件」）|
| copy place-once / out-of-store | copy が通常ファイル（書込可・mode に owner-write 付与）・place-once 冪等（ローカル編集を破棄しない）・out-of-store marker の live symlink（「copy モード」・「out-of-store symlink」）|
| HM module | home-manager standalone configuration を非 NixOS で評価・activate し、activation が `nput apply --manifest` で engine を起動して配置する（「モジュール別動作仕様」home-manager モジュール）|

- **NixOS VM テスト（`runNixOSTest`）は将来拡張**。NixOS / nix-darwin モジュール経路の実 activate は VM / sandbox を要し、本ハーネス（非 NixOS の単一ジョブ）のスコープ外。モジュール経路を本実装する段で別途追加する（→ docs/design.md「テスト戦略」）。

---

## 使用例（フル）

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url    = "github:NixOS/nixpkgs/nixos-unstable";
    nput.url       = "github:yasunori0418/nput";

    # 各ソースを独立して管理（flake = false でリポジトリをそのまま取得）
    vim-plugin-foo.url        = "github:foo/vim-plugin-foo";
    vim-plugin-foo.flake      = false;
    zsh-autosuggestions.url   = "github:zsh-users/zsh-autosuggestions";
    zsh-autosuggestions.flake = false;
    claude-skills.url         = "github:someone/claude-skills";
    claude-skills.flake       = false;
  };

  outputs = { self, nixpkgs, nput, home-manager, ... }@inputs:
  let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in
  {
    # パターン1: project mode（repo 内に配置・devShell キック）
    nput.${system} = {
      skills = nput.lib.mkManifest {
        root = nput.lib.projectRoot;
        entries = {
          ".claude/skills/nix" = { src = inputs.claude-skills; subpath = "skills/nix"; };
        };
      };
    };

    devShells.${system}.default = pkgs.mkShell {
      packages  = [ nput.packages.${system}.nput ];   # pin 版 nput を PATH へ（project mode は同梱が canonical・→ ADR-0015）
      shellHook = "nput apply skills --no-wait";
    };

    # パターン2: home mode（標準的な dotfiles 配置・別 profile）
    # nput.${system}.vim-plugins = nput.lib.mkManifest {
    #   root = nput.lib.homeRoot;
    #   entries = {
    #     ".local/share/nvim/site/pack/foo/start/foo" = { src = inputs.vim-plugin-foo; };
    #     ".zsh/plugins/autosuggestions" = { src = inputs.zsh-autosuggestions; };
    #   };
    # };

    # パターン3: home-manager モジュール（root は homeRoot を pin・外部リポジトリ + ローカル dotfiles 混在）
    homeConfigurations.<username> = home-manager.lib.homeManagerConfiguration {
      inherit pkgs;
      modules = [
        nput.homeManagerModules.default
        {
          nput.enable = true;
          nput.entries = {
            ".claude/skills/nix" = { src = inputs.claude-skills; subpath = "skills/nix"; };
            ".config/nvim" = { src = nput.lib.mkOutOfStoreSymlink "/home/me/dotfiles"; subpath = "home/.config/nvim"; };
          };
        }
      ];
    };
  };
}
```

```bash
# standalone CLI（PATH 常駐）で役割ごとに独立して適用・ロールバック
nput apply skills                 # project mode（git toplevel に配置）
nput apply vim-plugins            # home mode（$HOME に配置）
nput rollback vim-plugins         # home mode 限定
nput list-generations vim-plugins
nput gitignore skills             # .gitignore 向け target を stdout
nput apply --all                  # entrypoint の nput.* を全適用

# 環境セットアップ（テンプレート展開）
nput init project                 # nix flake init -t <nput>#project のラッパー
```
