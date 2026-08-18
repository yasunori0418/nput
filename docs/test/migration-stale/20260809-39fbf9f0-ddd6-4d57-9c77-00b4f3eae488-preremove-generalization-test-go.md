---
id: "CASE-39fbf9f0-ddd6-4d57-9c77-00b4f3eae488"
type: test_case
name: "internal/engine/preremove_generalization_test.go — レイアウト移行と method 変更"
target: "internal/engine/preremove_generalization_test.go"
covers:
  - "TC-b9d4ffaf-ac91-4bf1-9f27-5ea3964466ad"
  - "TC-76597b11-8199-4682-8f53-008e5208bd9c"
  - "TC-8a1f4b19-0ece-4057-a8bf-f9717855cade"
  - "TC-810d661d-6d3d-4199-828f-e44adcebad5a"
---
# CASE-39fbf9f0: preremove_generalization_test.go

## 対象

`internal/engine/preremove_generalization_test.go`

配置前除去の適用範囲が「自己記録の stale な祖先 symlink」から「配置 target を占有する自己記録
stale な FS オブジェクト全般」へ一般化されたこと（実ディレクトリ全体・method 変更）を、
実 tmpdir 上の Apply で確認する。

## 主な検証内容

- **per-file ↔ dir symlink の移行**: 新旧 target が leaf 名を共有するケース（2026-07-12 の
  実インシデント再現。readlink パターンの cleanup が誤判定する形）を 1 回の apply で移行し、
  除去・剪定の報告が各ディレクトリにつき 1 回ずつであること。dir symlink → per-file →
  dir symlink の 3 世代往復が各世代で収束すること
- **移行の全か無か**: 移行可能な記録済み symlink の兄弟に foreign 実ファイルが 1 つ混ざる
  だけで conflict 停止し、移行可能な兄弟が一つも除去されないこと。非空サブディレクトリが
  plan 段階で conflict になること
- **空サブツリー**: 多階層の空サブツリー（nput が作ったものでなくても可）が移行対象になること、
  root 直下の実 dir target も移行されること
- **method 変更**: symlink → copy は自動移行（記録済み symlink を配置前除去して place-once
  copy を配置）、copy → symlink は通常の上書き拒否 conflict のまま copy を保持、記録済み
  symlink が drift していた場合は移行せず foreign 扱いへ落ちること
- **drift のエラー化**: 配置前除去の rmdir / unlink 経路それぞれで drift が直接エラーになること
- **中断後の収束**: 配置前除去まで進んで中断した状態から再実行して目標状態へ収束すること
