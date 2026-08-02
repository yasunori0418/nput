---
id: "REQ-5c6b07da-3d06-414d-8770-4f438234b322"
type: requirement
name: "target 衝突の検出経路を同一 manifest 内と cross-config で分ける"
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
# REQ-5c6b07da: target 衝突の検出経路を同一 manifest 内と cross-config で分ける

## 仕様

属性キーによる一意性担保だけでは塞げない経路として、**別キー A/B が `target` フィールドを
同値に明示上書きした衝突**がある。これは正規化後 target 文字列の重複として eval 時に
`lib.throwIf` で検出・停止する（engine 実行時ではない）。

cross-config（別 profile・別 manifest）の同一 target 衝突は eval では検出不可で、
これは engine 実行時の後勝ち + foreign symlink warning になる。両者は別経路。

> **HM モジュールの `configs` は上の cross-config の例外**（→ ADR-0035 §4）。単一の HM config 内の
> `nput.configs.<A>` と `<B>` は全 config が同一のモジュール eval に載るため正規化後 target の
> 衝突を静的に検出でき、ADR-0035 §4 はこれを eval 時 assertion で停止すると決定している。
> 本 item の「cross-config は eval では検出不可」は別 entrypoint・別マシン・別ツールの場合を
> 指し、この例外を否定しない。`docs/spec.md` が ADR-0035 に未追従で対応記述を持たないため、
> 例外の item 化は原文の追従（epic #203 の段階 7）とあわせて別途扱う（→ REQ-c6891aeb の注記）。

## 出典

`docs/spec.md`「lib API」→「入力検査（`evalModules` + `normalizeManifest`）」。
