# nput 設計書

## 概要

nix store のパス（リポジトリ全体・サブディレクトリ・単一ファイル）を、root 相対の任意パスへ
symlink または copy で配置する Nix ライブラリ・モジュール群。
配置ロジックはテスト可能な純粋関数 + 単一の配置エンジンとして実装し、ユーザーが配置を明示的に握る。
`home.file` 相当（root = `$HOME`）はその一適用に過ぎない（→ `docs/adr/0004`）。

home-manager のような「Nix モジュールオプションから設定を生成する」モデルとは異なり、
リポジトリの内容をそのまま配置することに特化する。設定の生成・変換は行わない。

home-manager に依存せず単体で動作しつつ、home-manager / NixOS / nix-darwin のモジュールシステムとも
統合できる。ただし統合層は配置ロジックを持たず、nput エンジンを起動する薄い配線に徹する（→ `docs/adr/0003`）。

standalone では nix profile に乗せた世代管理（ロールバック）を提供する（→ `docs/adr/0002`）。

---

## 設計目標

| 目標 | 説明 |
|---|---|
| 純粋性・テスト可能性 | 配置ロジックを純粋関数 + 単一エンジンとして実装。モジュールに依存しない |
| 独立性 | home-manager に依存せず単体で動作する |
| 統合性 | HM / NixOS / nix-darwin モジュールから nput エンジンを起動できる |
| 柔軟性 | モジュールシステムを介さず関数として使える |
| 取得手段非依存 | npins / flake inputs / fetchFromGitHub / fetchGit など問わない |
| 粒度 | リポジトリ全体・サブディレクトリ・単一ファイルを配置できる |
| 更新の独立性 | 配置単位（`mkActivationScript`）ごとに個別 profile・個別更新・個別適用できる |
| 非生成 | ファイル内容に関与しない。リポジトリの内容をそのまま置く |
| 世代管理 | standalone は nix profile に乗せてロールバック可能（→ ADR-0002）|
| root 一般化 | 配置先 root を内部でパラメータ化。`$HOME` は一適用、将来 `/` へ拡張可（→ ADR-0004）|

---

## プロジェクト構成

```
<project>/
├── flake.nix              # エントリポイント。outputs を定義
├── flake.lock             # flake 入力のロック
├── lib/
│   ├── default.nix        # 公開 API のまとめ（mkActivationScript / mkOutOfStoreSymlink / listFilesInRepo）
│   ├── types.nix          # entries の型定義（各モジュールで共有）
│   ├── engine.nix         # 配置エンジン（ln / rsync / stale 除去 / 世代）の生成
│   └── out-of-store.nix   # mkOutOfStoreSymlink（マーカー構築子）
└── modules/
    ├── common.nix         # options 定義のみ（全モジュールが import）
    ├── home-manager.nix   # home.activation から nput エンジンを起動
    ├── nixos.nix          # （将来拡張）system.activationScripts から nput エンジンを起動
    └── nix-darwin.nix     # （将来拡張）system.activationScripts から nput エンジンを起動
```

今回の実装スコープは standalone + home-manager をコアとする。NixOS / nix-darwin は将来拡張（→ ADR-0004）。

---

## レイヤー構成

```
lib/engine.nix              ← nixpkgs のみ依存（純粋コア + 配置エンジン）
        ↑ 起動するだけ
modules/common.nix          ← options 型定義（nixpkgs.lib のみ依存）
        ↑
┌───────┼──────────────────┐
HM      NixOS    darwin    standalone
（root と activation hook を供給して nput エンジンを起動する薄い配線のみ）
```

上位層が下位層にのみ依存し、逆方向の依存は持たない。
**配置の振る舞いは全層で `lib/engine.nix` が単一の源**であり、各層はネイティブ機構へ翻訳しない（→ ADR-0003）。

---

## flake.nix outputs 設計

```nix
outputs = { ... }: {
  # モジュール統合
  homeManagerModules.default = ./modules/home-manager.nix;
  # nixosModules / darwinModules は将来拡張（ADR-0004）

  # 関数呼び出し（モジュールシステム不使用）
  lib = import ./lib;
  # lib.mkActivationScript    { pkgs, name, entries } → derivation
  # lib.mkOutOfStoreSymlink   "/abs/path"             → marker（src に渡す）
  # lib.listFilesInRepo       { src, dir? }           → attrset（builtins.readDir 互換）
};
```

---

## コアロジック設計（lib/engine.nix）

### entries スキーマ

各エントリは「どのリポジトリのどのパスを、どこへどのように置くか」という配置単位を表す。
エントリは互いに独立しており、`name` で識別する。

| フィールド | 型 | デフォルト | 必須 | 説明 |
|---|---|---|---|---|
| `name` | string | — | ✓ | エントリ識別子 |
| `src` | path \| set \| marker | — | ✓ | 配置元。デフォルトは store link。out-of-store はマーカー（下記）|
| `source` | string | `"."` | — | リポジトリ内のパス（ファイル・ディレクトリ両対応）|
| `target` | string | — | ✓ | root（デフォルト `$HOME`）からの相対パス |
| `mode` | enum | `"symlink"` | — | `"symlink"` または `"copy"` |

`src` の型による挙動の違い（→ ADR-0001）:

| `src` の値 | symlink の指す先 | 用途 |
|---|---|---|
| `path`（`inputs.foo` 等）| Nix ストア（不変）| flake inputs / `builtins.path` 等 |
| `set`（`fetchFromGitHub` 等）| Nix ストア（不変）| derivation |
| `mkOutOfStoreSymlink "/abs/path"`（marker）| ローカル FS（ライブ）| 開発中の手元 dotfiles |

`string` を直接渡して out-of-store にする暗黙分岐は廃止した。out-of-store は明示関数で opt-in する。

### source の判別ロジック

実行時にストアパスの種別を判定し、適切な処理を選択する。

```
mode = symlink, store/out-of-store → ln -s（ファイル・ディレクトリ問わず共通処理）
mode = copy, source がディレクトリ → place-once: target 不在時のみ rsync src/ dest/
mode = copy, source がファイル     → place-once: target 不在時のみ rsync src  dest
```

### 世代管理と state（→ ADR-0002）

- 純粋関数が **link farm derivation**（ストア内の symlink ツリー）を生成する。
  「配置したもの」のマニフェストは link farm の一部として **store 内に**埋め込む（可変 JSON は持たない）。
- activation スクリプトは:
  1. **前世代の store マニフェスト**と新世代を diff し、消えた entry の **symlink を除去**する（stale 除去）。
     - standalone は自 profile の前世代、モジュール時はホスト世代の旧世代パス（`$oldGenPath` 相当）を参照する。
     - 削除は保守的：前世代マニフェストが「nput が配置した」と記録し、かつ現状もその記録通りを指す symlink のみ。
       通常ファイル・nput 非管理の link には触れない。copy target は除去しない（orphan は警告）。初回は何も消さない。
  2. symlink / out-of-store / place-once copy を配置する。
  3. **standalone のみ** `nix-env --profile <profileDir> --set <link-farm-drv>` で nix profile に登録する。

| 機構 | 役割 | 適用層 |
|---|---|---|
| 世代由来の store マニフェスト | stale 除去のための前回状態（不変・GC-root 済み）| 全層共通 |
| nix profile | 世代番号・GC root・ロールバック | standalone 専用 |

- 配置・cleanup 機構は home-manager の `linkGeneration`/`cleanup` を参考に再実装する（`home.file` 自体は再利用しない）。
- モジュール時は nput 独自 profile を作らず、ホストの世代システムに委譲する。
- **GC**: profile 世代は GC root。旧世代を `nix profile wipe-history` 等で間引き、`nix-collect-garbage` で
  無参照 store パスを解放する。可変 JSON 方式は GC root を作らず rollback が壊れうるため採らない（→ ADR-0002）。

### mkActivationScript の生成物

`pkgs.writeShellApplication` を用いて、ストアパスと target を評価時に埋め込んだ
シェルスクリプトの derivation を返す。実行時にネットワークアクセスは発生しない。

```
mkActivationScript :: { pkgs, name, entries } -> derivation
```

- `name`: profile を一意特定する配置単位名（→ ADR-0002）。`mkActivationScript` 1 呼び出し = 1 profile。
- root は内部でパラメータ化するが、公開 API は当面 root = `$HOME` 固定。
  root 差し替え（将来の system 配置, root = `/`）は**拡張 seam**として残し、安定公開引数にはしない（→ ADR-0004）。

CLI は最小とする。任意世代切替・世代 GC は標準の `nix profile` / `nix-collect-garbage` に委譲する。

```bash
nput                    # 新世代を作って適用
nput --rollback         # 前世代へ戻す
nput --list-generations # 世代一覧
```

`--only`（一部 entry だけ適用）は profile 世代の atomic 性と衝突するため廃止した。
選択的更新は「役割ごとに別スクリプト（別 profile）に分ける」ことで担保する。

---

## モジュール統合設計

### 共通オプション（modules/common.nix）

```nix
# modules/common.nix（全モジュール共通）
options.nput = {
  enable  = mkEnableOption "nput";
  entries = mkOption { type = listOf ...; };
};

# modules/nixos.nix, modules/nix-darwin.nix（将来拡張・各モジュール内で追加定義）
options.nput.user = mkOption { type = str; };
```

`user` オプションは NixOS / nix-darwin のみ必要なため、`modules/common.nix` には含めず各モジュールに分離する。
home-manager と standalone は `$HOME` を直接参照するため不要。

### 各統合層の動作（→ ADR-0003）

すべての層で **nput エンジン**が配置を実行する。各層は root と activation タイミングを供給するだけ。

| 層 | エンジン起動方法 | root の解決 | 世代 |
|---|---|---|---|
| **standalone** | `nix run` でスクリプトを明示実行 | `$HOME`（実行時）| nix profile（あり）|
| **home-manager** | `home.activation` から起動 | `$HOME`（HM が解決）| ホスト（HM）世代に委譲 |
| **NixOS**（将来）| `system.activationScripts` から起動（ただし system パス symlink は `systemd.tmpfiles`）| `config.users.users.<user>.home` | ホスト（nixos）世代に委譲 |
| **nix-darwin**（将来）| `system.activationScripts` から起動 | `config.users.users.<user>.home` | ホスト世代に委譲 |

standalone / home-manager および $HOME レベル配置・copy・out-of-store では `home.file` は**明示的に採らない代替**であり、
stale 除去まで nput が所有する。正確性は世代由来の store マニフェスト（HM 同等のアルゴリズム）で担保する。

**NixOS の部分ハイブリッド（→ ADR-0003）**: 将来拡張の NixOS 層で root=`/` の system パスへ symlink を置く場合のみ、
nput エンジンの `ln` ではなく `systemd.tmpfiles`（`L`/`L+` 型）を使い NixOS の宣言性・追跡に乗せる。
ただし tmpfiles `L` は規則消滅時に作成済み symlink を自動削除しないため、NixOS の stale 除去には別途手当てが要る（将来作業で解決する open 事項）。

### 実行タイミング

| 層 | 実行タイミング |
|---|---|
| standalone | `nix run .#<name>` を明示実行 |
| home-manager | `home-manager switch`（home.activation）|
| NixOS（将来）| `nixos-rebuild switch` |
| nix-darwin（将来）| `darwin-rebuild switch` |

---

## 使用パターン

### パターン 1：standalone（役割ごとに分離して管理）

```nix
# flake.nix
packages.x86_64-linux = {
  vim-plugins = nput.lib.mkActivationScript {
    inherit pkgs;
    name = "vim-plugins";   # = profile 名
    entries = [
      { name = "vim-foo"; src = inputs.vim-foo; target = ".local/share/nvim/site/pack/foo/start/foo"; }
      { name = "vim-bar"; src = inputs.vim-bar; target = ".local/share/nvim/site/pack/bar/start/bar"; }
    ];
  };

  zsh-plugins = nput.lib.mkActivationScript {
    inherit pkgs;
    name = "zsh-plugins";
    entries = [
      { name = "zsh-autosuggestions";     src = inputs.zsh-autosuggestions;     target = ".zsh/plugins/autosuggestions"; }
      { name = "zsh-syntax-highlighting"; src = inputs.zsh-syntax-highlighting; target = ".zsh/plugins/syntax-highlighting"; }
    ];
  };
};
```

```bash
# それぞれ独立した profile として更新・適用・ロールバックできる
nix run .#vim-plugins
nix run .#vim-plugins -- --rollback
nix run .#zsh-plugins
```

### パターン 2：home-manager モジュール

```nix
imports = [ inputs.nput.homeManagerModules.default ];

nput = {
  enable = true;
  entries = [
    # 外部リポジトリ（store link）
    { name = "claude-skills"; src = inputs.skills-repo; source = "skills/nix"; target = ".claude/skills/nix"; }
    # テーマを copy（place-once、以後ユーザー管理）
    { name = "dark-theme";    src = inputs.themes;      source = "dark"; target = ".local/share/themes/dark"; mode = "copy"; }
    # 開発中の手元 dotfiles を out-of-store でライブ反映
    { name = "nvim-config";   src = nput.lib.mkOutOfStoreSymlink "/home/me/dotfiles"; source = "home/.config/nvim"; target = ".config/nvim"; }
  ];
};
```

HM モジュールは `home.activation` から nput エンジンを起動する。配置は nput 自身が行い、
`home.file` には委譲しない。世代は HM のものに乗る（nput 独自 profile は作らない）。

---

## 設計上の判断

### name フィールドを必須にする理由

エントリを識別するため。自動生成（インデックスベース等）にすると並び替えで意図せず名前が変わる危険があるため、明示指定を必須とする。

### mkActivationScript に `name`（profile 名）を要求する理由

世代の粒度を `mkActivationScript` 単位 = 1 profile としたため（→ ADR-0002）。関数は `packages.<name>` の attr 名を
知らないため、profile を一意特定する `name` を明示的に受け取る。

### src をユーザー側で渡す理由

取得手段（npins / flake inputs / fetchFromGitHub 等）を本プロジェクトが抱えると、取得方法の変更が
本プロジェクトの変更を要求する依存が生まれる。`src` を「フェッチ済みのパス」として受け取ることで取得手段の変化から独立する。

### out-of-store を明示関数に降格した理由（ADR-0001）

型ベースの暗黙分岐（`string` → out-of-store）を廃止し store link をデフォルトに統一した。
out-of-store は `mkOutOfStoreSymlink` で明示的に opt-in する。型マジックを排除し、Nix の再現性前提に揃える。

### 世代を nix profile に乗せる理由（ADR-0002）

profile symlink の差し替えだけで atomic な switch / rollback を実現し、GC root にもなる。Nix 標準機構を再利用できる。
純粋関数は derivation とスクリプトを生成するだけで、副作用（profile swap）は activation 実行時に閉じる。

### 配置ロジックをコアが所有する理由（ADR-0003）

振る舞いを単一コアに集約し、テスト可能性とクロスプラットフォームの一貫性を得るため。
ネイティブ機構へ翻訳すると層ごとに挙動が二重化し、nput の「単一コア・ユーザー管理」方針と逆行する。
ただし NixOS の system パス symlink のみ `systemd.tmpfiles` を使う部分例外を設ける（宣言性・標準追跡に乗せるため）。

### stale 除去を「世代由来の store マニフェスト + 保守的削除」にする理由（ADR-0002 / ADR-0003）

可変 JSON 方式は store 外可変で世代に捕捉されず、GC root を作らないため rollback が壊れうる。
代わりに「配置したもの」を link farm の一部として store に埋め込み、前世代の store マニフェストと diff する。
これは不変・GC-root 済みで、home.file を再利用せずとも home-manager の cleanup アルゴリズムを踏襲できる。
削除は保守的に行い（nput が張った link で、現状も記録通りを指す物のみ）、ユーザーの実ファイルを誤って消さない。

### copy を place-once・世代外にする理由（ADR-0002）

copy は元々再適用のたびに手編集を上書きしており、「ユーザー管理の副作用」と明示するのが整合的。
世代に含めると store 外スナップショット管理が重くなる。

### symlink と copy の両対応理由

- symlink：ストアの更新が即座に反映される。読み取り専用。vim プラグイン等に向く。
- copy：ファイルを直接編集したい場合（テーマ・設定の一時調整等）に必要。place-once でユーザーに委ねる。

### home-manager 非依存を優先する理由

NixOS サーバーや home-manager を使わない環境でも同一の設定を使い回せるようにするため。
リポジトリ内容をそのまま扱う用途では standalone の方が透明性が高い。
