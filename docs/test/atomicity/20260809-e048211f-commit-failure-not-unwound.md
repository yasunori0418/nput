---
id: "TC-e048211f-0844-4446-9957-fb444c1da4e5"
type: test_condition
name: "commit 段の失敗では配置を巻き戻さず、退避ファイルも掃除しない"
mitigates:
  - "RISK-68e810c5-4e68-4b25-9bc0-6b2613022b49"
---
# TC-e048211f: commit 段の失敗では配置を巻き戻さず、退避ファイルも掃除しない

## テスト条件

TC-3b02ab58 の裏返し。commit（`nix-env --set`）でのみ失敗させ、**巻き戻しが起きない**
ことを検証する。commit へ到達した時点でこの run の FS 書き込みは全て成功しており、FS は
既に整合している。ここで unwind すればその整合状態を壊すため、巻き戻さないのが正しい
（→ ADR-0044 §2）。世代が進まないだけで、冪等な再実行が収束する。

条件は次の 3 点。

- 配置済み symlink / copy が commit 失敗後も無傷で残る
- `--recopy` の rename 退避ファイルが残る。discardJournal は commit 成功後にしか走らず、
  unwind も起きないため、どちらの経路でも触られない。新しい copy が live のまま、
  pre-apply の内容は退避ファイルとして手元に残る
- 世代は進んでいない

commit 段は `Commit` 関数を差し替えて失敗させる。実 FS 条件で誘発すべき対象は FS 操作の
故障であり、commit の故障は nix 側の失敗を表す注入点として double で足りる。

## 対応する CASE

CASE-154af597（`internal/engine/undo_journal_test.go`）。commit 失敗時の部分 `Result`
の中身（`Unwound` が false であることを含む）は generations 側の
CASE-2008a909 が隣接して検証する。
