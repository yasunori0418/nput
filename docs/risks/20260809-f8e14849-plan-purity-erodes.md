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

DSG-836aa5cb へ張るのは、「planner と engine を分けて fake FS で table-driven に検証する」が
一つの設計選択だからである。engine を分割せず実 FS 統合テストだけで検証する設計へ差し替えれば
この乖離という失敗モードは消える（→ `docs/agents/sara-graph.md` の判別規約）。層分けそのものへの
懸念は設計に依らず残るため、REQ-c5dfcae6 へも張る。
