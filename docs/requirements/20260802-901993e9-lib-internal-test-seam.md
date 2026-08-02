---
id: "REQ-901993e9-771c-480a-ba0d-ca4be042e206"
type: requirement
name: "lib.__internal は private helper のテスト seam として公開する"
specification: |
  `lib.__internal` SHALL expose the private helpers (`escapesBase` / `pathChecks` /
  `anchorName` / `resolveEntry` / `farmEntries`) as a test seam for nix-unit and namaka.
  It SHALL NOT be a stable API and MUST NOT be covered by backward-compatibility
  guarantees.
specification_ja: |
  `lib.__internal` は private helper（`escapesBase` / `pathChecks` / `anchorName` /
  `resolveEntry` / `farmEntries`）を nix-unit / namaka のテスト seam として公開する
  内部 API でなければならない。これは安定 API ではなく、後方互換の保証対象に含めない。
issues:
  - "#71"
---
# REQ-901993e9: lib.__internal は private helper のテスト seam として公開する

## 仕様

`lib.__internal` は private helper（`escapesBase` / `pathChecks` / `anchorName` /
`resolveEntry` / `farmEntries`）を nix-unit / namaka のテスト seam として公開する
内部 API で、安定 API ではない。

- 目的は評価テストから内部関数を直接叩けるようにすること。
- 安定 API ではないため、後方互換の保証対象に含めない。

## 出典

`docs/spec.md`「lib API」。

> **本 item の根拠は ADR ではなく Issue #71**: 原文が根拠として挙げるのは `→ #71` のみで、
> `lib.__internal` を決めた ADR は存在しない。frontmatter の `issues` がこれを受けている。
> `justifies` を持たないのは張り漏れではなく、根拠が ADR 以外にあることによる。

> **`derives_from` を持たないのも意図的**（→ Issue #211）。この要求は評価テストから内部
> 関数を叩くための seam であり、ユーザーの使われ方（use_case）から導かれない。テストの
> ための要求という点で REQ-6419e4b0・REQ-2381d93a・REQ-690f2730 と同類で、いずれも
> use_case を持たない orphan として残す。
