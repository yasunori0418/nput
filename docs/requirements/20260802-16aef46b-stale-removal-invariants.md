---
id: "REQ-16aef46b-7bb8-4ca1-b962-e9f3ed1fd1d2"
type: requirement
derives_from:
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
name: "stale 除去は前世代の記録通りを指す symlink のみに限り、copy は消さず orphan を警告する"
specification: |
  Removal SHALL be conservative. Only a symlink that the previous-generation manifest
  records as having been placed by nput, and that still points at the destination so
  recorded — the store path of that generation, or the recorded out-of-store path — SHALL
  be removed. Regular files and links not managed by nput SHALL NOT be touched. Where the
  record and the actual entity disagree, the engine SHALL NOT remove it and SHALL warn.
  On a first run, or where there is no record, nothing SHALL be removed. When an entry
  disappears, a symlink — whether a store link or an out-of-store one — SHALL be removed
  under those invariants, whereas a copy SHALL NOT be removed, being data owned by the
  user, and its being orphaned SHALL be notified by a warning.
specification_ja: |
  削除は保守的に行わなければならない。前世代マニフェストが「nput が配置した」と記録し、かつ
  現状もその記録通りの先（その世代の store パス / 記録された out-of-store パス）を指す symlink
  のみを削除する。通常ファイルや nput 非管理の link には触れてはならない。記録と実体が不一致
  なら削除せず警告する。初回 / 記録なしは何も消さない。entry が消えたとき、symlink（store /
  out-of-store）は上記の不変条件を満たすもののみ除去し、copy は除去してはならない（ユーザー
  所有データのため）。ただし copy が orphan になったことは警告で通知する。
---
# REQ-16aef46b: stale 除去は前世代の記録通りを指す symlink のみに限り、copy は消さず orphan を警告する

## 仕様

削除は保守的に行う。前世代マニフェストが「nput が配置した」と記録し、**かつ現状もその記録通りの
先（その世代の store パス／記録された out-of-store パス）を指す symlink** のみ削除する。
通常ファイルや nput 非管理の link には触れない。記録と実体が不一致なら削除せず警告する。
初回／記録なしは何も消さない。

| 配置種別 | entry が消えたとき |
|---|---|
| symlink（store / out-of-store）| **除去する**（ただし上記の保守的不変条件を満たすもののみ）|
| copy | **除去しない**（ユーザー所有データ）。ただし orphan を警告で通知する |

> **上は原文の写しで、規範は frontmatter が正**。除去後の空親ディレクトリ剪定は
> REQ-8409db86、除去に途中失敗したときの巻き戻しは REQ-5e75aabc の担当。この保守的不変条件は
> `reset` の symlink 除去にも適用されるが（REQ-31f2882e）、copy を明示的に消す唯一の手段が
> `reset` であることも同 item の担当。

## 出典

`docs/spec.md`「世代管理仕様」→「stale 除去の対象と安全不変条件」節の導入文と表。

決定の実体は ADR-0002「世代管理を nix profile に乗せる」の保守的削除方針。
