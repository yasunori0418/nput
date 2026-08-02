---
id: "REQ-9dc7dac7-4e9e-494a-a17b-73853c119653"
type: requirement
name: "配置元の実在は判定できる層で検査し、いずれの層でも停止する"
specification: |
  The existence of the placement source SHALL be checked in the layer that can decide it,
  and SHALL stop the run in either layer. A `src` given as a store path (a path or a set)
  that does not exist SHALL be an error at Nix evaluation time. A `src` given as an
  out-of-store marker whose local path does not exist, and a `subpath` that does not exist
  inside `src`, SHALL be errors that stop the engine at run time, since neither is decidable
  until the actual filesystem is consulted. Stopping SHALL be the behaviour in every one of
  these cases, so that a missing source is not passed over silently or turned into a
  dangling symlink.
specification_ja: |
  配置元の実在は、それを判定できる層で検査し、いずれの層でも実行を停止させなければ
  ならない。store パス（path / set）として与えた `src` が存在しない場合は Nix 評価時の
  エラーとする。out-of-store marker として与えた `src` のローカルパスが存在しない場合と、
  `subpath` が `src` 内に存在しない場合は、実体を見るまで判定できないため engine 実行時の
  エラーとして停止する。いずれの場合も停止を動作とし、配置元の不在を黙って見過ごしたり
  dangling symlink に変えたりしないようにする。
---
# REQ-9dc7dac7: 配置元の実在は判定できる層で検査し、いずれの層でも停止する

## 仕様

| 条件 | 動作 |
|---|---|
| `src` が存在しないストアパス（path / set）| Nix 評価時にエラー |
| `src` が marker でローカルパスが存在しない | engine 実行時にエラーで停止 |
| `subpath` が `src` 内に存在しないパス | engine 実行時にエラーで停止 |

> **上は原文の写しで、規範は frontmatter が正**。`src` が取りうる 3 種（path / set /
> marker）と store link 既定は REQ-99ca5381、`subpath` が src 内相対で省略がリポジトリ
> 全体を表すことは REQ-27b75fe6、out-of-store のローカルパスが評価時に確定することは
> REQ-81249072、層分けの原則そのものは REQ-c5dfcae6 の担当。

## 出典

`docs/spec.md`「エラー仕様」節の表の `src` / `subpath` 実在に関する 3 行。

この 3 行に対応する決定を持つ ADR は無く、`docs/spec.md` が一次記述にあたる。層が分かれる
こと自体の一般則は REQ-c5dfcae6 の担当。
