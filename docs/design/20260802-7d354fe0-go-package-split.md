---
id: "DSG-7d354fe0-a333-495b-9f4b-14bba316dc47"
type: design
name: "Go 側は cmd/nput に CLI 面を、internal に配置ロジックを置く 2 パッケージ構成にする"
satisfies:
  - "REQ-f4d7d4ab-fbdb-48c6-b29f-08dd88e72645"
  - "REQ-b74a118a-1272-44eb-944c-7725163211c6"
  - "REQ-637599dc-a1ec-4af5-9e97-e882c7df56d0"
---
# DSG-7d354fe0: Go 側は cmd/nput に CLI 面を、internal に配置ロジックを置く 2 パッケージ構成にする

## 設計

```
cmd/nput/
└── main.go            # entrypoint 発見 + nix build/eval オーケストレーション + サブコマンド分岐
internal/              # 配置・diff・保守的 stale 除去の純ロジック（ユニットテスト対象）
```

REQ-f4d7d4ab が定める CLI と engine の 2 層を、そのまま Go のパッケージ境界へ落とす。
`packages.nput` は両者（`cmd/nput` + `internal`）をまとめてビルドした 1 バイナリになる。

分け方が実現手段として効くのは次の点。

- **依存の非対称を層ごとに固定できる**。REQ-637599dc が CLI 層に許す第三者依存（cobra）は
  `cmd/nput` に閉じ、REQ-b74a118a が engine へ課す stdlib-only は `internal/` に閉じる。
  1 パッケージだと同一の import ブロックに両方が混ざり、どちらの制約が効く範囲かが
  コードから読めなくなる
- **`internal/` は Go の言語仕様で外部リポジトリから import できない**ため、engine を
  内部層へ閉じる REQ-b74a118a がコンパイラに強制される。engine をライブラリとして
  公開しない意思が構造で表明される
- **テスト対象が `internal/` に集約される**。engine の振る舞い（REQ-6c4e174a が許す
  外部コマンドの範囲を含む）を検証する対象が 1 パッケージに収まるため、engine 側の
  ユニットテスト（DSG-836aa5cb）が engine の全経路を踏める。CLI 層と混在していると、
  検証対象と cobra を含む CLI の関心事が同じパッケージに同居する

`main.go` 単一ファイルにサブコマンド分岐まで置くのは、CLI 層が持つのが「entrypoint 発見・
nix オーケストレーション・フラグ解釈」に限られ、分岐の実体が `internal/` の呼び分けに
過ぎないため。

## 出典

`docs/design.md`「プロジェクト構成」の `cmd/nput/` / `internal/` の項と、
同「レイヤー構成」末尾の「配置ロジックは Go エンジンが単一の源として所有し、`lib/` は
データ生成に徹する」。
