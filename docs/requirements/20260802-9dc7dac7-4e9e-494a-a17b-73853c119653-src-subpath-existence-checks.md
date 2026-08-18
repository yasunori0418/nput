---
id: "REQ-9dc7dac7-4e9e-494a-a17b-73853c119653"
type: requirement
name: "配置元の実在は判定できる層で検査し、いずれの層でも停止する"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
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
  エラーとしなければならない。out-of-store marker として与えた `src` のローカルパスが存在
  しない場合と、`subpath` が `src` 内に存在しない場合は、実体を見るまで判定できないため
  engine 実行時のエラーとして停止しなければならない。いずれの場合も停止を動作としなければ
  ならず、配置元の不在を黙って見過ごしたり dangling symlink に変えたりしてはならない。
---
# REQ-9dc7dac7-4e9e-494a-a17b-73853c119653: 配置元の実在は判定できる層で検査し、いずれの層でも停止する

## 仕様

| 条件 | 動作 |
|---|---|
| `src` が存在しないストアパス（path / set）| Nix 評価時にエラー |
| `src` が marker でローカルパスが存在しない | engine 実行時にエラーで停止 |
| `subpath` が `src` 内に存在しないパス | engine 実行時にエラーで停止 |

> **上は原文の写しで、規範は frontmatter が正**。`src` が取りうる 3 種（path / set /
> marker）と store link 既定は REQ-99ca5381-6c53-426c-b145-7b4297c53868、`subpath` が src 内相対で省略がリポジトリ
> 全体を表すことは REQ-27b75fe6-6c36-44a8-8cd3-5cc98043022a、out-of-store のローカルパスが評価時に確定することは
> REQ-81249072-56b8-42f5-807b-ea623c1efe99、層分けの原則そのものは REQ-c5dfcae6-6094-4850-99e5-bf14530bc60a の担当。

## 出典

`docs/spec.md`「エラー仕様」節の表の `src` / `subpath` 実在に関する 3 行。

この 3 行に対応する決定を持つ ADR は無く、`docs/spec.md` が一次記述にあたる（原文の表でも
この 3 行は `→ ADR-00xx` の参照注記を持たない。同様に注記を持たない行は他にもある）。
よって本 item に `justifies` は張られないが、これは張り漏れではない。層が分かれること自体の
一般則は REQ-c5dfcae6-6094-4850-99e5-bf14530bc60a の担当。
