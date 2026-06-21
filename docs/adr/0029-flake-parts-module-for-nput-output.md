# ADR-0029: nput output を flake-parts module 化し flakeModules.default を公開する

- ステータス: 採用
- 日付: 2026-06-21
- 関連: ADR-0007（`nput.<name>` アドレッシング・専用 namespace）, ADR-0015（`nix flake check` の unknown output warning）, ADR-0006（lib は nixpkgs.lib のみ依存）, ADR-0023（passthru.rootKind）
- 起点 Issue: #26

## 背景

`flake.nix` の dogfood 用 `nput.<system>.<name>` output は、top-level の `flake` ブロックで `inputs.nixpkgs.lib.genAttrs systems (system: ...)` を使い、`inputs.nixpkgs.legacyPackages.${system}` を直書きして `mkManifest` の `pkgs` 引数に渡していた。

`nput` は標準 flake output でも flake-parts の flakeModules が型登録した output でもないため、`perSystem` ブロックに置いても top-level へ transpose されない。結果として:

- `pkgs` を perSystem（`packages.nput` 側）と二重に解決していた（`legacyPackages.${system}` 直書き）。
- dogfood の config 定義が CLI パッケージ（`packages.nput`）と別の層（top-level `flake`）に分散していた。

flake-parts には `mkTransposedPerSystemModule { name, option, file }` という、`perSystem.<name>` の値をそのまま `flake.<name>.<system>` へ転置する公式ヘルパーがある（`mkPerSystemOption` + `transposition.<name>` のラッパー）。これを使えば Issue が想定する「自前 module で `nput` を transpose する」を最小コードで実現できる。

## 決定

### 1. `nput` を flake-parts module として実装し、consumer 向けに公開する

- `modules/flake-parts.nix` に `flake-parts.lib.mkTransposedPerSystemModule { name = "nput"; option = ...; file = ...; }` で module を定義する。
- `flake.flakeModules.default = ./modules/flake-parts.nix;` で公開し、flake-parts を使う consumer も `imports = [ inputs.nput.flakeModules.default ]` してから `perSystem.nput.<name> = ...` を書けるようにする。dogfood 内部専用には留めない。

### 2. option は `mkManifest` の結果（derivation）を格納する

- option の型は `lazyAttrsOf package`。consumer は `perSystem = { pkgs, ... }: { nput.<name> = nput.lib.mkManifest { inherit pkgs; root = ...; entries = { ... }; }; }` と書く。
- 転置先 `flake.nput.<system>.<name>` は CLI が `nix eval … .rootKind` → `nix build .#nput.<system>.<name>` で叩く対象なので **build 可能な derivation でなければならない**。`mkTransposedPerSystemModule` は option 値をそのまま転置するため、derivation を格納する形が公式ヘルパーとそのまま噛み合う。
- これにより `mkManifest` を**唯一の公開 API**に保つ（option に `{ root, entries }` を持たせて module が `mkManifest` を内部呼びする案は採らない → 棄却案参照）。`pkgs` は perSystem 由来になり、二重解決が消える。

### 3. `mkManifest` の `pkgs` 引数は不変

- `mkManifest { pkgs, root, entries }` のシグネチャは変えない。module 化は「dogfood output をどこで定義するか」と「flakeModule を公開するか」の話で、lib API の変更を伴わない。

### 4. 本 repo も自身の flakeModule で dogfood する（self-dogfood）

- 本 repo の `flake.nix` も `imports` に `./modules/flake-parts.nix` を加え、`perSystem.nput.default = nputLib.mkManifest { inherit pkgs; ... }` で dogfood example を定義する。top-level `flake.nput = genAttrs systems (...)` の直書きは廃止する。
- **循環参照を避けるため、公開（`flakeModules.default`）も import も同一パス `./modules/flake-parts.nix` を参照する**。`imports = [ self.flakeModules.default ]` 経由にすると self outputs 評価の循環を招くため採らない。

### 5. module は純粋な transposition のみ。lib は注入しない

- module は `mkTransposedPerSystemModule` による transposition だけを行い、`mkManifest` / マーカー群を perSystem 引数として注入しない。consumer は `inputs.nput.lib`（module を import している同じ input）を直接参照する。

### 6. flake-parts 利用者には module を canonical とする

- flake-parts を使う repo では `perSystem.nput.<name>` 経由を推奨する（pkgs 一貫・将来の overlay / config 追従）。
- 直書き（`outputs.nput.<system>.<name> = mkManifest { ... }`）は **plain flake（非 flake-parts）と `shell.nix` / `default.nix` 経路の canonical** として存続する。CLI のアドレッシング（`nix build .#nput.<system>.<name>`）は両形で同一。
- テンプレート（`templates/standalone` / `templates/project`）は plain flake のまま変更しない（flake-parts は starter には過剰・ADR-0018）。module は「既に flake-parts を使っている repo」向けの導線。

## 根拠

- **公式ヘルパーで最小実装**: `mkTransposedPerSystemModule` は `packages` / `legacyPackages` 等の標準 output と同じ transposition 機構をそのまま流用でき、自前で `mkPerSystemOption` + `transposition` を手書きするより堅牢。
- **derivation 格納が制約から導かれる**: 転置先が build 可能な derivation でなければ `nput apply` が成立しない。option を derivation 型にすれば公式ヘルパーがそのまま使え、`mkManifest` 直呼び（spec の現行アドレッシング）と API 面が 1 つに揃う。
- **pkgs 一貫**: perSystem の `pkgs` を `mkManifest` に渡すことで、`packages.nput` と同じ pkgs（将来の overlay / config 含む）を使う。Issue の主目的を満たす。

## 影響

- `modules/flake-parts.nix` を追加。`flake.nix` は dogfood の `flake.nput` 直書きを廃し、`imports` + `perSystem.nput.default` へ移し、`flake.flakeModules.default` を公開する。
- `docs/design.md`: プロジェクト構成に `modules/flake-parts.nix`、flake.nix outputs 設計に `flakeModules.default`、使用パターンに flake-parts consumer 例を追記。
- `docs/spec.md`: アドレッシングに flake-parts 経路（`perSystem.nput.<name>` → `flake.nput.<system>.<name>` へ transpose・直書きと同一出力）を追記。「既存プロジェクトへの組み込み」に flake-parts 版を追記。`nix flake check` の節に「module 化しても warning は残る」を補足。
- **`nix flake check` の `warning: unknown flake output 'nput'` は本 ADR では解消しない**。nix 本体の flake check は known-output を hardcode しており、transposition で定義しても `nput` は非標準のまま warning が出る（exit 0・無害・ADR-0015）。本 Issue が解消するのは「pkgs 二重解決」と「dogfood 定義の perSystem 集約」の 2 点のみ。warning 除去は upstream flake-schemas (PR #8892) 待ちの別件。

## 棄却した代替案

- **option に `{ root, entries }` submodule を格納し module が `mkManifest` を内部呼びする（pkgs 完全隠蔽）**: consumer は `perSystem.nput.<name> = { root = ...; entries = { ... }; }` だけで済み HM の `nput.entries` と同じエルゴノミクスになるが、転置先を derivation にするため `mkTransposedPerSystemModule` が使えず自前 transposition + submodule→derivation 変換が要る。さらに `mkManifest` 直呼び（spec の現行形）と 2 つの API 面が生まれる。最小実装・単一 API を優先して却下。
- **dogfood 内部専用に留め flakeModules を公開しない**: consumer の pkgs 二重解決は解決されず、flake-parts 利用者の導線も増えない。公開する価値の方が大きい。
- **`self.flakeModules.default` 経由で self-import する**: self outputs 評価の循環リスク。公開・import とも同一パス参照で回避する。
- **module で `mkManifest` / マーカーを perSystem 引数に注入する**: magic arg と module 複雑化を招く。consumer は同じ input の `nput.lib` を直接参照すれば足りる。
- **templates を flake-parts ベースに変える**: starter には過剰（ADR-0018）。plain flake の直書きを維持する。
