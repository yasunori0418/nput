---
id: "TC-4b4709c9-1b7a-42f0-8430-62a26ead4eff"
type: test_condition
name: "engine 公開 API から見たロックの取得モードと保持区間が仕様どおりである"
mitigates:
  - "RISK-2b17fefb-6e92-4513-9e7c-de21897c9cfe"
---
# TC-4b4709c9: engine 公開 API から見たロックの取得モードと保持区間が仕様どおりである

## テスト条件

`Apply` / `Rollback` / `Reset` を外側から駆動し、`profileDir` の flock がどう取られ、
いつ解放されるかを観測する。内部ヘルパーには結合せず、公開 API 越しのブラックボックスで
条件を組む。

**取得モード** — `NoWait`（`shellHook` 経路）の `Apply` は、ロック保持中に走ると
`ErrSkipped` でスキップし、何も配置しない。`Rollback` / `Reset` は既定の blocking で、
ロック保持中は先へ進まず、解放後に処理を完了する。

**保持区間** — ロックは操作の全区間で保持され、完了時に解放される。commit をブロックして
`Apply` をロック内で停留させ、その間に走らせた `NoWait` の `Apply` がスキップすること
（操作中は保持）と、停留を解いて返ったあとの `NoWait` の `Apply` が成功すること
（完了で解放）の 2 点で観測する。

**sentinel の同一性** — `ErrSkipped` は `fmt.Errorf("%w")` のラップ（多重を含む）越しでも
`errors.Is` で到達できる。エラーへの文脈追加が sentinel の識別を切らないことの watchdog。
無関係なエラーをラップしたものが `ErrSkipped` に化けないことも併せて見る。

進行の同期は channel で決定的に取る。「保持中は進まない」だけは短い timeout 付きの
select で見るが、これは取りこぼしても偽陽性（block の見落とし）にしかならず、
flaky な失敗にはならない。

## 対応する CASE

CASE-6f6fabaa（`internal/engine/lock_test.go`）。
