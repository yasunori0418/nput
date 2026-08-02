---
id: "REQ-6be1cbf1-6c6e-498b-8acb-7f4b80037169"
type: requirement
name: "nput init は nix flake init -t への透明なラッパーとしファイルを生成しない"
specification: |
  `nput init <template>` SHALL be a transparent wrapper around
  `nix flake init -t github:yasunori0418/nput#<template>`. It is the nix templates
  mechanism that creates the files, and nput itself SHALL NOT generate them, so that the
  "does not generate configuration" thesis is maintained. It SHALL inherit the
  conservative behaviour of `nix flake init` of not overwriting existing files. Two
  templates SHALL be provided: `standalone` and `project`.
specification_ja: |
  `nput init <template>` は `nix flake init -t github:yasunori0418/nput#<template>` への
  透明なラッパーでなければならない。ファイルを作るのは nix の templates 機構であり、
  nput 自身が generate してはならない（「設定を生成しない」thesis を維持するため）。
  `nix flake init` の「既存ファイルを上書きしない」保守性を継承する。テンプレートは
  `standalone` と `project` の 2 つを提供する。
---
# REQ-6be1cbf1: nput init は nix flake init -t への透明なラッパーとしファイルを生成しない

## 仕様

```bash
nput init standalone   # nix flake init -t github:yasunori0418/nput#standalone のラッパー
nput init project      # nix flake init -t github:yasunori0418/nput#project のラッパー（devShell shellHook 配線 + .gitignore ガイド入り）
```

`nix flake init -t github:yasunori0418/nput#<template>` への**透明なラッパー**。ファイルを
作るのは nix の templates 機構であり nput 自身は generate しない（「設定を生成しない」
thesis を維持）。`nix flake init` の「既存ファイルを上書きしない」保守性を継承する。

固定 flake ref は REQ-cbd61281、template の中身は REQ-196ddabf、`--json` 時の出力は
REQ-fa181aa6 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「`nput init`（テンプレート展開）」のコードブロックと
箇条書き第 1・2 項。

決定の実体は ADR-0007 §6「`nput init` でテンプレートを展開する」。
