---
id: "TC-ed4992c0-8513-4383-be0a-e45acbbc229f"
type: test_condition
name: "--backup の退避が opt-in で発動し、commit 後は残り、途中失敗では戻る"
mitigates:
  - "RISK-3719d298-8175-4501-95d5-75f3a29568fe"
---
# TC-ed4992c0: --backup の退避が opt-in で発動し、commit 後は残り、途中失敗では戻る

## テスト条件

`apply --backup` の rename 退避が、発動条件・退避先の命名・run 終了後の扱いのそれぞれで
契約どおりに振る舞うことを検証する。

**発動と非発動** — 配置を塞ぐ記録外の通常ファイルは `<target>.nput-backup` へ退避されて
配置が進み、退避した target が `Result.BackedUp` に載る。`--backup` 無しの既定では同じ
配置が conflict で停止する（opt-in であることの裏取り）。`--backup=<suffix>` で退避先の
suffix が変わる。

**段の位置** — 世代スキップ（drift 修復）経路でも Backup 段は到達し、通常 apply と同じく
退避 + 配置を行う。PreRemove と違い「derivation 不変なら発火しない」という不変条件を
持たないため（記録外実体は manifest の変化と無関係に現れる → REQ-9b0046e0）。

**`--dryrun --backup`** — 塞いでいる target を conflict ではなく「退避 + 配置予定」として
報告し、exit 2 相当の conflict をゼロにする。FS には一切触れない。

**run 終了後の扱い** — commit 成功後も退避物は残る（`--recopy` の退避と異なり
discardJournal の掃除対象ではない。ユーザーのバックアップとして無期限に保持する）。
逆に同一バッチの後続配置が失敗したときは退避物が rename で戻り、target が pre-apply の
内容へ復元され、退避先パスは消える。

**巻き添えの誤警告が出ないこと** — 実 dir target の migration が foreign leaf 混在で
失敗し `--backup` がディレクトリごと退避したとき、前世代が自分の stale symlink として
記録している兄弟 leaf は removeStale から「planning 後に drift した」と報告されない
（親ごと 1 回の rename で去ったのは想定内の挙動であって drift ではない）。

上位の規範は TP-e7c25263（`internal/engine/` を実 FS の tmpdir で駆動する統合レベル）。
TP-deb05610 の射程は 4 つの不変条件に閉じており、退避ポリシーそのものはそちらの担当では
ない（巻き戻し対象としての退避は TC-3b02ab58 側で TP-deb05610 の下に立つ）。

## 対応する CASE

CASE-ed4b32fd（`internal/engine/backup_test.go`）。
