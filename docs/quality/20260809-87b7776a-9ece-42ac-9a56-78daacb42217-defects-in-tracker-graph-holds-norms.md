---
id: "QA-87b7776a-9ece-42ac-9a56-78daacb42217"
type: quality
name: "欠陥はトラッカーが持ちドキュメントグラフは規範のみを持つ。起票は分類語彙を経由する"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  A defect found in the product SHALL be recorded in the issue tracker and SHALL NOT be
  recorded as an item in the document graph, which holds norms rather than events. The
  tracker's reference to the items a defect traces back to SHALL be one-way; items SHALL
  NOT link back to defects. Before a defect is closed, whether a missing risk or test
  condition would have caught it earlier SHALL be asked, and any item found missing SHALL
  be added, so that the risk and test analysis does not decay into a static snapshot. An
  issue SHALL be classified by the project's fixed vocabulary, and on the filing routes
  where the tracker can enforce this, filing without it SHALL NOT be offered.
specification_ja: |
  製品に見つかった欠陥はイシュートラッカーに記録されなければならず、ドキュメントグラフの
  item として記録してはならない（グラフが持つのは出来事ではなく規範であるため）。欠陥が
  遡及する item へのトラッカー側の参照は片方向でなければならず、item 側から欠陥へ張り返して
  はならない。欠陥をクローズする前に、それをより早く捕まえられたはずの risk または
  test_condition の欠落がなかったかを問わなければならず、欠落が見つかった item は追加され
  なければならない（リスク分析とテスト分析が静的なスナップショットへ劣化しないようにする
  ため）。issue はプロジェクトが定める分類語彙によって分類されなければならず、トラッカーが
  それを強制できる起票経路では、分類を経ない起票が提供されてはならない。
---
# QA-87b7776a: 欠陥はトラッカーが持ちドキュメントグラフは規範のみを持つ。起票は分類語彙を経由する

## 仕様

グラフが持つのは「そうあるべき姿」であり、欠陥は「起きた出来事」になる。両者を同じ場所に
混ぜると、規範の集合に時系列の記録が紛れ、規範だけを読みたい参照が汚れる。所在をトラッカーと
グラフで分ける。

**参照を片方向にする**のが要点になる。トラッカー側から遡及先の item を名指しするのは追跡に
必要だが、item 側から欠陥へ張り返すと、規範の文書が個々の出来事の履歴を抱え込み、欠陥が
解決した後も残る。グラフは規範だけを持つという性質が、参照の向きで守られる。

**クローズ前のフィードバック**を規範に含めるのは、欠陥を直すだけでは分析が更新されないため。
その欠陥をより早く捕まえられたはずの分析の欠落を毎回問い、欠けていた item を足すことで、
リスクとテストの分析が最初に書いた時点のまま固まるのを防ぐ。

起票時の分類は、未分類の issue が滞留するのを防ぐために求める。**強制できる面では強制し、
できない面では運用規範として求める**という二段になるのは、起票経路によってトラッカーが介入
できる度合いが違うため。フォームを経由する経路は未分類の起票を提供しない形にでき、コマンド
経由の起票はそうできない。後者を理由に前者の強制を緩めることはしない。

語彙の具体（ラベル名と各ラベルの意味）・欠陥 issue に書く項目の様式・トラッカーの操作手順は
本 item の規範に含めない。`docs/agents/triage-labels.md` と `docs/agents/issue-tracker.md` が
持つ。規範がグラフ側だけに置かれることの裏返しの規範（規範をグラフが単独で持つこと）は
QA-67715bb3 が持つ。

## 出典

ADR-0051（欠陥は型として持たず GitHub Issues で管理する）が置いた方針と、
`docs/agents/issue-tracker.md` の Defect issues 節（片方向参照・クローズ前の shift-left
ループ）、`docs/agents/triage-labels.md` の分類語彙、`.github/ISSUE_TEMPLATE/config.yml` の
blank issue 禁止設定が実運用してきた規範を、Issue #272 で quality item として立てたもの。
