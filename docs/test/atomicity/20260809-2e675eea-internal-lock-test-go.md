---
id: "CASE-2e675eea-8e0e-41f6-bb94-5f2ce944255b"
type: test_case
name: "internal/lock/lock_test.go — flock ラッパーの try / blocking / 再取得"
covers:
  - "TC-5454b99c-f274-438a-9ba9-5be09aa71c98"
---
# CASE-2e675eea: internal/lock/lock_test.go — flock ラッパーの try / blocking / 再取得

## 対象

`internal/lock/lock_test.go`

engine から切り離した flock ラッパー単体のユニットテスト。

## 検証内容

- **try-lock の競合** — ロック保持中の try-lock が `ErrLocked` を返す
- **解放後の再取得** — 解放したロックを同じキーで再取得できる
- **blocking の待機** — blocking 取得は保持中に取得へ進まず、解放後に取得する
