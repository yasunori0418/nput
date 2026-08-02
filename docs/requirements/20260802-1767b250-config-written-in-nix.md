---
id: "REQ-1767b250-b475-4276-a551-20dc79e75a30"
type: requirement
name: "config は Nix で書き nix build で評価する"
specification: |
  Configuration SHALL be written in Nix and evaluated by `nix build`. What the CLI
  discovers SHALL be the entrypoint *file* (`flake.nix` / `shell.nix` / `default.nix`),
  not the configuration content. The CLI MUST NOT parse configuration in a format of
  its own.
specification_ja: |
  config は Nix で記述し、`nix build` によって評価されなければならない。CLI が発見する
  のは entrypoint *ファイル*（`flake.nix` / `shell.nix` / `default.nix`）であって
  config の内容ではない。CLI は config を独自形式でパースしてはならない。
---
# REQ-1767b250: config は Nix で書き nix build で評価する

## 仕様

config は **Nix で書き `nix build` で評価**される。CLI が発見するのは entrypoint
*ファイル*であって config 内容ではない。

CLI に独自の設定ファイル形式（YAML / TOML 等）を持たせず、評価は全て Nix に委ねる。
これにより Nix の型検査・モジュールシステム・入力固定がそのまま config の性質になる。

## 出典

`docs/spec.md`「アーキテクチャ概要」。
