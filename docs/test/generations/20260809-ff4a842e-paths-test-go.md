---
id: "CASE-ff4a842e-dd25-4c75-82ef-185507781d02"
type: test_case
name: "paths_test.go — state 基底の解決・roothash の性質・profileDir キーイング・世代リンク名と兄弟パス"
target: "internal/paths/paths_test.go"
covers:
  - "TC-40b762fc-3b48-471c-b495-c5ffe1f584b5"
---
# CASE-ff4a842e: paths_test.go — state 基底の解決・roothash の性質・profileDir キーイング・世代リンク名と兄弟パス

## 対象

`internal/paths/paths_test.go`

環境変数（`$XDG_STATE_HOME` / `$HOME`）と入力から profile のパス一式を決める層の
ユニットテスト。

## 検証内容

- **`<state>` の解決** — `$XDG_STATE_HOME` があれば `$HOME` の値によらずそれを使う。
  無ければ `$HOME/.local/state` へ落ちる。`$HOME` も解決できなければエラーにする
- **`<roothash>` の性質** — 同じ絶対 root に対して決定的で、長さが固定
- **キーイングの分岐** — project は `<roothash>`、home（`--root` なし）は `<name>` 直キー、
  home + `--root` 上書きは `<roothash>`、fixed root も `<roothash>`
- **世代リンクの命名** — profile リンクに対する `profile-N-link` の形
- **`.pending` out-link** — profileDir 内の `.pending` として解決される（project ケースで assert）
- **backref `.root`** — roothash 階層（`<name>` dir の親）の `.root` として解決される。
  `<roothash>` キーのモード（project / home + `--root` / fixed）でのみ生じ、home の
  `<name>` 直キーでは空になる
