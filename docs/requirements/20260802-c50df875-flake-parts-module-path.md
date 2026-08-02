---
id: "REQ-c50df875-2cb0-4e72-8a21-858359a11cae"
type: requirement
name: "flake-parts 経路は直書きと同一の derivation を生み CLI のアドレッシングを変えない"
derives_from:
  - "UC-d39c1994-f9a5-4860-80ba-f6e584adaf14"
specification: |
  A repository using flake-parts SHALL declare `perSystem.nput.<name> = mkManifest
  { inherit pkgs; ... }` on top of `imports = [ inputs.nput.flakeModules.default ]`.
  flake-parts SHALL transpose it to `flake.nput.<system>.<name>` and SHALL produce the
  same derivation as the directly written form, so that the CLI addressing is unchanged.
  In this path `pkgs` SHALL come from perSystem and be consistent with `packages.nput`,
  with no second resolution. For flake-parts users this form SHALL be canonical, while the
  directly written form SHALL be canonical for a plain flake and for `shell.nix` /
  `default.nix`. `mkManifest` SHALL remain the single public API in both forms, and the
  flakeModule option SHALL merely hold the derivation produced by `mkManifest`.
specification_ja: |
  flake-parts を使う repo は `imports = [ inputs.nput.flakeModules.default ]` の上で
  `perSystem.nput.<name> = mkManifest { inherit pkgs; ... }` を宣言しなければならない。
  flake-parts はこれを `flake.nput.<system>.<name>` へ transpose し、直書きと同一の
  derivation を生まなければならない（CLI のアドレッシングは不変）。この経路では `pkgs`
  が perSystem 由来になり `packages.nput` と一貫する（二重解決なし）。flake-parts
  利用者にはこの形が canonical で、直書きは plain flake と `shell.nix` /
  `default.nix` の canonical とする。`mkManifest` は両形で唯一の公開 API であり続け、
  flakeModule の option は `mkManifest` の derivation を格納するだけとする。
---
# REQ-c50df875: flake-parts 経路は直書きと同一の derivation を生み CLI のアドレッシングを変えない

## 仕様

flake-parts を使う repo は `imports = [ inputs.nput.flakeModules.default ]` の上で
`perSystem.nput.<name> = mkManifest { inherit pkgs; ... }` を宣言する。flake-parts が
`flake.nput.<system>.<name>` へ **transpose** し、直書きと**同一の derivation を生む**
（CLI のアドレッシングは不変）。`pkgs` が perSystem 由来になり `packages.nput` と
一貫する（二重解決なし）。

flake-parts 利用者にはこちらが canonical で、直書きは plain flake と `shell.nix` /
`default.nix` の canonical。`mkManifest` は両形で唯一の公開 API（flakeModule の option は
`mkManifest` の derivation を格納するだけ）。

## 出典

`docs/spec.md`「CLI 仕様」→「アドレッシング」の blockquote「flake-parts 経路」。

決定の実体は ADR-0029「nput output を flake-parts module 化し flakeModules.default を
公開する」。
