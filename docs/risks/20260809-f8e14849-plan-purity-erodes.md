---
id: "RISK-f8e14849-3bff-4755-b75f-c80b85990ce1"
type: risk
name: "plan と engine が乖離し、table-driven テストが実挙動を保証しなくなる"
threatens:
  - "DSG-836aa5cb-0389-4adb-990b-144fe5aeffe3"
  - "REQ-c5dfcae6-6094-4850-99e5-bf14530bc60a"
likelihood: medium
impact: medium
level: medium
---
# RISK-f8e14849: plan と engine が乖離し、table-driven テストが実挙動を保証しなくなる

配置ロジックは planner（manifest 2 世代 + FS プローブから plan を計算する純粋な層）と engine
（plan を実 FS へマテリアライズする層）に分かれており、分類の網羅は fake FS を使った
table-driven テストに重点配分されている。この分割は、planner が実 FS を触らないことと、engine が
planner の分類をそのまま実行することの両方に依存する。

planner が FS を直接触るようになれば table-driven テストは fake FS 越しに実挙動を測れなくなり、
engine が独自に分類をやり直せば table のケースが増えても実挙動は変わらない。どちらも
「テストは緑だが本番は壊れている」状態を作る。設定の誤りは評価時・実体の不整合は engine 実行時、
という層分けが崩れる場合も同じ経路で顕在化する。

## 想定する失敗

- planner が実 FS へ直接アクセスし、fake FS でのテストが実経路から外れる
- engine が plan を無視した独自判断で配置し、planner のテストが実挙動を担保しなくなる
- 未知の method が plan 計算を素通りし、engine 側で未定義の振る舞いになる
- 実体の不整合の検出が評価時へ漏れる／評価時に検出すべき設定の誤りが engine まで届く

## 張り先の判断

DSG-836aa5cb へ張るのは、「engine のテストを分類の table へ重点配分する」が一つの設計選択で
あり、それを実 FS 統合テストだけで検証する設計へ差し替えればこの乖離という失敗モードは消える
からである（→ `docs/agents/sara-graph.md` の判別規約）。

ただしこの張り先は消去法でもある。本来この risk が懸念しているのは planner（純粋計算）と
engine（FS 実行）という層分割そのものだが、その分割を記述した design item は現状存在せず、
DSG-836aa5cb 自身も planner 抽出より前の記述のまま「FS をモックしない」と述べていて現況と
食い違っている（分割の設計 item 新設と DSG-836aa5cb の追随は `docs/design/` の担当で、
本レーンの境界外）。したがってここで拾えているのは「table-driven テストが実挙動を保証しなく
なる」という失敗モードに限られる。

なお、FS に触れずに判断する層をユニットレベルで table-driven に覆うことは TP-e7c25263 が
テスト計画として規範化している。欠けているのは設計側の記述であって、規範そのものではない。

REQ-c5dfcae6 へも張るが、対応するのは「想定する失敗」の 4 点目（評価時に検出すべき設定の誤りと
engine 実行時に検出すべき実体の不整合の境界がずれる）だけである。同 REQ が規定するのは評価時と
実行時の層分けであって、planner / engine というプロセス内の層分けではない。
