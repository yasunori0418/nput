---
id: "CASE-364ebb9d-b9f8-4c79-8509-8bbf31e73698"
type: test_case
name: "generations_test.go — 世代一覧パース・rollback の再収束と失敗経路"
target: "internal/engine/generations_test.go"
covers:
  - "TC-746cb5b9-fe27-4d09-a51d-eb03d56a93a7"
  - "TC-36ea3609-d52e-42d4-975c-40fb89b23919"
  - "TC-fa7911c6-7347-47ff-b121-bec53562c063"
---
# CASE-364ebb9d: generations_test.go — 世代一覧パース・rollback の再収束と失敗経路

## 対象

`internal/engine/generations_test.go`

世代一覧のパースはテキストに対するユニットテスト、`Rollback` は tmpdir 上の実 FS へ
世代リンクと profile を手で組み立ててから駆動する統合テスト。
`ListGenerations` / `SwitchGeneration` は関数として差し替える。

`resolveRoot` を直接叩くテストは分岐網羅を 1 ファイルで追えるよう engine_test.go へ集約した
（集約先の集合は多くが純引数だが、cwd を入力に取る経路も含む。
→ CASE-31fdb776 / TC-b254a5a8、issue #328）。

## 検証内容

**世代一覧のパース**（TC-746cb5b9）

- 番号・日時・`(current)` マーカーの分解。日時に `(current)` を混入させない
- 空入力は 0 件でエラーにしない
- 世代番号が非数値の行はエラーにする

**rollback の再収束**（TC-36ea3609）

- gen1 = {a, b}・gen2（current）= {a, c} からの再収束。c の stale 除去・b の再配置・
  a の据え置き。`RollbackResult` の世代観測が From / To を映し、Entries が戻り先世代の全
  インベントリを持つ（→ issue #130）
- 祖先 symlink 世代（N）から per-file 世代（N-1）への復帰。plan の PreRemove が祖先
  symlink を除去してから子を実ディレクトリへ置き直す（→ ADR-0046、issue #173）

**rollback の失敗経路**（TC-fa7911c6）

- `SwitchGeneration` の失敗は unwind しない。部分の `RollbackResult` が From == To == 現世代を
  表し、既に行われた配置・stale 除去はそのまま残る（→ ADR-0044 §2）
- 祖先 migration 成功後の無関係な配置失敗で、migration ごと巻き戻る（→ ADR-0044、
  issue #168）
- 祖先の PreRemove が drift で失敗したときはポインタ移動へ進まず、配置を触らない
- 前世代が無い（最古世代）・profile が無い（未 apply）はエラーで停止する
- 戻り先の target が複数 foreign 実体で塞がれているとき、全 conflict を stderr へ
  列挙してから件数付きの集約エラーを返す（→ issue #176）
