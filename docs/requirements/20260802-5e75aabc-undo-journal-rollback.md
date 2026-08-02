---
id: "REQ-5e75aabc-0e8f-4a6c-92bd-a712dc68a940"
type: requirement
derives_from:
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
name: "途中失敗した apply / rollback はインメモリ undo ジャーナルで全 FS 変更を巻き戻す"
specification: |
  When an apply or a rollback fails part-way through any of its stages — pre-placement
  removal, backup, placement (symlink / copy), copy reflection or stale removal — every
  filesystem change that the run has made SHALL be rolled back so that the pre-apply state
  is restored, a failed apply thereby leaving no trace on the filesystem. This SHALL be
  always in effect and SHALL NOT be gated behind a flag. For that purpose the engine SHALL
  record one inverse operation in an in-memory undo journal for each filesystem write:
  deletion for a newly placed symlink or copy; re-creation with the old destination
  (captured before the unlink) for a replacement; re-creation with the destination
  recorded in the previous-generation manifest for a stale-removed link or a PreRemove
  unlink; re-creation of the empty directory for a PreRemove rmdir; and renaming back for
  the aside made by `--recopy` or `--backup`. Once an error occurs, the journal recorded so
  far SHALL be rolled back in reverse (LIFO) order before the error is returned. The
  journal SHALL be discarded once the commit through `nix-env --set` has succeeded, a
  failure of `--set` itself being outside the scope of rollback because an idempotent
  re-run converges. On the success of the run, the aside made by `--recopy` SHALL be
  deleted, whereas the aside made by `--backup` SHALL be left in place. Crashes such as
  SIGKILL or power loss SHALL be out of scope, the journal existing only in process memory
  with no persistent write-ahead log; in that case the guarantee SHALL continue to rest on
  the generation being left uncommitted and on convergence by idempotent re-run. The drift
  repair on the generation-skip path SHALL be rolled back by the same mechanism, and
  `rollback` SHALL use the same mechanism as apply, except that the move of the profile
  pointer SHALL be outside the scope of rollback because all filesystem changes have
  already succeeded by that point, and that the Backup stage SHALL always be empty for
  `rollback`, which has no flag equivalent to `--backup`.
specification_ja: |
  apply / rollback が各段（配置前除去・退避・配置（symlink / copy）・copy 反映・stale 除去）の
  いずれかで途中失敗したとき、その run が行った FS 変更を全て巻き戻して pre-apply 状態へ復元
  しなければならない（「失敗した apply は FS に痕跡を残さない」）。これは常時有効であり、
  フラグで制御してはならない。そのため engine は FS 書き込みごとにインメモリの undo ジャーナルへ
  逆操作を 1 件記録する。新規配置 symlink / copy は削除、張替えは unlink 前に捕捉した旧リンク先
  での symlink 再作成、stale 除去したリンクと PreRemove の unlink は前世代 manifest の記録 dest
  での symlink 再作成、PreRemove の rmdir は空 dir の再作成、`--recopy` / `--backup` の rename
  退避は退避物の rename back とする。エラーが発生したら、それまでに記録したジャーナルを逆順
  （LIFO）で巻き戻してからエラーを返す。`nix-env --set`（コミット）が成功した後はジャーナルを
  破棄する（`--set` 自体の失敗は冪等再実行で収束するため巻き戻し対象外）。`--recopy` の退避物は
  成功時に削除し、`--backup` の退避物は成功時も残置する。クラッシュ（SIGKILL・電源断）は対象外
  とする（ジャーナルはプロセスメモリ上にのみ存在し永続 WAL を持たないため）。この場合は
  「世代未コミット + 冪等再実行で収束」が保証を担う。世代スキップ経路の drift 修復も同じ機構で
  巻き戻し、`rollback` も apply と同じ機構を使う。ただし profile ポインタの移動はこの時点で
  全 FS 変更が成功済みのため巻き戻し対象外とし、`rollback` には `--backup` 相当のフラグが無いため
  Backup 段は常に空とする。
---
# REQ-5e75aabc: 途中失敗した apply / rollback はインメモリ undo ジャーナルで全 FS 変更を巻き戻す

## 仕様

apply / rollback が PreRemove・Backup（`--backup`）・配置（symlink / copy）・stale 除去の
いずれかの段で途中失敗すると、その run が行った FS 変更を**全て**巻き戻し、pre-apply 状態へ
復元する（「失敗した apply は FS に痕跡を残さない」）。フラグなし・常時有効。

```
各段（PreRemove / Backup / place / copy 反映 / stale 除去）の FS 書き込みごとに、インメモリの
undo ジャーナルへ逆操作を 1 件記録する:
  新規配置 symlink / copy         → 削除
  張替え（unlink 前に旧リンク先を捕捉）→ 旧リンク先で symlink を再作成
  stale 除去したリンク            → 前世代 manifest の記録 dest で symlink を再作成
  PreRemove の Unlink             → 記録 dest で symlink を再作成
  PreRemove の Rmdir              → 空 dir を再作成
  --recopy の rename 退避         → 退避物を rename back（成功時は退避物を削除）
  --backup の rename 退避         → 退避物を rename back（成功時も退避物は削除せず残置）

いずれかの段でエラーが発生したら、それまでに記録したジャーナルを逆順（LIFO）で巻き戻してから
エラーを返す。nix-env --set（コミット）が成功した後はジャーナルを破棄する（--set 自体の失敗は
巻き戻し対象外・冪等再実行で収束）。
```

- **クラッシュ（SIGKILL・電源断）は対象外**。undo ジャーナルはプロセスメモリ上にのみ存在し、
  永続 WAL は持たない。この場合は変わらず「世代未コミット + 冪等再実行で収束」が保証を担う
- 世代スキップ経路の drift 修復（repairDrift）も同じ機構で巻き戻される
- `rollback` も apply と同じ機構（PreRemove → place → stale 除去）で途中失敗時に巻き戻す。
  プロファイルポインタ移動（`--switch-generation`）はこの時点で全 FS 変更が成功済みのため
  巻き戻し対象外。`rollback` に `--backup` 相当のフラグはなく、Backup ステージは常に空

> **上は原文の写しで、規範は frontmatter が正**。巻き戻し自体が失敗したときの続行と報告は
> REQ-9fca28c9、drift 修復経路で Backup 段だけが残ることは REQ-9b0046e0 の担当。
> `rollback` の FS 収束を先・ポインタ移動を最後にする順序そのものは REQ-0e341430 の担当。

## 出典

`docs/spec.md`「配置動作仕様」→「途中失敗時の巻き戻し（インメモリ undo ジャーナル）」節の
導入文・コードブロックと、箇条書き第 3・4・5 項。

決定の実体は ADR-0044「apply 途中失敗の完全巻き戻し — インメモリ undo ジャーナル」。
