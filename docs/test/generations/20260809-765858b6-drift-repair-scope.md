---
id: "TC-765858b6-d596-4428-b4da-5d432944a714"
type: test_condition
name: "lstat ドリフト修復が壊れた entry だけを直し、ユーザーの編集には触れない"
mitigates:
  - "RISK-67e96e59-005f-439d-a230-834b0228801c"
---
# TC-765858b6: lstat ドリフト修復が壊れた entry だけを直し、ユーザーの編集には触れない

## テスト条件

世代スキップ時に走る `lstat` ベースのドリフト修復について、「何を直し、何を直さないか」を
検証する。同一の link farm を 2 回 apply して `generationUnchanged` を成立させ、2 回目の
run が修復だけを行う状況を作る。

**直すもの**

- 消された symlink を再張りする
- 別物へ張り替えられた foreign symlink を記録どおりへ戻す。この修復は warning を伴う
  （他ツールとの競合をユーザーへ知らせる）
- 不在になった copy target を place-once で再マテリアライズする

**直さないもの**

- 記録どおりの entry（ドリフト無し）— 何もせず、warning も出さない
- 内容が編集された copy target — 存在するが異なる場合は触らない（home mode の place-once
  と振る舞いを揃える）。`--recopy` を明示した run では逆に上書きする

修復対象は symlink と copy の両方であること（片方だけの修復に縮退していないこと）を
条件に含む。

上位の規範は TP-e7c25263（`internal/engine/` を実 FS の tmpdir で駆動する統合レベル）。

## 対応する CASE

CASE-2de3a3d8（`internal/engine/drift_test.go`）。`--backup` が同じ修復経路で発火する
側は atomicity の TC-ed4992c0 が担当する。
