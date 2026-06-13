# ADR-0010: manifest の型検査を `evalModules` + marker タグ方式で行う

- ステータス: 採用
- 日付: 2026-06-13
- 関連: ADR-0001（out-of-store を明示関数に降格）, ADR-0003（配置ロジックはエンジンが所有・モジュールは配線）, ADR-0006（lib は純データ生成・`manifest.json` 契約）, ADR-0007（root 明示必須・CLI 一次 UX）, ADR-0008（`src` / `subpath` 命名）, `docs/spec.md`, `docs/design.md`, `CONTEXT.md`
- 起点: 「manifest 内にセットする各 attrset（entries / root）の型検査に、NixOS モジュールシステムの `lib.types` / `mkOption` を再利用できるか」という確認

> **2026-06-14 改訂（ADR-0014）**: 本 ADR の以下の決定は ADR-0014 で反転した。検査機構（`mkManifest` 内で `evalModules`）・marker タグ方式・2 段分割・clean enum 契約・root 検査は不変。
> - 「entries を `attrsOf (submodule …)` でモデル化」を棄却 → **採用**（キー = target の attrset に変更）。
> - entry submodule の `name:str(必須)` フィールド → **廃止**（識別子は属性キー = target）。
> - 重複 `name` の `lib.throwIf` チェック → **撤廃**（一意性は Nix attrset が native 担保）。copy+marker / systemRoot の throwIf（→ ADR-0013）は残る。

## 背景

`lib.mkManifest { entries, root } -> derivation` は entries（配置定義のリスト）と root（配置先基準）を入力に取り、`manifest.json` + symlink farm を生成する純粋関数（→ ADR-0006）。この入力に対する型検査をどう実装するかが未決だった。

entries が流れる経路は 2 つある。

- **経路 1（CLI / entrypoint）**: ユーザーが `nput.lib.mkManifest { entries = [...]; root = ...; }` を直接呼ぶ。中心的ユースケース（project mode + standalone CLI・→ ADR-0007）。モジュールシステムを通らない。
- **経路 2（モジュール）**: `modules/common.nix` の `nput.entries` オプション。HM / NixOS の `evalModules` を通る。

`mkOption { type = ...; }` は単独では何も検査せず、型のマージ・デフォルト適用・検査が走るのは `lib.evalModules` を通したときだけ。型をオプションに書くだけだと検査が効くのは経路 2 のみで、コアである経路 1 が無検査で素通りする。

なお `lib.types` / `mkOption` / `lib.evalModules` は **`nixpkgs.lib` のコア**（`lib.types` / `lib.options` / `lib.modules`）であり、NixOS / home-manager に属するものではない。両者は `evalModules` の上に乗る利用者にすぎない。よってモジュールシステムを使っても「`lib/` は nixpkgs.lib のみに依存」（CLAUDE.md・→ ADR-0006）を満たす。

## 決定

### 検査機構: `mkManifest` 内で `evalModules` を回す

両経路で entries 検査を効かせるため、`mkManifest` 自身が `evalModules` を回して entries / root を検査・正規化する。型をオプションに書くだけの「経路 2 限定検査」は採らない。

### 型定義: `lib/types.nix` で共有、marker はタグ方式

- **entry submodule** を `lib/types.nix` に定義し、`mkManifest` の `evalModules` と `modules/common.nix` の `listOf (submodule …)` で共有する。フィールドは `name:str(必須)` / `src:srcType(必須)` / `subpath:str(default ".")` / `target:str(必須)` / `mode:enum ["symlink" "copy"](default "symlink")`。
- **strict（未知キー拒否）**: `types.submodule` のデフォルト挙動を採り、未宣言キー（タイポ・旧名 `source` / `dir`・→ ADR-0008）を評価時に弾く。`freeformType` で緩めない。
- **marker はタグフィールド方式**: marker（`mkOutOfStoreSymlink` / `projectRoot` / `homeRoot` / `systemRoot`）に `_nputMarker` 判別タグを持たせ、custom optionType の `check` で判別する。derivation（`fetchFromGitHub` の戻り）も marker も attrset であり、タグ無しでは区別できないため。
- **`srcType`** = `either storeBacked outOfStoreMarker`。`storeBacked.check` = `isPath ∥ isDerivation ∥ (isAttrs && ? outPath)`（`builtins.path` / derivation / flake input を 1 ブランチに集約）。**素の文字列は拒否**し、ADR-0001（`src` 文字列直渡しによる暗黙 out-of-store 分岐の廃止）を型で強制する。path と set は挙動が同一（ともに store link）なので型レベルで分けない。
- **`rootType`** = `either str rootMarker`。**`mkManifest` 専用**で、モジュールは root を pin する（→ ADR-0003）ため共有しない。

### クロスエントリ制約と必須性

- **重複 `name`**: `listOf` は一意性を検査しないため、`evalModules` 後に `lib.throwIf` で手動チェックする。NixOS 流の `assertions` 配管は lib に持ち込まない（検査がほぼ name 一意性 1 件で過剰なため）。
- **`root` 必須性**: `default` なしのオプションが未設定なら `evalModules` が "option used but not defined" で throw する標準挙動に委ねる。
- **モジュール経路の二重検査**（host `listOf` → `mkManifest` の `evalModules`）は許容する。`mkManifest` を「entries が必ず通る単一の検査・正規化ゲート」に保ち、経路ごとに検査の有無が分岐しないようにする。純粋・冪等なので再評価コストは無視できる。

### 2 段分割: `normalizeManifest` + `mkManifest`

- **`normalizeManifest { entries, root } -> attrset`**: `evalModules` 検査・デフォルト適用 + name 一意性 throw + 内部タグ → clean enum 変換を行う純データ関数。nix-unit / namaka の単体対象。
- **`mkManifest = args: derivation`**: `normalizeManifest` の出力を `manifest.json` に書き symlink farm を組む。

検査・正規化ロジックを derivation の外に出すことで、derivation 等価比較の難しさを回避し、「lib は純データ生成・テスト可能」（→ ADR-0006、design.md 第一目標）を直接担保する。

### `manifest.json` ワイヤ形式: 内部タグを漏らさない

`_nputMarker` は Nix 評価内で完結させ、Go が読む契約（`manifest.json`・→ ADR-0006）には clean enum で写す。`normalizeManifest` が変換層を担う。

```
# 内部（Nix の型判別用）                            →  manifest.json（Go が読む契約）
src  = { _nputMarker="outOfStore"; path="/abs"; }   →  { srcKind = "outOfStore"; src = "/abs"; }
src  = <store-backed>                               →  { srcKind = "store"; src = "/nix/store/…"; }
root = { _nputMarker="root"; kind="project"; }      →  { rootKind = "project"; }
root = { _nputMarker="root"; kind="home"; }         →  { rootKind = "home"; }
root = "/abs/fixed"                                 →  { rootKind = "fixed"; root = "/abs/fixed"; }
```

store-backed の解決値は `toString` / `outPath` で得た store パス文字列を Nix 側（`normalizeManifest`）で確定させ、Go は文字列を受け取るだけにする（`nix eval` の再呼び出し不要・→ ADR-0006）。

## 根拠

- **両経路で検査が効く**: コア（CLI + project mode）が無検査で素通りしない。spec.md のエラー仕様（`root` 省略・重複 `name` 等を「Nix 評価時にエラー」）と整合する。
- **marker タグの二重活用**: 型判別タグの kind は、そのまま `manifest.json` の `srcKind` / `rootKind` の元情報になる（root の kind / out-of-store 種別を manifest に記録する契約・spec.md）。
- **型で契約を強制**: 文字列 src の拒否（ADR-0001）・未知キー拒否（旧名・タイポ）を評価時に機械的に保証でき、「ユーザーが配置を明示的に握る」思想と一致する。
- **テスト可能性**: 2 段分割で検査・正規化が純 attrset として nix-unit / namaka に乗り、エラー系も `expectedError`（type / msg）で回帰テストできる。
- **契約の独立進化**: 内部タグを `manifest.json` に漏らさないことで、Nix 側の型実装（`_nputMarker` の名前 / 構造）を Go 契約と切り離して変更できる。

## 影響

- **`docs/spec.md`**: lib API（`mkManifest`）に検査機構（`evalModules` / `normalizeManifest` 2 段分割）を追記。entries スキーマ仕様に「型検査・strict・デフォルト適用」を明記。エラー仕様に「素の文字列 src」「未知キー」の評価時エラー行を追加し、既存エラーの機構（型 / throwIf / evalModules 標準）を明示。依存関係・テスト戦略を更新。
- **`docs/design.md`**: `lib/types.nix` は「entry submodule + 構成型」を持つ。検査機構の記述は spec.md に集約し、必要なら参照を足す（本 ADR では design.md の構成図は変更しない）。
- **`CONTEXT.md`**: 用語は不変。必要なら `manifest` 定義に「型検査済み・clean enum 契約」の注記を将来足す。
- **実装フェーズ**: `lib/types.nix`（entry submodule / srcType / rootType / marker custom type）、`lib/out-of-store.nix`（marker に `_nputMarker` タグ）、`lib/manifest.nix`（`normalizeManifest` + `mkManifest`）、`modules/common.nix`（共有 submodule を `listOf` で利用）。

## 棄却した代替案

- **型をオプションに書くだけ（経路 2 限定検査）**: コアの CLI / project mode 経路が無検査になり、型定義の意味が薄い。
- **手書き `assert` / `throw` で全検査**: `evalModules` のデフォルト適用・未知キー検出・型エラーメッセージを自前で再実装することになり、コードが増える割に品質が落ちる。
- **NixOS 流 `assertions` オプションを lib に配管**: name 一意性ほぼ 1 件のために NixOS の assertions 評価を lib に持ち込むのは過剰で、「nixpkgs.lib のみ」の軽さを損なう。
- **entries を `attrsOf (submodule …)`（name キー）でモデル化**: 一意性を型で担保できるが、確定済み公開 API（`{ name; … }` のリスト・spec.md / CONTEXT.md）を破壊する。
- **marker を判別タグ無しの素の attrset にする**: derivation（`fetchFromGitHub`）と marker が両方 attrset で区別できず、`srcType` の判別が破綻する。
- **`src` の `path` と `set` を型で分ける**: `manifest.json` に渡る段では両者とも store パス文字列に解決され区別が消えるため、型分割は無意味。
- **モジュール経路で `mkManifest` の検査を省く最適化**: 経路ごとに検査の有無が分岐し、`mkManifest` の単一ゲート性が崩れる。純粋関数の再評価コストは無視できる。
- **`mkManifest` を分割せず derivation だけ返す**: 検査・正規化ロジックが derivation 内に閉じ、nix-unit での不変条件アサートが難しくなる（ADR-0006 のテスト可能性に反する）。
- **内部タグ `_nputMarker` を `manifest.json` にそのまま出す**: Go 契約が Nix 側の型実装詳細に結合し、タグの名前 / 構造を変えると契約が壊れる。
