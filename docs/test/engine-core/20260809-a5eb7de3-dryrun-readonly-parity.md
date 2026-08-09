---
id: "TC-a5eb7de3-a1a7-41ae-8fb9-c2aa374ac894"
type: test_condition
name: "dryrun が副作用を持たず、本番 apply と同じ conflict を報告すること"
mitigates:
  - "RISK-33e30498-6fa1-450b-a251-5624cbe837b9"
---
# TC-a5eb7de3: dryrun の無副作用と本番との一致

`apply --dryrun` が FS へ一切の変更を残さないこと（配置物・profile ディレクトリ・state 配下の
いずれも生成しないこと）を検証する。副作用は成功時に表面化しないため、実行前後の FS 状態を
比較する形で条件を立てる。

同じ入力に対して dryrun が報告する conflict 集合が本番 apply の検出する集合と一致することを
併せて検証する。祖先移行・実ディレクトリ移行のように plan が事前除去を伴う分類でも、dryrun が
除去を実行せずに同じ判断へ至ることを含む。
