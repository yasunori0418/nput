---
id: "REQ-99ca5381-6c53-426c-b145-7b4297c53868"
type: requirement
name: "src は path / set / marker の 3 種を取り store link を既定として out-of-store は marker で opt-in する"
specification: |
  The `src` field SHALL accept three kinds of value: a `path`, a `set` (derivation), and
  a `marker`. For `path` and `set` the placement source SHALL be a Nix store path, which
  is the default behaviour. Only a marker returned by `mkOutOfStoreSymlink` SHALL select
  an out-of-store placement source pointing at a live local filesystem path, so
  out-of-store SHALL be opt-in and explicit. Passing a bare `string` as `src` MUST NOT be
  supported as an implicit way to select out-of-store.
specification_ja: |
  `src` フィールドは `path` / `set`（derivation）/ `marker` の 3 種の値を受け付け
  なければならない。`path` と `set` では配置元を Nix ストアのパスとし、これを既定の
  挙動とする。out-of-store（ローカル FS のライブなパス）を配置元にできるのは
  `mkOutOfStoreSymlink` が返す marker のみとし、opt-in の明示指定でなければならない。
  `src` に生の `string` を渡して暗黙に out-of-store を選ぶ形は提供してはならない。
---
# REQ-99ca5381: src は path / set / marker の 3 種を取り store link を既定として out-of-store は marker で opt-in する

## 仕様

`src` は必須フィールドで、配置元を表す。デフォルトは Nix ストアへの symlink で、
out-of-store は明示マーカーで opt-in する。

| `src` の値 | 例 | symlink の指す先 | 用途 |
|---|---|---|---|
| `path` | `inputs.myrepo` | Nix ストア（不変）| 外部リポジトリ（バージョン固定）|
| `path` | `builtins.path { path = /home/...; name = "..."; }` | Nix ストア（ローカルをコピー）| ローカルをストア経由で扱う |
| `set` | `pkgs.fetchFromGitHub { ... }` | Nix ストア（不変）| 外部リポジトリ（バージョン固定）|
| `marker` | `nput.lib.mkOutOfStoreSymlink "/abs/path"` | ローカル FS（ライブ）| 開発中の手元 dotfiles |

上の表は原文の写しで、規範は frontmatter が正。

`src` に string を直接渡して暗黙に out-of-store へ分岐する形は提供しない。

```nix
# 廃止: string を直接渡す暗黙の out-of-store 分岐は提供しない
# src = "/path/to/dotfiles";   # NG
```

marker が何を返すか（`mkOutOfStoreSymlink` の責務）は REQ-eb363122、marker と
derivation の判別方式は REQ-1dcc9a33 が持つ。本 item は `src` が取る値の種別と、
どれが store / out-of-store になるかを規定する。

## 出典

`docs/spec.md`「entries スキーマ仕様」→「フィールド定義（entry submodule）」→ `#### src`。

> **「entries を動的に生成する」節を item 化しない理由**: 同節が示す
> `builtins.listToAttrs` での target キー補間・`baseNameOf src` のアンチパターン・
> subdir 列挙の応用例は、いずれも利用者向けの idiom であって nput 実装が満たすべき
> 義務ではない。節の注記が述べる「eval 時 `readDir` の対象は既 realise の store パスに
> 限る（生 derivation は IFD を誘発し marker は展開できない）」も、`listFilesInSrc` を
> lib から除去して docs の応用 idiom へ降格した決定（`docs/publication-roadmap.md` ①・
> 完了済み）により、規範を負う実装主体が存在しない。`src` が `set` / marker を許容する
> 側（engine 実行時解決であるため成立する）は本 item の規範が担う。
