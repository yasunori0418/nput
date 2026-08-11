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
  コードで止まること。世代の情報を一切出力しないこと。名指しした config の配置が拒否の
  前後で変わらないこと
- **理由が利用者へ届く** — エラーが home mode を要する旨を述べること。および拒否の判断に
  使った root の種別を名指すこと。利用者が「コマンドが無い」ではなく「この config の root では
  使えない」と読み取れることがこの条件の一部である

拒否の判断は entrypoint を評価して root の種別を解決した後に効く。実際に評価を通す経路で
確かめる必要があり、評価を伴わない層では観測できない。

> **被覆の範囲**。3 つの残件がある。(1) 同じ拒否は fixed root にも効くが、fixed root の e2e
> fixture がリポジトリに無いため、本条件が実際に覆うのは project mode に限る。(2) 全 config を
> 対象とする列挙が project mode の config を含めないことは、home mode の config が実在する
> 配置でしか意味のある観測にならない（project mode だけの環境では「除外が効いた」と「元から
> 空」を区別できない）ため、本条件には含めず home mode 側の担当に残す。(3) 名指しの拒否が
> engine へ到達していないことは、project mode が generation skip で世代を 1 つしか持たない
> ため配置の不変からは示せない。到達の検知も世代を複数持つ home mode 側の担当になる。

> **区分の所在**。この条件は entrypoint の評価を通す経路でしか観測できず、その経路を持つ
> テスト資産は e2e 01-project だけである。その CASE が属する integration 区分に置く。
