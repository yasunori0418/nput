---
id: "REQ-e1e1114b-ba07-4d57-8e04-6e30e39a5da3"
type: requirement
name: "nput.backup は engine 起動の配線レイヤーのオプションで manifest には影響しない"
specification: |
  The `nput.backup` submodule, common to every module, SHALL wire `--backup=<suffix>` onto
  the `nput apply --manifest` invocation performed at activation, the suffix being the
  value of `nput.backup.suffix`, whose default REQ-fc1c7ce6 states. `nput.backup` SHALL be
  an option of the invocation-wiring layer, independent of `entries` (the manifest v1
  contract), and SHALL NOT affect the manifest itself.
specification_ja: |
  全モジュール共通の `nput.backup` submodule は、activation で行う
  `nput apply --manifest` 起動へ `--backup=<suffix>` を配線しなければならない。suffix は
  `nput.backup.suffix` の値とし、そのデフォルトは REQ-fc1c7ce6 が定める。`nput.backup` は
  `entries`（manifest v1 契約）とは独立な起動配線レイヤーのオプションであり、manifest 自体に
  影響してはならない。
---
# REQ-e1e1114b: nput.backup は engine 起動の配線レイヤーのオプションで manifest には影響しない

## 仕様

`nput.backup`（submodule・全モジュール共通）は activation の `nput apply --manifest` 起動へ
`--backup=<suffix>` を配線する。`enable = true` かつ `suffix` 省略で既定 `nput-backup` が
使われる。`entries`（manifest v1 契約・`lib/types.nix`）とは独立な起動配線レイヤーの
オプションで、manifest 自体には影響しない。

> **上は原文の写しで、規範は frontmatter が正**。オプションの型とデフォルト値そのものは
> REQ-fc1c7ce6、`--backup[=<suffix>]` が何を退避しどう振る舞うかは REQ-5dd5a4e9、
> module activation が `apply --manifest` で engine を kick すること自体は REQ-dec58330 /
> REQ-8085f194 の担当。本 item は `nput.backup` が manifest ではなく起動側に効くという
> レイヤーの所属を規定する。

## 出典

`docs/spec.md`「モジュールオプション仕様」→「共通オプション（全モジュール）」節の
`nput.backup` の段落。

決定の実体は ADR-0045「`apply --backup[=suffix]` — 配置を塞ぐ記録外実体の rename 退避」で、
module 経路への配線は同 ADR が Issue #169 とともに定めている。
