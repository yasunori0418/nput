---
id: "CASE-3b8d14a5-192b-46ba-94bd-ce1e78c4e8de"
type: test_case
name: "version_test.go — バージョンの供給元一致と --version の出力書式"
target: "cmd/nput/version_test.go"
covers:
  - "TC-0c7281f8-cffc-4a53-8fe0-f52d9ea6bd12"
  - "TC-b4a365cd-0b1c-40ff-92c6-91f6cffe8e98"
---
# CASE-3b8d14a5-192b-46ba-94bd-ce1e78c4e8de: version_test.go — バージョンの供給元一致と --version の出力書式

## 対象

`cmd/nput/version_test.go`

## 検証内容

- **埋め込み前の既定値** — リンク時の埋め込みが無いビルドで既定の値になること（テスト実行時は
  埋め込みを行わない前提を明示している）
- **供給元の一致** — CLI フレームワーク側のバージョンと内部のバージョン変数が同一であること。
  機械可読出力のツールバージョンがここから供給されるため、ドリフトすると文書の中身がずれる
- **出力書式** — `--version` の実出力が既定のテンプレートどおりの 1 行であること（独自
  テンプレートの混入を検出する）
- **提供しないものの固定** — 同名のサブコマンドが存在せず、フラグとしてのみ提供されること
