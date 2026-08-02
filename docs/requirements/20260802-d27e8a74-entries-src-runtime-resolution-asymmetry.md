---
id: "REQ-d27e8a74-20df-4a35-b121-17d1808a49e8"
type: requirement
name: "entries の src が derivation と marker を許容するのは engine 実行時解決だからであり eval 時 readDir は path 限定とする"
specification: |
  The `src` field of `entries` SHALL accept a `set` (derivation) and a marker because it
  is resolved by the engine at runtime. By contrast, any evaluation-time traversal with
  `builtins.readDir` SHALL be restricted to a `path` (an already realised store path,
  such as a `flake = false` flake input). Passing a raw derivation there would trigger
  import-from-derivation and break flake pure evaluation, and a marker cannot be expanded
  to a path at evaluation time at all. This asymmetry between evaluation-time `readDir`
  and runtime resolution SHALL be stated in the specification.
specification_ja: |
  `entries` の `src` が `set`（derivation）と marker を許容するのは、engine が実行時に
  解決するからでなければならない。対して評価時の `builtins.readDir` による走査の対象は
  `path`（既 realise の store パス。例: `flake = false` の flake input）に限らなければ
  ならない。生の derivation を渡すと IFD（import-from-derivation）を誘発して flake の
  pure eval が破綻し、marker はそもそも評価時にパスへ展開できないためである。この
  「評価時 readDir / 実行時解決」の非対称は仕様に明記しなければならない。
---
# REQ-d27e8a74: entries の src が derivation と marker を許容するのは engine 実行時解決だからであり eval 時 readDir は path 限定とする

## 仕様

`entries` の `src` は `path | set | marker` を許容する（→ REQ-99ca5381）。これは
配置元の解決が engine の実行時責務だから成立する。

一方、entries を動的に生成するために評価時に `builtins.readDir` で走査する対象は
**path（store パス）に限る**。

- `fetchFromGitHub` の生 derivation を直接 readDir すると IFD（import-from-derivation）
  を誘発し flake pure eval で破綻する。`flake = false` の flake input（= 既 realise の
  store path）を使う。
- `mkOutOfStoreSymlink` の marker も実行時解決の入れ物で、評価時に展開できないため
  readDir の対象にできない。

この「評価時 readDir / 実行時解決」の非対称は仕様に明記する。

`docs/spec.md` の当該節が示す動的生成の書き方そのもの（`builtins.listToAttrs` で
target キーへ名前を補間する形、`baseNameOf src` から target を導くアンチパターン、
subdir 列挙の応用例）は、規範ではなく利用ガイドのため item 化しない。target が属性キー
であること自体は REQ-cb77ea05 が持つ。

## 出典

`docs/spec.md`「entries スキーマ仕様」→「entries を動的に生成する（target キーの
string interpolation）」→「応用：subdir を列挙して動的に entry 展開する」の注記。
