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

### モジュール抽象がユーザーから配置の制御を奪う

home-manager / NixOS / nix-darwin / system-manager はいずれも **Nix モジュールシステムで配置を宣言し、
裏側の挙動を抽象で隠す**。これは簡易化と引き換えに次の代償を伴う。

- 「何がどこにどう置かれるか」がモジュールの内部実装に隠れ、ユーザーが直接握れない
- 配置ロジックがモジュールシステムと密結合し、純粋関数として切り出してテストすることが難しい
  （home-manager の file モジュールを standalone ライブラリとして抜き出した実装は事実上存在せず、
  多くの人が 20 行の bash や GNU Stow に落ちている）
- プラットフォームごとにネイティブ機構（`home.file` / `systemd.tmpfiles` 等）へ翻訳され、振る舞いが層ごとに二重化する

---

## コンセプトの核心

**「nix store の物を、任意の指定ディレクトリに配置する純粋関数である」**

**「役割を分離し、各役割を独立して管理・更新できるようにする」**

**「設定を生成しない。リポジトリの内容をそのまま置く」**

**「モジュール抽象で隠さず、テスト可能な純粋関数群として、ユーザーが配置を明示的に握る」**

nput の本質は、フレームワークではなく **配置プリミティブ**である。
「nix store のパス（リポジトリ全体・サブディレクトリ・単一ファイル）を、root 相対の任意パスへ symlink または copy で置く」
という単一の責務を、テスト可能な純粋関数として提供する。
`home.file` 相当（root = `$HOME`）はその一適用に過ぎず、root は `projectRoot` / `homeRoot` / `systemRoot` で**明示的に選ぶ**（暗黙デフォルトは持たない・→ ADR-0004 改訂, ADR-0007）。

nput は **project-first** に positioning する。中心的な使い方は「プロジェクト内に組み込み、repo 内の任意パスへ配置する」（project mode）であり、`$HOME` 配置（home mode）・system 配置は**明示マーカーで opt-in する例外**として位置づける（→ ADR-0007）。ユーザーは PATH 常駐の `nput` CLI を叩き、CLI が entrypoint（`flake.nix` / `shell.nix` / `default.nix`）を発見して配置する。

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

```
home-manager モデル:
  [全設定] → switch → 全てが一括で更新される

nput モデル:
  [vim-plugins]    → 任意のタイミングで独立更新
  [zsh-plugins]    → 任意のタイミングで独立更新
  [claude-skills]  → 任意のタイミングで独立更新
```

### store link をデフォルトとする（out-of-store は明示的退避路）

配置先のデフォルトは常に **Nix ストアへの symlink** であり、再現性を担保する（→ ADR-0001）。

手元の dotfiles リポジトリを直接参照してライブ編集したい場合に限り、
明示関数 `nput.lib.mkOutOfStoreSymlink "/abs/path"` で Nix ストアを経由しない out-of-store symlink を選べる。
これは headline 機能ではなく、開発中の dotfiles をライブ編集したいときの**明示的な退避路**である。

| 配置元 | 反映タイミング | 向いている用途 |
|---|---|---|
| store link（デフォルト）| flake 更新 + 適用時 | バージョン固定した外部リポジトリ |
| `mkOutOfStoreSymlink "/abs/path"`（明示）| ファイル編集と同時（ライブ）| 開発中の手元 dotfiles |

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
{ name = "claude-skills"; src = inputs.skills-repo; subpath = "skills/nix"; target = ".claude/skills"; }
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

各エントリは `name` で識別され、配置単位（`nput.<name>` 1 つ = 1 profile）ごとに独立した
profile・世代系列として管理される。役割ごとに config を分けることで、1つの更新が他に波及しない。

```bash
# 役割ごとに独立して更新・適用できる（それぞれ別 profile）
nput apply vim-plugins
nput apply zsh-plugins
nput apply claude-skills
```

`src` を更新（flake input の更新 / npins update 等）した後、
対象 config だけを再適用することで、他のツールへの影響を完全に排除できる。

---

## 世代管理（standalone）

初期方針の「世代管理をしない」を撤回し、**nix profile に乗せた世代管理**を導入する（→ ADR-0002）。

- 純粋関数 `lib.mkManifest` が link farm derivation（`manifest.json` + ストア内の symlink ツリー）を生成し、
  Go エンジン（CLI が import するライブラリ）が `nix-env --profile` で nix profile に登録する（→ ADR-0006, ADR-0007）。
- 世代番号・GC root・ロールバックを Nix 標準機構から得る。profile symlink の差し替えだけで atomic に切り替わる。
- 粒度は `nput.<name>` 単位 = 1 profile。役割ごとに独立したロールバック系列を持つ。

```bash
nput apply <name>            # 新世代を作って適用（name 省略時は nput.default）
nput rollback <name>         # 前世代へ戻す
nput list-generations <name> # 世代一覧
```

「純粋関数」と世代管理（副作用）は次の層で両立する。

```
純粋関数:  lib.mkManifest が link farm derivation（manifest.json + symlink farm）を生成（副作用なし）
実行時:    固定 Go エンジンが配置し profile を swap（副作用はここだけ・→ ADR-0006）
```

nput は **全モードで自前 profile を持つ**（前世代マニフェストの出所を統一するため。HM が NixOS submodule でも
自前 profile を持つのと同じ）。standalone では profile をユーザー向け rollback に使う。home-manager / NixOS / nix-darwin
モジュール時は profile を**内部機構**（stale 追跡）に留め、ユーザー向け rollback は host に一本化する
（host rollback は旧 config の再 activate で nput が自動追従する）。

| 配置種別 | 世代管理 | 備考 |
|---|---|---|
| symlink（store）| あり（profile）| ロールバック可能 |
| out-of-store symlink | あり（リンク先のみ）| 指す先の内容は設計上ライブ。版管理しない |
| copy | **なし（世代外）** | place-once・ユーザー管理の副作用（下記）|

### copy はユーザー管理の副作用

copy モードは「初回マテリアライズしたら以後 nput は触らない」place-once とする。

- target が既に在れば上書きしない。ストア更新の反映は明示的な再適用に委ねる。
- 世代管理の対象外であり、ロールバックされない。
- entry が消えても copy target は削除しない（手編集したデータのクロバーを避ける）。ただし orphan 化は警告で通知する。

---

## プロジェクトに閉じた配置（project mode）

nput の**中心的な配置モード**が **root = プロジェクトルート**の project mode（→ ADR-0005, ADR-0007）。
任意のプロジェクト内に nput を組み込み、repo 内の任意パスへ nix store の物を配置する用途のためのモード。
具体例は「repo 内の `.claude/skills/` をチームで共有」「project-local な tool 設定・hook を nix store から配置」など。

- **root の解決**: `root = nput.lib.projectRoot` で明示選択。実行時に git toplevel を root に解決する（`--root` で上書き可）。
  config ファイル相対は Nix で store path 化して成立せず、CWD 相対は冪等性を壊すため採らない（→ ADR-0005）。
- **主トリガは devShell**: `devShells.<name>` の `shellHook` から nput をキックする。`nix develop` / direnv で
  プロジェクトに入った瞬間に配置される。devShell は HM モジュールと同型の「エンジンを起動する配線」（→ ADR-0003, ADR-0005）。
- **ephemeral 配置**: project mode の配置物は per-clone で再生成される前提で、**プロジェクトにコミットされない**。
  ゆえに activation は `.gitignore` に触れず git 状態に干渉しない。`.gitignore` に入れるべき target は専用コマンド
  `nput gitignore`（stdout 出力のみ）で列挙し、プロジェクト管理者が一度登録する。
- **世代は内部機構のみ**: profile は解決済み root でキーしてクローン間衝突を避け、stale 除去と世代スキップ判定に使う。
  `--rollback` / `--list-generations` は公開しない（ephemeral な配置で rollback の意味が薄いため）。

```nix
# entrypoint(flake.nix)が manifest を公開し、devShell で nput apply する
nput.${system}.skills = nput.lib.mkManifest {
  root = nput.lib.projectRoot;
  entries = [
    { name = "nix-skills"; src = inputs.claude-skills; subpath = "skills/nix"; target = ".claude/skills/nix"; }
  ];
};
devShells.${system}.default = pkgs.mkShell {
  shellHook = "nput apply skills";
};
```

---

## north-star: 配置プリミティブから組むミニマル distro

nput の長期的な狙いは、**nixpkgs のパッケージ群（＝ストアパス）を活かしつつ、配置だけをユーザーに操作させ、
Arch / Gentoo 的なミニマル Linux ディストリビューションの基盤を作る**ことである（→ ADR-0004）。

- Linux ディストリビューションの本質はパッケージマネージャであり、Arch / Gentoo はユーザーのコマンド操作で
  ディレクトリ作成・カーネル・必要パッケージのインストールを行う。
- NixOS は同じことを巨大なモジュールシステムで行うが、その代償として nixpkgs への PR / リリースサイクルに縛られる。
- nixpkgs から取得できるパッケージ群を活かしつつ「配置」だけをユーザーに握らせれば、
  モジュール生態系から独立したシンプルな distro が組めそう、という見立てである。

このため、コアの中心抽象は root を `$HOME` に固定せず一般化する。project 配置（`projectRoot`）も
`home.file` 相当（`homeRoot`）も将来の system 配置（`systemRoot` = `/`）も、同じ関数に**明示マーカー**を渡して到達できるように設計する（→ ADR-0007）。

**スコープの線引き（honest な但し書き）**

- 今回の実装スコープは standalone CLI + project mode をコアとし、home mode（`homeRoot`）も対象。system 配置（`systemRoot` = `/`）は将来拡張（→ ADR-0007）。
- 「関数ベースのパッケージ導入・PATH 追加」の具体機構は未定義であり、今回の仕様には含めない。
- ブート / init / FS / パーティションの層は本構想でも空白であり、nput が担う領域ではない。

---

## 設計の哲学

### 取得と配置の分離

```
取得：Nix の評価フェーズ（src = ストアパス）
配置：Nix 管理下の実行フェーズ（symlink / copy）
```

取得手段（npins / flake inputs / fetchFromGitHub 等）をツール側が抱えることをやめ、
「フェッチ済みのストアパス（`src`）を受け取る」設計にすることで、取得方法の変化から独立する。

### 配置ロジックはコアが所有し、モジュールは配線に徹する

配置の実体は全層で **nput 自身の固定 Go エンジン（ネイティブ FS 操作）**が実行する（→ ADR-0003, ADR-0006）。
`home.file` / `systemd.tmpfiles` などプラットフォームのネイティブ機構には委譲しない。
モジュール（HM / NixOS / nix-darwin）は root と activation タイミングを供給して nput エンジンを起動するだけの薄い配線である。

```
nput コア（配置エンジン）= 振る舞いの単一の源
        ↑ 起動するだけ
モジュール層 = root と activation hook を供給する薄い配線
```

これによりネイティブ統合の恩恵（プラットフォーム標準の追跡・GC）は捨てるが、
振る舞いが単一コアに集約され、テスト可能性とクロスプラットフォームの一貫性を得る。

### home-manager に依存しない

home-manager はユーザー環境管理の強力なツールだが、すべての環境で使われるわけではない。
NixOS サーバー・最小構成の環境でも同じ設定定義で動くことを優先する。

### 統合は「オプション」

モジュールとして使えることは便利だが、コアロジックはモジュールシステムに依存しない関数として
実装する。統合層（HM / NixOS / nix-darwin）はコアの薄いラッパーに過ぎない。

```
lib（コア）は何にも依存しない（nixpkgs のみ）
モジュール層は nput エンジンの起動配線だけを持つ
```

### 冪等性

同じ設定を何度実行しても同じ結果になる。

- symlink：前世代の store マニフェストとの diff で、消えた entry の symlink を保守的に除去する（stale 除去、→ ADR-0002）。
  既存 symlink を置き換える。通常ファイル・ディレクトリがある場合はエラー停止
- copy：place-once。target が在れば触らない
- 世代（standalone）：適用のたびに新世代を作り、profile で atomic に切り替える

### 粒度の柔軟性

リポジトリ全体・サブディレクトリ・単一ファイルを同一インターフェースで扱う。
実行時にファイル・ディレクトリを判別するため、呼び出し側は型を意識しなくてよい。

---

## 既存ツールとの比較

比較軸は「機能の有無」ではなく **「モジュール抽象で隠すか、純粋関数でユーザーに握らせるか」**である。

| ツール | 役割 | アプローチ | nput との違い |
|---|---|---|---|
| npins / niv | ソースのバージョン固定 | — | 配置は行わない（nput と直交・併用可）|
| home-manager `home.file` | ファイル配置 + 世代 | モジュール（生成・宣言）| HM 必須。全体管理。file モジュールを standalone 切り出し不能 |
| `mkOutOfStoreSymlink`（HM）| out-of-store symlink | モジュール内ヘルパ | HM 文脈限定。nput は同等を非依存の明示関数で提供 |
| nixpkgs `linkFarm` / `symlinkJoin` | store 内 symlink ツリー生成 | 純粋関数 | 出力が**ストア内に閉じる**。store 外の任意パスへは置かない（nput は内部で利用可）|
| `nix profile` | 世代管理機構 | — | 配置先が `~/.nix-profile` 固定。任意パス配置はしない（nput が乗る対象）|
| `systemd.tmpfiles`（`L`）| 任意パスへの宣言的 symlink | モジュール（NixOS）| 低レベル・NixOS 専用・copy/世代/取得抽象なし |
| numtide/system-manager | 非NixOS の `/etc` + systemd + パッケージ | モジュール（`lib.evalModules`）| **ドメインは重なるがアプローチが逆**。任意パス配置・HOME dotfiles・サブディレクトリ取り出しはしない |
| `git clone`（シェル）| クローンと配置 | 命令的 | 再現性・Nix との統合がない |
| **本ツール** | 取得済みソースの独立配置 + 世代 + 明示 out-of-store | **純粋関数・ユーザー管理** | — |

nput とほぼ同一のツールは存在しない。構成要素（symlink farm / nix profile / out-of-store / 任意パス symlink）は
すべて既存だが、それらを「取得手段非依存 + 生成しない + エントリ個別適用 + HM 非依存の純粋関数コア +
クロスプラットフォーム共通スキーマ + 任意パス配置 × 世代管理」として束ねたものは無い。

特に **system-manager** とは、パッケージ / systemd / `/etc` というドメインこそ重なるが、
「モジュールで隠す」か「純粋関数でユーザーが握る」かというアプローチが思想レベルで異なるため競合しない。
distro 基盤を本気で狙う場合も、システム配置・サービス・パッケージ層は system-manager と併用 / 委譲し、
nput は「任意パスへの粒度自由な配置プリミティブ」に徹するのが妥当な切り分けである。

---

## 設計の変遷（会話の流れ）

| フェーズ | 検討内容 | 採用した方向 |
|---|---|---|
| 起点 | `fetchFromGitHub` + `lock.json` + シェルスクリプト | ロック管理をシェルで実装 |
| ロック管理 | シェルスクリプト vs Nix 関数 | 副作用が必要なため `npins` 等を使う |
| ソース管理 | npins の採用 | npins は `sources.json` でロック管理し attrset を返す。シェルより宣言的 |
| 配置手段 | home-manager 依存可否 | コアを純粋関数として切り出し、HM 非依存と HM 統合を両立 |
| src 設計 | npins を内包するか | `src` をストアパスとして受け取り取得手段を問わない設計に |
| 役割分離 | 全体管理 vs 役割ごとの独立管理 | エントリに `name` を持たせ、個別更新・個別適用できる設計に |
| out-of-store（ADR-0001）| 型ベース暗黙分岐 vs 明示関数 | store link をデフォルトに統一し、out-of-store は明示関数 `mkOutOfStoreSymlink` に降格 |
| 世代管理（ADR-0002）| 世代を取らない vs 取る | nix profile に乗せた standalone 世代管理を追加。copy は世代外 |
| 層モデル（ADR-0003）| ネイティブ翻訳 vs エンジン所有 | 配置ロジックは全層 nput エンジンが所有、モジュールは配線に徹する |
| 抽象（ADR-0004）| `$HOME` 固定 vs root 一般化 | root を一般化し配置プリミティブに。distro は純粋関数を合成して組む north-star |
| project mode（ADR-0005）| root=`$HOME` 固定 vs プロジェクト相対 | root を公開引数へ昇格し git toplevel 相対の project mode を追加。配置物は ephemeral・主トリガは devShell |
| エンジン実装（ADR-0006）| config ごとの生成 bash vs 固定バイナリ | 配置ロジックを固定 Go エンジンに集約。lib はデータ生成に徹し、契約は manifest.json。CLI はサブコマンド体系 |
| 露出 / root（ADR-0007）| per-config ラッパー vs 汎用 CLI、root 暗黙 vs 明示 | 汎用 `nput` CLI を一次 UX に昇格（entrypoint 発見・`nput.<name>`・`nput init`）。`mkActivationScript` 廃止・`mkManifest` 存続。root は明示必須（projectRoot/homeRoot/systemRoot）。positioning を project-first に |

---

## 想定ユースケース

**project mode（中心的な使い方・`root = projectRoot`）**

- プロジェクト repo 内に `.claude/skills` 等を nix store から配置し、devShell キックでチーム共有する
- project-local な tool 設定・hook をバージョン固定で repo 内に配置する（ephemeral・コミット対象外）
- 社内共有の設定リポジトリから特定ディレクトリだけを取り出してプロジェクトに配置する

**home mode（`root = homeRoot` で明示）**

- vim/neovim プラグインをバージョン固定で `~/.local/share/nvim` 以下に配置
- Claude Code や他のエージェントのスキルリポジトリから特定ディレクトリだけを `~/.claude/skills` に配置
- zsh/bash プラグインをプラグインごとに独立管理し、任意のタイミングで個別更新・ロールバック
- カラーテーマリポジトリから特定テーマだけを `~/.local/share/themes` に copy（place-once、以後ユーザー管理）
- 複数マシン（Linux / macOS）で同一のリポジトリ配置設定を共有する
- 開発中の手元 dotfiles を `mkOutOfStoreSymlink` でライブ反映しながら編集する
