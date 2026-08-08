---
id: "TP-403c55c7-d996-4951-8e6b-c3a7dddd387c"
type: test_plan
name: "lib.__internal は private helper のテスト seam として公開する"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  `lib.__internal` SHALL expose the private helpers (`escapesBase` / `pathChecks` /
  `anchorName` / `resolveEntry` / `farmEntries`) as a test seam for nix-unit and namaka.
  It SHALL NOT be a stable API and SHALL NOT be covered by backward-compatibility
  guarantees.
specification_ja: |
  `lib.__internal` は private helper（`escapesBase` / `pathChecks` / `anchorName` /
  `resolveEntry` / `farmEntries`）を nix-unit / namaka のテスト seam として公開する
  内部 API でなければならない。これは安定 API であってはならず、後方互換の保証対象に
  含めてはならない。
issues:
  - "#71"
---
# TP-403c55c7: lib.__internal は private helper のテスト seam として公開する

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

> **本 item は requirement から test_plan へ移設した**（→ Issue #239。旧 ID は
> `REQ-901993e9`）。`lib.__internal` を公開する唯一の動機は評価テストから内部関数を叩ける
> ようにすること、すなわちテスト対象へのアクセス手段の確保であり、ユーザーの使われ方
> （use_case）から導かれるプロダクトの振る舞いではないため、use_case を親に持てず orphan に
> なっていた（当時の判断は Issue #211）。specification が SHALL NOT で安定性・後方互換の
> 保証除外を宣言している点も、API 契約ではなくテスト容易性の確保が支配的であることを示す。
> 同じくテストのための要求だった TP-229b69c0・TP-d3000054・TP-b7f1dc79（→ Issue #238）と
> 同類として、テスト計画の型を新設して solution 直下で受けることにしたため、`derives_from`
> は SOL-9fcd1d6e を指す。
