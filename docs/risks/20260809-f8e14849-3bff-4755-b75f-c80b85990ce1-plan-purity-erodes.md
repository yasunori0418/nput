---
id: "RISK-f8e14849-3bff-4755-b75f-c80b85990ce1"
type: risk
name: "plan と engine が乖離し、table-driven テストが実挙動を保証しなくなる"
threatens:
  - "DSG-8b96869c-842e-4f78-8ff7-df1f1d6c1a68"
  - "REQ-c5dfcae6-6094-4850-99e5-bf14530bc60a"
likelihood: medium
impact: high
level: high
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

## 評価

- likelihood: medium — TC-b329cafd が「plan の計算が実 FS を触らない」ことを fake FS の
  table-driven で覆っている。ただし層分割の維持は個々のアサートではなく実装の書き方に
  依存し、engine 側へ判断を足す変更で少しずつ崩れうる
- impact: high — 乖離の帰結は「テストは緑だが本番は壊れている」という検証の空洞化なので、
  規約の継承則に従い「そのゲートが捕まえるはずの regression の impact」を継ぐ。この
  table-driven テストが守っているのは配置分類（RISK-e8449214・high）で、そちらの破れは通常ファイルを
  symlink で上書きしてユーザーのファイルを消す形をとり、再実行では回復しない。層分割が開発内の
  事柄であることを理由に low へは落とさない——失われるのはテストではなくテストが与えていた保護
  である。そのテストが本来の検出力を失っていること自体は誰も知らせない

## 張り先の判断

DSG-8b96869c へ張るのは、この risk が懸念しているのが planner（分類判定に閉じた純粋計算）と
engine（FS 意味論の実行）という層分割そのものであり、同 item がその分割を主張しているから
である。分割を採らず engine が分類と実行を一体で持つ設計へ差し替えれば、この乖離という失敗
モードは消える（→ `docs/agents/sara-graph.md` の判別規約）。

以前は分割を記述した design item が存在せず、テスト戦略の DSG-836aa5cb へ消去法で張っていた
（#282）。DSG-8b96869c の新設と DSG-836aa5cb のモック方針の改訂（対象を FS の意味論を実行する
層に限り、planner は fake FS の table-driven で覆うと明示）により、張り先を分割そのものへ
移し、拾える失敗モードを「想定する失敗」の 4 点に戻した。

なお、FS に触れずに判断する層をユニットレベルで table-driven に覆うことは TP-e7c25263 が
テスト計画として規範化している。DSG-8b96869c はその規範が前提とする層分割を設計側で述べる
item であり、規範そのものではない。

REQ-c5dfcae6 へも張るが、対応するのは「想定する失敗」の 4 点目（評価時に検出すべき設定の誤りと
engine 実行時に検出すべき実体の不整合の境界がずれる）だけである。同 REQ が規定するのは評価時と
実行時の層分けであって、planner / engine というプロセス内の層分けではない。
