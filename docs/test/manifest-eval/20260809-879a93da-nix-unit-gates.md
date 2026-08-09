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

拒否されるべき入力が `ThrownError` になり、メッセージが**拒否されたフィールド（または理由の
語）を名指しする**ことをアサートする（`expectedError.type` + `expectedError.msg` の部分一致）。

`msg` は部分一致なので、**フィールド名を本文に含む他ゲートのメッセージとも一致しうる**。
`"target"` を期待する 2 件（絶対パス / `..` 脱出）は互いに区別されないだけでなく、
`copy` × out-of-store と subpath の各メッセージが `(target: …)` を含むため、それらが誤って
発火した場合にも通る。ゲートの取り違えまで検出する粒度は持っていない。

- `systemRoot` を root に指定 → 未実装（`"system mode"`）。**このアサートは撤回済み決定の
  残骸**（下の注記を参照）
- `method = "copy"` と out-of-store marker の併用 → `"out-of-store"`
- 絶対パスの target（`/etc/x`）→ `"target"`
- `..` で root の外へ出る target（`../../etc/x`）→ `"target"`
- `..` で src の外へ出る subpath（`../escape`）→ `"subpath"`
- 別キーから同一 target を明示指定した衝突 → `"same target"`
- 未知キー（旧名 `source`）→ strict submodule で `"source"`
- 素の文字列 src → `"src"`
- `lib/types.nix` の `entriesType` を `evalModules` で直接評価し、未知キー `bogus` が
  モジュール経路（`modules/common.nix` が共有する型）でも eval エラーになること

配置元は TP-d3d06fe4 の fake flake-input double イディオムに従う。

## 既知の乖離: systemRoot の拒否テスト

`testSystemRootUnimplemented` は ADR-0013 §5（`root = systemRoot` は eval 時エラー）に基づく
が、この決定は **ADR-0036 が撤回済み**で、現行の規範は `rootKind = "system"` を正規の値と
する（REQ-37b56673。REQ-16faf428 / REQ-c5dfcae6 も同じ理由でこの拒否を規範から外している）。
ADR-0036 は「nix-unit / namaka テスト更新（拒否テスト → 変換テスト）」を影響範囲に挙げており、
本アサートと `lib/manifest.nix` の throwIf はその更新が未了のまま残ったもの。

したがって上の列挙に残しているのは**実装の現状を写すため**であって、規範を表すためではない。
TC-e7ff0e6d はこれを条件に含めない。テスト側の更新は本 item の対象外（`tests/**` は L1 の
境界外）で、別途 issue として起こす。

## 出典

`tests/nix-unit/gates.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。各ゲートの
設計判断は ADR-0008 / ADR-0010 / ADR-0013 / ADR-0019 / ADR-0024 が持つ。
