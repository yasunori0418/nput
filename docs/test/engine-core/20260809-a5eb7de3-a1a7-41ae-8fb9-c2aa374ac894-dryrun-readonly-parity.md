---
id: "TC-a5eb7de3-a1a7-41ae-8fb9-c2aa374ac894"
type: test_condition
name: "dryrun が副作用を持たず、本番 apply と同数の conflict を報告すること"
mitigates:
  - "RISK-33e30498-6fa1-450b-a251-5624cbe837b9"
---
# TC-a5eb7de3-a1a7-41ae-8fb9-c2aa374ac894: dryrun の無副作用と本番との conflict 件数の一致

`apply --dryrun` が FS へ一切の変更を残さないこと（配置物・profile ディレクトリのいずれも
生成しないこと）を検証する。副作用は成功時に表面化しないため、実行前後の FS 状態を比較する形で
条件を立てる。

同じ入力に対して dryrun と本番 apply が同数の conflict を報告することを併せて検証する。
祖先移行・実ディレクトリ移行のように plan が事前除去を伴う分類でも、dryrun が除去を実行せずに
同じ判断へ至ることを含む。現状の照合は件数までで、どの target が conflict になったかの集合
一致までは見ていない（RISK-33e30498-6fa1-450b-a251-5624cbe837b9 が恐れる乖離に対しては部分的な被覆にとどまる）。

conflict 検出時の終了コード（CI のゲートとして機能すること）は CLI 面の契約であり、
`cli-json` 対象の担当。ここでは engine が返す conflict の報告までを見る。
