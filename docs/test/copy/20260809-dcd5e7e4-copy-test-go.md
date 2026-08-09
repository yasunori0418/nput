---
id: "CASE-dcd5e7e4-9190-4b13-9521-9b8801ec85fb"
type: test_case
name: "internal/engine/copy_test.go — コピー経路の syscall 失敗注入"
covers:
  - "TC-4f315cfc-446c-4c3f-8dc9-c36b10073e9d"
  - "TC-d1eb1814-ac5e-4576-b092-7db4929fba43"
---
# CASE-dcd5e7e4: copy_test.go

対象: `internal/engine/copy_test.go`

コピー経路の各 syscall を実 FS の状態（通常ファイルで経路を塞いで ENOTDIR を誘発する・
ディレクトリを読み取り元にする等、root でも成立する条件）で失敗させ、エラーが伝播することを
確認する故障注入テスト。結果レコードが汚れないことは、
`Result` を経由する 2 経路（copy 配置・recopy）で併せて確認する。残る 5 本は `copyFile` /
`copyTree` / `copySymlink` のプリミティブを直接呼ぶため結果レコードを持たない。

## 主な検証内容

- **mkdir 失敗**: copy 配置時の親ディレクトリ作成失敗、ツリーコピー中のディレクトリ作成失敗
- **ファイル open 失敗**: 読み取り側 open の失敗、書き込み側 open-file の失敗
- **転送失敗**: read-write 中のコピー失敗
- **readlink 失敗**: symlink 複製時の readlink 失敗
- **recopy の lstat 失敗**: 非 ENOENT（経路上の通常ファイルによる ENOTDIR 等）が「target
  不在」と同一視されずエラーになること、およびこのとき `Recopied` / `Copied` のいずれにも
  記録が残らないこと
