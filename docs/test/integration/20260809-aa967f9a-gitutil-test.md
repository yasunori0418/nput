---
id: "CASE-aa967f9a-8020-44f8-b45f-57244711e1b5"
type: test_case
name: "gitutil_test.go — toplevel 解決の一意性とリポジトリ外・git 不在の失敗"
target: "internal/gitutil/gitutil_test.go"
covers:
  - "TC-ec0bded4-c4bc-4a61-a2b5-0b652134b223"
---
# CASE-aa967f9a: gitutil_test.go — toplevel 解決の一意性とリポジトリ外・git 不在の失敗

## 対象

`internal/gitutil/gitutil_test.go`（対象は `internal/gitutil/gitutil.go` の toplevel 解決）

## 検証内容

- リポジトリ内のサブディレクトリから呼んでも、リポジトリの toplevel へ絶対パスで解決される
  こと
- リポジトリの外で呼んだとき、曖昧に成功せず失敗すること。この検証は隔離を要し、一時
  ディレクトリの祖先がたまたまリポジトリだと偽陽性になるため、HOME と探索の上限を環境変数で
  切って隔離している
- 実行パスから git を辿れないとき、判別可能な値で失敗すること。この値は包まずそのまま返す
  契約になっており、テストが同一性で比較することでその契約を固定している

一時ディレクトリに実際の git リポジトリを作り、実際の git を起動する統合寄りの検証。git が
無い環境では skip する。プラットフォームによる一時ディレクトリの symlink の差を吸収してから
比較する。
