---
id: "REQ-14f0aec9-abae-4621-82f3-40536a1ad904"
type: requirement
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
name: "nput CLI は PATH 常駐の一次 UX で、project mode は devShell 同梱を canonical とする"
specification: |
  The `nput` CLI (`packages.nput`) SHALL be the primary UX, installed on PATH. Two
  installation forms SHALL be supported. In the standalone (home mode) form the CLI is
  installed globally (e.g. `nix profile install github:yasunori0418/nput`); the global CLI
  and the `nput.lib` pinned by the user flake are separate inputs and their `schemaVersion`
  MAY skew, so aligning the CLI and the flake pin to the same input SHOULD be recommended,
  and bundling the CLI into a devShell SHALL NOT be required for this form. In the project
  mode (canonical) form the pinned `nput` SHALL be bundled into the devShell of
  `templates/project` (`packages = [ nput.packages.${system}.nput ]`), so that the CLI and
  `nput.lib` come from the same flake input and their `schemaVersion` agrees. How the
  engine reacts to a skewed `schemaVersion` is stated by the manifest schema
  specification and is not restated here.
specification_ja: |
  `nput` CLI（`packages.nput`）は PATH に常駐する一次 UX でなければならない。導入形は
  2 つとする。standalone（home mode）ではグローバルに導入し（例:
  `nix profile install github:yasunori0418/nput`）、グローバル CLI とユーザー flake が
  pin する `nput.lib` は別入力になるため `schemaVersion` が skew し得る。よって CLI と
  flake pin の `nput` を同一 input から揃えることを推奨し、この形では devShell 同梱を
  強制しない。project mode（canonical）では `templates/project` の devShell に pin 版
  `nput` を同梱し（`packages = [ nput.packages.${system}.nput ]`）、CLI と `nput.lib`
  が同一 flake 入力から来て `schemaVersion` が一致するようにしなければならない。
  skew した `schemaVersion` に engine がどう反応するかは manifest スキーマ仕様の担当で、
  本 item では規定しない。
---
# REQ-14f0aec9: nput CLI は PATH 常駐の一次 UX で、project mode は devShell 同梱を canonical とする

## 仕様

`nput` CLI（`packages.nput`）は PATH に常駐する一次 UX。

- **standalone（home mode）**: `nix profile install github:yasunori0418/nput` 等で
  グローバルに導入する。グローバル CLI とユーザー flake が pin する `nput.lib`
  （manifest 生成側）は別入力になり `schemaVersion` が skew し得る。skew を避けるには
  **CLI と flake pin の `nput` を同一 input から揃える**ことを推奨する。project mode の
  ような devShell 同梱は強制しない（PATH 常駐の利便を優先）。
- **project mode（canonical）**: `templates/project` の devShell に **pin 版 `nput` を同梱**する
  （`packages = [ nput.packages.${system}.nput ]`）。`nix develop` / direnv 入室時に
  flake.lock で固定した `nput` が PATH に載り、CLI と `nput.lib`（manifest
  `schemaVersion`）が同一 flake 入力から来て一致する。グローバル install 依存だと
  flake.lock の pin を CLI で破り version skew を招くため、project mode では devShell
  同梱を canonical とする。

> **上は原文の写しで、規範は frontmatter が正**。原文が併記する次の 2 点は本 item の
> 規範ではない。
>
> - engine が自身の対応版より新しい `schemaVersion` を拒否すること（→ ADR-0006）は
>   REQ-79ce0a09、MVP が v1 のみであることは REQ-250d936c の担当
> - devShell 同梱そのものの配線（`templates/project` の内容）→
>   「`nput init`」→「テンプレートの内容」節の担当（REQ-196ddabf）

## 出典

`docs/spec.md`「CLI 仕様（一次 UX）」の導入 2 項目。

決定の実体は ADR-0007 §3「汎用 nput CLI を一次 UX に昇格する」（PATH 常駐・
`packages.nput`）と ADR-0015「project mode の `nput` は devShell 同梱が canonical」。
