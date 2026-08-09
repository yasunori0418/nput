---
id: "TC-06052178-ed49-4800-beb2-1d7c5d696ea8"
type: test_condition
name: "reset が保守的不変条件の範囲だけを消し、dryrun は副作用ゼロ、非同意では中断する"
mitigates:
  - "RISK-bb54245e-b284-4b4d-9896-8fec2b4e521c"
---
# TC-06052178: reset が保守的不変条件の範囲だけを消し、dryrun は副作用ゼロ、非同意では中断する

## テスト条件

`Reset`（profile を触らない FS-only teardown → REQ-31f2882e）を、一度 apply して世代を
コミットした状態から駆動して検証する。

**除去の範囲** — nput が配置した symlink と copy target は除去する。stale 除去と同じ
保守的不変条件を外れるもの（別物を指すよう張り替えられた foreign symlink）は残す。
config が entry を残す限り次の apply が再配置するので、消し過ぎだけが不可逆な誤りになる。

**target フィルタ** — 引数で指定した entry だけを対象にする。存在しない target 名は
エラーで停止する（黙って 0 件処理として成功しない）。

**`--dryrun`** — 除去予定の target を列挙するだけで FS を変更しない。preview が
preview であることの担保。

**同意** — 確認関数が false を返したら中断し、FS を変更しない。

**profile 不在** — 一度も apply していない状態での `reset` は no-op でエラーにしない。

## 対応する CASE

CASE-503c9021（`internal/engine/reset_test.go`）。`Reset` の返す `Result` の構造は
CASE-2008a909 が併せて覆う。
