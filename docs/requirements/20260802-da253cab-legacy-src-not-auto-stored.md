---
id: "REQ-da253cab-34d4-4d6e-96f0-de99e012b376"
type: requirement
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
name: "legacy entrypoint では相対 path の src が自動で store 化されない"
specification: |
  For a flake entrypoint the whole directory is copied into the store before evaluation,
  so a relative path literal (`src = ./foo;`) becomes a store symlink as is. A legacy
  entrypoint (`-f` eval) has no such prior copy, so a relative path SHALL be resolved
  relative to the actual location of the file and SHALL be carried into the manifest as a
  live working-tree path. To obtain a reproducible / store-backed `src` under a legacy
  entrypoint, an explicit means of adding to the store SHALL be used, such as
  `builtins.path { path = ./foo; }`, `fetchTarball` or `builtins.fetchGit`.
specification_ja: |
  flake entrypoint では評価前にディレクトリ全体が store へコピーされるため、相対 path
  リテラル（`src = ./foo;`）がそのまま store symlink になる。legacy entrypoint
  （`-f` eval）にはこの事前コピーが無く、相対 path はファイルの実位置基準で解決され、
  live な作業木パスのまま manifest に載る。legacy entrypoint で reproducible /
  store-backed な `src` を得るには、`builtins.path { path = ./foo; }` や `fetchTarball` /
  `builtins.fetchGit` など明示的に store へ add する手段を使わなければならない。
---
# REQ-da253cab: legacy entrypoint では相対 path の src が自動で store 化されない

## 仕様

**`src` の store 化は自動ではない**。flake は評価前にディレクトリ全体が store へ
コピーされるため相対 path リテラル（`src = ./foo;`）がそのまま store symlink になるが、
legacy（`-f` eval）にはこの事前コピーが無く、相対 path はファイルの実位置基準で
解決される live な作業木パスのまま manifest に載る。reproducible / store-backed に
したい場合は `src` に `builtins.path { path = ./foo; }` や `fetchTarball` /
`builtins.fetchGit` など明示的に store へ add する手段を使う。

## 出典

`docs/spec.md`「CLI 仕様」→「アドレッシング」の blockquote 末尾「`src` の store 化は
自動ではない」。

決定の実体は ADR-0032。
