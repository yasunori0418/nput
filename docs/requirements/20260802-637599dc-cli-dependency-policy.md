---
id: "REQ-637599dc-a1ec-4af5-9e97-e882c7df56d0"
type: requirement
name: "CLI の第三者依存は最小限に許可し、ビルドは vendorHash と pin した Go で固定する"
specification: |
  The CLI SHALL be allowed a minimal set of third-party dependencies — cobra, for
  subcommands and help — while importing the engine for placement, in contrast to the
  engine's stdlib-only constraint. It SHALL be built with `buildGoModule` and a vendorHash
  string, and the Go toolchain SHALL be pinned to the go of nixpkgs with the `toolchain`
  directive left unused, so that the build does not fetch a toolchain of its own and stays
  reproducible under Nix.
specification_ja: |
  CLI には第三者依存を最小限だけ許可しなければならない（サブコマンド / help のための
  cobra）。配置は engine を import して行う（engine の stdlib-only 制約とは対照的に扱う）。
  ビルドは `buildGoModule` と vendorHash 文字列で行い、Go ツールチェーンは nixpkgs の go に
  pin して `toolchain` ディレクティブを使ってはならない（ビルドが独自にツールチェーンを
  取得せず Nix の下で再現可能に保つため）。
---
# REQ-637599dc: CLI の第三者依存は最小限に許可し、ビルドは vendorHash と pin した Go で固定する

## 仕様

| コンポーネント | 依存 |
|---|---|
| `cmd/nput`（CLI = `packages.nput`）| 配置エンジンを import。最小依存を許可（**cobra** = サブコマンド / help）。entrypoint 発見と `nix`（build / eval）オーケストレーションを担う。`buildGoModule` + **vendorHash 文字列**でビルド。Go は nixpkgs の go に pin し `toolchain` ディレクティブ不使用 |

> **上は原文の写しで、規範は frontmatter が正**。CLI が entrypoint 発見と `nix` の
> オーケストレーションを担う役割分担そのものは REQ-f4d7d4ab、entrypoint の探索順は
> REQ-1cc080f6、実行フローにおける eval / build の順序は REQ-60c6b7ea、engine 側の
> stdlib-only 制約は REQ-b74a118a の担当。

## 出典

`docs/spec.md`「依存関係」節の表の `cmd/nput` 行。

決定の実体は ADR-0011「engine / CLI の技術スタック」で、CLI に cobra を許可し
`buildGoModule` + vendorHash・Go の pin でビルドを固定することを定めている。CLI を一次 UX と
する位置づけは ADR-0007 による。
