---
id: "REQ-81249072-56b8-42f5-807b-ea623c1efe99"
type: requirement
name: "out-of-store のローカルパスは評価時に確定し、target の root 解決には影響しない"
derives_from:
  - "UC-01b896b4-04b9-40d0-bf9e-966eaf64c3d4"
specification: |
  The local path of an out-of-store entry, settled at Nix evaluation time (REQ-eb363122),
  SHALL be hard-coded into the manifest as it stands. Because the root of a `target` is
  resolved at run time by the engine, that local path SHALL NOT affect the resolution on
  the `target` side; the two SHALL belong to separate points in time and SHALL be
  independent. Since the convention for the home directory differs between macOS and
  Linux, a local path SHOULD be resolved by discriminating the operating system within the
  flake.
specification_ja: |
  Nix 評価時に確定する out-of-store entry のローカルパス（REQ-eb363122）は、そのまま
  manifest へハードコードされなければならない。`target` の root 解決は engine が実行時に
  行うため、そのローカルパスは `target` 側の解決に影響してはならない（両者は別の時点に
  属し独立である）。macOS / Linux でホームの慣例が異なるため、ローカルパスは flake 内で
  OS を判別して解決するべきである。
---
# REQ-81249072: out-of-store のローカルパスは評価時に確定し、target の root 解決には影響しない

## 仕様

`mkOutOfStoreSymlink` の引数は Nix 評価時に確定する絶対パス。`$HOME` は使えない。
`target` の root 解決は実行時に行われるため `target` 側には影響しない。

```
Nix 評価時:  mkOutOfStoreSymlink "/path/to/dotfiles"  →  manifest にハードコード
実行時:      root を engine が解決                       →  target を絶対パス化
```

macOS / Linux でホームの慣例が異なるため、ローカルパスは flake 内で OS 判別して解決するのが
推奨。`builtins.getEnv "HOME"`（`--impure` 必要）や flake の `specialArgs` 注入も使えるが、
通常は OS 判別で十分。

> **上は原文の写しで、規範は frontmatter が正**。`mkOutOfStoreSymlink` がマーカーを返す純粋
> 関数であることと引数が評価時確定の絶対パスであること自体は REQ-eb363122、out-of-store entry
> の配置と版管理の範囲は REQ-a8a923ad、root マーカーの実行時解決は REQ-3f541d39 の担当。
> 本 item は「評価時に確定するローカルパスと実行時に解決される target root が独立した 2 つの
> 時点に属する」という関係を規定する。

## 出典

`docs/spec.md`「root の解決」→「ローカルパス（out-of-store）の扱い」節（本文・時点対比の
コードブロック・OS 判別の推奨）。同節の `nix` 使用例は要求ではないため規範に採らない。

決定の実体は root を marker 化して実行時解決へ寄せた ADR-0007「汎用 nput CLI を一次 UX に
昇格し、entrypoint 発見＋root 明示モデルへ移行する」と、out-of-store symlink をネイティブ
機構へ委譲せず engine が扱うと定めた ADR-0003。ただし OS 判別によるローカルパス解決の推奨
（`SHOULD`）は原文由来で、対応する ADR を持たない（ADR 全 48 本に macOS / Linux のホーム慣例差を
扱った決定が無い）。
