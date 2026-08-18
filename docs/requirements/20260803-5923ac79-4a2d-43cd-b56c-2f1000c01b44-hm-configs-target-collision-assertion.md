---
id: "REQ-5923ac79-4a2d-43cd-b56c-2f1000c01b44"
type: requirement
name: "単一 HM config 内の configs 間 target 衝突は eval 時 assertion で停止する"
derives_from:
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
specification: |
  Where `configs.<A>` and `configs.<B>` of a single home-manager configuration duplicate a
  normalized target — the string resolved after the attribute-key default and any explicit
  `target` override — the module evaluation SHALL be stopped by an assertion at eval time.
  This is possible because every config of one module configuration rides on a single
  module evaluation, and it is the natural extension of stopping at eval time a collision
  that eval can see, stated for a single manifest by REQ-5c6b07da-3d06-414d-8770-4f438234b322. It SHALL be an exception
  to the general cross-config collision, which does not ride on a single evaluation and
  which REQ-5c6b07da-3d06-414d-8770-4f438234b322 and REQ-fc1118b1-b0e8-4ddf-80f6-c70956651693 leave to last-writer-wins plus a foreign symlink
  warning at engine runtime; how a collision across separate entrypoints, machines or
  tools is treated is stated by those two items and is not restated or altered here.
  Without this detection, the two configs would
  take the same target from each other at every activation, making a flip-flop permanent,
  the introduction of role separation itself being what breeds the collision.
specification_ja: |
  単一の home-manager config 内の `configs.<A>` と `configs.<B>` が正規化後 target
  （属性キー既定値・明示 `target` 上書きを解決した後の文字列）を重複させた場合、
  モジュール eval 時に assertion で停止しなければならない。1 つのモジュール config の
  全 config が単一のモジュール eval に載るため静的検出が可能であり、eval で分かる衝突を
  eval で止めること（同一 manifest 内について REQ-5c6b07da-3d06-414d-8770-4f438234b322 が定める）の自然な延長で
  ある。これは単一の eval に載らない一般の cross-config 衝突（REQ-5c6b07da-3d06-414d-8770-4f438234b322 /
  REQ-fc1118b1-b0e8-4ddf-80f6-c70956651693 が engine 実行時の後勝ち + foreign symlink warning に委ねるもの）に
  対する例外でなければならない。別 entrypoint・別マシン・別ツールに跨る衝突をどう扱うかは
  同 2 item の担当であり、本 item では再掲も変更もしない。
  検出しなければ、activation のたびに A と B が同じ target を交互に奪い合う
  フリップフロップが恒常化する。役割分離の導入自体が衝突の温床になるためである。
---
# REQ-5923ac79-4a2d-43cd-b56c-2f1000c01b44: 単一 HM config 内の configs 間 target 衝突は eval 時 assertion で停止する

## 仕様

- 同一 HM config 内の `configs.<A>` と `configs.<B>` が**正規化後 target**（属性キー
  既定値・明示 `target` 上書きを解決した後の文字列）を重複させた場合、**モジュール
  eval 時に assertion で停止**する
- 一般の cross-config 衝突（別 entrypoint・別マシン・別ツール）が「engine 実行時の
  後勝ち + foreign symlink warning」であることは**不変**。HM の `configs` は全 config が
  単一のモジュール eval に載るため例外的に静的検出が可能であり、「eval で分かる衝突は
  eval で止める」の自然な延長として扱う
- 検出しなければ、activation のたびに A と B が同じ target を交互に奪い合う（毎回
  foreign symlink warning + 上書き）フリップフロップが恒常化する。役割分離の導入自体が
  衝突の温床になるため、eval 停止が正しい

> **本 item の出典は ADR-0035 §4 であり、`docs/spec.md` に対応記述は無い**。原文が
> ADR-0035 に未追従のため #209 の分割では item 化されず、REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a / REQ-5c6b07da-3d06-414d-8770-4f438234b322 /
> REQ-fc1118b1-b0e8-4ddf-80f6-c70956651693 の注記に申し送りとして残っていた（epic #203 / issue #228 で回収）。
>
> **REQ-5c6b07da-3d06-414d-8770-4f438234b322 / REQ-fc1118b1-b0e8-4ddf-80f6-c70956651693 との整合**: 両 item は規範文を「単一の eval に載らない
> 場合」へ限定したうえで、単一の eval に載る複数 config については「静的検出を妨げない」
> までを述べ、実際に eval 停止する規範そのものは持たないと明記している。本 item がその
> 欠けていた規範を担う。両 item が何を規定するかはそれぞれの specification が正で、
> ここでは再掲しない。検出層を「設定の誤りは評価時・実体の不整合は engine 実行時」に
> 分ける原則そのものは REQ-c5dfcae6-6094-4850-99e5-bf14530bc60a の担当。
>
> **他 item との担当分界**: 正規化後 target 文字列の同値判定と属性キー = target という
> 識別の体系は REQ-cb77ea05-bab8-4ccf-b09e-d23d8f71cdc7 / REQ-b232ec98-af3b-41f3-a050-29d417322002、`nput.configs` オプションの定義は
> REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10、`<name>` 次元と役割分離そのものは REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a、foreign symlink warning
> と単発の後勝ちは REQ-622787dc-4512-4ce9-9c7d-7b32bbb70557 の担当。

## 出典

ADR-0035「HM モジュールに `nput.configs.<name>` を導入し複数 profile（役割分離）を
可能にする」§4「configs 間の target 衝突は eval 時に検出して停止する」。

`docs/spec.md` には対応記述が無いため、原文の写しは持たない（規範は frontmatter が正で、
上の箇条書きは ADR 本文の要約）。同一 manifest 内の target 衝突を eval 時に検出すると
定めたのは ADR-0024 §5、cross-config を engine 実行時の後勝ち + foreign symlink warning
とするのは ADR-0015、未定義挙動を早期に弾く姿勢は ADR-0010 が定めている。
