---
id: "REQ-637599dc-a1ec-4af5-9e97-e882c7df56d0"
type: requirement
name: "CLI が持ち込む依存は許可した第三者ライブラリと pin した Go に限り、いずれも固定する"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  Everything the CLI brings in SHALL be confined to what has been explicitly allowed, and
  each SHALL be pinned, in contrast to the engine's stdlib-only constraint. A minimal set
  of third-party dependencies SHALL be allowed — cobra, for subcommands and help — while
  placement itself is done by importing the engine. Those dependencies SHALL be
  fixed by building with `buildGoModule` and a vendorHash string. The Go toolchain SHALL
  be pinned to the go of nixpkgs, and the `toolchain` directive SHALL NOT be used, so that
  the build does not fetch a toolchain of its own and stays reproducible under Nix.
specification_ja: |
  CLI が持ち込むものは明示的に許可したものだけに閉じ、いずれも固定しなければならない
  （engine の stdlib-only 制約とは対照的に扱う）。第三者依存は最小限だけ許可しなければ
  ならない（サブコマンド / help のための cobra）。配置そのものは engine を import して行う。
  それらの依存は `buildGoModule` と vendorHash 文字列でビルドすることによって固定しなければ
  ならない。
  Go ツールチェーンは nixpkgs の go に pin しなければならず、`toolchain` ディレクティブを
  使ってはならない（ビルドが独自にツールチェーンを取得せず Nix の下で再現可能に保つため）。
---
# REQ-637599dc: CLI が持ち込む依存は許可した第三者ライブラリと pin した Go に限り、いずれも固定する

## 仕様

| コンポーネント | 依存 |
|---|---|
| `cmd/nput`（CLI = `packages.nput`）| 配置エンジンを import。最小依存を許可（**cobra** = サブコマンド / help）。entrypoint 発見と `nix`（build / eval）オーケストレーションを担う。`buildGoModule` + **vendorHash 文字列**でビルド。Go は nixpkgs の go に pin し `toolchain` ディレクティブ不使用 |

> **上は原文の写しで、規範は frontmatter が正**。CLI が engine を import して配置を駆動
> すること、および entrypoint 発見と `nix` のオーケストレーションを担う役割分担そのものは
> REQ-f4d7d4ab、entrypoint の探索順は REQ-1cc080f6、実行フローにおける eval / build の
> 順序は REQ-60c6b7ea、engine 側の stdlib-only 制約は REQ-b74a118a の担当。規範文が
> engine の import に触れるのは、第三者依存を最小限に抑えられる前提が「配置を自前で
> 持たず engine に委ねること」にあるためで、その役割分担自体を規範化するものではない。

## 出典

`docs/spec.md`「依存関係」節の表の `cmd/nput` 行。

決定の実体は ADR-0011「engine / CLI の技術スタックを確定する」で、CLI に cobra を許可し
`buildGoModule` + vendorHash・Go の pin でビルドを固定することを定めている。CLI を一次 UX と
する位置づけは ADR-0007「汎用 nput CLI を一次 UX に昇格し、entrypoint 発見＋root 明示モデルへ
移行する」が定めるが、同 ADR は cobra / vendorHash / Go の pin のいずれにも触れておらず、
この item の規範を決めていないため、側面の根拠として `justifies` は張らない（位置づけそのものの
帰属は REQ-14f0aec9 / REQ-f4d7d4ab が担当する）。
