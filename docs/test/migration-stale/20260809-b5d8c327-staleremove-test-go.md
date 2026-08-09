---
id: "CASE-b5d8c327-1f52-4566-98d3-22216d609d94"
type: test_case
name: "internal/engine/staleremove_test.go — drift 再検証と空親剪定"
covers:
  - "TC-8a1f4b19-0ece-4057-a8bf-f9717855cade"
  - "TC-810d661d-6d3d-4199-828f-e44adcebad5a"
  - "TC-d160e18b-4c0c-4531-a506-e7d00d88788a"
---
# CASE-b5d8c327: staleremove_test.go

対象: `internal/engine/staleremove_test.go`

`removeStale` / `preRemove` を Apply パイプラインを通さず直接駆動し、実 tmpdir 上で drift の
同値類と剪定の境界を確認する。除去アクションは記録上の dest と実 FS 上の target を独立に
組めるヘルパで構成し、両者の食い違いを作り分ける。

## 主な検証内容

- **drift の同値類**: readlink 不一致 / 実ファイルへの置換 / ディレクトリへの置換 / target
  消失。`removeStale` では保持 + 警告（エラーにしない）、`preRemove` では同じ 4 類すべてで
  エラー停止（子が無条件配置として計画済みのため skip できない）
- **drift 後の継続**: drift で保持したアクションが後続アクションの除去を止めないこと
- **unlink 失敗**: 不変条件は保たれているが `os.Remove` が失敗する経路（親 dir の書き込み権限
  剥奪で誘発・root では skip）で、警告ではなくエラーになり除去記録も残らないこと
- **空親剪定**: 多階層の空チェーンをまとめて剪定、非空祖先は保持、root 境界で停止、symlink
  祖先で停止、foreign 実ファイルを含む祖先は保持、剪定失敗は警告に留めて致命化しない
- **剪定の適用範囲**: 自己記録祖先の移行が外側の祖先まで剪定しないこと、`preRemove` は
  祖先を剪定しないこと、`reset` は剪定すること（copy のみの構成を含む）
