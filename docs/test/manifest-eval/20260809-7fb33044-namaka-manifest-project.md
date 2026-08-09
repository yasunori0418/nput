---
id: "CASE-7fb33044-b68c-4518-bd93-b258072235ee"
type: test_case
name: "namaka: manifest-project — manifest 文書全体のスナップショット回帰"
covers:
  - "TC-de6514e2-9105-45a6-a5b9-d474911a401b"
---
# CASE-7fb33044: namaka manifest-project

## 対象

`tests/namaka/manifest-project/expr.nix`（スナップショット実体は
`tests/namaka/_snapshots/manifest-project`）

## 検証内容

`normalizeManifest` の出力全体を project root で 1 件スナップショットに固定し、以後の差分を
回帰として検出する。

- 入力は 2 entry。1 つは `subpath` を明示（`skills/nix`）、もう 1 つは `subpath` / `target` /
  `method` を全て省略して既定適用を通す。既定と明示の双方が文書へ現れる状態を固定する
- 配置元は固定 `outPath` を持つ fake flake-input で、2 つは store hash を違える
  （`000…-fake-src` と `111…-other`）。実パスを import すると store hash が揺れ、スナップ
  ショット比較そのものが成立しない
- どの不変条件も名指ししていないフィールドの追加・削除・改名が、ここで差分として現れる

規範として述べられている性質はこのスナップショットに委ねず、CASE-7a6c4957 /
CASE-59de34d4 / CASE-0c9d41d1 の名前付きアサートが守る。
