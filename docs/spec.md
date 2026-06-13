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
| `entries` | list of entry | — | 配置定義のリスト（後述）|
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
`manifest.json` は各 entry の解決済み配置元（store パス / out-of-store 絶対パス）・`subpath`・`target`・`mode`・root の kind（project / home / system / 固定パス）を記録する。
副作用は持たない（profile swap・FS 配置は engine の実行時責務）。

#### 入力検査（`evalModules` + `normalizeManifest`・→ ADR-0010）

`mkManifest` は内部で `lib.evalModules` を回して `entries` / `root` を検査・正規化する。型をオプションに書くだけだと検査が効くのはモジュール経路（`nput.entries`）のみだが、コアである CLI / entrypoint 経路（`mkManifest` 直呼び）でも検査を効かせるため `mkManifest` 自身が `evalModules` を回す。`lib.types` / `mkOption` / `evalModules` は `nixpkgs.lib` のコアなので「lib は nixpkgs.lib のみ依存」を満たす。

実装は 2 段に分かれる。

- **`normalizeManifest { entries, root } -> attrset`**: `evalModules` 検査・デフォルト適用（`subpath` → `"."` / `mode` → `"symlink"`）+ 重複 `name` の `lib.throwIf` チェック + 内部 marker タグ（`_nputMarker`）→ clean enum（`srcKind` / `rootKind`）変換を行う純データ関数。nix-unit / namaka の単体対象。
- **`mkManifest = args: derivation`**: `normalizeManifest` の出力を `manifest.json` に書き symlink farm を組む。

entry の型定義（`lib/types.nix` の submodule）は `modules/common.nix` の `listOf (submodule …)` と共有する。モジュール経路では host の `evalModules`（`listOf`）と `mkManifest` の `evalModules` で二重に検査されるが、純粋・冪等で害はなく、`mkManifest` を「entries が必ず通る単一の検査ゲート」に保つ。

> **marker のタグ方式**: `src` の `set`（derivation）と marker（`mkOutOfStoreSymlink`）はどちらも attrset で構造判別できないため、marker には `_nputMarker` 判別タグを持たせ custom optionType の `check` で判別する。`_nputMarker` は Nix 評価内で完結させ `manifest.json` には漏らさない（Go 契約は `srcKind` / `rootKind` の clean enum・→ ADR-0010）。

**使用例（entrypoint への公開）**

```nix
# flake.nix
outputs.nput.${system}.dotfiles = nput.lib.mkManifest {
  root = nput.lib.projectRoot;
  entries = [
    { name = "vim-foo";  src = inputs.vim-foo;  target = ".local/share/nvim/site/pack/foo/start/foo"; }
    { name = "zsh-sugg"; src = inputs.zsh-sugg; target = ".zsh/plugins/autosuggestions"; }
  ];
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
| `src` | path | — | フェッチ済みのストアパス |
| `subpath` | string | `"."` | リポジトリ内のディレクトリパス。`readDir` するため**ディレクトリ限定**（非 dir はエラー）。entries の `subpath` と同名（旧称 `dir`・→ ADR-0008）|

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

`nput` CLI（`packages.nput`）は PATH に常駐する一次 UX。`nix profile install github:<owner>/nput` 等で導入する。

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
nput apply --all               # entrypoint の nput.* を全て順に適用
nput rollback <name>           # 前世代へ戻す（home mode 限定）
nput list-generations <name>   # 世代一覧を表示（home mode 限定）
nput gitignore <name>          # 配置 target を .gitignore 向けに stdout 出力（書き込みなし）
nput init <template>           # nix flake init -t <nput>#<template> のラッパー（後述）
```

**グローバルフラグ**

```bash
-f, --file <path>   # entrypoint を明示（自動探索を上書き）
--root <path>       # project mode の root を明示上書き（git toplevel を使わない）
```

- `apply` の **name 省略時は `nput.default` を適用**する（flake の `default` 慣例に倣う。`default` が未定義ならエラー）。`<name>` を明示すればその config を、`--all` で `nput.*` 全てを適用する。profile は config 単位で atomic（→ ADR-0002）。
- `rollback` / `list-generations` は **home mode 限定**。project mode は世代を内部機構に留めユーザーに公開しない（→ ADR-0005）。
- `apply <name> --dryrun` は FS 書込・`--set`・flock いずれも行わない読み取り専用。`conflict` があれば非ゼロ終了（CI の事前 gate に使える）（→ ADR-0006）。
- `gitignore <name>` は配置 target を `.gitignore` 向けに列挙して stdout に出力するだけで、ファイルは書き込まない。更新責務はプロジェクト管理者（→ ADR-0005）。
- 透明性: `nput --help` 等で内部実行する nix コマンドを開示し、ユーザーが選択的に手で実行できる（→ ADR-0007）。
- 任意世代への切替・世代の GC は標準の `nix-env` / `nix-collect-garbage` を profile パスに対して使う。
- `--only`（一部 entry だけ適用）は profile 世代の atomic 性と衝突するため提供しない。選択的更新は config（`nput.<name>`）の分割で担保する。

### `nput init`（テンプレート展開）

```bash
nput init standalone   # nix flake init -t <nput>#standalone のラッパー
nput init project      # nix flake init -t <nput>#project のラッパー（devShell shellHook 配線 + .gitignore ガイド入り）
```

- `nix flake init -t <nput>#<template>` への**透明なラッパー**。ファイルを作るのは nix の templates 機構であり nput 自身は generate しない（「設定を生成しない」thesis を維持・→ ADR-0007）。
- `nix flake init` の「既存ファイルを上書きしない」保守性を継承する。

### 再現性スタンス（→ ADR-0007）

| entrypoint | eval | 再現性 |
|---|---|---|
| `flake.nix` | pure（root 解決はエンジン実行時なので eval は pure のまま）| flake.lock で固定 |
| `shell.nix` / `default.nix` | impure（NIX_PATH / channels 依存）を許容 | **ユーザー責任**。nput lib を含め nixpkgs を npins / fetchTarball / flake-compat 等で固定することを推奨 |

### 実行フロー

```
nput apply <name> [-f <ep>] [--root <p>]
  0. entrypoint 発見（-f 上書き）
  1. nix build <ep>#nput.<system>.<name>（または -A nput.<name>）→ link-farm store path
  2. engine を駆動:
     a. flock(profileDir) を try-lock（保持中ならスキップ）
     b. root 解決（manifest の kind: project=git rev-parse / home=$HOME / system=/ / 固定パス、--root 上書き）
     c. profileDir の前世代 manifest.json を読む（無ければ初回 = 削除対象ゼロ）
     d. project mode かつ新 link-farm が前世代と同一なら no-op で終了（世代スキップ）
     e. manifest.json を新旧 diff → 新規/張替を配置 → 保守的 stale 除去（ネイティブ FS）
     f. nix-env --profile <profileDir> --set <link-farm>（サブプロセス・コミット点）
```

---

## entries スキーマ仕様

### フィールド定義

```
entry :: {
  name   : string              # 必須
  src    : path | set | marker # 必須（type/marker によって挙動が変わる）
  subpath: string              # 省略可、デフォルト: "."（省略 = リポジトリ全体）
  target : string              # 必須
  mode   : "symlink"
         | "copy"              # 省略可、デフォルト: "symlink"
}
```

各フィールドは `lib/types.nix` の entry submodule（`lib.types`）として定義され、`mkManifest` の `evalModules` が検査・デフォルト適用する（→ ADR-0010）。submodule は **strict**（未知キー拒否）で、タイポや旧名（`source` / `dir`・→ ADR-0008）は評価時エラーになる。

#### `name`

- **型**: string
- **必須**: yes
- **説明**: エントリの識別子。同一 `entries` リスト内で一意であること。

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

- **型**: string / **必須**: yes
- **説明**: 配置先パス。**root**（`mkManifest` の `root` で明示選択した基準）からの相対パスで指定する。

#### `mode`

- **型**: `"symlink"` | `"copy"` / **必須**: no / **デフォルト**: `"symlink"`

| mode | `src` の種別 | 動作 | 世代管理 |
|---|---|---|---|
| `"symlink"` | path / set | Nix ストアへの symlink（読み取り専用）| あり（profile）|
| `"symlink"` | marker | ローカルパスへの out-of-store symlink（ライブ）| あり（リンク先のみ）|
| `"copy"` | path / set | place-once コピー（書き込み可・ユーザー管理）| **なし** |
| `"copy"` | marker | 非推奨（out-of-store は symlink で使う）| — |

### entries を動的に生成する（`subpath` / `target` の string interpolation）

`entries` は素の Nix の list。`src` / `subpath` / `target` はいずれも通常の文字列・式なので、Nix の `map` / `listToAttrs` と string interpolation で動的に組み立てられる。`target` を `src` に対して動的に決めたい場合の定石を示す。

#### 基本形：共有 name を補間する

配置先ディレクトリ名を、`src` 選択と同じ name 変数から導く。`name` / `src` / `target` を 1 つの変数で串刺しにする。

```nix
# nvim プラグインを名前リストから一括生成
let
  plugins = [ "telescope" "treesitter" "cmp" ];
in
nput.lib.mkManifest {
  root = nput.lib.homeRoot;
  entries = map (n: {
    name   = n;
    src    = inputs.${n};
    # subpath 省略 = リポジトリ全体
    target = ".local/share/nvim/site/pack/plugins/start/${n}";
  }) plugins;
}
```

#### ⚠️ アンチパターン：`src` の store パスから名前を導く

`baseNameOf src` で配置先名を作るのは**罠**。`src`（flake input / `fetchFromGitHub`）は `/nix/store/<hash>-source` のような **hash 前置の store 名**に解決されるため、`baseNameOf` は `<hash>-source` を返し target ディレクトリ名には使えない。

```nix
# NG: target が ".config/abcd1234...-source" のようになる
{ name = "foo"; src = inputs.foo; target = ".config/${baseNameOf inputs.foo}"; }
```

配置先名は **`src` の store 名ではなく、ユーザーが制御する name 変数から導く**（上の基本形）。

#### 応用：`listFilesInSrc` で subdir を列挙して展開する

リポジトリ内の subdir を `listFilesInSrc` で走査し、各 subdir を 1 entry に展開する。`subpath` と `target` の両方を列挙した名前で補間する。

```nix
let
  # claude-skills/skills 配下の dir 名を列挙
  skills = nput.lib.listFilesInSrc { src = inputs.claude-skills; subpath = "skills"; };
  names  = builtins.attrNames (nixpkgs.lib.filterAttrs (_: t: t == "directory") skills);
in
nput.lib.mkManifest {
  root = nput.lib.homeRoot;
  entries = map (n: {
    name    = "skill-${n}";
    src     = inputs.claude-skills;
    subpath = "skills/${n}";       # 取り出す側も補間
    target  = ".claude/skills/${n}";  # 配置先も同じ名前で補間
  }) names;
}
```

---

## 配置動作仕様

engine が**ネイティブ FS 操作**で行う（`ln` / `rsync` は使わない・→ ADR-0006）。

### symlink モード

```
1. target の親ディレクトリを作成（mkdir -p 相当）
2. target に既存の nput 管理 symlink があれば置き換える
3. <配置元>/<subpath> を指す symlink を <root>/<target> に作成（os.Symlink）
   - store link:        配置元 = Nix ストアパス
   - out-of-store:      配置元 = marker の絶対パス
```

- target に通常ファイルまたはディレクトリが存在する場合はエラーで停止（上書きしない）
- subpath がファイル・ディレクトリどちらでも同じ処理

### copy モード（place-once・ユーザー管理）

```
subpath がディレクトリの場合:
  target が不在のとき: <root>/<target> を作成しネイティブ再帰コピー（mode 保存。<src>/<subpath>/ → <root>/<target>/）
  target が存在するとき: 何もしない（ユーザー管理に委ねる）

subpath がファイルの場合:
  target が不在のとき: 親ディレクトリを作成しネイティブコピー（<src>/<subpath> → <root>/<target>）
  target が存在するとき: 何もしない
```

- **place-once**: 初回マテリアライズ後、target が在れば触らない。ストア更新の反映は明示再適用（target 削除後に再実行）に委ねる。
- 世代管理の対象外。ロールバックされない。
- `subpath` がディレクトリのとき `target` に通常ファイルが存在する場合、または `subpath` がファイルのとき `target` がディレクトリの場合は、構造の不一致としてエラーで停止。

### out-of-store symlink

- symlink として配置する。指す先は marker の絶対パス（ローカル FS）。
- 世代では「どの絶対パスを指すか」のリンク先マッピングのみ版管理する。指す先の内容は設計上ライブで、永遠にスナップショットしない（→ ADR-0002）。

---

## 世代管理仕様

→ `docs/adr/0002`

### 機構

- 純粋関数 `lib.mkManifest` が **link farm derivation**（`manifest.json` + 配置元への symlink ツリー）を生成する（→ ADR-0006）。
  「配置したもの」のマニフェストは `manifest.json` として link farm の一部に **store 内に**埋め込む（store 外の可変 JSON は持たない。`manifest.json` は不変）。
- **engine**（Go ライブラリ）が実行時の副作用として（→ ADR-0006, ADR-0007）:
  0. `profileDir` 単位の flock を try-lock（保持中ならスキップ）。
  1. **前世代の store マニフェスト**（`manifest.json`）と新世代を diff し、消えた entry の **symlink を除去**（stale 除去）
     - 前世代は **全モード共通で nput 自身の profile の前世代**から読む（standalone も module も同一。ホストの oldGenPath には依存しない）
  2. symlink / out-of-store / place-once copy を**ネイティブ FS 操作**で配置（`ln` / `rsync` は使わない。新規・張替を先に、stale 除去を最後に）
  3. 全て成功してから `nix-env --profile <profileDir> --set <link-farm-drv>` で nput の nix profile を更新（コミット点・全モード）
  - project mode は新 link-farm が前世代と同一なら 1〜3 をスキップ（世代スキップ）。途中失敗は 3 に到達せず前世代を保つ（部分失敗のコミット最後・→ ADR-0006）。

配置・cleanup アルゴリズムは home-manager の `linkGeneration`/`cleanup` を参考に Go で再実装する（`home.file` 自体は再利用しない）。`nix` / `git` 以外はサブプロセスを使わない（配置エンジン層）。

**nput は全モードで自前 profile を持つ**（→ ADR-0002）。standalone では profile をユーザー向け rollback に使う。
module（HM/NixOS/darwin）では profile を**内部機構**（前世代マニフェスト + stale 追跡）に留め、ユーザー向け rollback は
host（`home-manager --rollback` 等）に一本化する。`nput rollback <name>` は home mode 限定。host rollback は旧 config を
再 activate して nput を再 kick することで自動追従する（nput profile は前進のみ＝旧内容の新世代を積む）。

| 機構 | 役割 | 適用層 | 位置（推奨・未確定）|
|---|---|---|---|
| 世代由来の store マニフェスト | stale 除去のための前回状態（不変・GC-root 済み）| 全層共通 | `manifest.json` として link farm derivation 内に埋め込み（→ ADR-0006）|
| nput の nix profile | 前世代の保持・世代番号・GC root | 全モード（standalone はユーザー向け / module は内部）| home: `~/.local/state/nix/profiles/nput/<name>` / project: `.../<roothash>/<name>`（→ ADR-0005）|

> profile の具体パスは推奨値であり、実装時に確定する。

### stale 除去の対象と安全不変条件

削除は保守的に行う（→ ADR-0002）。前世代マニフェストが「nput が配置した」と記録し、
**かつ現状もその記録通りの先（その世代の store パス／記録された out-of-store パス）を指す symlink** のみ削除する。
通常ファイルや nput 非管理の link には触れない。記録と実体が不一致なら削除せず警告する。初回／記録なしは何も消さない。

| 配置種別 | entry が消えたとき |
|---|---|
| symlink（store / out-of-store）| **除去する**（ただし上記の保守的不変条件を満たすもののみ）|
| copy | **除去しない**（ユーザー所有データ）。ただし orphan を警告で通知する |

### GC とストレージ解放

profile の各世代は GC root。`nix profile wipe-history`／`nix-env --delete-generations` で旧世代を間引き、
`nix-collect-garbage` で無参照になった store パスを解放する。
（可変 JSON 方式は GC root を作らず参照中 store パスが GC で消えるため採らない。）

### ロールバック

- **standalone（home mode）**: `nput rollback <name>` で前世代に戻す（profile symlink を前世代に向け、その世代の link farm で再配置）。
  任意世代切替・世代 GC は標準の `nix profile` / `nix-collect-garbage` を profile パスに対して使う。
- **module**: `nput rollback` は公開しない。ユーザー向けロールバックは host（`home-manager --rollback` 等）に一本化する。
- **project mode**: rollback は公開しない（ephemeral・→ ADR-0005）。

### モジュール時

nput は自前 profile を**持つ**が、それは前世代マニフェスト + stale 追跡のための内部機構に留める（→ ADR-0002）。
ユーザー向けロールバックは host に一本化し、host rollback は旧 config を再 activate して nput を再 kick することで
FS を自動収束させる（nput profile は前進のみ＝旧内容の新世代を積む）。host から nput へ要求するのは「switch 時の kick」だけで、
ホストの oldGenPath 配管は不要。

### project mode の世代（→ ADR-0005）

- **profile は解決済み root でキーする**（例: `~/.local/state/nix/profiles/nput/<roothash>/<name>`）。同一 entrypoint を複数箇所に
  クローンしても profile が衝突せず、stale 除去が互いのクローンの配置を破壊しない。home mode（1 ユーザー 1 つ）では起きない問題。
- **世代はユーザーに公開しない**。profile は stale 除去 + 世代スキップ判定の内部機構に留め、`rollback` / `list-generations` を出さない。
  配置物が ephemeral で rollback の意味が薄く、devShell キック時は戻し先 host 世代も無いため。
- **世代スキップ短絡（必須）**: 新 link farm derivation が前世代と同一なら新世代を積まず no-op で終える。
  devShell / direnv 運用では `shellHook` がシェル再入のたびに走るため、毎回新世代を積むと世代が無限増殖する。
  home mode は従来通り「適用のたびに新世代」のまま（世代スキップは project mode 限定）。
- **orphan profile**: クローンを削除すると profile が `~/.local/state` に孤児として残る。store は `nix-collect-garbage` で解放されるが
  profile ディレクトリは残る。放置許容（または手動削除）とし、公開ドキュメントに注記する（→ ADR-0005）。

---

## モジュールオプション仕様

### 共通オプション（全モジュール）

```
nput.enable  :: bool      # デフォルト: false
nput.entries :: list      # デフォルト: []
```

モジュールは自分の性質で root を pin する（HM → `homeRoot` / devShell → `projectRoot`）ため、モジュール利用者は `root` を再指定しない（→ ADR-0003, ADR-0007）。

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
  entries = [
    { name = "nvim"; src = nput.lib.mkOutOfStoreSymlink dotfiles; subpath = "home/.config/nvim"; target = ".config/nvim"; }
  ];
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

| mode | `src` | 動作 |
|---|---|---|
| `"symlink"` | path / set | engine が store link をネイティブ symlink |
| `"symlink"` | marker | engine が out-of-store symlink をネイティブ作成（HM の mkOutOfStoreSymlink には委譲しない）|
| `"copy"` | path / set | engine が place-once ネイティブコピー |

- `home.activation.nput` から engine を起動する。配置ロジックは HM に依存しない。root は `homeRoot` を pin。
- 世代は HM のものに乗る（nput 独自 profile は内部機構）。

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
    nput apply skills
  '';
};
```

- `nix develop` / direnv（`use flake`）でシェルに入った瞬間にキックされ、git toplevel を root にプロジェクト内へ配置する。
- `shellHook` は高頻度で走るため、project mode の**世代スキップ短絡**（変更なしなら no-op）が前提（→「project mode の世代」）。

---

## エラー仕様

評価時エラー（`root` 省略・重複 `name`・不正な型・未知キー）は `mkManifest` の `evalModules` / `lib.throwIf` が検出する（→ ADR-0010）。

| 条件 | 動作 |
|---|---|
| `root` を省略（暗黙デフォルトなし）| Nix 評価時にエラー（`evalModules` の "option used but not defined"）|
| `src` に素の文字列を渡す（store パス文字列含む）| Nix 評価時にエラー（`srcType` が文字列を拒否・out-of-store は marker で opt-in・→ ADR-0001）|
| entry に未知キーがある（タイポ・旧名 `source` / `dir`）| Nix 評価時にエラー（submodule が strict・→ ADR-0008, ADR-0010）|
| `src` が存在しないストアパス（path / set）| Nix 評価時にエラー |
| `src` が marker でローカルパスが存在しない | engine 実行時にエラーで停止 |
| `subpath` が `src` 内に存在しないパス | engine 実行時にエラーで停止 |
| `target` に通常ファイル・ディレクトリが既存（symlink モード）| エラーで停止（上書きしない）|
| `subpath` がディレクトリのとき `target` に通常ファイルが既存（copy モード）| エラーで停止 |
| `subpath` がファイルのとき `target` がディレクトリとして既存（copy モード）| エラーで停止 |
| copy モードで `target` が既存 | place-once により何もしない（上書きしない）|
| stale 除去で copy entry が消えた | copy target は削除せず、orphan を警告で通知 |
| `target` の親ディレクトリが存在しない | `mkdir -p` で自動作成 |
| 同一 `entries` リスト内に重複する `name` がある | Nix 評価時にエラー |
| `rollback` で前世代が存在しない | エラーメッセージを出力して停止 |
| `apply` で name 省略かつ `nput.default` が未定義 | CLI がエラーで停止（config 名を要求）|
| `nput.<name>` が entrypoint に存在しない | CLI がエラーで停止 |
| entrypoint が発見できない（CWD に flake.nix/shell.nix/default.nix なし・`-f` 未指定）| CLI がエラーで停止 |
| nix-darwin（将来）で `users.users.<user>.home` が未設定 | Nix 評価時にエラー |
| project mode で git リポジトリ外かつ `--root` 未指定（git toplevel 解決失敗）| engine 実行時にエラーで停止 |
| project mode で `git` が PATH に無い | engine 実行時にエラーで停止 |

---

## 依存関係

| コンポーネント | 依存 |
|---|---|
| `lib/`（`mkManifest` / マーカー群 / `listFilesInSrc`）| nixpkgs.lib のみ（純データ生成。`rsync` 不要）。型検査に `lib.types` / `mkOption` / `evalModules`（nixpkgs.lib のコア）を使う（→ ADR-0010）|
| `lib/types.nix`（entry submodule / `srcType` / `rootType` / marker custom type）| nixpkgs.lib のみ（`modules/common.nix` と共有・→ ADR-0010）|
| `lib/out-of-store.nix`（`mkOutOfStoreSymlink` / `projectRoot` / `homeRoot` / `systemRoot` マーカー）| なし（`_nputMarker` タグ付きマーカー構築子・→ ADR-0010）|
| `internal/`（配置エンジン = Go ライブラリ）| Go 標準ライブラリ中心。`manifest.json` を入力に取り runtime に `nix`（profile）/ `git`（toplevel）をサブプロセスで要求（→ ADR-0006）|
| `cmd/nput`（CLI = `packages.nput`）| 配置エンジンライブラリを import。entrypoint 発見と `nix`（build / eval）オーケストレーションを担う。`buildGoModule` でビルド（→ ADR-0007）|
| `modules/common.nix` | nixpkgs.lib のみ |
| `modules/home-manager.nix` | home-manager の module system（起動配線のみ）|
| `modules/nixos.nix`（将来）| NixOS の module system（起動配線のみ）|
| `modules/nix-darwin.nix`（将来）| nix-darwin の module system（起動配線のみ）|

lib 層は home-manager / NixOS / nix-darwin に依存しない。配置ロジックは Go エンジン（ライブラリ）が単一の源として所有する（→ ADR-0003, ADR-0006, ADR-0007）。

---

## 使用例（フル）

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url    = "github:NixOS/nixpkgs/nixos-unstable";
    nput.url       = "github:<owner>/nput";

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
        entries = [
          { name = "nix-skills"; src = inputs.claude-skills; subpath = "skills/nix"; target = ".claude/skills/nix"; }
        ];
      };
    };

    devShells.${system}.default = pkgs.mkShell {
      shellHook = "nput apply skills";
    };

    # パターン2: home mode（標準的な dotfiles 配置・別 profile）
    # nput.${system}.vim-plugins = nput.lib.mkManifest {
    #   root = nput.lib.homeRoot;
    #   entries = [
    #     { name = "vim-plugin-foo"; src = inputs.vim-plugin-foo; target = ".local/share/nvim/site/pack/foo/start/foo"; }
    #     { name = "zsh-autosuggestions"; src = inputs.zsh-autosuggestions; target = ".zsh/plugins/autosuggestions"; }
    #   ];
    # };

    # パターン3: home-manager モジュール（root は homeRoot を pin・外部リポジトリ + ローカル dotfiles 混在）
    homeConfigurations.<username> = home-manager.lib.homeManagerConfiguration {
      inherit pkgs;
      modules = [
        nput.homeManagerModules.default
        {
          nput.enable = true;
          nput.entries = [
            { name = "nix-skills"; src = inputs.claude-skills; subpath = "skills/nix"; target = ".claude/skills/nix"; }
            { name = "nvim-config"; src = nput.lib.mkOutOfStoreSymlink "/home/me/dotfiles"; subpath = "home/.config/nvim"; target = ".config/nvim"; }
          ];
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
