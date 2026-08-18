---
id: "REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66"
type: requirement
name: "root は 3 マーカーと絶対パス文字列の union を取る"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The type of `root` SHALL be a union of `string` (fixed at evaluation time) and marker
  (resolved at runtime). `nput.lib.projectRoot` SHALL select project mode (git toplevel
  at runtime, overridable with `--root`), `nput.lib.homeRoot` SHALL select home mode
  (`$HOME` at runtime), `nput.lib.systemRoot` SHALL select system mode (`/` at runtime),
  and an absolute path string SHALL select a fixed root determined at evaluation time.
specification_ja: |
  `root` の型は `string`（評価時固定）と marker（実行時解決）の union でなければ
  ならない。`nput.lib.projectRoot` は project mode（実行時に git toplevel・`--root` で
  上書き可）、`nput.lib.homeRoot` は home mode（実行時の `$HOME`）、
  `nput.lib.systemRoot` は system mode（実行時の `/`）、絶対パス文字列は固定 root
  （評価時に確定）へそれぞれ対応しなければならない。
---
# REQ-37b56673: root は 3 マーカーと絶対パス文字列の union を取る

## 仕様

`root` の型は `string（評価時固定）| marker（実行時解決）` の union。

| `root` の値 | モード | root の解決 |
|---|---|---|
| `nput.lib.projectRoot` | project mode | 実行時に git toplevel（`--root` で上書き可）|
| `nput.lib.homeRoot` | home mode | 実行時の `$HOME`（standalone / HM 共通）|
| `nput.lib.systemRoot` | system mode | `/`（distro 構想・将来）|
| 絶対パス文字列 | 固定 root | 評価時に確定する絶対パス（任意固定 root の seam）|

上の表は原文の写しで、規範は frontmatter が正。`systemRoot` の「distro 構想・将来」の
限定は規範に含めない（理由は下の注記）。

home mode と project mode は世代の扱いが異なる。この差異そのものは REQ-46fccb80 の担当で、
本 item は root の値と対応するモードまでを規定する。

## 出典

`docs/spec.md`「lib API」→「`lib.mkManifest`」→「`root` の値」、および「root の解決」→
「system mode（`root = systemRoot`・将来）」節。後者は「root = `/`」「distro 構想の system
配置 seam」「今回の実装スコープ外」の 3 点からなり、規範として残る「root = `/`」が本 item の
表と同一のため、独立 item を立てず本 item に畳んだ（残る 2 点を規範に採らない理由は下の注記）。

> **`systemRoot` を「distro 構想・将来」としない理由**: `docs/spec.md` の当該表は
> `systemRoot` を「`/`（distro 構想・将来）」と記すが、system mode は **ADR-0036 が
> 実装を決定済み**（`rootKind = "system"` を正規値として通し、engine は root = `/` へ
> 解決する）。`docs/spec.md` 側がこの改訂に追従できていないため、分割にあたって
> 「将来」の限定を規範文へ持ち込まない判断をした（REQ-16faf428 で同じ ADR-0036 由来の
> 未追従を扱ったのと同じ扱い）。
>
> **marker の関数適用形（`subdir`）を含めない理由**: **ADR-0040** は root マーカーを
> `homeRoot { subdir = ".config"; }` の関数適用形へ拡張することを決定済みだが、
> `docs/spec.md` の当該表はこれにも未追従。本 item は原文の範囲（root の値と対応する
> モード）に留め、適用形は担当範囲外とする（REQ-3f541d39 に同じ注記）。
>
> いずれも `docs/spec.md` の追従は本 item の担当範囲外。
