---
id: "REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66"
type: requirement
name: "root は 3 マーカーと絶対パス文字列の union を取る"
specification: |
  The type of `root` SHALL be a union of `string` (fixed at evaluation time) and marker
  (resolved at runtime). `nput.lib.projectRoot` SHALL select project mode (git toplevel
  at runtime, overridable with `--root`), `nput.lib.homeRoot` SHALL select home mode
  (`$HOME` at runtime), `nput.lib.systemRoot` SHALL select system mode (`/`, future), and
  an absolute path string SHALL select a fixed root determined at evaluation time.
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

home mode と project mode は世代の扱いが異なる（→ `docs/spec.md`「世代管理仕様」）。

## 出典

`docs/spec.md`「lib API」→「`lib.mkManifest`」→「`root` の値」。
