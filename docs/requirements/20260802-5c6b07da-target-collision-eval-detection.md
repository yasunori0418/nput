---
id: "REQ-5c6b07da-3d06-414d-8770-4f438234b322"
type: requirement
name: "target 衝突の検出経路を同一 manifest 内と cross-config で分ける"
derives_from:
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
specification: |
  A collision in which two distinct keys A and B explicitly override the `target` field to
  the same value SHALL be detected as a duplicate of the normalized target string and
  SHALL stop evaluation via `lib.throwIf` at eval time, not at engine runtime. A
  cross-config collision that does not ride on a single evaluation (a same target across
  separate entrypoints, machines or tools) cannot be detected at eval time and SHALL
  instead be handled at engine runtime as last-writer-wins plus a foreign symlink warning.
  The two SHALL remain separate paths. Where several configs do ride on a single
  evaluation, as the `nput.configs` of one module configuration do, static detection is
  possible and SHALL NOT be precluded by this; that case is stated by REQ-5923ac79 and is
  not restated here.
specification_ja: |
  別キー A / B が `target` フィールドを同値に明示上書きした衝突は、正規化後 target
  文字列の重複として eval 時に `lib.throwIf` で検出し停止しなければならない
  （engine 実行時ではない）。単一の eval に載らない cross-config（別 entrypoint・別マシン・
  別ツールに跨る場合）の同一 target 衝突は eval では検出できず、engine 実行時の後勝ち +
  foreign symlink warning として扱う。両者は別経路とする。単一の eval に載る複数 config
  （1 つのモジュール config の `nput.configs` など）については静的検出が可能であり、
  本 item はそれを妨げない。その場合の規範は REQ-5923ac79 の担当で、本 item では
  規定しない。
---
# REQ-5c6b07da: target 衝突の検出経路を同一 manifest 内と cross-config で分ける

## 仕様

属性キーによる一意性担保だけでは塞げない経路として、**別キー A/B が `target` フィールドを
同値に明示上書きした衝突**がある。これは正規化後 target 文字列の重複として eval 時に
`lib.throwIf` で検出・停止する（engine 実行時ではない）。

cross-config（別 profile・別 manifest）の同一 target 衝突は eval では検出不可で、
これは engine 実行時の後勝ち + foreign symlink warning になる。両者は別経路。

> **上は原文の写しで、規範は frontmatter が正**。原文が「cross-config」を無条件に「eval では
> 検出不可」とするのに対し、規範文では**単一の eval に載らない場合**（別 entrypoint・別マシン・
> 別ツール）へ限定した。**ADR-0035 §4 が、1 つのモジュール config 内の `nput.configs.<A>` と
> `<B>` は全 config が同一のモジュール eval に載るため正規化後 target の衝突を静的に検出でき、
> eval 時 assertion で停止すると決定済み**で、原文の無条件な言い切りはこれを否定してしまう
> （原文が ADR-0035 に未追従・REQ-37b56673 / REQ-16faf428 で ADR-0036 由来の未追従を扱ったのと
> 同じ扱い）。本 item は静的検出が可能な場合を妨げないことまでを規範とし、その場合に実際に
> eval 停止する規範（ADR-0035 §4）そのものは持たない。**同 §4 の規範は REQ-5923ac79 が
> 持つ**。`docs/spec.md` の追従は本 item の担当範囲外。

## 出典

`docs/spec.md`「lib API」→「入力検査（`evalModules` + `normalizeManifest`）」。

決定の実体は同一 manifest 内の target 衝突を eval 時に検出すると定めた ADR-0024
「実装前残セマンティクス第6巡」§5 で、cross-config を「単一の eval に載らない場合」へ限定した
のは ADR-0035「HM モジュールに `nput.configs.<name>` を導入し複数 profile（役割分離）を
可能にする」§4（単一のモジュール eval に載る config 間は静的検出が可能）。
