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
