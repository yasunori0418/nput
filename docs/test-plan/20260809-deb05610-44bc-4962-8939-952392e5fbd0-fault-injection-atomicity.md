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
  the ones the operating system actually produces. Four invariants SHALL each be covered:
  a failure part-way through a filesystem-mutating batch rolls back every change that run
  had already made; a failure at commit does not unwind the placements, because the
  filesystem is already consistent and unwinding would destroy it; re-running after an
  interruption converges on the intended state rather than compounding the damage; and a
  concurrent run cannot interleave with any of the above, since the three invariants above
  assume the run is the only writer. Fault injection that a privileged user
  would bypass SHALL detect that condition and skip, rather than pass vacuously, and where
  a root-proof technique exists it SHOULD be preferred over one requiring the skip. For the
  fourth invariant the tests SHALL cover that a second run cannot acquire a held lock, that
  the lock is held for the whole operation and released afterwards, that the operations
  which wait for it do wait, that the one which declines to wait reports that it was
  skipped, and that this outcome and an ordinary failure cannot be mistaken for one another
  once they have been wrapped by the layers above. Contention SHALL be produced by actually
  holding the lock rather than by simulating the held state, so that the mechanism under
  test is the one that runs in production.
specification_ja: |
  実行の原子性は、正常系のアサートだけでなく故障注入によって検証しなければならない。注入
  する故障はエラーを返す double ではなく実ファイルシステムの条件で誘発しなければならない
  （テスト対象のエラー値を、OS が実際に返すものに保つため）。次の 4 つの不変条件をそれぞれ
  覆わなければならない。ファイルシステムを変更するバッチの途中で失敗したとき、その実行が
  それまでに加えた変更をすべて巻き戻すこと。commit で失敗したときは配置を巻き戻さないこと
  （ファイルシステムは既に整合しており、巻き戻せばそれを壊すため）。中断後の再実行が、被害を
  重ねるのではなく意図した状態へ収束すること。そして並行する実行が上記のいずれにも割り込め
  ないこと（上記の 3 つの不変条件は、その実行が唯一の書き手であることを前提としている
  ためである）。特権ユーザーでは迂回される故障注入は、空虚に成功するのではなくその条件を
  検出して skip しなければならず、root でも成立する手法がある場合はそちらを優先すべきである。
  4 つめの不変条件についてテストは、保持されたロックを 2 つめの実行が取得できないこと、
  ロックが操作全体にわたって保持され終了後に解放されること、待つべき操作が実際に待つこと、
  待たない操作が skip された旨を報告すること、そしてその結果と通常の失敗とが、上位層に
  包まれたあとも互いに取り違えられないことを覆わなければならない。競合は、保持状態を模擬
  するのではなく実際にロックを保持して作らなければならない（テスト対象の機構を、本番で
  動くものと同じに保つため）。
---
# TP-deb05610-44bc-4962-8939-952392e5fbd0: 原子性は実 FS の条件で故障を誘発して不変条件ごとに検証する

## 仕様

原子性の検証は故障注入で行い、故障はエラーを返す double ではなく実 FS の条件で誘発する。
OS が実際に返すエラー値（EACCES / ENOTDIR / EISDIR / EINVAL）を経路に流すため。

覆う不変条件は 4 つ。

| 不変条件 | 内容 |
|---|---|
| mid-batch 失敗の巻き戻し | FS 変更バッチの途中で失敗したら、その実行が加えた変更を全て undo してから返す |
| commit 失敗は unwind しない | commit 段の失敗では配置を巻き戻さない（FS は既に整合しており、巻き戻すと壊す）|
| 中断後の収束 | 中断後の再実行が意図した状態へ収束する（被害を重ねない）|
| 並行実行に対する原子性 | 上の 3 つは「その実行が唯一の書き手」を前提とするため、排他そのものも覆う（詳細は下記）|

故障の誘発手法は 2 系統ある。

- **権限による誘発**: 親ディレクトリを mode 0o555 にして、plan 時の `Lstat` / `ReadDir` は
  通したまま execute 時の書き込みだけを EACCES で落とす。plan 時に落ちると「バッチ途中の
  失敗」にならないため、通常ファイルで置き換える手法は採らない
- **root でも成立する誘発**: パス成分に通常ファイルを置いて ENOTDIR、ディレクトリの fd を
  読ませて EISDIR、非 symlink を readlink して EINVAL

権限による誘発は root で迂回されるため、`os.Geteuid() == 0` を検出して skip する（空虚に
成功させない）。CI が root で走る場合に被覆が消える範囲を狭めるため、root でも成立する手法を
取れるところではそちらを採る。

**並行実行に対する原子性**では、覆う点が 5 つある。保持されたロックを 2 つめの実行が取得
できないこと、ロックが操作全体にわたって保持され終了後に解放されること、待つべき操作が
実際に待つこと、待たない操作が skip された旨を報告すること、そしてその結果と通常の失敗とが
上位層に包まれたあとも互いに取り違えられないこと。最後の 1 点は両方向で見る（skip が失敗に
埋もれないことと、無関係なエラーが skip を騙らないこと）。競合は保持状態を模擬せず実際に
ロックを保持して作る（テスト対象の機構を本番で動くものと同じに保つため）。

実 FS で動かすという前提そのものは TP-e7c25263-6d2d-4a37-8275-26906889d912 の担当で、本 item はその上での故障注入体系を
定める。

## 出典

ADR-0044（apply の end-to-end undo journal）が定める「実行中の FS 変更を全て巻き戻す」
という決定を、テスト側でどう検証するかの規定。

現況の実装は次のとおり（代表を挙げる。故障注入は engine 全域に及ぶ横断規約で、権限による
誘発は `preflight_test.go` / `engine_test.go` / `staleremove_test.go` / `generations_test.go`
にも分布する）。

| 対象 | 実装 |
|---|---|
| mid-batch 失敗の巻き戻し・commit 失敗の非 unwind | `internal/engine/undo_journal_test.go` |
| journal の LIFO 巻き戻しと best-effort 継続 | `internal/engine/undo_test.go` |
| copy 系 syscall 失敗の網羅 | `internal/engine/copy_test.go` |
| 排他 | `internal/lock/lock_test.go`（取得競合・解放後の再取得・待機）・`internal/engine/lock_test.go`（no-wait の skip・保持と解放・wrapping 越しの skip 判別）|

手法の根拠もコメントが述べている。root 判定 skip は `undo_journal_test.go` の `blockWrite`、
root でも成立する誘発（ENOTDIR / EISDIR / EINVAL）の選好は `copy_test.go` の冒頭コメント。
