# nput 仕様書

## lib API

### `lib.mkActivationScript`

フェッチ済みリポジトリを実環境に配置するシェルスクリプトの derivation を返す。
ファイル内容の生成・変換は行わない。リポジトリの内容をそのまま配置する。
standalone では nix profile に乗せた世代管理を行う（→ `docs/adr/0002`）。

```
mkActivationScript :: { pkgs, name, entries } -> derivation
```

**引数**

| 引数 | 型 | 説明 |
|---|---|---|
| `pkgs` | nixpkgs | rsync など依存パッケージの解決に使用 |
| `name` | string | 配置単位名。profile を一意特定する（`mkActivationScript` 1 呼び出し = 1 profile）|
| `entries` | list of entry | 配置定義のリスト（後述）|

配置先 root は内部でパラメータ化されるが、公開 API は当面 root = `$HOME` 固定とする。
root 差し替え（将来の system 配置, root = `/`）は拡張 seam であり安定公開引数ではない（→ ADR-0004）。

**返り値**

`pkgs.writeShellApplication` で生成した derivation。`result/bin/nput` として実行可能。

**CLI**

```bash
nput                    # 新世代を作って全エントリを適用
nput --rollback         # 前世代へ戻す
nput --list-generations # 世代一覧を表示
```

任意世代への切替・世代の GC は標準の `nix profile` / `nix-collect-garbage` を profile パスに対して使う。
`--only`（一部 entry だけ適用）は profile 世代の atomic 性と衝突するため提供しない。選択的更新は配置単位（スクリプト）の分割で担保する。

**使用例**

```nix
let
  activate = inputs.nput.lib.mkActivationScript {
    inherit pkgs;
    name = "dotfiles";
    entries = [
      { name = "vim-foo";  src = inputs.vim-foo;  target = ".local/share/nvim/site/pack/foo/start/foo"; }
      { name = "zsh-sugg"; src = inputs.zsh-sugg; target = ".zsh/plugins/autosuggestions"; }
    ];
  };
in activate
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
- 実際の link 生成は nput エンジンが行う。プラットフォームのネイティブ機構（HM の
  `config.lib.file.mkOutOfStoreSymlink` 等）へは委譲しない（→ ADR-0003）。

```nix
src = nput.lib.mkOutOfStoreSymlink "/home/me/dotfiles";
```

> **制約**: 引数は Nix 評価時に確定する絶対パスの文字列。シェルの `$HOME` は使えない。
> ローカルパスをポータブルにしたい場合は flake 内で変数として定義する（後述の「ローカルパスの扱い」参照）。

---

### `lib.listFilesInRepo`

リポジトリ内の指定ディレクトリのファイル一覧を型情報付きで返す。

```
listFilesInRepo :: { src, dir? } -> { filename: fileType }
```

**引数**

| 引数 | 型 | デフォルト | 説明 |
|---|---|---|---|
| `src` | path | — | フェッチ済みのストアパス |
| `dir` | string | `"."` | リポジトリ内のディレクトリパス |

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
lib.listFilesInRepo { src = inputs.skills-repo; dir = "skills"; }
# => { "nix" = "directory"; "python" = "directory"; "README.md" = "regular"; }
```

---

## entries スキーマ仕様

### フィールド定義

```
entry :: {
  name   : string              # 必須
  src    : path | set | marker # 必須（type/marker によって挙動が変わる）
  source : string              # 省略可、デフォルト: "."
  target : string              # 必須
  mode   : "symlink"
         | "copy"              # 省略可、デフォルト: "symlink"
}
```

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

#### `source`

- **型**: string / **必須**: no / **デフォルト**: `"."`（リポジトリルート全体）
- **説明**: リポジトリ内の配置元パス。ファイル・ディレクトリどちらも指定可能。

```nix
source = ".";                  # リポジトリ全体
source = "skills/nix";         # サブディレクトリのみ取り出す
source = "themes/dark.json";   # 単一ファイル
```

#### `target`

- **型**: string / **必須**: yes
- **説明**: 配置先パス。root（standalone / HM では `$HOME`）からの相対パスで指定する。

#### `mode`

- **型**: `"symlink"` | `"copy"` / **必須**: no / **デフォルト**: `"symlink"`

| mode | `src` の種別 | 動作 | 世代管理 |
|---|---|---|---|
| `"symlink"` | path / set | Nix ストアへの symlink（読み取り専用）| あり（profile / standalone）|
| `"symlink"` | marker | ローカルパスへの out-of-store symlink（ライブ）| あり（リンク先のみ）|
| `"copy"` | path / set | place-once コピー（書き込み可・ユーザー管理）| **なし** |
| `"copy"` | marker | 非推奨（out-of-store は symlink で使う）| — |

---

## 配置動作仕様

### symlink モード

```
1. target の親ディレクトリを mkdir -p で作成
2. target に既存の nput 管理 symlink があれば置き換える
3. ln -s <配置元>/<source> <root>/<target> を実行
   - store link:        配置元 = Nix ストアパス
   - out-of-store:      配置元 = marker の絶対パス
```

- target に通常ファイルまたはディレクトリが存在する場合はエラーで停止（上書きしない）
- source がファイル・ディレクトリどちらでも同じ処理

### copy モード（place-once・ユーザー管理）

```
source がディレクトリの場合:
  target が不在のとき: <root>/<target> を mkdir -p し rsync -a <src>/<source>/ <root>/<target>/
  target が存在するとき: 何もしない（ユーザー管理に委ねる）

source がファイルの場合:
  target が不在のとき: 親ディレクトリを mkdir -p し rsync -a <src>/<source> <root>/<target>
  target が存在するとき: 何もしない
```

- **place-once**: 初回マテリアライズ後、target が在れば触らない。ストア更新の反映は明示再適用（target 削除後に再実行）に委ねる。
- 世代管理の対象外。ロールバックされない。
- `source` がディレクトリのとき `target` に通常ファイルが存在する場合、または `source` がファイルのとき `target` がディレクトリの場合は、構造の不一致としてエラーで停止。

### out-of-store symlink

- symlink として配置する。指す先は marker の絶対パス（ローカル FS）。
- 世代では「どの絶対パスを指すか」のリンク先マッピングのみ版管理する。指す先の内容は設計上ライブで、永遠にスナップショットしない（→ ADR-0002）。

---

## 世代管理仕様（standalone）

→ `docs/adr/0002`

### 機構

- 純粋関数が **link farm derivation**（ストア内の symlink ツリー）を生成する。
- activation スクリプトが副作用として:
  1. **state マニフェスト**に「配置したもの」を記録（全層共通）
  2. 新旧マニフェストを diff し、消えた entry の **symlink を除去**（stale 除去）
  3. symlink / out-of-store / place-once copy を配置
  4. **standalone のみ** `nix-env --profile <profileDir> --set <link-farm-drv>` で nix profile に登録

| 機構 | 役割 | 適用層 | 位置（推奨・未確定）|
|---|---|---|---|
| state マニフェスト | stale 除去のための前回状態 | 全層共通 | `<root>/.local/state/nput/<name>.json` |
| nix profile | 世代番号・GC root・ロールバック | standalone 専用 | `~/.local/state/nix/profiles/nput/<name>` |

> profile / マニフェストの具体パスは推奨値であり、実装時に確定する。

### stale 除去の対象

| 配置種別 | entry が消えたとき |
|---|---|
| symlink（store / out-of-store）| **除去する**（nput 管理ポインタ）|
| copy | **除去しない**（ユーザー所有データ）。ただし orphan を警告で通知する |

### ロールバック

- `nput --rollback` で前世代に戻す（profile symlink を前世代に向け、その世代の link farm で再配置）。
- 任意世代切替・世代 GC は標準の `nix profile` / `nix-collect-garbage` を profile パスに対して使う。

### モジュール時

nput 独自 profile は作らず、ホストの世代システム（home-manager generations 等）に委譲する。
ロールバックはホスト世代が旧 config で nput エンジンを再実行することで担保する。

---

## モジュールオプション仕様

### 共通オプション（全モジュール）

```
nput.enable  :: bool      # デフォルト: false
nput.entries :: list      # デフォルト: []
```

### NixOS / nix-darwin 追加オプション（将来拡張）

```
nput.user :: string       # 必須（配置先ユーザーの特定に使用）
```

home-manager と standalone は `$HOME` を参照するため `user` は不要。

---

## root（ホームディレクトリ）の解決

各層が target を絶対パスに変換する方法。配置の実体は全層で nput エンジンが実行する（→ ADR-0003）。

| 層 | root の解決方法 | 備考 |
|---|---|---|
| standalone | `$HOME`（実行時のシェル環境変数）| OS 問わず動作 |
| home-manager | `$HOME`（HM が内部解決）| OS 問わず動作 |
| NixOS（将来）| `config.users.users.${cfg.user}.home` | `isNormalUser = true` で `/home/<user>` |
| nix-darwin（将来）| `config.users.users.${cfg.user}.home` | デフォルト値なし。明示設定が必須 |

### ローカルパス（out-of-store）の扱い

`mkOutOfStoreSymlink` の引数は Nix 評価時に確定する絶対パス。`$HOME` は使えない。
`target` の root 解決は実行時に行われるため `target` 側には影響しない。

```
Nix 評価時:  mkOutOfStoreSymlink "/path/to/dotfiles"  →  スクリプトにハードコード
実行時:      $HOME = /home/<user>                       →  シェルが root を展開
```

macOS / Linux でホームの慣例が異なるため、ローカルパスは flake 内で OS 判別して解決するのが推奨。

```nix
let
  username = "<username>";
  homeDir  = if pkgs.stdenv.isDarwin then "/Users/${username}" else "/home/${username}";
  dotfiles = "${homeDir}/dotfiles";
in
nput.lib.mkActivationScript {
  inherit pkgs;
  name = "dotfiles";
  entries = [
    { name = "nvim"; src = nput.lib.mkOutOfStoreSymlink dotfiles; source = "home/.config/nvim"; target = ".config/nvim"; }
  ];
}
```

`builtins.getEnv "HOME"`（`--impure` 必要）や flake の `specialArgs` 注入も使えるが、通常は上の OS 判別で十分。

---

## モジュール別動作仕様

すべての層で nput エンジンが配置を実行する。各層は root と activation タイミングを供給するだけ（→ ADR-0003）。
`systemd.tmpfiles` / `home.file` は明示的に採らない代替である。

### home-manager モジュール

| mode | `src` | 動作 |
|---|---|---|
| `"symlink"` | path / set | nput エンジンが store link を `ln -s` |
| `"symlink"` | marker | nput エンジンが out-of-store symlink を `ln -s`（HM の mkOutOfStoreSymlink には委譲しない）|
| `"copy"` | path / set | nput エンジンが place-once rsync |

- `home.activation.nput` から nput エンジンを起動する。配置ロジックは HM に依存しない。
- 世代は HM のものに乗る（nput 独自 profile は作らない）。

### NixOS モジュール（将来拡張）

- `system.activationScripts.nput` から nput エンジンを起動する。
- root は `config.users.users.${cfg.user}.home`。世代は nixos 世代に委譲。

### nix-darwin モジュール（将来拡張）

- `system.activationScripts.nput` から nput エンジンを起動する。
- root は `config.users.users.${cfg.user}.home`（明示設定が必須）。世代はホスト世代に委譲。

### standalone

`nix run .#<name>` または `./result/bin/nput` をユーザーが明示的に実行する。
nix profile による世代管理を行い、`--rollback` / `--list-generations` を提供する。

---

## エラー仕様

| 条件 | 動作 |
|---|---|
| `src` が存在しないストアパス（path / set）| Nix 評価時にエラー |
| `src` が marker でローカルパスが存在しない | スクリプト実行時にエラーで停止 |
| `source` が `src` 内に存在しないパス | スクリプト実行時にエラーで停止 |
| `target` に通常ファイル・ディレクトリが既存（symlink モード）| `ln` がエラーで停止（上書きしない）|
| `source` がディレクトリのとき `target` に通常ファイルが既存（copy モード）| エラーで停止 |
| `source` がファイルのとき `target` がディレクトリとして既存（copy モード）| エラーで停止 |
| copy モードで `target` が既存 | place-once により何もしない（上書きしない）|
| stale 除去で copy entry が消えた | copy target は削除せず、orphan を警告で通知 |
| `target` の親ディレクトリが存在しない | `mkdir -p` で自動作成 |
| 同一 `entries` リスト内に重複する `name` がある | Nix 評価時にエラー |
| `--rollback` で前世代が存在しない | エラーメッセージを出力して停止 |
| nix-darwin（将来）で `users.users.<user>.home` が未設定 | Nix 評価時にエラー |

---

## 依存関係

| コンポーネント | 依存 |
|---|---|
| `lib/engine.nix` | nixpkgs（rsync パッケージ）のみ |
| `lib/out-of-store.nix` | なし（純粋なマーカー構築子）|
| `modules/common.nix` | nixpkgs.lib のみ |
| `modules/home-manager.nix` | home-manager の module system（起動配線のみ）|
| `modules/nixos.nix`（将来）| NixOS の module system（起動配線のみ）|
| `modules/nix-darwin.nix`（将来）| nix-darwin の module system（起動配線のみ）|

lib 層は home-manager / NixOS / nix-darwin に依存しない。

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
  let pkgs = nixpkgs.legacyPackages.x86_64-linux; in
  {
    # パターン1: 役割ごとに分離した standalone スクリプト（それぞれ別 profile）
    packages.x86_64-linux = {
      vim-plugins = nput.lib.mkActivationScript {
        inherit pkgs;
        name = "vim-plugins";
        entries = [
          { name = "vim-plugin-foo"; src = inputs.vim-plugin-foo; target = ".local/share/nvim/site/pack/foo/start/foo"; }
        ];
      };

      zsh-plugins = nput.lib.mkActivationScript {
        inherit pkgs;
        name = "zsh-plugins";
        entries = [
          { name = "zsh-autosuggestions"; src = inputs.zsh-autosuggestions; target = ".zsh/plugins/autosuggestions"; }
        ];
      };

      claude-skills = nput.lib.mkActivationScript {
        inherit pkgs;
        name = "claude-skills";
        entries = [
          # リポジトリ内の特定ディレクトリだけを取り出す
          { name = "nix-skills"; src = inputs.claude-skills; source = "skills/nix"; target = ".claude/skills/nix"; }
        ];
      };
    };

    # パターン2: home-manager モジュール（外部リポジトリ + ローカル dotfiles 混在）
    homeConfigurations.<username> = home-manager.lib.homeManagerConfiguration {
      inherit pkgs;
      modules = [
        nput.homeManagerModules.default
        {
          nput.enable = true;
          nput.entries = [
            # 外部リポジトリ（store link）
            { name = "nix-skills"; src = inputs.claude-skills; source = "skills/nix"; target = ".claude/skills/nix"; }
            # 開発中の手元 dotfiles（out-of-store symlink、明示関数で opt-in）
            { name = "nvim-config"; src = nput.lib.mkOutOfStoreSymlink "/home/me/dotfiles"; source = "home/.config/nvim"; target = ".config/nvim"; }
          ];
        }
      ];
    };
  };
}
```

```bash
# 独立して更新・適用・ロールバック
nix run .#vim-plugins
nix run .#vim-plugins -- --rollback
nix run .#zsh-plugins
nix run .#claude-skills -- --list-generations
```
