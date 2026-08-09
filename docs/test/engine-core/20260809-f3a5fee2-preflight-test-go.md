---
id: "CASE-f3a5fee2-1981-42c3-ba5b-81b75fd7a3ad"
type: test_case
name: "internal/engine/preflight_test.go — out-of-store 事前検査の非 ENOENT エラー"
covers:
  - "TC-405606f0-3ac8-4cc2-998e-d4759a62a171"
---
# CASE-f3a5fee2: preflight_test.go

対象: `internal/engine/preflight_test.go`

配置前検査が out-of-store marker のパスを stat する際、ENOENT（不在）以外のエラーを
握り潰さずエラーとして返すことを確認する。不在は「marker のパスが無い」というユーザー向けの
診断へ落ちるが、権限エラー等をそれと同一視すると原因を誤って伝える。

## 主な検証内容

- **非 ENOENT の伝播**: 権限エラー等の stat 失敗が不在扱いにされず、そのままエラーになること
