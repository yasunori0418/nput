---
id: "RISK-2b17fefb-6e92-4513-9e7c-de21897c9cfe"
type: risk
name: "flock による直列化が破れ、同一 config への同時実行が互いの配置を壊す"
threatens:
  - "REQ-1c1526b1-59e3-4264-bb7c-65a10a4aa461"
  - "REQ-60c6b7ea-e936-4ce8-bd75-ad35e9c693b9"
likelihood: medium
impact: high
level: high
---
# RISK-2b17fefb-6e92-4513-9e7c-de21897c9cfe: flock による直列化が破れ、同一 config への同時実行が互いの配置を壊す

## リスク

同一 `profileDir` に対する apply / rollback / reset は flock で直列化される
（→ REQ-60c6b7ea-e936-4ce8-bd75-ad35e9c693b9 (2)(a)）。この直列化が破れると、2 つの run が同じ target 集合へ同時に
書き込む。片方の stale 除去がもう片方の配置直後の symlink を消す、片方の undo journal が
もう片方の書き込みを「自分が置いたもの」として巻き戻す、といった形で、どちらの run の
巻き戻し保証も成立しなくなる。原子性の他のリスクが「1 run の中の失敗」を扱うのに対し、
本リスクはその前提である「1 config を触るのは同時に 1 run だけ」の側が崩れる。

破れ方は取得モードの取り違えで起きる。既定は blocking（`LOCK_EX`）で待ち、`--no-wait`
のときだけ try-lock（`LOCK_NB`）でスキップする（→ REQ-1c1526b1-59e3-4264-bb7c-65a10a4aa461）。blocking のはずの経路が
try-lock になっていれば競合時に黙って何もせず「成功した」と見え、try-lock のはずの
`shellHook` 経路が blocking になっていればシェル入室がロック保持者の完了までブロックされる。
後者は破壊ではないが、devShell / direnv 運用を実質的に壊す。

保持区間の誤りも同じ経路で効く。ロックは操作の全区間で保持され完了時に解放される必要が
あり、早すぎる解放は上の同時実行を許し、解放漏れは以後の全 run を永久にブロックする。
スキップを表す `ErrSkipped` が `fmt.Errorf("%w")` のラップ越しに `errors.Is` で届かなく
なるのも、呼び出し側がスキップを失敗と誤認する形で本リスクに含む。

## 実現性

**likelihood: medium** — 競合は devShell の `shellHook` が明示 apply と重なる、複数の
シェルを同時に開く、といった日常操作で成立する。

**impact: high** — 同時書き込みの結果は配置の破壊であり、巻き戻しも効かない。

## 緩和

TC-4b4709c9-1b7a-42f0-8430-62a26ead4eff（engine 公開 API から見たロック取得モードと保持区間の観測）と
TC-5454b99c-f274-438a-9ba9-5be09aa71c98（`internal/lock` プリミティブ単体の try / blocking / 再取得）が緩和する。
