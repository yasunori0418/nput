---
id: "TC-9504e908-a4bc-4d2d-80d3-af07264284f8"
type: test_condition
name: "undo の逆操作が、対応する前方操作の直前状態を単体で復元する"
mitigates:
  - "RISK-68e810c5-4e68-4b25-9bc0-6b2613022b49"
---
# TC-9504e908-a4bc-4d2d-80d3-af07264284f8: undo の逆操作が、対応する前方操作の直前状態を単体で復元する

## テスト条件

undo ジャーナルの各エントリ種別に対応する逆操作を、`Apply` を通さず個別に駆動し、
前方操作の直前状態が復元されることを検証する。

| 逆操作 | 対応する前方操作 | 復元すべき状態 |
|---|---|---|
| unlinkNew | 新規 symlink 配置（PlaceNew）| symlink が無い状態 |
| relinkOld | 張替え（PlaceReplace / PlaceForeign）・stale 除去・PreRemove の Unlink | 旧 dest を指す symlink |
| removeCopy | copy のマテリアライズ | copy tree が無い状態 |
| restoreRename | `--recopy` の rename 退避 | 退避元パスに退避前の内容 |
| mkdir | PreRemove の Rmdir | 空ディレクトリの再作成 |

undo ジャーナルは 6 種目として `--backup` の退避戻しを持つが、これは
`restoreRename` と実装（rename で戻す case 分岐）を共有するため単体駆動のテストを
持たず、`Apply` を通した end-to-end で覆う（TC-ed4992c0-8513-4383-be0a-e45acbbc229f の担当）。

前方操作の対象が既に消えている場合（別プロセスに動かされた・そもそも配置に到達して
いなかった）に逆操作がエラーにならないことも条件に含む。「消すものが無い」「unlink すべき
現物が無いが再作成はできる」は復元の成功であり、失敗として扱えば RISK-a1622cdd-c7e2-4178-a50d-85bc2a35b29f の
「1 件の失敗が全体を道連れにする」を自ら誘発する。

上位の規範は TP-deb05610-44bc-4962-8939-952392e5fbd0（原子性の故障注入体系）と TP-e7c25263-6d2d-4a37-8275-26906889d912（`internal/engine/` の
実 FS 統合レベル）。

## 対応する CASE

CASE-02475ac2-5555-4fa1-a3e5-cda5015919c5（`internal/engine/undo_test.go`）。
