---
id: "CASE-2008a909-b45e-49e1-adf1-39191a8ddd95"
type: test_case
name: "result_extensions_test.go — インベントリ・世代観測・到達状態・構造化 warning"
covers:
  - "TC-1d19aebc-e1e6-4d6f-9440-3efbd69b18a8"
---
# CASE-2008a909: result_extensions_test.go — インベントリ・世代観測・到達状態・構造化 warning

## 対象

`internal/engine/result_extensions_test.go`

issue #130 が niface envelope（#131 / #132 が共有）のために足した `Result` 拡張の
テスト。`fakeGenCommit` が `nix-env --set` の世代簿記を模し、兄弟の `<profile>-<N>-link`
を作って profile リンクを（`nix-env` と同じく相対で）張り替えるため、世代番号を読み戻せる。
manifest は `rootKind=fixed`（root を明示的に渡し git に依存しない）。

## 検証内容

- **インベントリと世代観測** — `Entries` が全 entry を持つ。初回 apply（nil → 1）・
  2 回目（1 → 2）・`--dryrun`（before == after でポインタ不動）
- **観測不能時の nil** — profile リンクが世代リンクでない（テスト差し替えの commit）
  `Reset` は before / after ともに nil（→ niface の nil-able Generation）
- **失敗時の到達状態** — 配置途中の失敗で、完了済み操作リストは完了の記録として残り、
  失敗 entry は `FailedTarget`（`Placed` には入らない）、以降の予定 target は
  `Unreached`。部分 `Result` がエラーと並んで返り、`Unwound` が journal 巻き戻しを表す
- **`--recopy` の Unreached** — place-once の分類で `CopyAction` が生まれなかった copy
  entry も、`materializeCopies` 到達前に失敗した run では `Unreached` に載る
  （recopy の実行対象は plan ではなく manifest であるため）
- **commit 失敗の形** — インベントリは埋まり、到達状態フィールドは空（entry スコープの
  失敗ではない）、`Unwound` は false（→ ADR-0044 §2）、世代は nil のまま。配置済み
  symlink は残る
- **conflict / 部分失敗の部分 Result** — apply の conflict、reset の部分失敗、rollback の
  conflict のいずれも部分 `Result` を返す。`Apply` は張替えた dest と除去 entry も記録する
- **構造化 warning** — planner の entry warning が kind + target の構造で `Result` に
  載り、`Warnf` のテキスト出力は従来どおり流れる。`Reset` でも同じ形で露出する
