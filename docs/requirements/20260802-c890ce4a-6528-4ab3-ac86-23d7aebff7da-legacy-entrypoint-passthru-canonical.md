---
id: "REQ-c890ce4a-6528-4ab3-ac86-23d7aebff7da"
type: requirement
name: "legacy entrypoint は mkShell passthru 形を canonical とし CLI の attr path を分岐させない"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  For `shell.nix` / `default.nix`, the canonical form SHALL be a passthru form that
  coexists with the nixpkgs `mkShell`: `shell.nix` SHALL return a plain `mkShell`
  derivation and carry the named manifest at `passthru.nput.<name>`, so that it is
  compatible with bundling the CLI into the `project` template devShell. Because
  `passthru` is merged into the attrset of the host derivation, the attr path the CLI
  invokes (`nput.<name>`) SHALL be identical to that of the top-level attrset form, and
  the CLI SHALL NOT branch its implementation between the two. A bare `nix-shell` (without
  `-A`) SHALL keep working for both forms. Handling multiple manifests at once SHALL be
  served by the existing batched eval of `apply --all`, and no helper that composes
  manifests via `inputsFrom` of `mkShell` SHALL be provided, because it would conflict
  with the "one profile = one config" atomicity.
specification_ja: |
  `shell.nix` / `default.nix` は nixpkgs `mkShell` と共存できる passthru 形を canonical と
  しなければならない。`shell.nix` は素の `mkShell` derivation を返し、
  `passthru.nput.<name>` に named manifest を載せなければならない（`project` template の
  devShell 同梱と両立する）。`passthru` はホスト derivation の attrset にマージされるため、
  CLI が叩く attr path（`nput.<name>`）はトップレベル attrset 形と同一でなければならず、
  CLI は実装分岐を持ってはならない。素の `nix-shell`（`-A` 不要）はどちらの形でも壊れては
  ならない。複数 manifest の一括処理は既存の `apply --all` の一括 eval でそのまま充足させ
  なければならず、`mkShell` の `inputsFrom` で manifest を合成するヘルパは持ってはならない
  （1 profile = 1 config の atomic 性と衝突するため）。
---
# REQ-c890ce4a-6528-4ab3-ac86-23d7aebff7da: legacy entrypoint は mkShell passthru 形を canonical とし CLI の attr path を分岐させない

## 仕様

`shell.nix` / `default.nix` は **nixpkgs `mkShell` と共存できる passthru 形**を canonical と
する。`shell.nix` は素の `mkShell` derivation を返し、`passthru.nput.<name>` に named
manifest を載せる（`project` template の devShell 同梱と両立する）。`passthru` はホスト
derivation の attrset にマージされるため、**CLI が叩く attr path（`nput.<name>`）は
トップレベル attrset 形と同一**で実装分岐を持たない。素の `nix-shell`（`-A` 不要）は
どちらの形でも壊れない。

```nix
# shell.nix（canonical: passthru 形）
{ pkgs ? import <nixpkgs> {}, nput ? import (fetchTarball "https://github.com/yasunori0418/nput/archive/....tar.gz") }:
pkgs.mkShell {
  packages = [ ];
  shellHook = "nput apply skills --no-wait";
  passthru.nput = {
    skills = nput.lib.mkManifest { inherit pkgs; root = nput.lib.projectRoot; entries = { }; };
  };
}
```

複数 manifest の一括処理は既存の `apply --all` の一括 eval
（`nix eval -f <ep> nput --apply … --json`）でそのまま充足し、`mkShell` の `inputsFrom` で
manifest を合成するヘルパは持たない（1 profile = 1 config の atomic 性と衝突するため）。

> **上は原文の写しで、規範は frontmatter が正**。原文が併記する次の点は本 item の
> 規範ではない。
>
> - legacy entrypoint の addressing がフラットな `nput.<name>` になること → REQ-496b1a07-5b74-416b-9e5f-3952b4c03737
> - `apply --all` の一括 eval そのもの → REQ-535b811d-dfc5-4eac-92db-737e70eb5415
> - 1 profile = 1 config の atomic 性（→ ADR-0002）→ REQ-1be4d678-959c-44d7-a346-44bfd95af56e

## 出典

`docs/spec.md`「CLI 仕様」→「アドレッシング」の blockquote「legacy entrypoint の
canonical 形」。

決定の実体は ADR-0032「legacy entrypoint（shell.nix / default.nix）は mkShell passthru を
canonical とし、addressing を `nix build -f` に統一する」。
