---
id: "CASE-2de3a3d8-3b67-485a-8ddf-a97dd182af23"
type: test_case
name: "drift_test.go — 世代スキップの発火条件と lstat ドリフト修復の対象範囲"
covers:
  - "TC-765858b6-d596-4428-b4da-5d432944a714"
  - "TC-1ee7a729-5629-4c92-b02a-4f6e11c0b57d"
---
# CASE-2de3a3d8: drift_test.go — 世代スキップの発火条件と lstat ドリフト修復の対象範囲

## 対象

`internal/engine/drift_test.go`

実 FS（tmpdir）を使い nix は使わない統合テスト。`fakeCommit` が profile リンクを直接
link farm へ張るため、同じ link farm を 2 回 apply すると `generationUnchanged` が成立し
（project mode 限定）、新世代を積まずに `lstat` 修復だけが走る状態を作れる。共通ヘルパー
`applyOnce` が project mode（roothash キー）の 1 回分の apply を回す。

## 検証内容

**発火条件**（TC-1ee7a729）

- project mode で同一 link farm を 2 回 apply → 世代を積まない
- link farm が変われば新世代を積む
- home mode では同一 link farm でも新世代を積む（世代スキップは project 限定）

**修復の対象範囲**（TC-765858b6）

- ドリフト無し（記録どおり）→ 何もせず warning も出さない
- 消された symlink → 再張りする
- foreign symlink へ張り替えられた entry → 記録どおりへ戻し、warning を出す
- 不在になった copy target → place-once で再マテリアライズする
- 内容が編集された copy target → 触らない
- `--recopy` を明示した run → 編集済み copy を上書きする
