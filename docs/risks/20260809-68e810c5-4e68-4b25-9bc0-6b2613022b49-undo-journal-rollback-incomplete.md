---
id: "RISK-68e810c5-4e68-4b25-9bc0-6b2613022b49"
type: risk
name: "途中失敗した run の巻き戻しが不完全で、FS に部分適用の痕跡が残る"
threatens:
  - "REQ-5e75aabc-0e8f-4a6c-92bd-a712dc68a940"
  - "REQ-61856da1-8883-401e-ad57-9f326b96d400"
likelihood: high
impact: high
level: high
---
# RISK-68e810c5: 途中失敗した run の巻き戻しが不完全で、FS に部分適用の痕跡が残る

## リスク

apply / rollback が FS 変更バッチの途中で失敗したとき、その run が既に加えた変更が
巻き戻されずに残る。逆操作そのものが誤っている（新規 symlink を消し忘れる・張替えの
prevDest を復元しない・PreRemove が消した実ディレクトリを作り直さない・`--recopy` /
`--backup` の rename 退避を戻さない）場合と、逆操作は正しいが適用順序が誤っている
（LIFO でないため親ディレクトリを作り直す前に子 symlink を復元しようとする）場合の
どちらでも起こる。

被害は「失敗した apply が痕跡を残さない」という保証の喪失に留まらない。失敗 run が
新規に置いた symlink は前世代 manifest に記録が無いため、次の apply の stale 除去でも
拾えず、FS 上に恒久的なゴミとして残る（→ ADR-0044 背景）。PreRemove の migration が
半分だけ進んだ状態は更に悪く、実ディレクトリが消えたまま新しい dir symlink も無い、
という配置前より劣化した状態を作る。

逆に、巻き戻すべきでない段で巻き戻すのも同じ深刻度の欠陥である。commit（`nix-env
--set`）の失敗時点では FS の全書き込みが成功しており整合しているため、ここで unwind
すると整合状態を壊す（→ ADR-0044 §2）。この非対称性が崩れる方向の欠陥も本リスクに含む。

`design` ではなく `requirement` へ張るのは、undo ジャーナルという機構そのものが
REQ-5e75aabc の `specification` に「インメモリ undo ジャーナルへ逆操作を 1 件記録する」と
書き下ろされて要求へ昇格しており、対応する design item が `docs/design/` に存在しない
ため（REQ-5e75aabc を satisfies するのは DSG-836aa5cb というテスト戦略の design だけ）。

## 実現性

**likelihood: high** — 逆操作は 6 種（unlinkNew / relinkOld / removeCopy / restoreRename /
mkdir / backup 戻し）あり、それぞれの前方操作が置かれる段（PreRemove / Backup / place /
materializeCopies / removeStale）と組み合わせで機能する。段の追加・順序変更のたびに
journal 記録漏れが入りうる構造で、実際に ADR-0046 / ADR-0047 の PreRemove 追加時に
journal 側の対応が必要になっている。

**impact: high** — 復旧はユーザーの手作業になり、消えた実ディレクトリは復旧手段が無い。

## 緩和

TC-9504e908（逆操作単体の正しさ）・TC-83fe0d4a（LIFO 順序と journal のライフサイクル）・
TC-3b02ab58（実 FS 故障注入による end-to-end 巻き戻し）・TC-e048211f（commit 失敗は
unwind しない非対称性）が緩和する。検証方針そのものは TP-deb05610 が定める。
