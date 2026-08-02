---
id: "REQ-81249072-56b8-42f5-807b-ea623c1efe99"
type: requirement
name: "out-of-store のローカルパスは評価時に確定し、target の root 解決には影響しない"
specification: |
  The argument of `mkOutOfStoreSymlink` SHALL be an absolute path settled at Nix
  evaluation time and hard-coded into the manifest, `$HOME` being unusable. Because the
  root of a `target` is resolved at run time by the engine, the local path of an
  out-of-store entry MUST NOT affect the resolution on the `target` side; the two SHALL be
  independent. Since the convention for the home directory differs between macOS and
  Linux, a local path SHOULD be resolved by discriminating the operating system within the
  flake.
specification_ja: |
  `mkOutOfStoreSymlink` の引数は Nix 評価時に確定して manifest へハードコードされる絶対パス
  でなければならず、`$HOME` は使えない。`target` の root 解決は engine が実行時に行うため、
  out-of-store entry のローカルパスは `target` 側の解決に影響してはならない（両者は独立で
  ある）。macOS / Linux でホームの慣例が異なるため、ローカルパスは flake 内で OS を判別して
  解決するべきである。
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
機構へ委譲せず engine が扱うと定めた ADR-0003。
