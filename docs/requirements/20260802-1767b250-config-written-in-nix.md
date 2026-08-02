---
id: "REQ-1767b250-b475-4276-a551-20dc79e75a30"
type: requirement
name: "config は Nix で書き nix build で評価する"
specification: |
  Configuration SHALL be written in Nix and evaluated by `nix build`. What the CLI
  discovers SHALL be the entrypoint *file*, not the configuration content.
specification_ja: |
  config は Nix で記述し、`nix build` によって評価されなければならない。CLI が発見する
  のは entrypoint *ファイル*であって config の内容ではない。
---
# REQ-1767b250: config は Nix で書き nix build で評価する

## 仕様

config は **Nix で書き `nix build` で評価**される。CLI が発見するのは entrypoint
*ファイル*であって config 内容ではない。

評価は全て Nix に委ねる。どのファイルを entrypoint として発見するか（`flake.nix` /
`shell.nix` / `default.nix`・CWD 既定・`-f` 上書き）は `docs/spec.md`「CLI 仕様」→
「entrypoint の発見」節の担当で、本 item は「発見の対象が config 内容ではなく
ファイルである」という一点のみを規定する。

## 出典

`docs/spec.md`「アーキテクチャ概要」。
