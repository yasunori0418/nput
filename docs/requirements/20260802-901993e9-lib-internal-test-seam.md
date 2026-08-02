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
- 名前・シグネチャ・存在自体を予告なく変更してよい。ユーザーコードが依存することを
  想定しない。

## 出典

`docs/spec.md`「lib API」。
