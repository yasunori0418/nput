---
id: "CASE-efe9e9c8-cc0a-4b1b-bf2b-aa1d15428252"
type: test_case
name: "internal/planner/planner_test.go — fake FS による plan 計算の table-driven 網羅"
covers:
  - "TC-b329cafd-06c3-4b87-8357-229b69e5ba5c"
  - "TC-9df804ce-35ee-44a7-87b1-17935d53fab2"
  - "TC-b9d4ffaf-ac91-4bf1-9f27-5ea3964466ad"
  - "TC-d160e18b-4c0c-4531-a506-e7d00d88788a"
  - "TC-596d697f-4ba6-4ec1-b71e-8b5375806c08"
---
# CASE-efe9e9c8: planner_test.go

対象: `internal/planner/planner_test.go`

plan 計算を fake FS（`Lstat` / `Readlink` / `ReadDir` を差し替えたマップ）に対して回す
table-driven テスト。実 FS へ触らずに分類の全パターンを列挙できることが、この CASE の存在
そのものによって示される。

## 主な検証内容

- **配置分類の網羅**: 初回 apply（前世代 manifest 無し）で除去ゼロ、自己記録 symlink の
  無警告張替え、foreign symlink の警告付き後勝ち、通常ファイル占有の conflict、祖先 symlink の
  conflict、自己記録 stale 祖先の移行（複数子での祖先の重複排除を含む）
- **stale 除去の分類**: 記録どおりを指す symlink の除去、空 manifest による全クリア（無警告）、
  記録先と食い違う symlink の保持 + 警告、symlink でなくなった target の保持 + 警告、
  既に消えた target の no-op、copy entry の orphan 警告（除去しない）
- **copy の分類**: target 不在での place-once コピー、記録済み target の no-op（place-once）、
  foreign 実ファイルの skip + 警告、構造不一致（src dir × target file / src file × target dir）
  両方向の conflict、記録済みディレクトリの no-op
- **除去順序**: ディレクトリ移行の事前除去がボトムアップ順で組まれること
- **未知の method**: 未定義の method がエラーとして弾かれること
- **祖先が symlink でない場合**: 実ディレクトリ祖先が symlink 祖先と別分類になること

なお同じ table には backup 有効時の分類（退避先の衝突・実 dir 全体の退避など）のブロックも
あるが、これは `atomicity` 対象の担当であり、この CASE では数えない。
