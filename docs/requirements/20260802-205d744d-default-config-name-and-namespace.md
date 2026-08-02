---
id: "REQ-205d744d-5a53-4511-bc09-892ba01d4e6f"
type: requirement
derives_from:
  - "UC-1c280dce-7c72-44c0-95ea-d06344f62a47"
name: "config 名 default を慣例の解決先とし専用 nput 名前空間で packages を汚さない"
specification: |
  The config name `default` SHALL be a special name following the flake `default`
  convention, and SHALL be the resolution target used by `nput apply` when the name is
  omitted. Named manifests SHALL live in a dedicated `nput` namespace rather than in
  `packages`, so that a manifest does not appear as an ordinary package in
  `nix flake show` / `nix build`.
specification_ja: |
  config 名 `default` は flake の `default` 慣例に倣う特別な名前とし、`nput apply`
  （name 省略）が解決先に使うものでなければならない。named manifest は `packages` では
  なく専用の `nput` 名前空間に置き、manifest が通常パッケージとして `nix flake show` /
  `nix build` に混ざらないようにしなければならない。
---
# REQ-205d744d: config 名 default を慣例の解決先とし専用 nput 名前空間で packages を汚さない

## 仕様

`<name>` = `default` は flake の `default` 慣例に倣う特別な名前で、`nput apply`
（name 省略）が解決先に使う。専用 `nput` 名前空間を使い `packages` を汚さない
（manifest が通常パッケージとして `nix flake show` / `nix build` に混ざらない）。

name 省略時に `default` が未定義ならエラーになることは `nput apply` の config 選択の
規範であり、REQ-c2d44626 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「アドレッシング」節の末尾段落。

決定の実体は ADR-0007 §4（専用 `nput` 名前空間・`default` 慣例）と、棄却した代替案
「アドレッシングで `packages.<system>.<name>` を流用」の項。
