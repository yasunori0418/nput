# ADR-0009: `listFilesInRepo` を `listFilesInSrc` に改名する

- ステータス: 採用
- 日付: 2026-06-13
- 関連: ADR-0008（lib 命名整理・`src` / `subpath`）, `docs/spec.md`, `docs/design.md`, `CONTEXT.md`
- 起点: 「`listFilesInRepo` は初期コンセプト（git リポジトリを fetch して配置）由来の名で、`pkgs.hello` 等の store パッケージも走査できる今の射程より狭い」という指摘

## 背景

`listFilesInRepo :: { src, subpath? } -> { filename: fileType }` は、`src` 内の `subpath` ディレクトリの内容を `builtins.readDir` 互換の attrset で返すヘルパー。

命名の「Repo」は、初期コンセプト（git リポジトリを Nix の厳密性で fetch して配置する）に由来する。しかし現行の `src` は `path | set`（flake input / `fetchFromGitHub` / `builtins.path` / `pkgs.hello` のような derivation の store パス）まで一般化されており、必ずしも「リポジトリ」ではない。entries の `src` は **配置元（どの store パスか）** という確定語で、`listFilesInRepo` の引数 `src` も同じ概念。

つまり関数名の「Repo」だけが `src` の射程より狭く、語が古いコンセプトに引きずられている。ADR-0008 で `dir` → `subpath` を直したのと同じ「lib 命名を実態に合わせる」整理の続き。

## 決定

- `listFilesInRepo` を **`listFilesInSrc`** に改名する。
- `InRepo`（狭い）を、entries / 引数で確定済みのドメイン語 `src` に合わせた `InSrc` に置き換える。`src` 内の subpath を走査する、という実態を名前で表す。
- 引数（`{ src, subpath? }`）・返り値（`readDir` 互換）・dir 限定制約は変更しない。改名のみ。

## 根拠

- **射程に合った名前**: `src` は repo に限らない任意の store パス。`InSrc` なら `pkgs.hello` 等を渡しても語義が破綻しない。
- **ドメイン語との一貫**: entries の `src` と関数の対象が同一概念であることが名前で繋がる。
- **`Files` は据え置き**: 今回フラグが立ったのは「Repo」（射程）だけ。`Files` は元設計が許容した語（readDir 互換で dir/symlink も返すが、ファイル列挙ヘルパーとして通る）で、`Dir` への二次変更はサーフェスを広げるため行わない。
- **今が最安**: 実装ゼロのため改名コストが無い。

## 影響

- **`docs/spec.md`**: `lib.listFilesInRepo` の見出し・シグネチャ・使用例、および string interpolation サンプル（応用節）の関数名を `listFilesInSrc` に改名。lib 概要・依存関係表の関数列挙も更新。
- **`docs/design.md`**: プロジェクト構成（`lib/default.nix` の公開 API 列挙）・flake outputs の lib コメントを更新。
- **`CONTEXT.md`**: `subpath` 項の関数言及を `listFilesInSrc` に更新。
- **ADR-0008**: 本文は当時の名（`listFilesInRepo`）で記述された歴史的記録として残し、ヘッダに本 ADR への改訂注記を足す。
- **実装フェーズ**: `lib/default.nix` は最初から `listFilesInSrc` で公開する。

## 棄却した代替案

- **`listFiles`（修飾語を全部落とす）**: 最小だが、`{ src, subpath }` を見ないとスコープが分からない。ドメイン語 `src` を名前に残す方が discoverable。
- **`listDir`（`readDir` ミラー）**: 実態に最も正確だが、`Files` → `Dir` の二次変更になり、指摘の範囲（Repo）を超える。
- **`listFilesInStore`**: store 走査の意図は明示できるが、`builtins.path` 等 store 以外の path を渡す余地を名前で狭める。
- **`listFilesInRepo` を維持**: 「Repo」が `src` の射程より狭く、初期コンセプトの名残のまま実態とズレる。
