---
id: "REQ-e1e1114b-ba07-4d57-8e04-6e30e39a5da3"
type: requirement
name: "nput.backup は engine 起動の配線レイヤーのオプションで manifest には影響しない"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The `nput.backup` submodule, common to every module, SHALL wire `--backup=<suffix>` onto
  the `nput apply --manifest` invocation performed at activation only where
  `nput.backup.enable` is true, setting it aside being an explicit opt-in by the user; the
  suffix SHALL be the value of `nput.backup.suffix`, whose default REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10 states.
  `nput.backup` SHALL be an option of the invocation-wiring layer, independent of `entries`
  (the manifest v1 contract), and SHALL NOT affect the manifest itself.
specification_ja: |
  全モジュール共通の `nput.backup` submodule は、`nput.backup.enable` が true のときに限り、
  activation で行う `nput apply --manifest` 起動へ `--backup=<suffix>` を配線しなければ
  ならない（退避はユーザーの明示 opt-in であるため）。suffix は `nput.backup.suffix` の値で
  なければならず、そのデフォルトは REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10 が定める。`nput.backup` は `entries`
  （manifest v1 契約）とは独立な起動配線レイヤーのオプションであり、manifest 自体に影響して
  はならない。
---
# REQ-e1e1114b-ba07-4d57-8e04-6e30e39a5da3: nput.backup は engine 起動の配線レイヤーのオプションで manifest には影響しない

## 仕様

`nput.backup`（submodule・全モジュール共通）は activation の `nput apply --manifest` 起動へ
`--backup=<suffix>` を配線する。`enable = true` かつ `suffix` 省略で既定 `nput-backup` が
使われる。`entries`（manifest v1 契約・`lib/types.nix`）とは独立な起動配線レイヤーの
オプションで、manifest 自体には影響しない。

> **上は原文の写しで、規範は frontmatter が正**。オプションの型とデフォルト値そのものは
> REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10、`--backup[=<suffix>]` が何を退避しどう振る舞うかは REQ-5dd5a4e9-6162-4fa5-b295-66844f5a4f3b、
> module activation が `apply --manifest` で engine を kick すること自体は REQ-dec58330-6dad-47f7-8f56-2402764a89c7 /
> REQ-8085f194-c903-4ecb-abd8-c719fe7b3292 の担当。本 item は `nput.backup` が manifest ではなく起動側に効くという
> レイヤーの所属と、その**配線の発動条件**（`enable` が true のときに限る）を規定する。
> 発動条件は REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10 が定めるオプションの型・デフォルト値とは別の規範で、退避が
> ユーザーの明示 opt-in であるという ADR-0045 の設計意図（既定 false・記録外実体を勝手に
> 動かさない）を担保する。

## 出典

`docs/spec.md`「モジュールオプション仕様」→「共通オプション（全モジュール）」節の
`nput.backup` の段落。

決定の実体は ADR-0045「`apply --backup[=suffix]` — 配置を塞ぐ記録外実体の rename 退避」で、
module 経路への配線は同 ADR が Issue #169 とともに定めている。
