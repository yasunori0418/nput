---
id: "REQ-cbd61281-64b0-4487-a4b7-ce76e70dc4f9"
type: requirement
name: "init のテンプレート参照はバイナリにハードコードした固定 flake ref とする"
specification: |
  The template reference of `nput init` SHALL be the fixed flake ref
  `github:yasunori0418/nput`, hardcoded into the binary, so that it works without
  depending on registry registration. Since `init` is for bootstrapping something new, a
  divergence between the CLI version and the template version (which always refers to the
  latest main) SHALL be accepted.
specification_ja: |
  `nput init` のテンプレート参照は、バイナリにハードコードした固定 flake ref
  `github:yasunori0418/nput` でなければならない（registry 登録に依存せず動くように）。
  `init` は新規 bootstrap 用途であるため、CLI 版と template 版のズレ（常に latest main
  参照）は許容する。
---
# REQ-cbd61281: init のテンプレート参照はバイナリにハードコードした固定 flake ref とする

## 仕様

**テンプレート参照はバイナリにハードコードした固定 flake ref
`github:yasunori0418/nput`**。registry 登録に依存せず動く。`init` は新規 bootstrap 用途で、
CLI 版と template 版のズレ（常に latest main 参照）は許容する。apply 時の
`schemaVersion` 整合は project mode の devShell 同梱 pin が担う（REQ-14f0aec9）。

## 出典

`docs/spec.md`「CLI 仕様」→「`nput init`（テンプレート展開）」の箇条書き第 3 項。

決定の実体は ADR-0025「nput init 固定 ref」。
