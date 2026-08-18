---
id: "CASE-82fdfb27-0bea-4833-b5e9-5f9e142a08df"
type: test_case
name: "internal/engine/copytree_test.go — 再帰コピーの構造・mode・symlink 再現"
target: "internal/engine/copytree_test.go"
covers:
  - "TC-b1b8c163-9d37-47ee-9838-7168569df03a"
---
# CASE-82fdfb27-0bea-4833-b5e9-5f9e142a08df: copytree_test.go

## 対象

`internal/engine/copytree_test.go`

コピーのプリミティブ（`copyTree` / `copyFile` / `copySymlink`）を実 tmpdir に対して直接
呼び、結果のツリーを lstat / readlink で確認する。engine の配置経路を通さないため、
コピー自体の忠実性だけを切り出して見る。

## 主な検証内容

- **構造の再現**: ネストしたディレクトリの構造保存、空ディレクトリ、ネストした空ディレクトリ
- **ディレクトリ属性**: ディレクトリ mode への owner-write 付与
- **symlink の非 deref**: ツリー内部の symlink を symlink のまま複製、トップレベルが symlink
  である src を `copySymlink` へ振り分け、リンク先文字列を保存
- **ファイル属性**: 元 mode を保存しつつ owner-write を付与、実行ビットの保存
- **symlink の上書き**: 既存 dst がある場合の symlink 複製
