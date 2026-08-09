---
id: "CASE-0c2cec08-db65-427c-9480-ac0dcacc8d31"
type: test_case
name: "internal/engine/dryrun_test.go — dryrun の無副作用と分類一致"
covers:
  - "TC-a5eb7de3-a1a7-41ae-8fb9-c2aa374ac894"
---
# CASE-0c2cec08: dryrun_test.go

対象: `internal/engine/dryrun_test.go`

`apply --dryrun` 相当の呼び出しが実 FS へ何も残さないこと、および事前除去を伴う分類でも
本番と同じ判断へ至ることを実 tmpdir で確認する。

## 主な検証内容

- **無副作用**: dryrun 実行後に配置物・profile ディレクトリ・state 配下のいずれも生成されて
  いないこと
- **conflict 一致**: dryrun が検出する conflict が本番 apply と同じであること
- **祖先移行の判断**: 自己記録 stale 祖先の移行を、実際に除去せず plan 上で判断すること
- **実ディレクトリ移行の判断**: 実 dir が占有する target の移行判断を、除去なしで行うこと
