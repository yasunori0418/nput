---
id: "REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10"
type: requirement
name: "全モジュールは共通オプションの同一集合を公開し、root はモジュールが pin する"
specification: |
  Every module (home-manager / NixOS / nix-darwin) SHALL expose the same set of common
  options: `nput.enable :: bool` defaulting to false, `nput.entries :: attrsOf entry`
  defaulting to `{}`, `nput.backup.enable :: bool` defaulting to false, and
  `nput.backup.suffix :: str` defaulting to `"nput-backup"`. A module SHALL pin the root
  by its own nature — home-manager to `homeRoot`, a devShell to `projectRoot` — and
  SHALL NOT expose a `root` option, so that a user of a module never restates it. This
  is a departure from `mkManifest` and the CLI entrypoint, where `root` is required to be
  stated explicitly (REQ-4ec3accc).
specification_ja: |
  全モジュール（home-manager / NixOS / nix-darwin）は共通オプションとして同一の集合を
  公開しなければならない: `nput.enable :: bool`（デフォルト false）・
  `nput.entries :: attrsOf entry`（デフォルト `{}`）・`nput.backup.enable :: bool`
  （デフォルト false）・`nput.backup.suffix :: str`（デフォルト `"nput-backup"`）。
  モジュールは自分の性質で root を pin しなければならず（home-manager は `homeRoot`・
  devShell は `projectRoot`）、`root` オプションを公開してはならない。モジュール利用者が
  root を再指定しないようにするためである。これは `root` を明示必須とする `mkManifest` /
  CLI entrypoint の層（REQ-4ec3accc）との差分である。
---
# REQ-fc1c7ce6: 全モジュールは共通オプションの同一集合を公開し、root はモジュールが pin する

## 仕様

```
nput.enable  :: bool          # デフォルト: false
nput.entries :: attrsOf entry # デフォルト: {}（属性キー = target・→ ADR-0014）
nput.backup.enable :: bool    # デフォルト: false（→ ADR-0045）
nput.backup.suffix :: str     # デフォルト: "nput-backup"（→ ADR-0045）
```

モジュールは自分の性質で root を pin する（HM → `homeRoot` / devShell → `projectRoot`）ため、
モジュール利用者は `root` を再指定しない。

> **上は原文の写しで、規範は frontmatter が正**。`entries` の属性キーが target であることと
> entry submodule のフィールド定義は REQ-cb77ea05 / REQ-a33a11e3、`root` が
> `mkManifest` / CLI 層で明示必須であることは REQ-4ec3accc、`--backup` の退避契約そのものは
> REQ-5dd5a4e9 の担当。`nput.backup` が manifest ではなく engine 起動の配線に効くことは
> REQ-e1e1114b、HM の `entries` が単一 profile に限られることは REQ-c6891aeb の担当。

## 出典

`docs/spec.md`「モジュールオプション仕様」→「共通オプション（全モジュール）」節の
オプション表と、その直後の root pin の段落。

決定の実体は root 明示必須とモジュール側 pin を定めた ADR-0007「汎用 nput CLI を一次 UX に
昇格し、entrypoint 発見＋root 明示モデルへ移行する」と、配置ロジックを engine が所有し
モジュールは配線に徹すると定めた ADR-0003。`nput.backup` の追加は ADR-0045。
