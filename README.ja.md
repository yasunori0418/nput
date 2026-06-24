# nput

> フェッチ済みの git リポジトリを、symlink または copy で任意のパスへ配置する。

*この文書は英語版 [`README.md`](README.md) の日本語訳。仕様・用語の一次参照は英語版とし、両者に差異があれば英語版が優先する。*

nput は、**フェッチ済みの Nix store パスの内容を `root` 相対の target へ配置する** Nix ライブラリ・モジュール群(symlink もしくは copy)。設定の生成は **行わない**。リポジトリの内容を、加工せず、指定した場所に置くだけ。

コアは **placement primitive** ——`root` 相対の `target` に Nix store パスを配置する純粋関数。モジュール抽象の裏に隠さず、ユーザーが直接合成して使う。`home.file` 風の配置(`root` = `$HOME`)はその一適用にすぎず、`root` は `projectRoot` / `homeRoot` / `systemRoot` マーカーで**明示的に**選ぶ(**暗黙のデフォルトは持たない**)。

> **ステータス: MVP / 実装フェーズ。** 実装済みの範囲は standalone CLI ＋ **project mode** をコアとし、**home mode** もサポート。NixOS / nix-darwin モジュールおよび **system mode** は将来対応。全体のマトリクスは [MVP ステータス](#mvp-ステータス) を参照。API は今後も変わりうる。

---

## なぜ nput か

Nix はリポジトリを *フェッチ* できる(`fetchFromGitHub`、`fetchGit`、flake inputs、npins など)が、その内容をファイルシステム上に *配置する* のは別問題。既存の解にはそれぞれコストがある。

- **home-manager `home.file`** はファイルを配置できるが、home-manager を要求し、*環境まるごと* のモデルを前提とする。`home-manager switch` 一発ですべてが一斉に更新されるため、一つの変更が無関係なツールへ波及——あるいはそれを壊す——ことがある。file モジュールを standalone なライブラリとして切り出すこともできない。
- **シェルの `git clone`** は動くが、バージョン固定・再現性・Nix 統合を失う。
- **モジュール抽象**(home-manager / NixOS / nix-darwin / system-manager)は配置を Nix モジュールシステムで宣言し、挙動を抽象の裏に隠して、プラットフォーム固有の機構(`home.file`、`systemd.tmpfiles` など)へ翻訳する。これは「何をどこへ、どう置くか」の制御をユーザーの手から奪い、挙動を層ごとに重複させる。

nput は **フェッチ**(Nix 評価：`src` は store パス)と **配置**(固定の実行時エンジン)を分離し、配置の挙動を **単一のコア** に閉じてユーザーが明示的に駆動する。

- **設定生成をしない。** nput はモジュールオプションを設定ファイルへ翻訳しない。リポジトリにすでにある内容を配置する。
- **独立した単位。** 各配置 config(`nput.<name>`)はそれ自体が独立した Nix profile。役割ごとに独立して更新・適用でき、ある更新が別へ波及しない。
- **home-manager 非依存。** `lib/` コアは nixpkgs のみに依存する。standalone で動き、module 統合(home-manager、devShell、将来の NixOS / nix-darwin)はエンジンを *起動するだけ* の薄い配線であって、自身でファイルを配置することはない。

---

## 仕組み

nput は 2 層構成。

```
[nput CLI]  packages.nput — PATH 上に乗る一次 UX
  · entrypoint を発見する(flake.nix / shell.nix / default.nix)
  · 内部で `nix build` / `nix eval` を回して named manifest を取得する
  · エンジンを駆動して配置・stale link 除去・profile 切替を行う
   ↓ manifest.json
[engine]    Go ライブラリ(stdlib-only)
  · manifest.json を入力に取り、`nix`(profile)と `git`(toplevel)のみを叩く
  · place / replace / remove のネイティブなファイルシステム操作 ＋ 保守的な stale 除去
```

- **engine** が配置と stale 除去を所有する。`manifest.json`——安定した Nix↔Go の契約——を読み、ネイティブなファイルシステム操作を実行する。
- `lib.mkManifest` は link-farm derivation(`manifest.json` ＋ symlink farm)を生成する **純粋関数**。副作用を持たない。
- **entrypoint** は CLI が読む Nix ファイル(`flake.nix` / `shell.nix` / `default.nix`)で、`nput.<name>` に named manifest を公開する。config は依然として Nix で書かれ、`nix build` で評価される。

---

## 要件

- **Nix**。環境で experimental features を有効化していること：`experimental-features = nix-command`(flake entrypoint には `flakes` も)。nput は `--extra-experimental-features` を黙って注入 **しない**。機能が無効なら、前提条件と有効化方法を明示するメッセージを出して停止する。
- **git** が `PATH` 上にあること(project mode でプロジェクトルートを解決するために使う)。

---

## インストール

### Standalone(home mode)

CLI をグローバルにインストールし、`nput` を `PATH` に乗せる。

```bash
nix profile install github:yasunori0418/nput
```

`schemaVersion` のずれを避けるため、CLI と flake が固定する `nput` を **同一 input** に揃える(グローバル CLI と flake 側の `nput.lib` は別 input で drift しうる。エンジンは自分より新しい `schemaVersion` を拒否する)。

### Project mode(canonical: devShell に固定する)

project mode の canonical な形は、プロジェクトの devShell に **固定した `nput` を同梱** すること。これで CLI と `nput.lib` が同一 flake input(`flake.lock` で固定)から来る。

```nix
devShells.${system}.default = pkgs.mkShell {
  packages  = [ nput.packages.${system}.nput ];   # 固定した nput を PATH に
  shellHook = "nput apply <name> --no-wait";        # `nix develop` / direnv 進入時に配置
};
```

### 新規プロジェクトの scaffold

`nput init` は `nix flake init -t` の透過的なラッパー。nput 自身は何も生成せず、既存ファイルを上書きしない。

```bash
nput init standalone   # homeRoot の例
nput init project      # projectRoot の例 ＋ devShell 配線 ＋ .gitignore ガイド
```

---

## Quickstart

### Project mode(中心的な使い方)

**project mode** では root はプロジェクトルート(git toplevel)。配置物は **ephemeral**——クローンごとに再生成され、コミットされない——なので activation が git 状態に干渉しない。これが nput の中心的な使い方：リポジトリに組み込み、devShell から起動して、store パスを任意の in-repo パスへ配置する。

```nix
# flake.nix — リポジトリ進入時に、フェッチ済み store パスから .claude/skills/nix を配置する
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.nput.url    = "github:yasunori0418/nput";

  inputs.claude-skills.url   = "github:someone/claude-skills";
  inputs.claude-skills.flake = false;

  outputs = { self, nixpkgs, nput, ... }@inputs:
    let
      system = "x86_64-linux";
      pkgs   = nixpkgs.legacyPackages.${system};
    in
    {
      nput.${system}.skills = nput.lib.mkManifest {
        root = nput.lib.projectRoot;        # 実行時に git toplevel へ解決される
        entries = {
          ".claude/skills/nix" = { src = inputs.claude-skills; subpath = "skills/nix"; };
        };
      };

      devShells.${system}.default = pkgs.mkShell {
        packages  = [ nput.packages.${system}.nput ];   # 固定した nput(canonical)
        shellHook = "nput apply skills --no-wait";       # shell 進入時に配置
      };
    };
}
```

```bash
# shell に入る — shellHook 経由で配置が自動実行される
nix develop          # または: direnv allow

# プロジェクト所有者が .gitignore に加えるべき target を一覧する(stdout のみ・書き込みなし)
nput gitignore skills >> .gitignore
```

- `--root <path>` は任意モードで解決済み root を上書きする(git なしの木やデバッグ用の退避路)。
- ここでは世代は内部機構であり、`rollback` / `list-generations` は project mode では **公開されない**(ephemeral な配置に rollback は無意味)。
- devShell では **named apply**(`nput apply skills`)か `nput apply --all --project-root` を使う。裸の `--all` は home-mode config も `$HOME` へ配置してしまうため、混在 entrypoint では footgun。

### Home mode(standalone、役割を別 profile に分ける)

**home mode** では root は `$HOME`(`homeRoot` で選ぶ)。各 config はそれ自体が独立した profile で、apply ごとにコミットされ、ユーザー公開の `rollback` を持つ。

```nix
# flake.nix — 各役割が named manifest = 独立した profile
outputs.nput.${system} = {
  vim-plugins = nput.lib.mkManifest {
    root = nput.lib.homeRoot;
    entries = {
      ".local/share/nvim/site/pack/foo/start/foo" = { src = inputs.vim-foo; };
      ".local/share/nvim/site/pack/bar/start/bar" = { src = inputs.vim-bar; };
    };
  };

  zsh-plugins = nput.lib.mkManifest {
    root = nput.lib.homeRoot;
    entries = {
      ".zsh/plugins/autosuggestions"     = { src = inputs.zsh-autosuggestions; };
      ".zsh/plugins/syntax-highlighting" = { src = inputs.zsh-syntax-highlighting; };
    };
  };
};
```

```bash
# 各役割を独立して更新 / 適用 / rollback できる(別 profile)
nput apply vim-plugins
nput rollback vim-plugins          # home mode のみ
nput apply zsh-plugins
nput list-generations vim-plugins
```

`src` を更新した後(flake input 更新、`npins update` など)は、影響を受けた config だけを再適用すれば、変更が他のツールに触れずに済む。

### home-manager モジュール

モジュールは `root = homeRoot` に固定される(`root` を再指定しない)。`home.activation` から `nput apply --manifest <link-farm>` でエンジンを起動し、`home.file` へは委譲しない。nput は自前の profile を **内部機構** として保持するが、ユーザー公開の rollback はホスト側に一本化される(`home-manager --rollback`)。

```nix
imports = [ inputs.nput.homeManagerModules.default ];

nput = {
  enable = true;
  entries = {
    # 外部リポジトリ(store link)
    ".claude/skills/nix" = { src = inputs.skills-repo; subpath = "skills/nix"; };
    # テーマを copy で(一度配置したら以降ユーザー管理)
    ".local/share/themes/dark" = { src = inputs.themes; subpath = "dark"; method = "copy"; };
    # ローカル dotfiles を out-of-store symlink でライブ編集
    ".config/nvim" = { src = nput.lib.mkOutOfStoreSymlink "/home/me/dotfiles"; subpath = "home/.config/nvim"; };
  };
};
```

> home-manager モジュールは MVP では **単一 manifest = 一つの profile**(固定名 `default`)であり、`<name>` 次元を持たないため、**役割分離はモジュール経由では使えない**。複数の独立 profile が要るなら standalone CLI 経路(`nput.<name>` entrypoint)を使う。

### 既存の flake に nput を追加する

`nput init` は新規プロジェクト向け。既存の `flake.nix` にあとから組み込むには、次の 4 ステップを手で行う(nput は flake を自動マージしない——「設定を生成しない」)。

1. **input を追加**：`inputs.nput.url = "github:yasunori0418/nput";`
2. **manifest を公開**：
   `outputs.nput.<system>.<name> = nput.lib.mkManifest { root = nput.lib.projectRoot; entries = { ... }; };`
3. **固定した nput を devShell に同梱**：`packages = [ nput.packages.${system}.nput ];`
4. **named apply を配線**：`shellHook = "nput apply <name> --no-wait";`

リポジトリが **flake-parts** を使うなら、ステップ 2 を flake module 経由で書く(`pkgs` を `perSystem` と整合させられる)。

```nix
imports = [ inputs.nput.flakeModules.default ];
perSystem = { pkgs, ... }: {
  nput.<name> = inputs.nput.lib.mkManifest {
    inherit pkgs;
    root = inputs.nput.lib.projectRoot;
    entries = { ... };
  };
};
# flake-parts はこれを flake.nput.<system>.<name> へ転置する — CLI のアドレッシングは変わらない。
```

> `nix flake check` は `warning: unknown flake output 'nput'` を報告するが exit 0(無害・想定どおり——`nput` 名前空間は manifest を `packages` の外に保つ)。成果物は `nix build .#nput.<system>.<name>` で検証する。

---

## `entries` スキーマ

`entries` は **`target` をキーとする attrset**——属性キーが識別子であり、既定の `target` を与える。一意性は Nix の attrset キーで native に担保され、手動の `name` フィールドは無い。

```nix
entries = {
  "<target>" = {
    src     = ...;          # 必須
    subpath = ".";          # 任意・デフォルト "."(リポジトリ全体)
    target  = "<target>";   # 任意・デフォルト = 属性キー
    method  = "symlink";    # 任意・デフォルト "symlink" | "copy"
  };
  # ...
};
```

entry submodule は **strict**(未知のキーを拒否する)。typo や旧名(`name` / `source` / `dir` / `mode`)は評価時エラー。

### `src` — 必須

配置元。デフォルトは **store link**(Nix store への symlink。再現性を担保する)。out-of-store は明示マーカーで opt-in する。

| `src` の値 | symlink が指す先 | 用途 |
|---|---|---|
| `path`(例：`inputs.myrepo`) | Nix store(不変) | バージョン固定の外部リポジトリ |
| `builtins.path { path = /home/...; name = "..."; }` | Nix store(ローカルを取り込み) | store 経由のローカル木 |
| `set`(例：`pkgs.fetchFromGitHub { ... }`) | Nix store(不変) | バージョン固定の外部リポジトリ |
| `marker`(`nput.lib.mkOutOfStoreSymlink "/abs/path"`) | ローカル FS(ライブ) | 開発中のローカル dotfiles |

```nix
src = inputs.myrepo;                                    # store link
src = pkgs.fetchFromGitHub { owner = "..."; repo = "..."; rev = "..."; hash = "..."; };
src = builtins.path { path = /path/to/dotfiles; name = "dotfiles"; };
src = nput.lib.mkOutOfStoreSymlink "/path/to/dotfiles"; # out-of-store(ライブ)・明示

# 廃止: 裸の文字列で暗黙の out-of-store にするのはサポートしない。
# src = "/path/to/dotfiles";   # エラー
```

### `subpath` — デフォルト `"."`

**`src` 内** のどのパスを取り出すかを選ぶ相対パス。file / dir 両対応。省略(または `"."`)でリポジトリ全体を選ぶ。

```nix
subpath = ".";                  # リポジトリ全体(明示形)
subpath = "skills/nix";         # サブディレクトリ
subpath = "themes/dark.json";   # 単一ファイル
```

`src` と `subpath` は直交する：`src` = *どの物*(store パス / リポジトリ)、`subpath` = *その中のどのパス*。

### `target` — デフォルト = 属性キー

配置先。**`root` からの相対パス**。属性キーが既定の `target` であり、entry の identity(stale 除去の diff キーであり、一意性のキー)でもある。キーを論理ラベルにして `target` を明示上書きすることもできる(`home.file` と同様)。

### `method` — デフォルト `"symlink"`

配置の種別を選ぶ。

| `method` | `src` の種類 | 挙動 | 世代 |
|---|---|---|---|
| `"symlink"` | path / set | Nix store への symlink(read-only) | あり(profile) |
| `"symlink"` | marker | ローカルパスへの out-of-store symlink(ライブ) | あり(link 先のみ) |
| `"copy"` | path / set | **一度だけ配置する** copy(書き込み可・ユーザー管理) | **なし** |
| `"copy"` | marker | 評価時エラー(矛盾) | — |

**copy は place-once でユーザー管理。** 一度実体化したら、nput は target に触れない。store の read-only モード(`0444` / `0555`)は保持しつつ owner-write を加えるので、copy は編集できる。上流 `src` の更新に追従するには `nput apply --recopy`(全 copy target を無条件に上書き)か `nput reset` 後に再適用する。copy は世代管理されず、rollback もされない。copy target に外部の実ファイルがすでにある場合、nput はスキップしつつ **警告** を出す(あなたのファイルを上書きしない)。

### entry を動的に生成する

名前を target キーに展開する。target は **あなたが** 制御する変数から導出する——`baseNameOf src` からではない(store パスは `/nix/store/<hash>-source` に解決されるので、`baseNameOf` は `<hash>-source` を返す)。

```nix
let plugins = [ "telescope" "treesitter" "cmp" ]; in
nput.lib.mkManifest {
  root = nput.lib.homeRoot;
  entries = builtins.listToAttrs (map (n: {
    name  = ".local/share/nvim/site/pack/plugins/start/${n}";  # キー = target
    value = { src = inputs.${n}; };
  }) plugins);
}
```

サブディレクトリを列挙するには、**実体化済みの store パス / `flake = false` input** を `builtins.readDir` する(生の `fetchFromGitHub` derivation は import-from-derivation を起こし、純粋な flake 評価を壊す)。

```nix
let
  skills = builtins.readDir "${inputs.claude-skills}/skills";
  names  = builtins.attrNames (nixpkgs.lib.filterAttrs (_: t: t == "directory") skills);
in
nput.lib.mkManifest {
  root = nput.lib.homeRoot;
  entries = builtins.listToAttrs (map (n: {
    name  = ".claude/skills/${n}";
    value = { src = inputs.claude-skills; subpath = "skills/${n}"; };
  }) names);
}
```

---

## コマンドリファレンス

CLI は CWD の entrypoint を発見する(`flake.nix` → `shell.nix` → `default.nix`)。`-f` で上書き可。各 `nput.<name>` は独立した profile で、`<name> = default` は名前省略時に `nput apply` が解決する。

```bash
nput apply [<name>]            # nput.<name> を適用(省略 = nput.default)。build・新世代コミット・配置を行う
nput apply <name> --dryrun     # read-only な計画: place/replace/remove/conflict/no-op、副作用ゼロ
nput apply <name> --recopy     # さらに全 copy target を src から無条件に上書きする
nput apply --manifest <farm>   # ビルド済み link-farm を直接適用(entrypoint 発見 / eval / build なし)
nput apply --all               # 全 nput.* を辞書順で適用。失敗しても続行する
nput apply --all --project-root # projectRoot config のみ適用(--home-root / --system-root も同様)
nput reset <name> [target...]  # 配置を撤去(profile 変更なし)。target 省略 = 全 entry
nput reset <name> --dryrun     # 削除対象を表示。副作用ゼロ
nput rollback <name>           # 直前の世代へ戻す(home mode のみ・name 必須)
nput list-generations <name>   # 世代を一覧(home mode のみ)
nput list-generations --all    # 全 home-mode config の世代を一覧
nput gitignore <name>          # 配置 target を .gitignore 用に stdout へ出力(書き込みなし・project mode のみ)
nput gitignore --all           # 全 projectRoot config の target をソート＋重複排除して出力
nput init <template>           # `nix flake init -t github:yasunori0418/nput#<template>` のラッパー
```

### グローバルフラグ

```text
-f, --file <path>   # entrypoint を明示指定(自動発見を上書き)
--root <path>       # 任意モードで解決済み root を上書き
--no-wait           # ロック競合時に待たずスキップ(shellHook 用。明示 apply は既定でブロック)
-v, --verbose       # 配置レポートを出力(サマリ＋target ごとの行)。既定は成功時サイレント
--debug             # 内部 nix コマンドを stderr に出す(トラブルシュート用)
--project-root      # --all の限定子: projectRoot config のみ(--home-root / --system-root も同様)
--recopy            # apply の限定子: 全 copy target を src から上書き
--manifest <path>   # apply 専用: ビルド済み link-farm を直接適用
-y, --yes           # reset の確認プロンプトを省略(スクリプト / CI 用)
```

### 出力と終了コード

- **既定では成功時サイレント**("silence is golden")。配置レポート、try-lock のスキップ通知、`apply --all` のサマリは `-v` / `--verbose` を付けない限り **出力しない**。`-v` で stderr のレポートに opt-in する。
- **`--debug`** は内部 nix コマンドを明らかにする(verbosity の `-v` とデバッグは直交)。`--quiet` は **無い**(成功時サイレントが既定になった時点で廃止)。MVP に `--json` も **無い**。
- **ストリーム規律**：stdout は機械可読出力(`gitignore` 一覧、`apply --dryrun` 計画)専用で、既定 verbosity でも出力される。よって `nput gitignore <name> >> .gitignore` や `nput apply <name> --dryrun | ...` は安全にパイプできる。**警告(例：外部 symlink)とエラーは常に stderr へ出力され、サイレンスされない。**

| 終了コード | 意味 |
|---|---|
| `0` | 成功 / no-op / `--no-wait` の try-lock スキップ |
| `1` | 一般エラー(eval エラー、エンジン実行時エラー、`apply --all` の部分失敗) |
| `2` | `apply --dryrun` が conflict を検出(CI の事前ゲートに使える) |

### 挙動メモ

- **冪等。** 再適用は同じ結果に収束する。symlink については、nput は「配置したと記録し、かつ記録どおりを指している」stale link だけを保守的に除去する——あなたの実ファイルや外部 link には決して触れない。既存の nput symlink は置き換えられ、外部 symlink は警告付きで置き換えられ、target にある実ファイル / ディレクトリはエラー(上書きしない)。
- **世代** は nput 自前の Nix profile(`nix-env --profile <dir>`)に乗る。任意世代への切替・間引き・GC は、profile パスに対して標準の `nix-env` / `nix-collect-garbage` で行う。project mode は link-farm が不変なら新世代コミットをスキップする(ただし drift した entry は `lstat` で修復する)。
- **`apply --all`** は各 config を独立に適用し(各々が自分の profile 上で atomic)、失敗しても続行し、いずれかが失敗すれば非ゼロで終了する。全体としては **atomic ではない**。
- **`reset`** はファイルシステムのみの撤去：nput 管理 symlink を(保守的に)除去し、**さらに** copy target を削除する(copy を消す唯一の明示手段)。name 必須(`--all` 非対応)、確認または `-y` 必須、profile / 世代には触れない——config に残っている entry は次の apply で再配置される。

---

## 他ツールとの比較

軸は「機能の有無」ではなく、**「配置をモジュール抽象の裏に隠すか、ユーザーが制御する純粋関数として露出させるか」**。

| ツール | 役割 | アプローチ | nput との違い |
|---|---|---|---|
| npins / niv | ソースのバージョン固定 | — | ファイルを配置しない(直交——nput と組み合わせる) |
| home-manager `home.file` | ファイル配置 ＋ 世代 | module(生成 / 宣言) | HM 必須。環境まるごとモデル。file モジュールを standalone に切り出せない |
| `mkOutOfStoreSymlink`(HM) | out-of-store symlink | module 内のヘルパー | HM 専用。nput は同等物を依存なしの明示関数として提供 |
| nixpkgs `linkFarm` / `symlinkJoin` | store 内 symlink ツリー | 純粋関数 | 出力は store *内* に留まり、任意の out-of-store パスへは置かない(nput は内部で使う) |
| `nix profile` | 世代管理 | — | 配置先が `~/.nix-profile` に固定。任意パス配置なし(nput はこれに乗る) |
| `systemd.tmpfiles`(`L`) | 任意パスへの宣言的 symlink | module(NixOS) | 低水準・NixOS 専用。copy / 世代 / フェッチ抽象なし |
| numtide/system-manager | 非 NixOS の `/etc` ＋ systemd ＋ パッケージ | module(`lib.evalModules`) | ドメインは重なるが **逆** のアプローチ。任意パス配置・HOME dotfiles・サブディレクトリ抽出なし |
| `git clone`(シェル) | clone して配置 | 命令的 | 再現性も Nix 統合もない |
| **nput** | フェッチ済みソースの独立配置 ＋ 世代 ＋ 明示的 out-of-store | **純粋関数・ユーザー管理** | — |

ほぼ同一の既存ツールは無い。構成要素(symlink farm、nix profile、out-of-store、任意パス symlink)はすべて存在するが、それらを「フェッチ非依存 ＋ 非生成 ＋ entry ごとの適用 ＋ HM 非依存の純粋関数コア ＋ クロスプラットフォーム共有スキーマ ＋ 任意パス配置 × 世代管理」として束ねるのは nput だけ。特に nput は system-manager と **競合しない**：ドメイン(パッケージ / systemd / `/etc`)は重なるが、思想(「モジュールに隠す」vs.「純粋関数として露出する」)が設計レベルで異なる。実用的なディストロ基盤では、system / service / package 層は system-manager へ委譲または結合し、nput は **粒度の細かい任意パス配置 primitive** に留まる。

---

## MVP ステータス

| 領域 | ステータス |
|---|---|
| Standalone CLI(`apply` / `reset` / `rollback` / `list-generations` / `gitignore` / `init`) | 実装済み(コア) |
| project mode(`projectRoot`) | 実装済み(コア) |
| home mode(`homeRoot`) | 実装済み |
| home-manager モジュール | 実装済み——単一 profile(固定名 `default`)・役割分離なし |
| 世代 / rollback(home mode) | 実装済み |
| copy(place-once)/ out-of-store symlink | 実装済み |
| flake-parts モジュール | 実装済み |
| `manifest.json` スキーマ | v1 のみ。migration / 後方互換の仕組みはまだ無い |
| `--json` 機械可読出力 | MVP 対象外(将来) |
| NixOS / nix-darwin モジュール | 将来 |
| system mode(`systemRoot` = `/`) | 将来(seam のみ。今選ぶと評価時エラー) |

**既知の制限 / 正直な注意点**

- ディストロ north-star 向けの関数ベース「パッケージインストール ＋ PATH」機構は未定義でスコープ外。
- boot / init / filesystem / partition 層は nput のドメインではない。
- クローンを削除すると `<state>/nix/profiles/nput/` 下に orphan な profile ディレクトリが残る(store は `nix-collect-garbage` で解放されるが、profile ディレクトリは残る)。MVP に `prune` コマンドは無い——手で消す。
- home-manager モジュールは MVP では役割を複数 profile に分けられない——その用途には standalone CLI を使う。

---

## ドキュメント

完全な設計書・仕様書は現在日本語で保守されている。

- `docs/concept.md` — コンセプト、設計哲学、既存ツールとの比較
- `docs/design.md` — 設計(レイヤー、flake outputs、モジュール設計、使用パターン)
- `docs/spec.md` — 仕様(lib API、entries スキーマ、配置動作、エラー仕様)
- `docs/glossary.md` — 正準な英語用語(日本語対訳は `docs/glossary.ja.md`)
