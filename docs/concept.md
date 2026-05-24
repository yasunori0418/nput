# nput コンセプト

## 解決したい課題

### Nix での外部リポジトリ配置に既製手段がない

- `fetchFromGitHub` / `fetchGit` でリポジトリは取得できるが、ファイルシステムへの配置は別の話
- home-manager の `home.file` を使えば配置できるが、home-manager への依存が前提になる
- シェルスクリプトで `git clone` すれば動くが、バージョン固定・再現性・Nix との統合が失われる

### home-manager の「全体管理」モデルの問題

home-manager は強力だが、全てを管理対象にすることを前提とするモデルである。

- vim プラグインも、シェル設定も、テーマも、全て home-manager の管理下に入れる必要がある
- 設定が大きくなるにつれて管理が煩雑になる
- `home-manager switch` は全てを一括で更新するため、**1つの変更が全体に影響する**
- あるツールの更新が別のツールの設定を壊す「共倒れ」が発生しうる
- 更新のタイミングをツールごとに制御できない

---

## コンセプトの核心

**「役割を分離し、各役割を独立して管理・更新できるようにする」**

**「設定を生成しない。リポジトリの内容をそのまま置く」**

home-manager のように全体を一括管理するのではなく、用途ごとに独立した配置単位（エントリ）を定義する。
各エントリは互いに独立しており、任意のタイミングで個別に更新・適用できる。

また、home-manager はユーザー設定を **Nix モジュールオプションから生成する** モデルである
（例：`programs.neovim.enable = true` → neovim 設定ファイルを生成する）。
本ツールはそのような設定生成を行わない。リポジトリに存在する内容をそのまま指定した場所に置くだけであり、
ファイルの内容に関与しない。

```
home-manager モデル:
  Nix モジュールオプション → 設定ファイルを生成 → 配置

nput モデル:
  リポジトリの内容 → そのまま配置（生成・変換なし）
```

対象は「既に存在するリポジトリの内容」であり、Nix の設定言語で表現された設定ではない。

また、`src` に string 型のローカルパスを渡すことで Nix ストアを経由しない
out-of-store symlink を作成できる。手元の dotfiles リポジトリを直接参照するため、
ファイルを編集すると即座に反映される。

| `src` の型 | 反映タイミング | 向いている用途 |
|---|---|---|
| path / set（ストア経由） | flake 更新 + 適用時 | バージョン固定した外部リポジトリ |
| string（ローカルパス） | ファイル編集と同時（ライブ） | 手元の dotfiles リポジトリ |

```
home-manager モデル:
  [全設定] → switch → 全てが一括で更新される

nput モデル:
  [vim-plugins]    → 任意のタイミングで独立更新
  [zsh-plugins]    → 任意のタイミングで独立更新
  [claude-skills]  → 任意のタイミングで独立更新
```

---

## 役割の分離 — 具体的な用途

### vim プラグイン管理

vim/neovim のプラグインは従来 `git clone` で特定ディレクトリに配置するパターンが多い。
これを Nix で再現し、バージョンを固定する。

```nix
{ name = "vim-plugin-foo"; src = inputs.vim-plugin-foo; target = ".local/share/nvim/site/pack/foo/start/foo"; }
```

### コーディングエージェントのスキル

Claude Code などのコーディングエージェントはスキルをリポジトリで管理することがある。
リポジトリ全体ではなく、特定のサブディレクトリだけを取り出して配置したい。

```nix
{ name = "claude-skills"; src = inputs.skills-repo; source = "skills/nix"; target = ".claude/skills"; }
```

### zsh / bash プラグイン

zsh や bash のプラグインは特定ディレクトリに配置し、設定ファイルから `source` で読み込む
パターンが一般的。プラグインごとに独立管理することで、1つの更新が他に波及しない。

```nix
{ name = "zsh-autosuggestions"; src = inputs.zsh-autosuggestions; target = ".zsh/plugins/autosuggestions"; }
{ name = "zsh-syntax-highlighting"; src = inputs.zsh-syntax-highlighting; target = ".zsh/plugins/syntax-highlighting"; }
```

---

## 独立した更新サイクル

各エントリは `name` で識別され、個別に適用できる。

```bash
# 全エントリを一括適用
nput

# 特定エントリのみ更新（他には影響しない）
nput --only zsh-autosuggestions
```

`src` を更新（flake input の更新 / npins update 等）した後、
対象エントリだけを再適用することで、他のツールへの影響を完全に排除できる。

---

## 設計の哲学

### 取得と配置の分離

```
取得：Nix の評価フェーズ（src = ストアパス）
配置：Nix 管理下の実行フェーズ（symlink / copy）
```

取得手段（npins / flake inputs / fetchFromGitHub 等）をツール側が抱えることをやめ、
「フェッチ済みのストアパス（`src`）を受け取る」設計にすることで、取得方法の変化から独立する。

### home-manager に依存しない

home-manager はユーザー環境管理の強力なツールだが、すべての環境で使われるわけではない。
NixOS サーバー・最小構成の環境でも同じ設定定義で動くことを優先する。

### 統合は「オプション」

モジュールとして使えることは便利だが、コアロジックはモジュールシステムに依存しない関数として
実装する。統合層（HM / NixOS / nix-darwin）はコアの薄いラッパーに過ぎない。

```
lib（コア）は何にも依存しない
モジュール層は lib + 各プラットフォームの変換ルールだけを持つ
```

### 冪等性

同じ設定を何度実行しても同じ結果になる。

- symlink：既存 symlink を置き換える。通常ファイル・ディレクトリがある場合はエラー停止
- copy：rsync `--delete` により常にストアの内容と一致させる

### 粒度の柔軟性

リポジトリ全体・サブディレクトリ・単一ファイルを同一インターフェースで扱う。
実行時にファイル・ディレクトリを判別するため、呼び出し側は型を意識しなくてよい。

---

## 既存ツールとの比較

| ツール | 役割 | 問題点 |
|---|---|---|
| npins / niv | ソースのバージョン固定 | 配置は行わない |
| home-manager `home.file` | ファイル配置 | HM 必須。全体管理モデル。一括更新で共倒れリスク |
| `git clone`（シェル） | クローンと配置 | 再現性・Nix との統合がない |
| **本ツール** | 取得済みソースの独立配置 + ローカル out-of-store symlink | — |

本ツールは home-manager の代替ではなく、「特定のリポジトリを特定の場所に置く」という
単一の責務に集中した補完的なツールである。home-manager と併用も可能。

---

## 設計の変遷（会話の流れ）

| フェーズ | 検討内容 | 採用した方向 |
|---|---|---|
| 起点 | `fetchFromGitHub` + `lock.json` + シェルスクリプト | ロック管理をシェルで実装 |
| ロック管理 | シェルスクリプト vs Nix 関数 | 副作用が必要なため Nix 純粋関数では不可。`pkgs.writeShellApplication` でパッケージ化か `npins` を使う |
| ソース管理 | npins の採用 | npins は `sources.json` でロック管理し `import ./npins` で attrset を返す。シェルより宣言的 |
| 配置手段 | home-manager 依存可否 | `home.file`（symlink）/ `home.activation`（copy）で対応可能だが HM 依存が前提になる |
| 独立性 | HM 非依存 + HM 統合の両立 | コアを純粋関数として切り出し、モジュール層で各プラットフォームに変換する3層設計を採用 |
| スコープ拡大 | NixOS / nix-darwin 対応 | 同一 `entries` スキーマを共有し、変換先のみを各モジュールが担う |
| src 設計 | npins を内包するか | `src` をストアパスとして受け取ることで取得手段を問わない設計に |
| 役割分離 | home-manager 的全体管理 vs 役割ごとの独立管理 | エントリに `name` を持たせ、個別更新・個別適用できる設計に |

---

## 想定ユースケース

- vim/neovim プラグインをバージョン固定で `~/.local/share/nvim` 以下に配置
- Claude Code や他のエージェントのスキルリポジトリから特定ディレクトリだけを取り出して配置
- zsh/bash プラグインをプラグインごとに独立管理し、任意のタイミングで個別更新
- カラーテーマリポジトリから特定テーマだけを `~/.local/share/themes` にコピー
- 社内共有の設定リポジトリから個人環境用の設定を取り込む
- 複数マシン（Linux / macOS）で同一のリポジトリ配置設定を共有する
