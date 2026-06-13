# nput 設計書

## 概要

nix store のパス（リポジトリ全体・サブディレクトリ・単一ファイル）を、root 相対の任意パスへ
symlink または copy で配置する Nix ライブラリ・モジュール群。
配置ロジックはテスト可能な純粋関数 + 単一の配置エンジンとして実装し、ユーザーが配置を明示的に握る。
root は `projectRoot` / `homeRoot` / `systemRoot` で**明示的に選ぶ**（暗黙デフォルトなし）。`home.file` 相当（`homeRoot`）はその一適用に過ぎない（→ `docs/adr/0004`, `docs/adr/0007`）。

nput は **project-first** に positioning する（→ ADR-0007）。中心的な使い方はプロジェクト内配置（project mode）であり、ユーザーは PATH 常駐の `nput` CLI を叩く。CLI が entrypoint（`flake.nix` / `shell.nix` / `default.nix`）を発見し、内部で `nix build` して得た manifest を配置エンジンに渡す。

home-manager のような「Nix モジュールオプションから設定を生成する」モデルとは異なり、
リポジトリの内容をそのまま配置することに特化する。設定の生成・変換は行わない。

home-manager に依存せず単体で動作しつつ、home-manager / NixOS / nix-darwin のモジュールシステムとも
統合できる。ただし統合層は配置ロジックを持たず、nput エンジンを起動する薄い配線に徹する（→ `docs/adr/0003`）。

standalone（home mode）では nix profile に乗せた世代管理（ロールバック）を提供する（→ `docs/adr/0002`）。

---

## 設計目標

| 目標 | 説明 |
|---|---|
| 純粋性・テスト可能性 | 配置ロジックを純粋関数 + 単一エンジン（Go ライブラリ）として実装。モジュールに依存しない |
| 独立性 | home-manager に依存せず単体で動作する |
| 統合性 | HM / NixOS / nix-darwin モジュールから nput エンジンを起動できる |
| 柔軟性 | モジュールシステムを介さず CLI + entrypoint ファイルで使える |
| 取得手段非依存 | npins / flake inputs / fetchFromGitHub / fetchGit など問わない |
| 粒度 | リポジトリ全体・サブディレクトリ・単一ファイルを配置できる |
| 更新の独立性 | 配置単位（`nput.<name>`）ごとに個別 profile・個別更新・個別適用できる |
| 非生成 | ファイル内容に関与しない。リポジトリの内容をそのまま置く |
| 世代管理 | standalone（home mode）は nix profile に乗せてロールバック可能（→ ADR-0002）|
| root 明示 | 配置先 root を明示マーカーで選ぶ。project / home / system / 固定パスに同じ関数で到達（→ ADR-0004, ADR-0007）|

---

## プロジェクト構成

```
<project>/
├── flake.nix              # エントリポイント。outputs を定義（packages.nput / lib / templates / modules）
├── flake.lock             # flake 入力のロック
├── lib/                   # 純データ生成（nixpkgs.lib のみ依存・→ ADR-0006）
│   ├── default.nix        # 公開 API のまとめ（mkManifest / mkOutOfStoreSymlink / projectRoot / homeRoot / systemRoot / listFilesInSrc）
│   ├── types.nix          # entries の型定義（各モジュールで共有）
│   ├── manifest.nix       # mkManifest（manifest.json + symlink farm derivation を生成する純粋関数）
│   └── out-of-store.nix   # mkOutOfStoreSymlink / projectRoot / homeRoot / systemRoot（マーカー構築子）
├── cmd/nput/              # nput CLI のエントリポイント（packages.nput・一次 UX・→ ADR-0007）
│   └── main.go            # entrypoint 発見 + nix build/eval オーケストレーション + サブコマンド分岐（apply [--dryrun] / rollback / list-generations / gitignore / init）
├── internal/              # 配置・diff・保守的 stale 除去の純ロジック（Go ライブラリ・ユニットテスト対象・→ ADR-0006, ADR-0007）
├── templates/             # nix flake init -t 用（standalone / project・nput init が叩く・→ ADR-0006, ADR-0007）
└── modules/
    ├── common.nix         # options 定義のみ（全モジュールが import）
    ├── home-manager.nix   # home.activation から nput エンジンを起動（root = homeRoot を pin）
    ├── nixos.nix          # （将来拡張）system.activationScripts から nput エンジンを起動
    └── nix-darwin.nix     # （将来拡張）system.activationScripts から nput エンジンを起動
```

今回の実装スコープは standalone CLI + project mode をコアとし、home mode（`homeRoot`）も対象とする。NixOS / nix-darwin・system mode（`systemRoot`）は将来拡張（→ ADR-0004, ADR-0007）。
配置ロジックは Go エンジン（`internal/`・`cmd/nput` が import）が単一の源として所有し、`lib/` はデータ（`manifest.json`）生成に徹する（→ ADR-0006, ADR-0007）。

---

## レイヤー構成

```
nput CLI (packages.nput)      ← 一次 UX。entrypoint 発見 + nix build/eval + エンジン駆動（→ ADR-0007）
        │ import
配置エンジン (internal/ Go ライブラリ)  ← 配置・stale 除去・profile swap の単一の源（manifest.json in・→ ADR-0006）
        ↑ manifest.json を渡して起動
lib/ (mkManifest 他)          ← nixpkgs.lib のみ依存（純データ生成。manifest.json + symlink farm）
        ↑ 起動配線
modules/common.nix            ← options 型定義（nixpkgs.lib のみ依存）
        ↑
┌───────┼──────────────────┐
HM   NixOS  darwin  devShell  standalone(CLI)
（root と activation hook を供給して nput エンジンを起動する薄い配線のみ）
```

上位層が下位層にのみ依存し、逆方向の依存は持たない。
**配置の振る舞いは全層で配置エンジン（`internal/` の Go ライブラリ）が単一の源**であり、各層はネイティブ機構へ翻訳しない（→ ADR-0003, ADR-0006）。
`lib/` は配置ロジックを持たず、エンジンの入力データ（`manifest.json`）を生成するだけ。`nput` CLI はエンジンを import し、entrypoint 発見と nix オーケストレーションを足す（→ ADR-0007）。

---

## flake.nix outputs 設計

```nix
outputs = { ... }: {
  # nput CLI（一次 UX・配置エンジンを import・→ ADR-0007）
  packages.<system>.nput = ...;   # buildGoModule（cmd/nput + internal）

  # 環境セットアップ（nput init = nix flake init -t のラッパー・nput はファイルを生成しない・→ ADR-0006, ADR-0007）
  templates.standalone = { path = ./templates/standalone; description = "..."; };
  templates.project    = { path = ./templates/project;    description = "..."; };  # devShell 配線入り

  # モジュール統合
  homeManagerModules.default = ./modules/home-manager.nix;
  # nixosModules / darwinModules は将来拡張（ADR-0004）

  # 関数呼び出し（モジュールシステム不使用）
  lib = import ./lib;
  # lib.mkManifest          { entries, root }   → derivation（manifest.json + symlink farm・純データ。root は必須）
  # lib.mkOutOfStoreSymlink "/abs/path"         → marker（src に渡す）
  # lib.projectRoot                             → marker（root に渡す: project mode / git toplevel）
  # lib.homeRoot                                → marker（root に渡す: home mode / $HOME）
  # lib.systemRoot                              → marker（root に渡す: system mode / / ・将来）
  # lib.listFilesInSrc     { src, subpath? }       → attrset（builtins.readDir 互換）
};
```

ユーザーの entrypoint 側は `nput.<name>` に named manifest を公開する（→ ADR-0007）。

```nix
# ユーザーの flake.nix
outputs.nput.<system>.<name> = nput.lib.mkManifest { root = ...; entries = { ... }; };
# ユーザーの default.nix / shell.nix
{ nput.<name> = nput.lib.mkManifest { root = ...; entries = { ... }; }; }
```

---

## コアロジック設計（lib データ生成 + Go エンジン）

### entries スキーマ

各エントリは「どのリポジトリのどのパスを、どこへどのように置くか」という配置単位を表す。
`entries` は **属性キー = target の attrset**で、**属性キーが識別子**になる（home-manager `home.file` 同型・→ ADR-0014）。エントリは互いに独立。

| フィールド | 型 | デフォルト | 必須 | 説明 |
|---|---|---|---|---|
| （属性キー）| string | — | ✓ | root 相対の target パス。識別子（= `target` の既定値）|
| `src` | path \| set \| marker | — | ✓ | 配置元。デフォルトは store link。out-of-store はマーカー（下記）|
| `subpath` | string | `"."` | — | リポジトリ内のパス（ファイル・ディレクトリ両対応）|
| `target` | string | 属性キー | — | root（`mkManifest` の `root` で明示選択）からの相対パス。省略時は属性キー |
| `mode` | enum | `"symlink"` | — | `"symlink"` または `"copy"` |

`src` の型による挙動の違い（→ ADR-0001）:

| `src` の値 | symlink の指す先 | 用途 |
|---|---|---|
| `path`（`inputs.foo` 等）| Nix ストア（不変）| flake inputs / `builtins.path` 等 |
| `set`（`fetchFromGitHub` 等）| Nix ストア（不変）| derivation |
| `mkOutOfStoreSymlink "/abs/path"`（marker）| ローカル FS（ライブ）| 開発中の手元 dotfiles |

`string` を直接渡して out-of-store にする暗黙分岐は廃止した。out-of-store は明示関数で opt-in する。

### subpath の判別ロジック（Go エンジン・ネイティブ FS・→ ADR-0006）

Go エンジンが実行時にパスの種別を判定し、適切な処理を**ネイティブ FS 操作**で選択する（`ln` / `rsync` は使わない）。

```
mode = symlink, store/out-of-store → os.Symlink（ファイル・ディレクトリ問わず共通処理）
mode = copy, subpath がディレクトリ → place-once: target 不在時のみネイティブ再帰コピー（mode 保存）
mode = copy, subpath がファイル     → place-once: target 不在時のみネイティブコピー
```

### 世代管理と state（→ ADR-0002）

- 純粋関数 `lib.mkManifest` が **link farm derivation**（`manifest.json` + ストア内の symlink ツリー）を生成する（→ ADR-0006）。
  「配置したもの」のマニフェストは `manifest.json` として link farm の一部に **store 内に**埋め込む（store 外の可変 JSON は持たない）。
- **配置エンジン**（`internal/` の Go ライブラリ。`nput` CLI が manifest を渡して起動）が実行時に:
  1. **前世代の store マニフェスト**と新世代を diff し、消えた entry の **symlink を除去**する（stale 除去）。
     - 前世代は **全モード共通で nput 自身の profile の前世代**から読む（standalone も module も同一。ホストの oldGenPath には依存しない）。
     - 削除は保守的：前世代マニフェストが「nput が配置した」と記録し、かつ現状もその記録通りを指す symlink のみ。
       通常ファイル・nput 非管理の link には触れない。copy target は除去しない（orphan は警告）。初回は何も消さない。
  2. symlink / out-of-store / place-once copy をネイティブ FS 操作で配置する（新規・張替を先に、stale 除去を最後に）。
  3. 全成功後に `nix-env --profile <profileDir> --set <link-farm-drv>` で nput の nix profile を更新する（コミット点・全モード）。
     途中失敗は 3 に到達せず前世代を保つ（部分失敗のコミット最後・→ ADR-0006）。並行実行は**解決後 `profileDir` 単位**の flock で直列化する（明示 apply は blocking wait / shellHook は try-lock skip・衝突時は後勝ち・→ ADR-0013）。

| 機構 | 役割 | 適用層 |
|---|---|---|
| 世代由来の store マニフェスト | stale 除去のための前回状態（不変・GC-root 済み）| 全層共通 |
| nput の nix profile | 前世代の保持・世代番号・GC root | 全モード（standalone はユーザー向け rollback、module は内部機構）|

- 配置・cleanup 機構は home-manager の `linkGeneration`/`cleanup` を参考に再実装する（`home.file` 自体は再利用しない）。
- **nput は全モードで自前 profile を持つ**（→ ADR-0002）。module 時もホスト世代に依存せず自 profile を保持し、
  前世代マニフェストの出所を統一する（HM が NixOS submodule で自前 profile を持つのと同じ）。
  module 時の profile は内部機構（stale 追跡）に留め、ユーザー向け rollback は host に一本化する。
  host rollback は旧 config を再 activate して nput を再 kick することで自動追従する。
- **GC**: profile 世代は GC root。旧世代を `nix-env --profile <dir> --delete-generations` 等で間引き、`nix-collect-garbage` で
  無参照 store パスを解放する。可変 JSON 方式は GC root を作らず rollback が壊れうるため採らない（→ ADR-0002）。
- **project mode（→ ADR-0005）**: profile を解決済み root でキーし（クローン間衝突回避）、世代はユーザー非公開の内部機構に留める。
  `shellHook` 高頻度実行に備え、新 derivation が前世代と同一なら新世代を積まない（世代スキップ）。home mode は従来通り毎回新世代。

### nput CLI の実行モデル（一次 UX・→ ADR-0007）

`nput` CLI（`packages.nput`）は PATH に常駐し、entrypoint を発見して manifest をビルドし、配置エンジンを駆動する。

```
nput apply <name> [-f <ep>] [--root <p>]
  0. entrypoint 発見（既定: CWD で flake.nix → shell.nix → default.nix。-f で上書き）
  1. nix build <ep>#nput.<system>.<name>（flake）/ nix-build <ep> -A nput.<name>（非 flake）→ link-farm store path
  2. 配置エンジン（import）を駆動: flock → root 解決 → 前世代 diff → 配置 → stale 除去 → nix-env --set
```

- `root` は `manifest.json` に記録された kind（project / home / system / 固定パス）をエンジンが実行時に解決する。マーカーは評価時にパスへ展開しない（→ ADR-0005, ADR-0007）。
- `name` は entrypoint の `nput.<name>` 属性キーが供給する。`name` 省略時は `nput.default` を使う（flake の `default` 慣例。未定義ならエラー）。
- 透明性: `nput --help` 等で内部実行する nix コマンドを開示する（→ ADR-0007）。

CLI はサブコマンド体系（→ ADR-0006, ADR-0007）。任意世代切替・世代 GC は標準の `nix-env` / `nix-collect-garbage` に委譲する。

```bash
nput apply [<name>]     # 配置（name 省略時 nput.default）。--dryrun は副作用ゼロのプラン表示
nput apply --all        # entrypoint の nput.* を全適用
nput rollback <name>    # 前世代へ（home mode 限定）
nput list-generations <name>   # 世代一覧（home mode 限定）
nput gitignore <name>   # .gitignore 向け列挙（stdout のみ）
nput init <template>    # nix flake init -t <nput>#<template> のラッパー
```

`--only`（一部 entry だけ適用）は profile 世代の atomic 性と衝突するため提供しない。
選択的更新は「役割ごとに別 config（`nput.<name>`）に分ける」ことで担保する。

### 再現性スタンス（→ ADR-0007）

- `flake.nix` entrypoint は pure eval（root 解決はエンジン実行時なので eval は pure）。flake.lock で固定。
- `shell.nix` / `default.nix` entrypoint は NIX_PATH 依存の impure eval を許容する best-effort。nput lib を含め nixpkgs を npins / fetchTarball / flake-compat 等で固定するのはユーザー責任。

---

## モジュール統合設計

### 共通オプション（modules/common.nix）

```nix
# modules/common.nix（全モジュール共通）
options.nput = {
  enable  = mkEnableOption "nput";
  entries = mkOption { type = attrsOf (submodule ...); };  # 属性キー = target（→ ADR-0014）
};

# modules/nixos.nix, modules/nix-darwin.nix（将来拡張・各モジュール内で追加定義）
options.nput.user = mkOption { type = str; };
```

`user` オプションは NixOS / nix-darwin のみ必要なため、`modules/common.nix` には含めず各モジュールに分離する。
home-manager と standalone は `$HOME` を直接参照するため不要。モジュールは自分の性質で root を pin する（HM → `homeRoot` / devShell → `projectRoot`）ため、`nput.entries` 利用者は `root` を再指定しない（→ ADR-0007）。

### 各統合層の動作（→ ADR-0003）

すべての層で **配置エンジン**が配置を実行する。各層は root と activation タイミングを供給するだけ。

いずれも nput 自身の profile を使う（→ ADR-0002）。standalone（home mode）は profile をユーザー向け rollback に使い、
module は内部機構に留め rollback は host に一本化する。

| 層 | エンジン起動方法 | root の解決 | nput profile | ユーザー向け rollback |
|---|---|---|---|---|
| **standalone（CLI）** | `nput apply <name>` を明示実行 | マーカー（`homeRoot` / `projectRoot` 等）| あり（home はユーザー向け）| `nput rollback <name>`（home mode 限定）|
| **home-manager** | `home.activation` から起動 | `$HOME`（`homeRoot` を pin）| あり（内部）| host（`home-manager --rollback`）|
| **devShell**（→ ADR-0005）| `shellHook` から `nput apply` | project mode: git toplevel（`--root` 可）| あり（内部・root でキー）| なし（ephemeral 配置）|
| **NixOS**（将来）| `system.activationScripts` から起動 | `config.users.users.<user>.home` | あり（内部）| host（`nixos-rebuild` 世代）|
| **nix-darwin**（将来）| `system.activationScripts` から起動 | `config.users.users.<user>.home` | あり（内部）| host 世代 |

devShell は project mode（root = プロジェクトルート）の主トリガ。配置物は ephemeral でコミット対象外のため rollback は持たず、
profile は解決済み root でキーしてクローン間衝突を避ける。`shellHook` の高頻度実行に備え、変更なしなら新世代を積まない世代スキップを必須とする（→ ADR-0005）。

全モジュール（HM / NixOS / nix-darwin）は **一律「nput エンジンをキックするだけ」のランチャー**であり、
プラットフォームごとのネイティブ機構（`home.file` / `systemd.tmpfiles`）へは翻訳しない。これらは**明示的に採らない代替**である。
配置の振る舞いは全環境で nput エンジン + 世代由来の store マニフェスト（HM 同等のアルゴリズム）に統一され、stale 除去まで nput が所有する。

`systemd.tmpfiles` は OS（NixOS）自身の宣言的ファイル管理ツールであって nput の関心事ではない（→ ADR-0003）。
nput は「OS とは別の一機構」として、どの環境でも同じく振る舞う。

**モジュール対応の位置づけ**: 基本的な利用方法は project mode と standalone CLI を中心に考える（→ ADR-0007）。モジュール対応は、
他のモジュールシステムの switch と**一括で動いてほしいユースケース**を拾うためだけに存在し、各モジュールの内部事情は設計に持ち込まない。

### 実行タイミング

| 層 | 実行タイミング |
|---|---|
| standalone | `nput apply <name>` を明示実行 |
| home-manager | `home-manager switch`（home.activation）|
| devShell | `nix develop` / direnv 入室（shellHook）|
| NixOS（将来）| `nixos-rebuild switch` |
| nix-darwin（将来）| `darwin-rebuild switch` |

---

## 使用パターン

### パターン 1：project mode（中心的な使い方・→ ADR-0005, ADR-0007）

```nix
# flake.nix — repo に入ると .claude/skills を nix store から配置する
outputs.nput.${system}.skills = nput.lib.mkManifest {
  root = nput.lib.projectRoot;   # git toplevel を root に解決（project mode）
  entries = {
    ".claude/skills/nix" = { src = inputs.claude-skills; subpath = "skills/nix"; };
  };
};

devShells.${system}.default = pkgs.mkShell {
  shellHook = "nput apply skills --no-wait";   # direnv / nix develop 入室で配置
};
```

```bash
# .gitignore に入れるべき target を列挙（stdout 出力のみ・書き込みはしない）
nput gitignore skills
```

- root はプロジェクトルート（git toplevel）。`--root` で上書き可。
- 配置物は ephemeral（コミット対象外）。`.gitignore` への登録は `nput gitignore` の出力を見てプロジェクト管理者が一度行う。
- 世代は内部機構のみ（rollback 非公開）。`shellHook` 高頻度実行に備え変更なしなら新世代を積まない。

### パターン 2：standalone CLI（home mode・役割ごとに分離して管理）

```nix
# flake.nix — entrypoint が役割ごとに named manifest を公開（それぞれ別 profile）
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
# それぞれ独立した profile として更新・適用・ロールバックできる
nput apply vim-plugins
nput rollback vim-plugins
nput apply zsh-plugins
```

### パターン 3：home-manager モジュール

```nix
imports = [ inputs.nput.homeManagerModules.default ];

nput = {
  enable = true;   # root は homeRoot を pin（再指定不要）
  entries = {
    # 外部リポジトリ（store link）
    ".claude/skills/nix" = { src = inputs.skills-repo; subpath = "skills/nix"; };
    # テーマを copy（place-once、以後ユーザー管理）
    ".local/share/themes/dark" = { src = inputs.themes; subpath = "dark"; mode = "copy"; };
    # 開発中の手元 dotfiles を out-of-store でライブ反映
    ".config/nvim" = { src = nput.lib.mkOutOfStoreSymlink "/home/me/dotfiles"; subpath = "home/.config/nvim"; };
  };
};
```

HM モジュールは `home.activation` から nput エンジンを起動する。配置は nput 自身が行い、
`home.file` には委譲しない。nput は自前 profile を**内部機構**として持つ（前世代マニフェスト + stale 追跡）が、
ユーザー向け rollback は HM（`home-manager --rollback`）に一本化する。

---

## テスト戦略（→ ADR-0006, ADR-0007）

| 対象 | 手段 | 重点 |
|---|---|---|
| lib（純データ生成）| **nix-unit**（評価テスト）+ **namaka**（スナップショット）| `mkManifest` の不変条件 / `manifest.json` 全体の回帰 |
| 配置エンジン（Go ライブラリ）| Go ユニット + tmpdir 統合テスト（実 FS・偽 source・nix 不使用）| **保守的 stale 除去の安全不変条件**（誤削除防止）を table-driven |
| E2E | 非 NixOS（ubuntu-latest）+ `cachix/install-nix-action` の別ジョブで `nput apply` → FS / profile / rollback をアサート | 「非 NixOS でも nix さえあれば動く」主張の検証（→ ADR-0012）|

NixOS VM テスト（`runNixOSTest`）はモジュール経路を実装する段で追加する。

---

## 設計上の判断

### entries を target キーの attrset にする理由（ADR-0014）

エントリの識別子を別フィールド（`name`）で手動定義させると、命名と一意性管理がユーザー負担になる。home-manager `home.file` 同型に **属性キー = target** とすることで、識別子を考える行為自体が消え、一意性は Nix の attrset キー重複不可で native に担保される。target は配置先として元々一意であるべき値で identity に過不足なく、順序非依存なので index ベース命名の「並び替えで名前が変わる」問題も起きない。`target` フィールドはキーから既定値を取り、キーを論理ラベルにして明示上書きする逃げ道も残す。

### 配置単位を `nput.<name>` = 1 profile とする理由（ADR-0002, ADR-0007）

世代の粒度を配置単位 = 1 profile としたため（→ ADR-0002）。entrypoint が `nput.<name>` に named manifest を公開し、
CLI が選択した `<name>` を profile の一意特定に使う。`name` 省略時は flake 慣例の `nput.default` を解決先にする（→ ADR-0007）。

### root を明示マーカー必須にする理由（ADR-0007）

暗黙の root デフォルトは「ユーザーが配置を明示的に握る」核心思想に反する隠れた選択であり、ephemeral/永続・rollback 有無という重い差を暗黙に選ばせる。
`projectRoot` / `homeRoot` / `systemRoot` の明示マーカー（＋固定パス文字列）で `mkOutOfStoreSymlink` と同じ「マーカー opt-in」パターンに揃える。

### CLI を一次 UX に昇格した理由（ADR-0007）

`mkActivationScript` の per-config ラッパー（`nix run .#x`）を呼ばないと `nput` が使えなかった（#3）。flake.nix 以外（shell.nix / default.nix）からも動かしたい（#4）。
汎用 `nput` CLI を PATH に置き entrypoint を発見させることで、config は依然 Nix 評価で確定しつつ standalone のエルゴノミクスを得る。config は Nix で書くモデルを保つため「設定を生成しない」thesis は不変。

### src をユーザー側で渡す理由

取得手段（npins / flake inputs / fetchFromGitHub 等）を本プロジェクトが抱えると、取得方法の変更が
本プロジェクトの変更を要求する依存が生まれる。`src` を「フェッチ済みのパス」として受け取ることで取得手段の変化から独立する。

### out-of-store を明示関数に降格した理由（ADR-0001）

型ベースの暗黙分岐（`string` → out-of-store）を廃止し store link をデフォルトに統一した。
out-of-store は `mkOutOfStoreSymlink` で明示的に opt-in する。型マジックを排除し、Nix の再現性前提に揃える。

### 世代を nix profile に乗せる理由（ADR-0002）

profile symlink の差し替えだけで atomic な switch / rollback を実現し、GC root にもなる。Nix 標準機構を再利用できる。
純粋関数は derivation を生成するだけで、副作用（profile swap）は activation 実行時に閉じる。

### 配置ロジックをコアが所有する理由（ADR-0003）

振る舞いを単一コアに集約し、テスト可能性とクロスプラットフォームの一貫性を得るため。
ネイティブ機構へ翻訳すると層ごとに挙動が二重化し、nput の「単一コア・ユーザー管理」方針と逆行する。
nput は「OS とは別の一機構」であり、`systemd.tmpfiles` など OS のファイル管理ツールへは翻訳しない（全モジュールは一律ランチャー）。

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
