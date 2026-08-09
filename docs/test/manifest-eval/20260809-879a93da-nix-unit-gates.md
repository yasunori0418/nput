---
id: "CASE-879a93da-d22f-4397-84da-3544f8249af1"
type: test_case
name: "nix-unit: gates.nix — 評価時に throw する検査ゲート群"
covers:
  - "TC-e7ff0e6d-32d7-4ed6-8c2f-449dba34b2f6"
---
# CASE-879a93da: nix-unit gates.nix

## 対象

`tests/nix-unit/gates.nix`

## 検証内容

拒否されるべき入力が `ThrownError` になり、メッセージが理由を名指しすることをアサートする
（`expectedError.type` + `expectedError.msg` の部分一致）。

- `systemRoot` を root に指定 → 未実装（`"system mode"`）
- `method = "copy"` と out-of-store marker の併用 → `"out-of-store"`
- 絶対パスの target（`/etc/x`）→ `"target"`
- `..` で root の外へ出る target（`../../etc/x`）→ `"target"`
- `..` で src の外へ出る subpath（`../escape`）→ `"subpath"`
- 別キーから同一 target を明示指定した衝突 → `"same target"`
- 未知キー（旧名 `source`）→ strict submodule で `"source"`
- 素の文字列 src → `"src"`
- `lib/types.nix` の `entriesType` を `evalModules` で直接評価し、未知キー `bogus` が
  モジュール経路（`modules/common.nix` が共有する型）でも eval エラーになること

配置元は固定 `outPath` を持つ fake flake-input。
