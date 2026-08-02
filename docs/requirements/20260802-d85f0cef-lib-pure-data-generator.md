---
id: "REQ-d85f0cef-0f1e-4897-a841-41b61a8dae51"
type: requirement
name: "lib は nixpkgs.lib のみに依存する純データ生成器である"
specification: |
  `lib` (`mkManifest` and the marker functions) SHALL be a pure data generator that
  depends only on nixpkgs.lib. It MUST NOT carry placement logic (filesystem operations,
  profile swap, stale removal), and MUST NOT introduce a dependency on home-manager,
  NixOS or nix-darwin.
---
# REQ-d85f0cef: lib は nixpkgs.lib のみに依存する純データ生成器である

## 仕様

`lib`（`mkManifest` / マーカー群）は nixpkgs.lib のみに依存する純データ生成器。
配置ロジックは持たない。

- 依存は nixpkgs.lib に限る。`lib.types` / `mkOption` / `evalModules` は nixpkgs.lib の
  コアなのでこの制約を満たす。
- 実際の配置（place / replace / remove・profile swap・stale 除去）は engine の責務で、
  lib は一切行わない。
- 統合層（home-manager モジュール等）へ依存しないため、lib 単体を任意の Nix 環境から
  取り込める。

## 出典

`docs/spec.md`「lib API」。
