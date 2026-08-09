---
id: "ADR-0051"
type: adr
name: "defect 型を廃止する — 欠陥は GitHub Issues で管理し、グラフは規範のみを持つ"
status: 採用
issues:
  - "#268"
  - "#203"
  - "#267"
origin: "Issue #268（epic #203 残ブロッカー）の判断（2026-08-09）で確定"
justifies:
  - "INF-659b139d-0cf8-4c65-b30d-93c5ee2dfc71"
revises:
  - "ADR-0048"
  - "ADR-0049"
  - "ADR-0050"
---
# ADR-0051: defect 型を廃止する — 欠陥は GitHub Issues で管理し、グラフは規範のみを持つ

- ステータス: 採用
- 日付: 2026-08-09
- 関連: ADR-0048, ADR-0049, ADR-0050, GitHub Issue #268, #203, #267
- 改訂対象: ADR-0048 の型テーブルから defect を除去（12 型 → 11 型）。ADR-0049 §1 の
  defect 例外の記述と再検討条件「defect の `is_revealed_by`」を解消。ADR-0050 の
  再検討条件のうち defect 起因のトリガー（最初の defect item での strict 衝突）を消滅
- 起点: Issue #268「defect の upstream relation（is_revealed_by）の要否を判断する」

## 背景

defect は ADR-0048 で ISTQB のテスト実行・欠陥管理工程に対応する型として定義されたが、
他の型と異なり `allowed_targets` が空で自分から upstream relation を張れず、親側の
test_case が downstream の `reveals`（primary）で接続する唯一の構造だった。この構造は
3 つの例外として顕在化していた（→ Issue #268）:

1. defect item は作った時点で必ず orphan warning になる（model.yaml 冒頭の既知例外）
2. strict_mode（ADR-0050）稼働後は defect 新設が即 `sara check` exit 1 になり、
   テスト工程の着手自体をブロックする
3. 型関係図の機械生成（#267）で `CASE -->|reveals| D` だけが「upstream の辺のみ描く」
   規約から漏れ、生成側に特例が要る

Issue #268 は「`is_revealed_by` を足すか・primary をどちらに置くか・足さない場合の代替」
の 3 論点を挙げ、最初の defect item を起こす前の判断を必須としていた。

## 決定

**defect 型を廃止し、発生した欠陥は GitHub Issues で管理する。**

1. **model.yaml から defect 型を削除する**（12 型 → 11 型）。使い手がいなくなる
   relation 定義 `reveals` / `is_revealed_by` と、test_case の allowed_target
   `reveals → defect` も同時に削除する。グラフのテスト系統は
   `risk → test_condition → test_case` で終端する
2. **欠陥は GitHub Issues（`bug` label）で管理する**。トレーサビリティは片方向で、
   欠陥 issue 本文に発見元の `CASE-xxxxxxxx` と遡及先の REQ / DSG を記載する。
   docs 側から issue への参照は張らない（欠陥発生のたびに docs へ PR を出さない）
3. **フィードバックループを規約化する**。欠陥 issue のクローズ時に「この欠陥を事前に
   捕まえられたはずの risk / test_condition の欠落」を検討し、欠けていれば item を
   追補する。運用規約の本体は `docs/agents/issue-tracker.md` の「Defect issues」節が持つ
4. Issue #268 の論点 1・2（`is_revealed_by` の追加・primary の向き）は型の廃止により
   消滅する。#267 の図生成特例も不要になる

## 理由

このプロジェクトのドキュメント管理は、要求・設計から始まりリスク・テスト観点を
シフトレフトで管理する**予防の体系**であり、グラフが持つのは規範（あるべき姿）である。
defect は他の型と唯一異なり「規範」ではなく「発生した出来事の記録」で、出来事の記録・
状態遷移・修正作業の追跡は issue tracker の領分。この区別に沿えば defect をグラフに
置く必然性は無く、構造例外（orphan・strict 衝突・図生成特例）はこの不整合の症状だった。

型の廃止により「接続漏れ = orphan warning」の不変条件が**例外なしに全型で成立**し、
ADR-0050 の strict_mode とも無条件に整合する。

## 検討した代替案

### `is_revealed_by` を defect の allowed_targets へ足す（#268 論点 1 の採用側）

orphan 例外と strict 衝突は解消するが、「defect は test_case が発見するもの（下流からの
発見）」という意味論を defect 側から親を張る形に反転させ、かつ出来事の記録をグラフに
持ち込む不整合自体は残る。GitHub Issues との二重管理も解消しない。

### GitHub Issue を一次とし defect item をグラフ接続用スタブとして残す

item の存在意義が「issue への転送」だけになり、欠陥発生のたびに docs へ PR を出す
運用コストと二重管理が残る。トレーサビリティは issue 本文の CASE / REQ / DSG 記載
（片方向参照）で足りる。

### strict_mode を解除する（#268 論点 3・ADR-0050 の再検討条件どおり）

defect 1 型のために全型の機械的担保を弱める本末転倒。採らない。

## 影響

- `docs/model.yaml`: defect 型・`reveals` / `is_revealed_by` の削除、冒頭コメントの更新
- `CLAUDE.md`・`docs/agents/domain.md`: 型テーブルから defect 行を削除、orphan 例外の
  記述を撤去
- `docs/agents/issue-tracker.md`: 「Defect issues」節を新設
- Issue #268 は本 ADR でクローズ。#267 へ特例消滅を通知
- テスト工程着手時に作る成果物は risk / test_condition / test_case の 3 型となる
