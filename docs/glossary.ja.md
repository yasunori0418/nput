# 用語集（日本語版）

nput の正準（canonical）英語用語を日本語で対訳したもの。README・コードコメント・コマンド出力で概念に言及するときは、ここで定義した **canonical 用語**（見出しの英語表記）を使い、列挙した別名を避けて、プロジェクト全体で表記をそろえる。正準の綴り自体は英語版 `docs/glossary.md` が一次（authoritative）の参照先。

各エントリは canonical な綴りを 1 つ固定する。定義は意図的に短い。詳細な根拠は `docs/spec.md`（仕様書）と `docs/adr/`（設計判断）を参照する。英語版（一次）は `docs/glossary.md` にある。

## 配置の中心抽象

### placement primitive
nput のコア。`root` 相対の `target` に Nix store のパスを配置する純粋関数。モジュール抽象の裏に隠さず、ユーザーが直接合成して使う。
- **Avoid**: 「placement framework（配置フレームワーク）」「configuration management（設定管理）」（nput は設定を生成しない）。

### engine
配置（ネイティブなファイルシステム操作）と stale 除去の両方を一手に所有する配置コア。`manifest.json` を入力に取り、`nix`（profile）と `git`（toplevel）のみを叩く。**nput CLI** が駆動する Go ライブラリとして実装される。config ごとに bash スクリプトを生成しない。ここでの「ライブラリ」は `internal/` 配下のバイナリ内部の層分離を指し、公開 import 可能な再利用モジュールではない——安定面は `manifest.json` 契約に閉じる。エンジンは stdlib-only。
- **Avoid**: 「per-config generated bash script（config ごとに生成される bash スクリプト）」「per-layer placement logic（層ごとの配置ロジック）」「a single flat implementation fused with the CLI（CLI と一体の平らな単一実装）」「importing the engine as a public Go module（engine を公開 Go モジュールとして import する）」。

### nput CLI
ユーザーが直接触れる一次 UX。`PATH` 上の `packages.nput` バイナリ。**entrypoint** を発見し、内部で `nix build` / `eval` を回して named manifest を取得し、エンジンに配置させる。サブコマンドは `apply [<name>]`・`apply --all`・`rollback`・`list-generations`・`gitignore`・`init`。
- **Avoid**: config ごとの `nix run .#x` ラッパーを一次 UX と説明すること、`apply` を「常に entrypoint を build する」と説明すること（ビルド済み link-farm は `--manifest` で適用できる）。

### entrypoint
nput CLI が読む Nix の config ファイル。`flake.nix` / `shell.nix` / `default.nix` のいずれか。`nput.<name>` に named manifest を公開する。config は依然として Nix で書かれ、`nix build` で評価される。
- **Avoid**: 「nput が CWD から config の内容を発見する」と説明すること——発見するのは entrypoint *ファイル* であり、config は Nix 評価で確定する。

### module
standalone・home-manager・将来の NixOS モジュール・devShell の `shellHook` といった統合層。エンジンを起動する配線（wiring）に徹し、自身ではファイルを配置せず、`home.file` や `systemd.tmpfiles` へ翻訳することもない。
- **Avoid**: 「the module places files（モジュールがファイルを配置する）」「the module translates to native mechanisms（モジュールがネイティブ機構へ翻訳する）」。

## 配置の入出力

### entry / entries
配置定義。`entries` は **target** をキーとする attrset で、各値が entry（`{ src; subpath?; target?; method?; }`）。属性キーが識別子であり、既定の `target` を与える。手動の `name` フィールドは持たない。識別子の一意性は Nix の attrset キーで native に担保される。
- **Avoid**: 「file entry（ファイルエントリ）」（配置元はディレクトリのこともある）、entry に `name` フィールドを持たせること、`entries` を `{ name; … }` のリストと呼ぶこと（旧形）。

### src
**entry** の配置元（どの store パス / リポジトリか）。デフォルトは Nix ストアへの **store link**。out-of-store は明示マーカーで opt-in する。**subpath** とは直交する別概念であり、`src` を `source` の短縮形と読まない。
- **Avoid**: **subpath** と混同すること、`source` と呼ぶこと。

### subpath
**entry** で **src** 内のどのパスを取り出すかを選ぶ相対パス。file / dir 両対応。デフォルト `"."`。リポジトリ全体を選ぶ canonical な方法は **subpath を省略**すること（`subpath = "."` は明示形）。
- **Avoid**: 旧名 `source` / `dir` を使うこと、**src** と混同すること、「全体を表すには専用のトークン / marker が要る」と考えること（省略で表せる）。

### target
**entry** の配置先。**root** からの相対パスで指定する。`entries` の属性キーが既定の `target` であり、entry の identity（stale 除去の diff キーであり、一意性のキー）でもある。

### root
配置の基準パス。公開 API の `root` 引数で**明示的に**選ぶ（**暗黙のデフォルトは持たない**）。型は `string`（絶対パス・評価時に固定）と `marker`（実行時に解決）の union。マーカーは **projectRoot** / **homeRoot** / **systemRoot**。
- **Avoid**: 「`$HOME` is fixed（`$HOME` 固定）」「defaults to home mode（既定は home mode）」「the default when `root` is omitted（`root` 省略時の既定）」と説明すること、マーカーを「パス文字列を返す糖衣（sugar）」と説明すること（実行時解決の種別であって、評価時にパスへ展開しない）。

## 配置モード

### home mode
**root** = `$HOME` の配置モード。`homeRoot` マーカーで明示選択する。standalone も home-manager 等の module もこのモードを使う。世代を毎回コミットし、`--rollback` をユーザーに公開する。
- **Avoid**: 「standalone-only（standalone 専用）」「the default when `root` is omitted（`root` 省略時の既定）」と説明すること。

### project mode
**root** = プロジェクトルートの配置モード。`projectRoot` マーカーで選ぶ。配置物は **ephemeral placement**（コミットしない）であり、世代・rollback はユーザーに公開しない。root は git toplevel に解決される。
- **Avoid**: 「relative to CWD（CWD 相対）」「relative to the config file（設定ファイル相対）」と説明すること。

### system mode
**root** = `/` の配置モード。`systemRoot` マーカーで選ぶ。distro 構想（root = `/`）に使う。

### projectRoot
**project mode** を選ぶ root マーカー。実行時に git toplevel を **root** に解決する（`--root` で上書き可）。`homeRoot` / `systemRoot` と並ぶ root マーカーの一つで、`mkOutOfStoreSymlink` と同じ「マーカーを渡して挙動を opt-in する」パターンに従う。
- **Avoid**: 設定ファイルの場所を指すと解釈すること。

### homeRoot
**home mode** を選ぶ root マーカー。実行時に `$HOME` を **root** に解決する。従来は暗黙だった `$HOME` の既定を明示マーカーへ昇格したもの。

### systemRoot
**system mode**（root = `/`）を選ぶ root マーカー。ADR-0004 が「将来の絶対パス文字列 seam」と呼んでいたものを第一級のマーカーへ昇格したもの。

### ephemeral placement
**project mode** における配置物の性質。クローンごとに再生成される前提で、プロジェクトにコミットされない。ゆえに activation は git 状態に干渉しない。
- **Avoid**: 「vendoring」や「成果物をコミットする配置」と混同すること。

## 配置の種別

### store link
コア・デフォルトの配置。配置先が Nix store パスである symlink。再現性を担保するデフォルト経路。「unification（統一）」とは、ストアをデフォルト / コアにし、out-of-store を明示的な例外へ降格することを意味する。
- **Avoid**: out-of-store symlink と混同すること、「copy（コピー）」と呼ぶこと。

### out-of-store symlink
ローカルの絶対パスへのライブ symlink。`nput.lib.mkOutOfStoreSymlink "/abs/path"` でのみ opt-in する（開発中の dotfiles をライブ編集する用途）。明示的な退避路（escape hatch）であり、第一級機能ではない。
- **Avoid**: デフォルト挙動として扱うこと、`src` の型による暗黙の分岐で生み出すこと。

## 状態管理

### generation
rollback の単位。nput 自身の Nix profile（`nix-env --profile <dir>` 式）に乗せて管理する。コミット（`--set`）・rollback・任意世代への切替・一覧・間引きはすべて `nix-env --profile <dir>` 系で統一され、store GC のみ `nix-collect-garbage` を使う。
- **Avoid**: 「stateless スクリプト」前提で語ること（初期方針からは覆っている）、新 `nix profile` CLI で管理すること（profile-manifest を要求し、`nix-env --set` 製の profile では動かない）。

### store manifest
「nput が何を配置したか」を記録する世代由来のデータ。実体は link-farm derivation 内の `manifest.json`（`schemaVersion` を持つ）で、Nix（`lib.mkManifest`）が生成し Go エンジンが読む——Nix↔Go の契約。エンジンの保守的な stale 除去（記録通りを指す nput 管理 symlink だけを削除し、ユーザーの実ファイルには決して触れない）を支える。

### method
配置の種別を選ぶ entry のフィールド（旧名 `mode` からの改名）。unix の file mode との誤読を避けるために改名した。
- **Avoid**: 旧名 `mode`。

## Flagged ambiguities

- **「symlink」単独では曖昧**。必ず **store link** か **out-of-store symlink** のどちらかに寄せる。デフォルトの store link 自体も symlink として実現されるため、裸の語では種別を識別できない。
- **「unification（統一）」は「removal（廃止）」ではない**。store link の統一は out-of-store を削除しない。デフォルトから降格し、明示的な関数の裏に隔離するだけ。
- **`src` と `subpath` は別概念**。`src` = どの物（store パス / リポジトリ）、`subpath` = その中のどのパス。名前は似ているが直交する概念。旧名 `source`（= 現在の `subpath`）は使わない。
- **「standalone」は配置モードではなく起動形態**。standalone（CLI を直接叩くこと）は配置モード（home / project / system）と直交する。モードは `root` マーカーが決める。
