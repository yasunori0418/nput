# nput 設計書

## 概要

任意の git リポジトリを特定バージョンで固定し、リポジトリ全体または特定のサブディレクトリ・ファイルを
実環境の任意のパスへ symlink または copy で配置する Nix ライブラリ・モジュール群。

home-manager のような「Nix モジュールオプションから設定を生成する」モデルとは異なり、
リポジトリの内容をそのまま配置することに特化する。設定の生成・変換は行わない。

home-manager に依存せず単体で動作しつつ、home-manager / NixOS / nix-darwin のモジュールシステムとも
統合できる設計とする。モジュールシステムを介さない関数呼び出しにも対応する。

---

## 設計目標

| 目標 | 説明 |
|---|---|
| 独立性 | home-manager に依存せず単体で動作する |
| 統合性 | HM / NixOS / nix-darwin モジュールとして呼び出せる |
| 柔軟性 | モジュールシステムを介さず関数として使える |
| 取得手段非依存 | npins / flake inputs / fetchFromGitHub / fetchGit など問わない |
| 粒度 | リポジトリ全体・サブディレクトリ・単一ファイルを配置できる |
| 更新の独立性 | エントリごとに個別更新・個別適用できる。他エントリに影響しない |
| 非生成 | ファイル内容に関与しない。リポジトリの内容をそのまま置く |

---

## プロジェクト構成

```
<project>/
├── flake.nix              # エントリポイント。outputs を定義
├── flake.lock             # flake 入力のロック
├── lib/
│   ├── default.nix        # 公開 API のまとめ（外部向けエントリ）
│   ├── types.nix          # entries の型定義（各モジュールで共有）
│   └── deploy.nix         # mkActivationScript などコアロジック
└── modules/
    ├── common.nix         # options 定義のみ（全モジュールが import）
    ├── home-manager.nix   # home.file / home.activation への変換
    ├── nixos.nix          # systemd.tmpfiles / system.activationScripts への変換
    └── nix-darwin.nix     # system.activationScripts への変換
```

---

## レイヤー構成

```
lib/deploy.nix              ← nixpkgs のみ依存（純粋コア）
        ↑
modules/common.nix          ← options 型定義（nixpkgs.lib のみ依存）
        ↑
┌───────┼──────────────────┐
HM      NixOS    darwin    standalone
（各層固有のプリミティブへの変換のみを追加）
```

上位層が下位層にのみ依存し、逆方向の依存は持たない。

---

## flake.nix outputs 設計

```nix
outputs = { ... }: {
  # モジュール統合
  homeManagerModules.default = ./modules/home-manager.nix;
  nixosModules.default       = ./modules/nixos.nix;
  darwinModules.default      = ./modules/nix-darwin.nix;

  # 関数呼び出し（モジュールシステム不使用）
  lib = import ./lib;
  # lib.mkActivationScript { pkgs, entries } → derivation
  # lib.listFilesInRepo    { src, dir? }     → attrset（builtins.readDir 互換）
};
```

---

## コアロジック設計（lib/deploy.nix）

### entries スキーマ

各エントリは「どのリポジトリのどのパスを、どこへどのように置くか」という配置単位を表す。
エントリは互いに独立しており、`name` で識別して個別に適用できる。

| フィールド | 型 | デフォルト | 必須 | 説明 |
|---|---|---|---|---|
| `name` | string | — | ✓ | エントリ識別子。選択的適用に使用 |
| `src` | path \| set \| string | — | ✓ | 配置元リポジトリ。Nix の型によって symlink 先が変わる（下記参照） |
| `source` | string | `"."` | — | リポジトリ内のパス（ファイル・ディレクトリ両対応） |
| `target` | string | — | ✓ | `$HOME` からの相対パス |
| `mode` | enum | `"symlink"` | — | `"symlink"` または `"copy"` |

`src` の型による挙動の違い:

| Nix の型 | symlink の指す先 | 用途 |
|---|---|---|
| `"path"` | Nix ストア（不変） | flake inputs / `builtins.path` 等 |
| `"set"` | Nix ストア（不変） | `fetchFromGitHub` 等の derivation |
| `"string"` | ローカルファイルシステム（ライブ） | 手元の dotfiles リポジトリ等 |

`src` はユーザーが任意の手段で用意したパスまたはローカルパス文字列を渡す設計とする。
これにより本プロジェクトは取得手段に依存しない。
string 型を渡した場合はストアを経由しない out-of-store symlink となり、ファイル編集が即座に反映される。

### エントリの独立性

各エントリの `src` は独立した flake input または npins エントリとして管理することを想定する。
あるエントリの `src` を更新しても、他のエントリは影響を受けない。

```
inputs.vim-plugin-foo       → entry "vim-plugin-foo" のみ影響
inputs.zsh-autosuggestions  → entry "zsh-autosuggestions" のみ影響
inputs.claude-skills        → entry "claude-skills" のみ影響
```

### source の判別ロジック

実行時にストアパスの種別を判定し、適切な処理を選択する。

```
source がディレクトリ → rsync src/ dest/  （内容物をコピー）
source がファイル     → rsync src  dest   （単体コピー）
symlink モード時      → ln -s（ファイル・ディレクトリ問わず共通処理）
```

### mkActivationScript の生成物

`pkgs.writeShellApplication` を用いて、ストアパスと target を評価時に埋め込んだ
シェルスクリプトの derivation を返す。実行時にネットワークアクセスは発生しない。

生成されるスクリプトは `--only <name>` 引数による選択的適用をサポートする。

```bash
nput              # 全エントリを適用
nput --only foo   # "foo" エントリのみ適用
```

---

## モジュール統合設計

### 共通オプション（modules/common.nix）

```nix
# modules/common.nix（全モジュール共通）
options.nput = {
  enable  = mkEnableOption "nput";
  entries = mkOption { type = listOf ...; };
};

# modules/nixos.nix, modules/nix-darwin.nix（各モジュール内で追加定義）
options.nput.user = mkOption { type = str; };
```

`user` オプションは NixOS / nix-darwin のみ必要なため、`modules/common.nix` には含めず各モジュールに分離する。
home-manager と standalone は `$HOME` を直接参照するため不要。

### 各統合層の変換先

| 層 | symlink の変換先 | copy の変換先 |
|---|---|---|
| **standalone** | `ln -s` | rsync |
| **home-manager** | `home.file."target".source` | `home.activation` + rsync |
| **NixOS** | `systemd.tmpfiles.rules`（`L` 型） | `system.activationScripts` + rsync |
| **nix-darwin** | `system.activationScripts` + ln | `system.activationScripts` + rsync |

### 実行タイミング

| 層 | 実行タイミング | 選択的適用 |
|---|---|---|
| standalone | `nix run .#nput` を明示実行 | `--only <name>` で可能 |
| home-manager | `home-manager switch` | 不可（switch 全体が対象） |
| NixOS | `nixos-rebuild switch` | 不可（switch 全体が対象） |
| nix-darwin | `darwin-rebuild switch` | 不可（switch 全体が対象） |

モジュール統合時の独立性は「`src` を個別に更新し、switch する」という flake レベルで担保する。
選択的適用が必要な場合は standalone を使う。

---

## 使用パターン

### パターン 1：standalone（役割ごとに分離して管理）

```nix
# flake.nix
packages.x86_64-linux = {
  vim-plugins = nput.lib.mkActivationScript {
    inherit pkgs;
    entries = [
      { name = "vim-foo"; src = inputs.vim-foo; target = ".local/share/nvim/site/pack/foo/start/foo"; }
      { name = "vim-bar"; src = inputs.vim-bar; target = ".local/share/nvim/site/pack/bar/start/bar"; }
    ];
  };

  zsh-plugins = nput.lib.mkActivationScript {
    inherit pkgs;
    entries = [
      { name = "zsh-autosuggestions";     src = inputs.zsh-autosuggestions;     target = ".zsh/plugins/autosuggestions"; }
      { name = "zsh-syntax-highlighting"; src = inputs.zsh-syntax-highlighting; target = ".zsh/plugins/syntax-highlighting"; }
    ];
  };

  claude-skills = nput.lib.mkActivationScript {
    inherit pkgs;
    entries = [
      { name = "claude-skills"; src = inputs.skills-repo; source = "skills/nix"; target = ".claude/skills/nix"; }
    ];
  };
};
```

```bash
# それぞれ独立して更新・適用できる
nix run .#vim-plugins
nix run .#zsh-plugins
nix run .#claude-skills
```

### パターン 2：home-manager モジュール

```nix
imports = [ inputs.nput.homeManagerModules.default ];

nput = {
  enable = true;
  entries = [
    { name = "claude-skills"; src = inputs.skills-repo; source = "skills/nix"; target = ".claude/skills/nix"; }
    { name = "dark-theme";    src = inputs.themes;      source = "dark"; target = ".local/share/themes/dark"; mode = "copy"; }
  ];
};
```

### パターン 3：NixOS モジュール

```nix
imports = [ inputs.nput.nixosModules.default ];

nput = {
  enable = true;
  user   = "yasunori";
  entries = [
    { name = "myconfig"; src = inputs.myrepo; source = "config"; target = ".config/myapp"; }
  ];
};
```

---

## 設計上の判断

### name フィールドを必須にする理由

エントリを選択的に適用するためには識別子が必要。
自動生成（インデックスベース等）にすると、エントリの並び替えで意図せず名前が変わる危険があるため、
明示的な指定を必須とする。

### src をユーザー側で渡す理由

ソース取得手段（npins / flake inputs / fetchFromGitHub 等）を本プロジェクトが
抱えることで、取得方法の変更が本プロジェクトの変更を要求する依存が生まれる。
`src` を「フェッチ済みのパスまたはローカルパス文字列」として受け取ることで、
取得手段の変化から完全に独立できる。
string 型のローカルパスを渡した場合は Nix ストアを経由しない out-of-store symlink となり、
ファイル編集が即座に反映されるライブ用途にも対応できる。

### symlink と copy の両対応理由

- symlink：ストアの更新が即座に反映される。読み取り専用。vim プラグイン等に向く。
- copy：ファイルを直接編集したい場合（テーマ・設定の一時調整等）に必要。

### home-manager 非依存を優先する理由

NixOS サーバーや home-manager を使わない環境でも同一の設定を使い回せるようにするため。
また、home-manager の「設定生成モデル」に縛られず、リポジトリ内容をそのまま扱う用途では
standalone の方が透明性が高い。
