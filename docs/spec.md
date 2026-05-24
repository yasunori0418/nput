# nput 仕様書

## lib API

### `lib.mkActivationScript`

フェッチ済みリポジトリを実環境に配置するシェルスクリプトの derivation を返す。
ファイル内容の生成・変換は行わない。リポジトリの内容をそのまま配置する。

```
mkActivationScript :: { pkgs, entries } -> derivation
```

**引数**

| 引数 | 型 | 説明 |
|---|---|---|
| `pkgs` | nixpkgs | rsync など依存パッケージの解決に使用 |
| `entries` | list of entry | 配置定義のリスト（後述） |

**返り値**

`pkgs.writeShellApplication` で生成した derivation。
`result/bin/nput` として実行可能。

**CLI**

```bash
nput              # 全エントリを適用
nput --only <name>  # 指定エントリのみ適用
```

**使用例**

```nix
let
  activate = inputs.nput.lib.mkActivationScript {
    inherit pkgs;
    entries = [
      { name = "vim-foo";  src = inputs.vim-foo;  target = ".local/share/nvim/site/pack/foo/start/foo"; }
      { name = "zsh-sugg"; src = inputs.zsh-sugg; target = ".zsh/plugins/autosuggestions"; }
    ];
  };
in activate
```

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
| `"unknown"` | 特殊ファイル（デバイスファイル・パイプ等） |

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
  src    : path | set | string # 必須（型によって挙動が変わる）
  source : string              # 省略可、デフォルト: "."
  target : string              # 必須
  mode   : "symlink"
         | "copy"              # 省略可、デフォルト: "symlink"
}
```

#### `name`

- **型**: string
- **必須**: yes
- **説明**: エントリの識別子。`--only <name>` による選択的適用に使用する。
  同一 `entries` リスト内で一意であること。

```nix
name = "vim-plugin-foo";
name = "zsh-autosuggestions";
name = "claude-skills";
```

#### `src`

- **必須**: yes
- **説明**: 配置元リポジトリ。**Nix の型（`builtins.typeOf`）によって挙動が変わる。**

| Nix の型 | 例 | symlink の指す先 | 用途 |
|---|---|---|---|
| `"path"` | `inputs.myrepo` | Nix ストア（不変） | 外部リポジトリ（バージョン固定） |
| `"path"` | `builtins.path { path = "/home/..."; name = "..."; }` | Nix ストア（ローカルをコピー） | ローカルリポジトリをストア経由で扱いたい場合 |
| `"set"` | `pkgs.fetchFromGitHub { ... }` | Nix ストア（不変） | 外部リポジトリ（バージョン固定） |
| `"string"` | `"/path/to/dotfiles"` | ローカルファイルシステム（ライブ） | 手元の dotfiles リポジトリ等 |

`string` 型はローカルパスとして扱い、Nix ストアを経由しない out-of-store symlink になる。
`builtins.path` を使うと、ローカルパスを評価時にストアへコピーしてからストア symlink にできる。
`path` リテラル（クォートなしの `/path/to/...`）は Nix がストアにコピーするため、**out-of-store が目的なら必ず文字列リテラルで渡す。**

> **制約**: `src` は Nix 評価時に確定する必要があるため、シェルの `$HOME` は使えない。
> ローカルパスは絶対パスの文字列リテラルで指定する。
> `target` の `$HOME` 解決（standalone）は実行時にシェルが行うため、`target` 側には影響しない。

```nix
# 外部リポジトリ（ストア経由）
src = inputs.myrepo;
src = pkgs.fetchFromGitHub { owner = "..."; repo = "..."; rev = "..."; hash = "..."; };
src = builtins.fetchGit { url = "https://github.com/..."; rev = "..."; };

# ローカルリポジトリ - out-of-store symlink（ライブ反映、編集が即時反映される）
src = "/path/to/dotfiles";   # string リテラル必須。クローン先の絶対パスを指定

# ローカルリポジトリ - ストア経由 symlink（評価時点の内容をストアにコピー）
src = builtins.path { path = /path/to/dotfiles; name = "dotfiles"; };

# src = /path/to/dotfiles;   # NG: 意図が不明確。builtins.path を明示的に使うこと
```

#### `source`

- **型**: string
- **必須**: no
- **デフォルト**: `"."`（リポジトリルート全体）
- **説明**: リポジトリ内の配置元パス。ファイル・ディレクトリどちらも指定可能。

```nix
source = ".";                  # リポジトリ全体
source = "skills/nix";         # サブディレクトリのみ取り出す
source = "themes/dark.json";   # 単一ファイル
```

#### `target`

- **型**: string
- **必須**: yes
- **説明**: 配置先パス。`$HOME` からの相対パスで指定する。

```nix
target = ".local/share/nvim/site/pack/foo/start/foo";  # vim プラグイン
target = ".zsh/plugins/autosuggestions";               # zsh プラグイン
target = ".claude/skills/nix";                         # claude スキル
```

#### `mode`

- **型**: `"symlink"` | `"copy"`
- **必須**: no
- **デフォルト**: `"symlink"`

| mode | `src` の型 | 動作 | 向いている用途 |
|---|---|---|---|
| `"symlink"` | path / set | Nix ストアへの symlink（読み取り専用） | vim プラグイン、zsh プラグイン、スキル等 |
| `"symlink"` | string | ローカルパスへの out-of-store symlink（ライブ反映） | 手元の dotfiles リポジトリ |
| `"copy"` | path / set | ストアからコピー（書き込み可） | 後から手動編集したいテーマ・設定等 |
| `"copy"` | string | ローカルパスから実行時点の内容をコピー | ほぼ非推奨（ライブ反映されない） |

---

## 配置動作仕様

### symlink モード

```
1. target の親ディレクトリを mkdir -p で作成
2. target に既存の symlink があれば削除
3. ln -s <store-path>/<source> $HOME/<target> を実行
```

- target に通常ファイルまたはディレクトリが存在する場合はエラーで停止（上書きしない）
- source がファイル・ディレクトリどちらでも同じ処理

### copy モード

```
source がディレクトリの場合:
  1. $HOME/<target> を mkdir -p で作成
  2. rsync -a --delete <src>/<source>/ $HOME/<target>/ を実行

source がファイルの場合:
  1. $HOME/<target> の親ディレクトリを mkdir -p で作成
  2. rsync -a <src>/<source> $HOME/<target> を実行
```

- `--delete` により、ストアに存在しないファイルが target から削除される（ディレクトリの場合）
- コピーされたファイルはユーザーが直接編集可能
- `source` がディレクトリのとき `target` に通常ファイルが存在する場合、または `source` がファイルのとき `target` がディレクトリの場合は、構造の不一致としてエラーで停止（上書きしない）

---

## モジュールオプション仕様

### 共通オプション（全モジュール）

```
nput.enable  :: bool      # デフォルト: false
nput.entries :: list      # デフォルト: []
```

### NixOS / nix-darwin 追加オプション

```
nput.user :: string       # 必須（配置先ユーザーの特定に使用）
```

home-manager と standalone は `$HOME` を参照するため `user` は不要。

---

## ホームディレクトリの解決

各層が target を絶対パスに変換する方法。

| 層 | 解決方法 | 備考 |
|---|---|---|
| standalone | `$HOME`（実行時のシェル環境変数） | OS 問わず動作 |
| home-manager | `$HOME`（HM が内部解決） | OS 問わず動作 |
| NixOS | `config.users.users.${cfg.user}.home` | `isNormalUser = true` で `/home/<user>` が自動設定 |
| nix-darwin | `config.users.users.${cfg.user}.home` | デフォルト値なし。ユーザー側で明示設定が必須 |

### NixOS での設定例

```nix
users.users.<username> = {
  isNormalUser = true;
  # home は省略可。/home/<username> が自動設定される
};
```

### nix-darwin での設定例（明示必須）

```nix
users.users.<username> = {
  home = "/Users/<username>";  # macOS の慣例パスを明示
};
```

---

## モジュール別動作仕様

### home-manager モジュール

| mode | `src` の型 | 使用する HM 機能 | symlink の指す先 |
|---|---|---|---|
| `"symlink"` | path / set | `home.file."<target>".source` | Nix ストア |
| `"symlink"` | string | `home.file` + `mkOutOfStoreSymlink` | ローカルパス（ライブ） |
| `"copy"` | path / set | `home.activation.nput` + rsync | —（コピー） |
| `"copy"` | string | `home.activation.nput` + rsync | —（コピー） |

`builtins.typeOf src == "string"` で分岐し、string のとき `config.lib.file.mkOutOfStoreSymlink` を使う。

```nix
home.file."${entry.target}".source =
  if builtins.typeOf entry.src == "string"
  then config.lib.file.mkOutOfStoreSymlink "${entry.src}/${entry.source}"
  else "${entry.src}/${entry.source}";
```

ローカルリポジトリをストア経由にしたい場合は `src = builtins.path { ... }` を使う。
この場合 `builtins.typeOf` は `"path"` になるため、上の分岐で自動的にストア経由になる。

### NixOS モジュール

| mode | 使用する NixOS 機能 | 実行タイミング |
|---|---|---|
| `"symlink"` | `systemd.tmpfiles.rules`（`L` 型） | 起動時 / `nixos-rebuild switch` |
| `"copy"` | `system.activationScripts.nput` + rsync | `nixos-rebuild switch` |

ホームディレクトリは `config.users.users.${cfg.user}.home` から取得する（「ホームディレクトリの解決」参照）。

```nix
"${config.users.users.${cfg.user}.home}/${entry.target}"
```

### nix-darwin モジュール

| mode | 使用する nix-darwin 機能 | 実行タイミング |
|---|---|---|
| `"symlink"` | `system.activationScripts.nput` + ln | `darwin-rebuild switch` |
| `"copy"` | `system.activationScripts.nput` + rsync | `darwin-rebuild switch` |

ホームディレクトリは `config.users.users.${cfg.user}.home` から取得する。
明示設定が必須な点に注意（「ホームディレクトリの解決」参照）。

### standalone

`nix run .#<script-name>` または `./result/bin/nput` を
ユーザーが明示的に実行する。自動実行機構は持たない。

`--only <name>` で指定エントリのみ適用できる。他エントリは実行されない。

**`src` と `target` の解決タイミングの違い**

`src`（ローカルパス）は Nix 評価時に確定する絶対パスで、`$HOME` は使えない。
`target` の `$HOME` 解決はスクリプト実行時に行われる。

```
Nix 評価時:  src = "/path/to/dotfiles"  →  スクリプトにハードコード
実行時:      $HOME = /home/<user>        →  シェルが展開
```

以下の entries 定義に対して生成されるスクリプトの実行内容：

```nix
{ name = "nvim-config"; src = "/path/to/dotfiles"; source = "home/.config/nvim"; target = ".config/nvim"; }
```

```bash
# 生成スクリプトの実行内容（イメージ）
mkdir -p "$(dirname "$HOME/.config/nvim")"
if [[ -L "$HOME/.config/nvim" ]]; then rm "$HOME/.config/nvim"; fi
ln -s "/path/to/dotfiles/home/.config/nvim" "$HOME/.config/nvim"
# ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  src: eval時に確定した絶対パス
#                                                    $HOME: 実行時にシェルが展開
```

`src` のローカルパスをポータブルにしたい場合は、flake 内で変数として定義する：

```nix
let
  dotfiles = "/path/to/dotfiles";  # 1箇所で管理
in
nput.lib.mkActivationScript {
  inherit pkgs;
  entries = [
    { name = "nvim";  src = dotfiles; source = "home/.config/nvim";  target = ".config/nvim"; }
    { name = "zsh";   src = dotfiles; source = "home/.zshrc";        target = ".zshrc"; }
  ];
}
```

**macOS / Linux の環境差異への対応パターン**

Linux（`/home/<user>`）と macOS（`/Users/<user>`）でホームディレクトリの慣例が異なるため、
`src` のローカルパスを環境に応じて解決する方法が必要になる。

---

**パターン 1：OS 判別 + ユーザー名変数（推奨）**

`pkgs.stdenv.isDarwin` で OS を判別し、ホームプレフィックスを切り替える。
pure eval のまま動作する。

```nix
let
  username = "<username>";
  homeDir  = if pkgs.stdenv.isDarwin
             then "/Users/${username}"
             else "/home/${username}";
  dotfiles = "${homeDir}/dotfiles";
in
nput.lib.mkActivationScript {
  inherit pkgs;
  entries = [
    { name = "nvim"; src = dotfiles; source = "home/.config/nvim"; target = ".config/nvim"; }
  ];
}
```

---

**パターン 2：`builtins.getEnv "HOME"`（impure）**

`$HOME` 環境変数を Nix 評価時に読み込む。OS を意識せず書けるが、
flakes のデフォルトである pure eval モードでは空文字列が返るため `--impure` が必要。

```nix
let
  dotfiles = "${builtins.getEnv "HOME"}/dotfiles";
in ...
```

```bash
nix run --impure .#nput
```

---

**パターン 3：flake の `specialArgs` / `extraSpecialArgs` で注入**

NixOS / home-manager のモジュール文脈では、flake 引数としてパスを渡す方法もある。
standalone の `mkActivationScript` は単純な関数呼び出しのため、通常はパターン 1 で十分。

```nix
# 複数の設定ファイルで共有したい場合
outputs = { self, nixpkgs, nput, ... }:
let
  username = "<username>";
  dotfilesFor = system:
    let pkgs = nixpkgs.legacyPackages.${system}; in
    if pkgs.stdenv.isDarwin then "/Users/${username}/dotfiles"
    else "/home/${username}/dotfiles";
in { ... }
```

---

| パターン | pure eval | OS 対応 | ユーザー名の扱い |
|---|---|---|---|
| 1. OS 判別 | ✓ | ✓ | flake 内変数 |
| 2. `builtins.getEnv` | ✗（`--impure` 必要） | ✓ | 不要 |
| 3. specialArgs 注入 | ✓ | 任意 | 引数として渡す |

---

## エラー仕様

| 条件 | 動作 |
|---|---|
| `src` が存在しないストアパス（path / set） | Nix 評価時にエラー |
| `src` が string でローカルパスが存在しない | スクリプト実行時にエラーで停止 |
| `source` が `src` 内に存在しないパス | スクリプト実行時にエラーで停止 |
| `target` に通常ファイル・ディレクトリが既存（symlink モード） | `ln` がエラーで停止（上書きしない） |
| `source` がディレクトリのとき `target` に通常ファイルが既存（copy モード） | エラーで停止（上書きしない） |
| `source` がファイルのとき `target` がディレクトリとして既存（copy モード） | エラーで停止（上書きしない） |
| `target` の親ディレクトリが存在しない | `mkdir -p` で自動作成 |
| `--only` で指定した `name` が存在しない | エラーメッセージを出力して停止 |
| 同一 `entries` リスト内に重複する `name` がある | Nix 評価時にエラー |
| nix-darwin で `users.users.<user>.home` が未設定 | Nix 評価時にエラー（空文字列でパスが壊れる） |

---

## 依存関係

| コンポーネント | 依存 |
|---|---|
| `lib/deploy.nix` | nixpkgs（rsync パッケージ）のみ |
| `modules/common.nix` | nixpkgs.lib のみ |
| `modules/home-manager.nix` | home-manager の module system |
| `modules/nixos.nix` | NixOS の module system |
| `modules/nix-darwin.nix` | nix-darwin の module system |

lib 層は home-manager / NixOS / nix-darwin に依存しない。

---

## 使用例（フル）

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url    = "github:NixOS/nixpkgs/nixos-unstable";
    nput.url  = "github:<owner>/nput";

    # 各ソースを独立して管理（flake = false でリポジトリをそのまま取得）
    vim-plugin-foo.url        = "github:foo/vim-plugin-foo";
    vim-plugin-foo.flake      = false;
    zsh-autosuggestions.url   = "github:zsh-users/zsh-autosuggestions";
    zsh-autosuggestions.flake = false;
    claude-skills.url         = "github:someone/claude-skills";
    claude-skills.flake       = false;
  };

  outputs = { self, nixpkgs, nput, ... }@inputs:
  let pkgs = nixpkgs.legacyPackages.x86_64-linux; in
  {
    # パターン1: 役割ごとに分離した standalone スクリプト
    packages.x86_64-linux = {
      vim-plugins = nput.lib.mkActivationScript {
        inherit pkgs;
        entries = [
          { name = "vim-plugin-foo"; src = inputs.vim-plugin-foo; target = ".local/share/nvim/site/pack/foo/start/foo"; }
        ];
      };

      zsh-plugins = nput.lib.mkActivationScript {
        inherit pkgs;
        entries = [
          { name = "zsh-autosuggestions"; src = inputs.zsh-autosuggestions; target = ".zsh/plugins/autosuggestions"; }
        ];
      };

      claude-skills = nput.lib.mkActivationScript {
        inherit pkgs;
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
            # 外部リポジトリ（ストア経由 symlink）
            { name = "nix-skills"; src = inputs.claude-skills; source = "skills/nix"; target = ".claude/skills/nix"; }
            # ローカル dotfiles（out-of-store symlink、mkOutOfStoreSymlink を自動適用）
            # src は string リテラルで渡す。/path/to/dotfiles はクローン先の絶対パス
            { name = "nvim-config"; src = "/path/to/dotfiles"; source = "home/.config/nvim"; target = ".config/nvim"; }
          ];
        }
      ];
    };
  };
}
```

```bash
# 独立して更新・適用
nix run .#vim-plugins
nix run .#zsh-plugins
nix run .#claude-skills

# 特定エントリのみ
nix run .#zsh-plugins -- --only zsh-autosuggestions
```
