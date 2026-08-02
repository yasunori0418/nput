---
id: "REQ-0e341430-17f0-498b-9439-65491652163a"
type: requirement
name: "rollback は FS を先に収束させてから profile ポインタを最後に移す"
specification: |
  `nput rollback <name>` SHALL return the placement to the previous generation. Because
  nput places into an arbitrary root rather than into the profile directory itself, moving
  the profile pointer alone would not change the filesystem, so re-placement SHALL be
  mandatory. The diff for stale removal SHALL take
  the manifest of the current generation N — the present state of the filesystem — as the
  `baseline` and the manifest of the generation N-1 being returned to as the `target`, and
  the planner SHALL be computed from that pair and reflected onto the filesystem in the
  same order as an apply: pre-placement removal, then placement and replacement, then
  stale removal. Where generation N migrated a self-recorded ancestor symlink into nested
  entries below it, the pre-placement removal SHALL remove, before placement, the
  self-recorded symlink that blocks the ancestor of the destination N-1, through the same
  execution path as the pre-placement removal of an apply, stopping with an error on drift
  in the same way. Only at the very end SHALL the profile pointer be moved through
  `nix-env --profile <profileDir>/profile --rollback` (or `--switch-generation N-1`);
  moving the pointer first would shift the baseline to N-2 and make stale removal wrong,
  so the filesystem SHALL converge first and the pointer move SHALL come last. The apply
  engine SHALL be reused by substituting the pair (`baseline`, `target`).
specification_ja: |
  `nput rollback <name>` は配置を前世代へ戻すものである。nput は profile
  dir 自体ではなく任意 root に配置するため、profile ポインタの移動だけでは FS が変わらず、
  再配置が必須である。stale 除去の diff は、`baseline` を現世代 N の manifest（FS の現状）、
  `target` を戻る世代 N-1 の manifest とし、この組で planner を計算して apply と同順
  （配置前除去 → 配置 / 張替え → stale 除去）で FS へ反映しなければならない。世代 N が自己記録の
  祖先 symlink を配下ネスト entries へ移行した世代であれば、配置前除去は戻り先 N-1 の祖先を
  塞いでいる自己記録 symlink を配置前に除去する（apply の配置前除去と同一実行経路・drift 時は
  同じくエラー停止）。最後に `nix-env --profile <profileDir>/profile --rollback`（または
  `--switch-generation N-1`）で profile ポインタを移す。ポインタを先に動かすと baseline が N-2 へ
  ずれ stale 除去が誤るため、FS 収束を先に、ポインタ移動を最後にしなければならない。apply
  エンジンは（`baseline`, `target`）の差し替えで再利用する。
---
# REQ-0e341430: rollback は FS を先に収束させてから profile ポインタを最後に移す

## 仕様

**standalone（home mode）**: `nput rollback <name>` で前世代に戻す。nput は profile dir 自体では
なく任意 root に配置するため、profile ポインタ移動だけでは FS が変わらず**再配置が必須**。
stale 除去の diff は次の基準・順序で行う:

1. `baseline` = 現世代 N の manifest（FS の現状）／ `target` = 戻る世代 N-1 の manifest
2. `(baseline, target)` で planner を計算し、apply と同順（**PreRemove → place/replace →
   stale 除去**）で FS へ反映する。N が自己記録の祖先 symlink を配下ネスト entries へ移行した
   世代なら、`plan.PreRemove` は戻り先 N-1 の祖先 symlink を塞いでいる自己記録 symlink を配置前
   除去する（apply の PreRemove と同一実行経路・drift 時は同じく error 停止）
3. **最後に** `nix-env --profile <profileDir>/profile --rollback`（または
   `--switch-generation N-1`）で profile ポインタを移す

ポインタを先に動かすと baseline が N-2 へずれ stale 除去が誤るため、FS 収束を先に・ポインタ移動を
最後にする。apply エンジンを `(baseline, target)` 差し替えで再利用する。

> **上は原文の写しで、規範は frontmatter が正**。原文が併記する次の点は本 item の担当ではない。
>
> - 任意世代切替・世代 GC に `nix-env --profile` 系 / `nix-collect-garbage` を使うこと →
>   REQ-706de717
> - module で `nput rollback` を公開せず host へ一本化すること → REQ-844ee375
> - project mode で rollback を公開しないこと → REQ-46fccb80
> - `rollback` が名指し必須で `--all` に対応しないこと・`list-generations` が home mode 限定で
>   あること → REQ-89c7baf9 / REQ-05abce3e
> - 途中失敗時に undo ジャーナルで巻き戻すこと → REQ-5e75aabc
> - 配置前除去の判定そのもの → REQ-c9ab91c1

## 出典

`docs/spec.md`「世代管理仕様」→「ロールバック」節の standalone（home mode）の箇条書き。

決定の実体は ADR-0015「実装前レビューで surfaced した残セマンティクス」の rollback diff 基準で、
戻り先の祖先 symlink を配置前除去する経路は ADR-0046 が定めている。
