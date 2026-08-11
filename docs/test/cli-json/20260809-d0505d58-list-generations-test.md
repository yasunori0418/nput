---
id: "CASE-d0505d58-ef48-4fe2-ab7c-77002109d1c4"
type: test_case
name: "list-generations_test.go — 世代一覧の info 表現と二重エンコードの禁止"
target: "cmd/nput/list-generations_test.go"
covers:
  - "TC-4e0a14d6-342c-448b-964b-b8e87520e89b"
  - "TC-cf8189c4-d680-4d7e-bcd3-810543762c50"
---
# CASE-d0505d58: list-generations_test.go — 世代一覧の info 表現と二重エンコードの禁止

## 対象

`cmd/nput/list-generations_test.go`

## 検証内容

- **世代一覧の表現** — 世代が番号・日付・現行フラグを持つ行として `info` に載ること。日付は
  取得元の出力をそのまま載せること。現行フラグは偽のときもキーを省略しないこと
- **二重エンコードの禁止** — 同じ世代番号を結果の世代スロットへ重ねて載せないこと。個別の
  item を作らず空配列のままであること
- **`info` の境界** — 列挙に到達せず失敗した実行では `info` がキーごと不在であること、
  0 件で列挙した実行では空配列がキーごと残ること

`info` の境界の扱いはパス列挙の CASE と対称であり、読み取り系コマンド全体で同じ規約が成り立つ
ことをこの 2 ファイルで押さえている。
