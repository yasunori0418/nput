---
id: "REQ-5c6b07da-3d06-414d-8770-4f438234b322"
type: requirement
name: "同一 manifest 内の target 衝突は評価時に検出して停止する"
specification: |
  A collision in which two distinct keys A and B explicitly override the `target` field to
  the same value SHALL be detected as a duplicate of the normalized target string and
  SHALL stop evaluation via `lib.throwIf` at eval time, not at engine runtime. A
  cross-config collision (a same target across separate profiles or manifests) cannot be
  detected at eval time and SHALL instead be handled at engine runtime as last-writer-wins
  plus a foreign symlink warning. The two SHALL remain separate paths.
specification_ja: |
  別キー A / B が `target` フィールドを同値に明示上書きした衝突は、正規化後 target
  文字列の重複として eval 時に `lib.throwIf` で検出し停止しなければならない
  （engine 実行時ではない）。cross-config（別 profile・別 manifest）の同一 target 衝突は
  eval では検出できず、engine 実行時の後勝ち + foreign symlink warning として扱う。
  両者は別経路とする。
---
# REQ-5c6b07da: 同一 manifest 内の target 衝突は評価時に検出して停止する

## 仕様

属性キーによる一意性担保だけでは塞げない経路として、**別キー A/B が `target` フィールドを
同値に明示上書きした衝突**がある。これは正規化後 target 文字列の重複として eval 時に
`lib.throwIf` で検出・停止する（engine 実行時ではない）。

cross-config（別 profile・別 manifest）の同一 target 衝突は eval では検出不可で、
これは engine 実行時の後勝ち + foreign symlink warning になる。両者は別経路。

## 出典

`docs/spec.md`「lib API」→「入力検査（`evalModules` + `normalizeManifest`）」。
