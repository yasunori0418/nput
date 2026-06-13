# ADR-0008: entries の `source` を `subpath` に改名し、リポジトリ全体は subpath 省略で表す

- ステータス: 採用
- 日付: 2026-06-13
- 関連: ADR-0001（marker パターンの起点）, ADR-0004 / ADR-0005 / ADR-0007（root マーカー）, ADR-0006（`manifest.json` 契約）, `docs/spec.md`, `docs/design.md`, `CONTEXT.md`
- 起点: 「リポジトリ全体を示す文字列として `"."` 以外の表現は無いか」という問いの設計検討
- 改訂注記: 本 ADR が `listFilesInRepo` と呼ぶ関数は ADR-0009 で `listFilesInSrc` に改名した（本文は当時の名のまま歴史的記録として残す）

## 背景

entries スキーマには 2 つの「source 系」フィールドがある。

- `src`: 配置元（どの store パス / リポジトリか）。`path | set | marker`。Nix では derivation の `src` 属性そのもので idiomatic。
- `source`: リポジトリ内のどのパスか（subpath）。`string`、デフォルト `"."`（リポジトリ全体）。file / dir 両対応。

`src` は英語的に `source` の短縮形そのもので、読み手は両者が直交概念（どの“物”か / その中の どの“パス”か）であると判別できず、`src` を `source` の別名・typo と誤読する。

さらに lib 内で不整合がある。`listFilesInRepo :: { src, dir? }` は同じ「リポジトリ内パス」概念を `dir` と呼んでおり、entries（`source`）と二重命名されている。しかも `dir` は dir 限定の語なのに entries の `source` は file も取れる。

この命名衝突が、「リポジトリ全体を示す文字列として `"."` 以外の表現は無いか」という問いの背後にある実体だった。`"."` 自体ではなく、`source` という名と `src` との近さが意味を不明瞭にしていた。

実装はまだ無く（`lib/` は未実装、`internal/` 空）、documented API のみ。改名コストが最小のタイミングである。

## 決定

### 1. `source` を `subpath` に改名する（`src` は維持）

- entries の `source` フィールドを **`subpath`** に改名する。`src` は Nix 慣例として維持。
- 「`src` = どの物 / `subpath` = その中のどのパス」と直交が名前で明確化する。`subpath` は file / dir 両対応という語義にも合う。

### 2. `listFilesInRepo` の `dir` も `subpath` に統一する

- `listFilesInRepo :: { src, subpath? }` とし、lib 全体で「リポジトリ内パス」概念を **`subpath` 1 語**に固定する。
- `listFilesInRepo` は `readDir` するため subpath にはディレクトリを要求するが、その制約は名前ではなくドキュメントと実行時エラー（`readDir` が非 dir で明確に失敗する）で担保する。`source` / `dir` の二重命名を再生産しない方を優先する。

### 3. リポジトリ全体の canonical 表現は「`subpath` 省略」とする

- whole-repo の canonical 表現は **`subpath` を省略**すること（サブパス未指定 = 全体）。語として自然に読める。
- `subpath = "."` はルート subpath の明示形として合法に残す（path 直感に合う）。
- 型は `subpath : string`（デフォルト `"."`）の最小形を保つ。新トークン（`null` / `""`）も union も導入しない。

### 4. whole-repo を表す marker（`wholeRepo`）は採用しない

- marker は ADR-0007 が定義したとおり **「実行時解決の種別を運ぶ入れ物」**であって、評価時にパスへ展開できないもの（`$HOME` / git toplevel）に対して使う。
- `subpath` は store パスのサブパス選択で、store パスは評価時に既知 → `subpath` は完全に評価時に確定する。`wholeRepo` marker を作っても評価時に `"."` へ畳まれる**純粋な糖衣**にしかならない。
- これは既存 marker の意味（実行時解決の種別）と真逆で、導入すれば「初の糖衣 marker」となり `root` / `src` で確立した marker パターンを濁す。よって採らない。

### 5. `manifest.json` のキーも `subpath` に揃える

- `manifest.json`（ADR-0006 の Nix↔Go 契約）が記録する各 entry のキーも `source` → `subpath` に揃える。
- Nix 記述・`manifest.json`・Go エンジンで名が一貫し、`manifest.json` を config と照らしてデバッグしやすい。engine 未実装かつ `schemaVersion` 初期のため移行コストはゼロ。

## 根拠

- **命名衝突の解消が本質**: 問いは `"."` の是非ではなく `source`/`src` の近さにあった。`subpath` は直交を名前で表し、`"."` 以外の表現（= 省略）を語義から自然に導く。
- **marker 不採用は marker 思想の保全**: marker を eval-time 糖衣に広げると ADR-0007 の「実行時解決の種別を運ぶ入れ物」という定義が壊れ、`root` / out-of-store の marker と意味がぶれる。`subpath` は素の string で十分。
- **一語統一の学習コスト**: 「リポジトリ内パス」が `subpath` の 1 語に固定され、entries と `listFilesInRepo` をまたいで同じ語で理解できる。
- **今が最安**: 実装ゼロのため、改名・`manifest.json` キー変更とも破壊コストが無い。

## 影響

- **`docs/spec.md`**: entries スキーマ・`#### source` 節・配置動作の prose・エラー仕様・`listFilesInRepo`・`manifest.json` 記述（解決済み配置元 / `source` / `target` / `mode`）を `subpath` に改名。`subpath` を `src` に対し動的決定する string interpolation サンプル（共有 name 補間 / `baseNameOf` の罠注記 / `listFilesInRepo` 応用）を追加。
- **`docs/design.md`**: entries スキーマ表・「source の判別ロジック」見出し・使用パターンの例を `subpath` に改名。
- **`CONTEXT.md`**: `subpath` を新規 glossary 化（_Avoid_: `source` / `dir`）。`src` 項に subpath との対比を追記。marker 語彙に「`subpath` に糖衣 marker を足さない」注記。
- **実装フェーズ**: `lib/types.nix`・`lib.mkManifest`・`listFilesInRepo`・Go エンジンの `manifest.json` パーサは最初から `subpath` で実装する。

## 棄却した代替案

- **`wholeRepo` marker を導入する**: 評価時に `"."` へ畳まれる純粋な糖衣で、marker の定義（実行時解決の種別）を濁す。§決定 4。
- **`null` / `""` を whole-repo の canonical トークンにする**: 省略で自然に表せるのに別トークンを足す。`""` は typo 誘発・パス結合で曖昧。
- **`source` を維持し `src` を `repo` 等に改名する**: `src` は repo 以外の store path / out-of-store local dir も取るため `repo` は狭い。Nix 慣例からも離れる。
- **両方維持し docs で区別する**: 命名衝突という根本を残す。最小変更だが学習コストが下がらない。
- **`listFilesInRepo` の `dir` を維持する**: dir 限定を名前で表せるが、lib 内に `subpath` / `dir` の二重命名が残り、今回潰した `source` / `dir` 分裂を再生産する。
