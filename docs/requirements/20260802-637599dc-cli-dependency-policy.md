---
id: "REQ-637599dc-a1ec-4af5-9e97-e882c7df56d0"
type: requirement
name: "CLI が持ち込む依存は許可した第三者ライブラリと pin した Go に限り、いずれも固定する"
specification: |
  Everything the CLI brings in SHALL be confined to what has been explicitly allowed, and
  each SHALL be pinned, in contrast to the engine's stdlib-only constraint. A minimal set
  of third-party dependencies SHALL be allowed — cobra, for subcommands and help — while
  placement itself SHALL be done by importing the engine. Those dependencies SHALL be
  fixed by building with `buildGoModule` and a vendorHash string. The Go toolchain SHALL
  be pinned to the go of nixpkgs, and the `toolchain` directive SHALL NOT be used, so that
  the build does not fetch a toolchain of its own and stays reproducible under Nix.
specification_ja: |
  CLI が持ち込むものは明示的に許可したものだけに閉じ、いずれも固定しなければならない
  （engine の stdlib-only 制約とは対照的に扱う）。第三者依存は最小限だけ許可する
  （サブコマンド / help のための cobra）。配置そのものは engine を import して行う。
  それらの依存は `buildGoModule` と vendorHash 文字列でビルドすることによって固定する。
  Go ツールチェーンは nixpkgs の go に pin しなければならず、`toolchain` ディレクティブを
  使ってはならない（ビルドが独自にツールチェーンを取得せず Nix の下で再現可能に保つため）。
---
# REQ-637599dc: CLI が持ち込む依存は許可した第三者ライブラリと pin した Go に限り、いずれも固定する

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
