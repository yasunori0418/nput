---
id: "TC-f304d682-bc5e-4369-a427-f766fdfdd961"
type: test_condition
name: "project mode では世代を公開するコマンドが拒否され理由が利用者へ届く"
mitigates:
  - "RISK-cdcc6faf-9164-409b-b584-2921fa036d10"
---
# TC-f304d682: project mode では世代を公開するコマンドが拒否され理由が利用者へ届く

## 条件

世代をユーザーへ見せるコマンド（`rollback` / `list-generations`）が、project mode の
config に対して実行できないことを確かめる。engine 層が観測値の nil を正しく運ぶかどうかとは
別に、公開そのものを CLI が入口で止めているかを見る。

- **拒否が成立する** — project mode の config を名指した実行が成功せず、一般エラーの終了
  コードで止まること。世代の一覧を出さないこと、および配置物が巻き戻らないこと
- **理由が利用者へ届く** — エラーが home mode 限定である旨を述べること。および拒否の判断に
  使った root の種別を示すこと。利用者が「コマンドが無い」ではなく「この config の root では
  使えない」と読み取れることがこの条件の一部である
- **全件の列挙にも現れない** — 全 config を対象とする読み取り専用の列挙が、project mode の
  config を対象に含めないこと。名指しの拒否とは別機構（profile の配置構造による除外）で
  成り立つため、名指しの拒否が通っても退行しうる

拒否の判断は entrypoint を評価して root の種別を解決した後に効く。実際に評価を通す経路で
確かめる必要があり、評価を伴わない層では観測できない。

> **被覆の範囲**。同じ拒否は fixed root にも効くが、fixed root の e2e fixture がリポジトリに
> 無いため、本条件が実際に覆うのは project mode に限る。fixed は未被覆の残件である。

> **区分の選択**。Issue #284 の本文は追補先を cli-json 区分としていたが、拒否の実経路が
> 評価を通す e2e 01-project にしか無いため、その CASE が属する integration 区分へ置いた
> （同 issue の設計決着コメントによる読み替え）。
