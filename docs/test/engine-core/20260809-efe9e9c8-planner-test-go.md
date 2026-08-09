---
id: "CASE-efe9e9c8-cc0a-4b1b-bf2b-aa1d15428252"
type: test_case
name: "internal/planner/planner_test.go — fake FS による plan 計算の table-driven 網羅"
covers:
  - "TC-b329cafd-06c3-4b87-8357-229b69e5ba5c"
  - "TC-9df804ce-35ee-44a7-87b1-17935d53fab2"
  - "TC-b9d4ffaf-ac91-4bf1-9f27-5ea3964466ad"
---
# CASE-efe9e9c8: planner_test.go

対象: `internal/planner/planner_test.go`

plan 計算を fake FS（`Lstat` / `Readlink` / `ReadDir` を差し替えたマップ）に対して回す
table-driven テスト。実 FS へ触らずに分類の全パターンを列挙できることが、この CASE の存在
そのものによって示される。

## 主な検証内容

- **table-driven の分類網羅**: 初回 apply（前世代 manifest 無し）で除去ゼロ、自己記録
  symlink の無警告張替え、foreign symlink の警告付き後勝ち、通常ファイル占有の conflict、
  祖先 symlink の conflict、自己記録 stale 祖先の移行（複数子での祖先の重複排除を含む）
- **除去順序**: ディレクトリ移行の事前除去がボトムアップ順で組まれること
- **未知の method**: 未定義の method がエラーとして弾かれること
- **祖先が symlink でない場合**: 実ディレクトリ祖先が symlink 祖先と別分類になること
