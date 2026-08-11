---
id: "CASE-f3a5fee2-1981-42c3-ba5b-81b75fd7a3ad"
type: test_case
name: "internal/engine/preflight_test.go — out-of-store 配置直前検査の非 ENOENT エラー"
target: "internal/engine/preflight_test.go"
covers:
  - "TC-8435052a-5dcc-49e2-ac26-82f645cb6890"
---
# CASE-f3a5fee2: preflight_test.go

## 対象

`internal/engine/preflight_test.go`

配置直前検査が out-of-store marker のリンク先を lstat する際、ENOENT（不在）以外のエラーを
不在と同一視しないことを確認する。不在は「dangling symlink を作らずに止める」というユーザー
向けの診断へ落ちるが、権限エラー等をそれと同一視すると原因を誤って伝える。

## 主な検証内容

- **非 ENOENT の分岐**: 検索権限を落とした親ディレクトリで誘発した EACCES が、「リンク先が
  存在しない」ではなく「リンク先を検査できない」として報告されること（root では skip）
- **原因の保存**: 返るエラーが元の権限エラーを包んでおり、`errors.Is` で辿れること
