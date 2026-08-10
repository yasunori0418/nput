---
id: "RISK-fbf029f6-866c-4a08-a4eb-f09e3c7e907e"
type: risk
name: "rollback が FS を前世代へ収束させ損ね、世代表示と実配置が食い違う"
threatens:
  - "REQ-0e341430-17f0-498b-9439-65491652163a"
likelihood: high
impact: high
level: high
---
# RISK-fbf029f6: rollback が FS を前世代へ収束させ損ね、世代表示と実配置が食い違う

## リスク

nput は profile ディレクトリ自体ではなく任意 root へ配置するため、profile ポインタを
戻しただけでは FS は何も変わらない。`rollback` は「現世代 N の manifest を baseline、
戻る世代 N-1 を target」として planner を回し、apply と同順（配置前除去 → 配置 / 張替え →
stale 除去）で FS を収束させてから、最後にポインタを移す（→ REQ-0e341430）。

この収束が欠けると、`list-generations` は N-1 を current と表示するのに FS には N の
配置が残る、という嘘の状態になる。ポインタを先に動かすと更に悪く、次の apply が baseline を
N-2 と読んで stale 除去を誤る。

収束の失敗形は経路ごとに異なる。**祖先 symlink の migration** — 世代 N が配下 entries を
まとめて祖先 symlink へ移行していた場合、戻り先 N-1 の配下 entries を置くには先にその
祖先 symlink を除去する必要がある（plan の PreRemove を落とすと、symlink 越しに解決した
store パスへ書こうとして EEXIST / EROFS で落ちる → ADR-0046、issue #173）。**途中失敗の
巻き戻し** — rollback も apply と同じ undo journal を使う必要があり、migration だけ済んで
配置が失敗した状態を残してはならない。**ポインタ移動の失敗** — `SwitchGeneration` の
失敗時点では FS の書き込みが全て成功しているため巻き戻さない（apply の commit 失敗と同じ
非対称性 → ADR-0044 §2）。ここで unwind すると整合状態を壊す。

前提が満たされない場合に静かに進むのも本リスクに含む。前世代が無い（最古世代）・profile が
そもそも無い（未 apply）・戻り先の target が foreign な実体で塞がれている、のいずれも
エラーで停止すべきであり、部分的に配置してから諦めるのが最悪の結果になる。

同じ FS 操作系でも `reset` の teardown は脅威が別で、あちらは「消してはいけないものを
消す」側にある（→ RISK-bb54245e）。

## 実現性

**likelihood: high** — rollback は apply エンジンの再利用で成立しており、apply 側の
変更（PreRemove の追加・undo journal の導入）のたびに rollback 側の配線漏れが実際に
発生している（→ issue #173、#168）。

**impact: high** — 誤収束の帰結は配置の破壊か、復旧手段だと思われている機能が壊れて
いること自体である。

## 緩和

TC-36ea3609（rollback の再収束と祖先 migration）・TC-fa7911c6（rollback の途中失敗・
前提不成立・conflict 全件報告）・TC-527b5034（e2e での世代往復）が緩和する。
