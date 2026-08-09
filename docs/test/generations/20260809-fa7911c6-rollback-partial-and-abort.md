---
id: "TC-fa7911c6-7347-47ff-b121-bec53562c063"
type: test_condition
name: "rollback は途中失敗を巻き戻し、ポインタ移動の失敗は巻き戻さず、前提不成立では停止する"
mitigates:
  - "RISK-fbf029f6-866c-4a08-a4eb-f09e3c7e907e"
---
# TC-fa7911c6: rollback は途中失敗を巻き戻し、ポインタ移動の失敗は巻き戻さず、前提不成立では停止する

## テスト条件

TC-36ea3609 の正常系に対し、`Rollback` の失敗経路を検証する。

**途中失敗の巻き戻し** — 祖先 migration（PreRemove の unlink + それによって可能になった
子の配置）が成功したあとで、同じバッチの無関係な配置が失敗したとき、migration ごと
巻き戻る。rollback も apply と同じ undo journal に配線されている必要があり、half-migrated
のまま「次の冪等な再実行が収束する」に頼ってはならない（→ ADR-0044、issue #168）。

**ポインタ移動の失敗は巻き戻さない** — `SwitchGeneration` が失敗した時点では
PreRemove / place / removeStale の全 FS 書き込みが成功している。ここは apply の commit
失敗と同じ非対称性で、巻き戻さない。部分の `RollbackResult` は「遷移は起きなかった（From == To ==
現世代）」「ポインタは動いていない」「失敗は entry スコープではない」を表し、既に置いた
配置・除去済みの stale はそのまま残る。

**前提不成立での停止** — 次の 3 つはいずれもエラーで停止する。

- 現世代が最古で前世代が無い
- profile がそもそも無い（一度も apply していない）
- 祖先の PreRemove が drift で失敗した（この場合はポインタ移動へ進まず、現状の配置を
  触らずに残す）

**conflict の全件報告** — 戻り先の target が複数 foreign な実体で塞がれているとき、
apply と同じく全 conflict を stderr へ列挙してから、件数を持つ 1 本の集約エラーを返す
（→ issue #176）。1 件目で打ち切らない。

## 対応する CASE

CASE-364ebb9d（`internal/engine/generations_test.go`）。conflict / 部分失敗時の結果
の構造は CASE-2008a909 も隣接して検証する。
