---
id: "TC-1d19aebc-e1e6-4d6f-9440-3efbd69b18a8"
type: test_condition
name: "Result のインベントリ・世代観測・到達状態・構造化 warning が実態を表す"
mitigates:
  - "RISK-cdcc6faf-9164-409b-b584-2921fa036d10"
---
# TC-1d19aebc: Result のインベントリ・世代観測・到達状態・構造化 warning が実態を表す

## テスト条件

`Apply` / `Reset` / `Rollback` が返す `Result` の各フィールドが、その run で実際に
起きたことと一致することを検証する（→ issue #130。niface envelope の素材）。commit は
`nix-env --set` の世代簿記を模した double へ差し替え、世代番号を読み戻せる形にする。

**インベントリと世代観測** — `Entries` が manifest の全 entry を持つ。世代観測は
初回 apply（nil → 1）・2 回目（1 → 2）・`--dryrun`（before == after でポインタ不動）を
覆う。観測できない状況（profile リンクが世代リンクでない）では before / after ともに
nil であり、0 や現在値で埋めない。

**失敗時の到達状態の分割** — 配置の途中で失敗した run では、完了済みの操作リストは
「完了した記録」として残り、失敗した entry は `FailedTarget` に入って `Placed` には
入らない。以降に予定されていた target は `Unreached` に入る。部分 `Result` がエラーと
並んで返り、`Unwound` が journal の巻き戻しの有無を表す。`--recopy` では、place-once の
分類で `CopyAction` が生まれなかった copy entry も `Unreached` に載る（recopy の実行
対象は plan ではなく manifest であるため）。

**commit 失敗の形** — 全 FS 操作が成功しているため到達状態のフィールドは空
（entry スコープの失敗ではない）、`Unwound` は false（→ ADR-0044 §2）、世代は動かない。

**conflict と部分失敗** — apply の conflict、reset の部分失敗、rollback の conflict の
いずれでも部分 `Result` が返る。`Apply` は張替えた dest と除去 entry も記録する。

**構造化 warning** — planner の entry warning が kind + target の構造で `Result` に
載り、テキストの stderr 出力とは別に消費できる。

## 対応する CASE

CASE-2008a909（`internal/engine/result_extensions_test.go`）。
