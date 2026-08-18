---
id: "TC-5454b99c-f274-438a-9ba9-5be09aa71c98"
type: test_condition
name: "lock プリミティブが try で競合を弾き、blocking で待ち、解放後に再取得できる"
mitigates:
  - "RISK-2b17fefb-6e92-4513-9e7c-de21897c9cfe"
---
# TC-5454b99c: lock プリミティブが try で競合を弾き、blocking で待ち、解放後に再取得できる

## テスト条件

engine から切り離した `internal/lock` パッケージ単体で、flock ラッパーの 3 つの基本
振る舞いを検証する。engine 側の観測（TC-4b4709c9）が仕様レベルの契約を見るのに対し、
こちらは土台のプリミティブが正しいことを固定する。

- **try-lock の競合** — 保持中に try-lock すると `ErrLocked` を返す
- **解放後の再取得** — 解放したロックは同じキーで再取得できる（fd / ファイルの後始末が
  次の取得を妨げない）
- **blocking の待機** — blocking 取得は保持中に取得へ進まず、解放されてから取得する

上位の規範は TC-4b4709c9 と同じく TP-deb05610 の「並行実行に対する原子性」。

## 対応する CASE

CASE-2e675eea（`internal/lock/lock_test.go`）。
