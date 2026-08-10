---
id: "RISK-33e30498-6fa1-450b-a251-5624cbe837b9"
type: risk
name: "dryrun が副作用を残す、または本番 apply と異なる結果を報告する"
threatens:
  - "REQ-02a33511-0941-4813-b289-a05eb8e9aa57"
  - "REQ-7a71a049-5876-4cfc-a65e-44e9a0349856"
likelihood: medium
impact: high
level: high
---
# RISK-33e30498: dryrun が副作用を残す、または本番 apply と異なる結果を報告する

`apply --dryrun` はユーザーが「実行前に安全を確認する」ための唯一の手段であり、2 つの性質に
依存している。読み取り専用であること（FS も flock も pending gcroot も触らない）と、報告する
conflict / change が本番 apply の結果と一致することである。どちらが崩れても、dryrun は
「安全確認」の役に立たなくなり、むしろ確認したつもりで壊す経路になる。

副作用側の失敗は特に見えにくい。dryrun が profile ディレクトリを作る・flock を取る・gcroot を
張る、といった副作用は成功時には表面化せず、並行実行や read-only 環境で初めて露見する。

## 想定する失敗

- dryrun が FS へ書き込み、確認だけのつもりの実行が状態を変える
- dryrun が flock を取り、確認中に本番 apply がブロックされる
- dryrun が報告した conflict と本番 apply が検出する conflict が食い違う
- conflict を検出しても終了コードが 0 のままで、CI のゲートとして機能しない

## 評価

- likelihood: medium — TC-a5eb7de3 が副作用の不在と conflict 件数の一致を覆っているが、
  dryrun は本番 apply と同じ経路を分岐で共有するため、apply 側に段を足すたびに
  「dryrun でも走ってしまう」形の混入が起こりうる。加えて同 TC は照合が件数までで
  「どの target が conflict になったか」の集合一致は見ていないことを自ら残余として挙げて
  おり、「想定する失敗」4 点目の終了コードは RISK-bd1e4ada 側が持つ
- impact: high — 副作用側の破れは沈黙の不整合になる。確認だけのつもりの実行が profile を
  作る・flock を取るといった変化は成功時に表面化せず、並行実行や read-only 環境で初めて
  露見する

## 張り先の判断

2 本とも requirement へ張る。「dryrun は読み取り専用で conflict 時に非ゼロ終了する」は
CLI の観測可能な契約であり、内部でどう plan を計算するかを差し替えても懸念は残る。
