# ADR-0007: 汎用 nput CLI を一次 UX に昇格し、entrypoint 発見＋root 明示モデルへ移行する

- ステータス: 採用（2026-06-14 追記: project mode の `nput` は devShell 同梱が canonical → ADR-0015）
- 日付: 2026-06-13
- 関連: ADR-0002, ADR-0003, ADR-0004, ADR-0005, ADR-0006, ADR-0015, `docs/concept.md`, `docs/design.md`, `docs/spec.md`
- 改訂対象: ADR-0006「nput の露出と環境セットアップ」節と棄却案（本 ADR が反転）、ADR-0004 / ADR-0005 の root モデル
- 起点 Issue: #2（root デフォルト）, #3（CLI 化）, #4（flake.nix 以外の entrypoint）

> **2026-06-14 追記（ADR-0015）**: 「`nput` を PATH に常駐させる」手段を具体化。**project mode は `templates/project` の
> devShell に pin 版 `nput`（`packages = [ nput.packages.${system}.nput ]`）を同梱するのが canonical**。CLI と `nput.lib`
> （manifest schemaVersion）が同一 flake 入力から来て一致する。グローバル install（`nix profile install`）は standalone
> （home mode）の利便として残す（→ ADR-0015）。

> **2026-06-17 追記（ADR-0026）**: 本 ADR の entrypoint 発見モデルは **entrypoint 駆動**の kick（standalone / devShell が
> `nput.<name>` を公開し `nput apply <name>` で build→配置）を指す。これとは別に、HM 等の**モジュールはビルド済み manifest を
> `nput apply --manifest <link-farm>` で kick する**（モジュール評価時に `mkManifest` でビルドし、entrypoint output を持たない）。
> モジュールの engine kick invocation は ADR-0026 で確定（→ ADR-0026）。

## 背景

ADR-0006 は次の露出モデルを決定していた。

- エンジン `nput` は PATH に常駐せず、`mkActivationScript` が生成する **per-config ラッパー**（`nix run .#x`）経由でのみ起動する。
- グローバル `nput` に「CWD / ファイルから config を発見する」機構は足さない（棄却案）。
- 環境セットアップは `nix flake init -t <nput>#standalone` で賄い、`nput init` スキャフォルドは採らない（棄却案）。

この設計に対し、3 つの Issue で再検討要求が出た。

- **#2**: root のデフォルトが `$HOME`（home mode）固定で、ミニマル／スコープ利用に合わない。
- **#3**: `mkActivationScript` を呼ばないと `nput` コマンドが作れない。エンジンとは別に、ユーザーインターフェイスとしての CLI が欲しい。
- **#4**: 現状は flake.nix しか読まない。standalone を前提にすると `shell.nix` / `default.nix` からも動く必要がある。

これらは個別の小改善ではなく、**nput の positioning と露出モデルそのものの転換**を要求していた。

## 決定

### 1. positioning を project-first に転換する

- nput の中心的な使い方を **「プロジェクト内に組み込み、repo 内の任意パスへ配置する」** に置く。home-manager と `$HOME` で正面競合せず、project-local 配置のニッチを一次ターゲットにする。
- `$HOME` 配置（dotfiles）・system 配置（`/`）は**明示マーカーで opt-in する例外**として位置づける。
- `docs/concept.md` の主用途リスト・north-star の語り口をこの positioning に合わせて改訂する。

### 2. root は暗黙デフォルトを廃止し、明示必須にする（#2 の解決）

- **Issue #2 の「デフォルトを projectRoot にする」は、「暗黙デフォルトを廃し root を明示必須にする」へ書き換える**（タイトルとは表現が反転するが、`$HOME` 固定/暗黙デフォルトをやめる本質は満たす）。
- `root` の型は `string（絶対パス・評価時固定）| marker（実行時解決）` の union。**省略は許さない**（評価時エラー）。
- マーカーは 3 つ：`projectRoot`（git toplevel、`--root` で上書き可）/ `homeRoot`（`$HOME`）/ `systemRoot`（`/`）。
  - マーカーは**「実行時解決の種別を運ぶ入れ物」**であってパス文字列を返す糖衣ではない。`homeRoot` / `projectRoot` は評価時にパスへ展開できない（`$HOME` / git toplevel は実行環境依存・ADR-0005）。kind を `manifest.json` に記録し、エンジンが実行時に解決する。
  - 絶対パス文字列だけが評価時固定。`systemRoot` は ADR-0004 が「将来の絶対パス文字列 seam」としていたものを正式なマーカーへ昇格。
- **明示必須は `mkManifest` / CLI entrypoint の層に課す。モジュール（HM → `homeRoot` / devShell → `projectRoot`）は自分の性質で root を pin する**（ADR-0003 の「配線が root を供給する」役割）。モジュール利用者は root を再指定しない。

### 3. 汎用 nput CLI を一次 UX に昇格する（#3 の解決・ADR-0006 露出モデルの反転）

- **`nput` CLI を PATH に常駐させ、一次 UX とする**（`packages.nput` を `nix profile install` 等で導入）。per-config ラッパー（`nix run .#x`）を主経路とする ADR-0006 を反転する。
- **`mkActivationScript`（per-config ラッパー derivation）は廃止**する。`lib.mkManifest`（純データ生成）が entrypoint の公開面として残る。
- CLI の責務：
  1. entrypoint ファイルを発見（既定: CWD で `flake.nix` → `shell.nix` → `default.nix` の優先順。`-f` / `--file <path>` で明示上書き）。
  2. 内部で `nix build` / `nix eval` を回して named manifest（`manifest.json` + symlink farm derivation）の store path を得る。
  3. エンジンに manifest を渡して配置・stale 除去・profile swap を実行。
- **透明性**: `nput --help` 等で内部実行する nix コマンドを開示し、ユーザーが選択的に手で実行できるようにする。

### 4. アドレッシング = 専用 `nput` 名前空間・named config（#4 の解決）

- entrypoint は **`nput.<name>`** に named manifest を公開する。1 プロジェクトに複数 config（skills / vim-plugins …）を置ける（concept の「役割ごと独立 profile」を維持）。
  - flake.nix: `outputs.nput.<system>.<name> = mkManifest { ... }`。標準 flake output ではないため CLI が現行 `<system>` を差し込む（`nix build .#nput.<system>.<name>`）。`packages` を汚さない。
  - default.nix / shell.nix: トップレベル `{ nput.<name> = mkManifest { ... }; }`。CLI は `nix-build -A nput.<name>`。
- **config 選択**: `nput apply <name>` で named 適用。**name 省略時は `nput.default` を適用**する（flake の `default` 慣例に倣う。`default` 未定義ならエラー）。一括適用は `nput apply --all` を明示フラグで提供。profile は config 単位で atomic（ADR-0002）。

### 5. shell.nix / default.nix は best-effort（再現性はユーザー責任・#4）

- flake entrypoint は pure eval（root 解決はエンジン実行時なので eval は pure のまま）。
- `shell.nix` / `default.nix` は NIX_PATH（channels）依存の **impure eval を許容**する。nput lib の取り込みも含め、**再現性は「nixpkgs を固定する」ユーザー責任**として `docs/spec.md` に明記する（npins / fetchTarball / flake-compat 等を推奨）。

### 6. `nput init` でテンプレートを展開する（#3 後段・ADR-0006 棄却案の反転）

- `nput init <template>` を提供し、**`nix flake init -t <nput>#<template>`（standalone / project）への透明なラッパー**とする。
- ファイルを作るのは nix の templates 機構であり nput 自身は generate しない。`nix flake init` の「既存ファイルを上書きしない」保守性を継承する。これにより「設定を生成しない」thesis を厳密に保つ。

### 7. バイナリ構成 = エンジンはライブラリ、CLI が import する

- 配置エンジンを **Go ライブラリ**（単体テスト＋ tmpdir 統合テスト付き）として実装し、`manifest.json` を入力に取り `nix`(profile) / `git`(toplevel) のみ叩く ADR-0006 の契約をライブラリ API として温存する。
- `cmd/nput` の CLI バイナリがこのエンジンライブラリを import し、entrypoint 発見・nix オーケストレーション（`nix build` / `eval`）・`nput init` を担う。
- 配布は単一バイナリ。CLI 層が増やすサブプロセスは `nix`(build / eval) で、配置エンジン層の契約（manifest.json in）は変えない。

## 根拠

- **ADR-0006 棄却理由の前提が外れている**: ADR-0006 は「CWD から config を発見＝Nix 評価を捨てる」と捉えて棄却した。本 ADR のモデルは config を依然 Nix で書き（flake/shell/default）`nix build` で評価する。発見するのは *entrypoint ファイル* であって config 内容ではない。Nix 評価 thesis は保たれる。
- **`nput init` も thesis を傷つけない**: nput はファイルを書かず `nix flake init -t` を呼ぶだけ。生成は nix の templates が行い、上書きもしない。
- **明示 root が thesis と整合**: 暗黙の root デフォルトは「モジュール抽象で隠さずユーザーが配置を明示的に握る」（concept.md）に反する隠れた選択だった。明示マーカーの方が思想に合う。
- **project-first の動機**: standalone のエルゴノミクス（`nix run .#x` の per-config ラッパーより、PATH 上の `nput apply <name>` が素直）、#4 の非 flake entrypoint 対応、home-manager と正面競合しない positioning。
- **エンジン＝ライブラリ**: 安全クリティカルな保守的 stale 除去をライブラリ単体でテストでき、CLI の nix オーケストレーションと配置ロジックの境界が明確になる（ADR-0006 のテスト可能性 thesis を強化）。

## 影響

- **ADR-0004 改訂**: root を「明示必須・3 マーカー（projectRoot / homeRoot / systemRoot）」へ。`$HOME` は `homeRoot` マーカーへ、system seam は `systemRoot` マーカーへ昇格。
- **ADR-0005 改訂**: 「既定 `$HOME`（home mode）」前提を廃し、暗黙デフォルト撤廃。project mode は 3 マーカーの一つに。
- **ADR-0006 改訂**: 「nput の露出と環境セットアップ」節・実行フロー・棄却案（グローバル CLI / `nput init`）を本 ADR で反転と明記。`mkActivationScript` ラッパーは廃止、`mkManifest` は存続。エンジンはライブラリ化。
- **`docs/spec.md`**: lib API から `mkActivationScript` を削除、CLI（entrypoint 発見・`-f`・`apply <name>` / `--all` / `init` / `gitignore` / `rollback`・`--root`）・アドレッシング（`nput.<name>`）・root 明示必須・shell.nix/default.nix の再現性注記・実行フローを反映。
- **`docs/concept.md`**: positioning を project-first に、主用途リスト・north-star の語りを改訂。
- **`CONTEXT.md`**: `root` / モード / マーカー語彙は更新済み。`engine` の露出・`mkActivationScript` 廃止・CLI / entrypoint / `nput init` の語彙を追記。
- **`flake.nix`**: `templates`（standalone / project）output と `packages.<system>.nput`（CLI バイナリ）の追加は実装フェーズ。

## 棄却した代替案

- **ADR-0006 のまま per-config ラッパーを主経路に保つ**: #3（mkActivationScript 必須）・#4（非 flake entrypoint）・standalone エルゴノミクスを満たせない。
- **root に暗黙デフォルトを残す（project または home）**: ephemeral / 永続・rollback 有無という重い差を暗黙に選ばせる。明示思想に反する。「省略 = project」も恣意的なデフォルトで、エントリモデルからの自然な帰結として説明しにくい。
- **生の絶対パス文字列エスケープハッチを落として 3 マーカーのみにする**: 任意固定 root の seam を失う。`string | marker` union を維持する（現行 spec の型を踏襲）。
- **アドレッシングで `packages.<system>.<name>` を流用**: `nix build .#<name>` の system 自動注入は効くが、manifest が通常パッケージとして `nix flake show` / `nix build` に混ざり誤ビルドしうる。専用 `nput` 名前空間で分離する。
- **CLI とエンジンを別バイナリに分離**: 配布が増える。Go ライブラリ＋単一 CLI バイナリで責務分離とテスト可能性は両立する。
- **CLI とエンジンを単一の平らなバイナリに混ぜる**: `manifest.json` だけ受け取る契約が崩れ、eval テストと配置テストの境界が曖昧になる。ライブラリ境界で分ける。
- **テンプレート展開を別 Issue に分離**: CLI の全体像（init を含むサブコマンド体系）を一度に固める方が一貫する。`nix flake init -t` ラッパーで実装も薄い。
