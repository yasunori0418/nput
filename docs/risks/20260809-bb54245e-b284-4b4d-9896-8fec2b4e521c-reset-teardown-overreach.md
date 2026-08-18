---
id: "RISK-bb54245e-b284-4b4d-9896-8fec2b4e521c"
type: risk
name: "reset の teardown が管理外の実体まで消し、preview と同意のゲートも素通りする"
threatens:
  - "REQ-31f2882e-d2e3-4e3b-b783-feb627d73ac6"
  - "REQ-6a950d6d-c191-4235-a1b4-73ffc7c2bb38"
  - "REQ-31dae599-f3a3-4bbe-b367-c955535265da"
likelihood: medium
impact: high
level: high
---
# RISK-bb54245e-b284-4b4d-9896-8fec2b4e521c: reset の teardown が管理外の実体まで消し、preview と同意のゲートも素通りする

## リスク

`reset` は配置物を無い状態へ戻す FS-only teardown で、copy target を消す唯一の明示手段
でもある（→ REQ-31f2882e-d2e3-4e3b-b783-feb627d73ac6）。削除だけを行うコマンドであるぶん、誤りは片方向にしか効かない。
消し過ぎは不可逆で、消し足りないぶんは次の apply が再配置するので実害が無い。

**除去範囲の逸脱** — symlink の除去は stale 除去と同じ保守的不変条件（nput 管理・前世代の
記録どおりを指すもののみ・foreign は warning で残す）に従う必要がある。ここを緩めると、
ユーザーが別途張った symlink や他ツールの配置物が消える。target フィルタの解釈違い
（存在しない target 名を黙って 0 件処理として成功させる）も、ユーザーが「消えたつもり」の
まま実体を残す方向の誤りになる。

**preview の破れ** — `--dryrun` が副作用を持てば、削除対象を確認するつもりの操作が削除
そのものになる。teardown の preview は、この操作を安全に試す唯一の手段である。

**同意ゲートの破れ** — 確認プロンプトの中断（no を選ぶ・非 TTY で `-y` が無い）が効かず
処理が進めば、ユーザーの同意なくデータが消える。

このリスクは rollback の誤収束（RISK-fbf029f6-866c-4a08-a4eb-f09e3c7e907e）とは脅威が別で、あちらは「意図した状態へ
持っていけない」、こちらは「消してはいけないものを消す」。

## 実現性

**likelihood: medium** — 保守的不変条件の実装は stale 除去と共有されており、そちらの
変更が reset 側へ波及しうる。

**impact: high** — 削除は不可逆で、対象が nput 管理外のユーザーの実データになりうる。

## 緩和

TC-06052178-ed49-4800-beb2-1d7c5d696ea8（reset の teardown 安全性）が緩和する。ただし覆えるのは engine 層まで
（`Confirm` コールバックが false を返したときの中断）で、TTY 判定と `-y` / `--yes` の
解釈は CLI 側の責務のため、同意ゲートの残り半分は cli-json 区分の TC が担う。
