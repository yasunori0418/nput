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
├── flake.nix              # エントリポイント。outputs を定義（packages.nput / lib / templates / modules）
├── flake.lock             # flake 入力のロック
├── lib/                   # 純データ生成（nixpkgs.lib のみ依存・→ ADR-0006）
│   ├── default.nix        # 公開 API のまとめ（mkManifest / mkActivationScript / mkOutOfStoreSymlink / projectRoot / listFilesInRepo）
│   ├── types.nix          # entries の型定義（各モジュールで共有）
│   ├── manifest.nix       # mkManifest（manifest.json + symlink farm derivation を生成する純粋関数）
│   ├── activation.nix     # mkActivationScript（mkManifest + packages.nput を配線する writeShellApplication ラッパー）
│   └── out-of-store.nix   # mkOutOfStoreSymlink / projectRoot（マーカー構築子）
├── cmd/nput/              # Go エンジンのエントリポイント（packages.nput・→ ADR-0006）
│   └── main.go            # サブコマンド分岐（apply [--dryrun] / rollback / list-generations / gitignore）
├── internal/              # 配置・diff・保守的 stale 除去の純ロジック（Go ユニットテスト対象）
├── templates/             # nix flake init -t 用（standalone / project・→ ADR-0006）
└── modules/
    ├── common.nix         # options 定義のみ（全モジュールが import）
    ├── home-manager.nix   # home.activation から nput エンジンを起動
    ├── nixos.nix          # （将来拡張）system.activationScripts から nput エンジンを起動
    └── nix-darwin.nix     # （将来拡張）system.activationScripts から nput エンジンを起動
```

今回の実装スコープは standalone + home-manager をコアとする。NixOS / nix-darwin は将来拡張（→ ADR-0004）。
配置ロジックは Go エンジン（`packages.nput`）が単一の源として所有し、`lib/` はデータ（`manifest.json`）生成に徹する（→ ADR-0006）。

---

## レイヤー構成

```
packages.nput (Go エンジン)   ← 配置・stale 除去・profile swap の単一の源（→ ADR-0006）
        ↑ manifest.json を渡して起動
lib/ (mkManifest 他)          ← nixpkgs.lib のみ依存（純データ生成。manifest.json + symlink farm）
        ↑ 起動配線
modules/common.nix            ← options 型定義（nixpkgs.lib のみ依存）
        ↑
┌───────┼──────────────────┐
HM   NixOS  darwin  devShell  standalone
（root と activation hook を供給して nput エンジンを起動する薄い配線のみ）
```

上位層が下位層にのみ依存し、逆方向の依存は持たない。
**配置の振る舞いは全層で Go エンジン（`packages.nput`）が単一の源**であり、各層はネイティブ機構へ翻訳しない（→ ADR-0003, ADR-0006）。
`lib/` は配置ロジックを持たず、エンジンの入力データ（`manifest.json`）を生成するだけ。

---

## flake.nix outputs 設計

```nix
outputs = { ... }: {
  # 配置エンジン（Go バイナリ・配置ロジックの単一の源・→ ADR-0006）
  packages.<system>.nput = ...;   # buildGoModule（cmd/nput + internal）

  # 環境セットアップ（nix flake init -t・nput はファイルを生成しない・→ ADR-0006）
  templates.standalone = { path = ./templates/standalone; description = "..."; };
  templates.project    = { path = ./templates/project;    description = "..."; };  # devShell 配線入り

  # モジュール統合
  homeManagerModules.default = ./modules/home-manager.nix;
  # nixosModules / darwinModules は将来拡張（ADR-0004）

  # 関数呼び出し（モジュールシステム不使用）
  lib = import ./lib;
  # lib.mkManifest            { name, entries, root? }       → derivation（manifest.json + symlink farm・純データ）
  # lib.mkActivationScript    { pkgs, name, entries, root? } → derivation（mkManifest + nput を配線するラッパー）
  # lib.mkOutOfStoreSymlink   "/abs/path"                    → marker（src に渡す）
  # lib.projectRoot                                          → marker（root に渡す: project mode）
  # lib.listFilesInRepo       { src, dir? }                  → attrset（builtins.readDir 互換）
};
```

---

## コアロジック設計（lib データ生成 + Go エンジン）

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

### source の判別ロジック（Go エンジン・ネイティブ FS・→ ADR-0006）

Go エンジンが実行時にパスの種別を判定し、適切な処理を**ネイティブ FS 操作**で選択する（`ln` / `rsync` は使わない）。

```
mode = symlink, store/out-of-store → os.Symlink（ファイル・ディレクトリ問わず共通処理）
mode = copy, source がディレクトリ → place-once: target 不在時のみネイティブ再帰コピー（mode 保存）
mode = copy, source がファイル     → place-once: target 不在時のみネイティブコピー
```

### 世代管理と state（→ ADR-0002）

- 純粋関数 `lib.mkManifest` が **link farm derivation**（`manifest.json` + ストア内の symlink ツリー）を生成する（→ ADR-0006）。
  「配置したもの」のマニフェストは `manifest.json` として link farm の一部に **store 内に**埋め込む（store 外の可変 JSON は持たない）。
- **固定 Go エンジン**（`packages.nput`）が実行時に:
  1. **前世代の store マニフェスト**と新世代を diff し、消えた entry の **symlink を除去**する（stale 除去）。
     - 前世代は **全モード共通で nput 自身の profile の前世代**から読む（standalone も module も同一。ホストの oldGenPath には依存しない）。
     - 削除は保守的：前世代マニフェストが「nput が配置した」と記録し、かつ現状もその記録通りを指す symlink のみ。
       通常ファイル・nput 非管理の link には触れない。copy target は除去しない（orphan は警告）。初回は何も消さない。
  2. symlink / out-of-store / place-once copy をネイティブ FS 操作で配置する（新規・張替を先に、stale 除去を最後に）。
  3. 全成功後に `nix-env --profile <profileDir> --set <link-farm-drv>` で nput の nix profile を更新する（コミット点・全モード）。
     途中失敗は 3 に到達せず前世代を保つ（部分失敗のコミット最後・→ ADR-0006）。並行実行は `profileDir` 単位 flock の try-lock で直列化（保持中はスキップ）。

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

### mkActivationScript の生成物（ラッパー・→ ADR-0006）

`lib.mkManifest` で配置データ（`manifest.json` + symlink farm の derivation）を作り、それを固定 Go エンジン（`packages.nput`）に
渡す **`pkgs.writeShellApplication` ラッパー** derivation を返す。配置ロジックはラッパーではなくエンジンが所有する。
実行時にネットワークアクセスは発生しない。

```
mkActivationScript :: { pkgs, name, entries, root ? <$HOME> } -> derivation
# 中身: exec ${packages.nput}/bin/nput --manifest ${mkManifest {...}} "$@"
```

- `name`: profile を一意特定する配置単位名（→ ADR-0002）。`mkActivationScript` 1 呼び出し = 1 profile。
- `root`: 配置先の基準を選ぶ公開引数（→ ADR-0004 改訂, ADR-0005）。省略時は home mode（`$HOME`）。
  `nput.lib.projectRoot` で project mode（実行時に git toplevel 解決、`--root` で上書き可）、絶対パス文字列で固定 root
  （将来の system 配置 root = `/` の seam）。home mode と project mode は世代の扱いが異なる（後述）。

CLI はサブコマンド体系（→ ADR-0006）。任意世代切替・世代 GC は標準の `nix-env` / `nix-collect-garbage` に委譲する。

```bash
nput                    # = apply（省略時デフォルト）
nput apply [--dryrun]   # 配置（--dryrun は副作用ゼロのプラン表示）
nput rollback           # 前世代へ（home mode 限定）
nput list-generations   # 世代一覧（home mode 限定）
nput gitignore          # .gitignore 向け列挙（stdout のみ）
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

いずれも nput 自身の profile を使う（→ ADR-0002）。standalone は profile をユーザー向け rollback に使い、
module は内部機構に留め rollback は host に一本化する。

| 層 | エンジン起動方法 | root の解決 | nput profile | ユーザー向け rollback |
|---|---|---|---|---|
| **standalone** | `nix run` でスクリプトを明示実行 | `$HOME`（実行時）| あり（ユーザー向け）| `nput --rollback` |
| **home-manager** | `home.activation` から起動 | `$HOME`（HM が解決）| あり（内部）| host（`home-manager --rollback`）|
| **devShell**（→ ADR-0005）| `shellHook` から起動 | project mode: git toplevel（`--root` 可）| あり（内部・root でキー）| なし（ephemeral 配置）|
| **NixOS**（将来）| `system.activationScripts` から起動 | `config.users.users.<user>.home` | あり（内部）| host（`nixos-rebuild` 世代）|
| **nix-darwin**（将来）| `system.activationScripts` から起動 | `config.users.users.<user>.home` | あり（内部）| host 世代 |

devShell は project mode（root = プロジェクトルート）の主トリガ。配置物は ephemeral でコミット対象外のため rollback は持たず、
profile は解決済み root でキーしてクローン間衝突を避ける。`shellHook` の高頻度実行に備え、変更なしなら新世代を積まない世代スキップを必須とする（→ ADR-0005）。

全モジュール（HM / NixOS / nix-darwin）は **一律「nput エンジンをキックするだけ」のランチャー**であり、
プラットフォームごとのネイティブ機構（`home.file` / `systemd.tmpfiles`）へは翻訳しない。これらは**明示的に採らない代替**である。
配置の振る舞いは全環境で nput エンジン + 世代由来の store マニフェスト（HM 同等のアルゴリズム）に統一され、stale 除去まで nput が所有する。

`systemd.tmpfiles` は OS（NixOS）自身の宣言的ファイル管理ツールであって nput の関心事ではない（→ ADR-0003）。
nput は「OS とは別の一機構」として、どの環境でも同じく振る舞う。

**モジュール対応の位置づけ**: 基本的な利用方法は standalone を中心に考える。モジュール対応は、
他のモジュールシステムの switch と**一括で動いてほしいユースケース**を拾うためだけに存在し、各モジュールの内部事情は設計に持ち込まない。

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
`home.file` には委譲しない。nput は自前 profile を**内部機構**として持つ（前世代マニフェスト + stale 追跡）が、
ユーザー向け rollback は HM（`home-manager --rollback`）に一本化する。

### パターン 3：project mode（プロジェクトに閉じた配置・→ ADR-0005）

```nix
# flake.nix — repo に入ると .claude/skills を nix store から配置する
devShells.default = pkgs.mkShell {
  shellHook = ''
    ${nput.lib.mkActivationScript {
      inherit pkgs;
      name = "skills";
      root = nput.lib.projectRoot;   # git toplevel を root に解決（project mode）
      entries = [
        { name = "nix-skills"; src = inputs.claude-skills; source = "skills/nix"; target = ".claude/skills/nix"; }
      ];
    }}/bin/nput
  '';
};
```

```bash
# direnv (use flake) / nix develop でシェルに入ると配置される
# .gitignore に入れるべき target を列挙（stdout 出力のみ・書き込みはしない）
nix run .#skills -- gitignore
```

- root はプロジェクトルート（git toplevel）。`--root` で上書き可。
- 配置物は ephemeral（コミット対象外）。`.gitignore` への登録は `nput gitignore` の出力を見てプロジェクト管理者が一度行う。
- 世代は内部機構のみ（`--rollback` 非公開）。`shellHook` 高頻度実行に備え変更なしなら新世代を積まない。

---

## テスト戦略（→ ADR-0006）

| 対象 | 手段 | 重点 |
|---|---|---|
| lib（純データ生成）| **nix-unit**（評価テスト）+ **namaka**（スナップショット）| `mkManifest` の不変条件 / `manifest.json` 全体の回帰 |
| Go エンジン | Go ユニット + tmpdir 統合テスト（実 FS・偽 source・nix 不使用）| **保守的 stale 除去の安全不変条件**（誤削除防止）を table-driven |
| E2E | 非 NixOS + nix のコンテナで `nix run .#x` → FS / profile / rollback をアサート | 「非 NixOS でも nix さえあれば動く」主張の検証 |

NixOS VM テスト（`runNixOSTest`）はモジュール経路を実装する段で追加する。

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
