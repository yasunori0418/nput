---
id: "TP-deb05610-44bc-4962-8939-952392e5fbd0"
type: test_plan
name: "原子性は実 FS の条件で故障を誘発して不変条件ごとに検証する"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
depends_on:
  - "TP-e7c25263-6d2d-4a37-8275-26906889d912"
specification: |
  The atomicity of a run SHALL be verified by injecting faults, not by asserting the happy
  path alone, and the injected faults SHALL be induced through real filesystem conditions
  rather than through an error-returning double, so that the error values under test are
  the ones the operating system actually produces. Three invariants SHALL each be covered:
  a failure part-way through a filesystem-mutating batch rolls back every change that run
  had already made; a failure at commit does not unwind the placements, because the
  filesystem is already consistent and unwinding would destroy it; and re-running after an
  interruption converges on the intended state rather than compounding the damage. Fault
  injection that a privileged user would bypass SHALL detect that condition and skip,
  rather than pass vacuously, and where a root-proof technique exists it SHOULD be
  preferred over one requiring the skip.
specification_ja: |
  実行の原子性は、正常系のアサートだけでなく故障注入によって検証しなければならない。注入
  する故障はエラーを返す double ではなく実ファイルシステムの条件で誘発しなければならない
  （テスト対象のエラー値を、OS が実際に返すものに保つため）。次の 3 つの不変条件をそれぞれ
  覆わなければならない。ファイルシステムを変更するバッチの途中で失敗したとき、その実行が
  それまでに加えた変更をすべて巻き戻すこと。commit で失敗したときは配置を巻き戻さないこと
  （ファイルシステムは既に整合しており、巻き戻せばそれを壊すため）。中断後の再実行が、被害を
  重ねるのではなく意図した状態へ収束すること。特権ユーザーでは迂回される故障注入は、空虚に
  成功するのではなくその条件を検出して skip しなければならず、root でも成立する手法がある
  場合はそちらを優先すべきである。
---
# TP-deb05610: 原子性は実 FS の条件で故障を誘発して不変条件ごとに検証する

## 仕様

原子性の検証は故障注入で行い、故障はエラーを返す double ではなく実 FS の条件で誘発する。
OS が実際に返すエラー値（EACCES / ENOTDIR / EISDIR / EINVAL）を経路に流すため。

覆う不変条件は 3 つ。

| 不変条件 | 内容 |
|---|---|
| mid-batch 失敗の巻き戻し | FS 変更バッチの途中で失敗したら、その実行が加えた変更を全て undo してから返す |
| commit 失敗は unwind しない | commit 段の失敗では配置を巻き戻さない（FS は既に整合しており、巻き戻すと壊す）|
| 中断後の収束 | 中断後の再実行が意図した状態へ収束する（被害を重ねない）|

故障の誘発手法は 2 系統ある。

- **権限による誘発**: 親ディレクトリを mode 0o555 にして、plan 時の `Lstat` / `ReadDir` は
  通したまま execute 時の書き込みだけを EACCES で落とす。plan 時に落ちると「バッチ途中の
  失敗」にならないため、通常ファイルで置き換える手法は採らない
- **root でも成立する誘発**: パス成分に通常ファイルを置いて ENOTDIR、ディレクトリの fd を
  読ませて EISDIR、非 symlink を readlink して EINVAL

権限による誘発は root で迂回されるため、`os.Geteuid() == 0` を検出して skip する（空虚に
成功させない）。CI が root で走る場合に被覆が消える範囲を狭めるため、root でも成立する手法を
取れるところではそちらを採る。

実 FS で動かすという前提そのものは TP-e7c25263 の担当で、本 item はその上での故障注入体系を
定める。

## 出典

ADR-0044（apply の end-to-end undo journal）が定める「実行中の FS 変更を全て巻き戻す」
という決定を、テスト側でどう検証するかの規定。

現況の実装は `internal/engine/undo_journal_test.go`（mid-batch 失敗の巻き戻し・commit
失敗の非 unwind）・`undo_test.go`（journal の LIFO 巻き戻しと best-effort 継続）・
`copy_test.go`（copy 系 syscall 失敗網羅）で、root 判定 skip と root-proof 手法の使い分けも
これらのファイルのコメントが根拠を述べている。
