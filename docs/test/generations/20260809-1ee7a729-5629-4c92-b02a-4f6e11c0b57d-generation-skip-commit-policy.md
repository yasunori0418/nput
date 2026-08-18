---
id: "TC-1ee7a729-5629-4c92-b02a-4f6e11c0b57d"
type: test_condition
name: "世代スキップは project mode で derivation 同一のときだけ発火し、それ以外は新世代を積む"
mitigates:
  - "RISK-67e96e59-005f-439d-a230-834b0228801c"
---
# TC-1ee7a729-5629-4c92-b02a-4f6e11c0b57d: 世代スキップは project mode で derivation 同一のときだけ発火し、それ以外は新世代を積む

## テスト条件

世代を積むか積まないかの判定を、mode と derivation の同一性の掛け合わせで検証する。
commit の呼び出し記録を観測点にして「新世代が積まれたか」を判定する。

| mode | link farm | 期待 |
|---|---|---|
| project | 前回と同一 | 世代を積まない（`shellHook` の高頻度実行で無限増殖させない）|
| project | 前回と異なる | 新世代を積む |
| home | 前回と同一 | 新世代を積む（世代スキップは project 限定）|

スキップ時も完全な no-op にはならず、`lstat` によるドリフト検査だけは走ること
（検査の中身は TC-765858b6-d596-4428-b4da-5d432944a714 が担当）。逆にスキップしない run では、修復ではなく通常の
plan / place 経路を通ること。

## 対応する CASE

CASE-2de3a3d8-3b67-485a-8ddf-a97dd182af23（`internal/engine/drift_test.go`）。
