---
id: "TC-de6514e2-9105-45a6-a5b9-d474911a401b"
type: test_condition
name: "manifest 文書全体をスナップショットに固定し、どの不変条件も名指ししない変更を差分として出す"
mitigates:
  - "RISK-5df2d02b-e5d4-40eb-86ad-e8bc96e4c34d"
  - "RISK-3de9753f-3fd3-4364-b1ab-64c68c15ec77"
---
# TC-de6514e2: 文書全体をスナップショットに固定する

## テスト条件

`normalizeManifest` の出力全体をスナップショットとして保存し、以後の変更が差分として現れる
ようにする。個別アサートは名指しした性質しか守らないため、フィールドの追加・削除・改名の
ように「アサートを書き忘れた箇所」の変化はここでしか捕まらない。

- 入力は既定適用の対象（`subpath` / `target` / `method` を省略した entry）と明示指定した
  entry を混在させ、既定と明示の双方が文書へ現れる状態を固定する
- 配置元は固定 `outPath` を持つ fake flake-input で与える。実パスを store へ import すると
  マシン・nixpkgs revision で store hash が揺れ、スナップショット比較そのものが成立しない
- 2 つの配置元は store hash を違えて置き、entry どうしの取り違えが差分として出るようにする

**このスナップショットに委ねてよい範囲**: スナップショットは「あるべき姿」ではなく「現に
そうなっている姿」を記録し、変わったときは丸ごと再承認される。規範として述べられている性質は
TC-4e7cfae7 / TC-d9175bb5 / TC-81be084d 側の名前付きアサートで守り、ここへは委ねない。

## 覆う CASE

- CASE-7fb33044（`tests/namaka/manifest-project/expr.nix`）
