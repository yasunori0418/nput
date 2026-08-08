---
id: "ADR-0049"
type: adr
name: "型構成を 12 型へ改訂する — quality / test_plan の新設・infrastructure の再接続・design→test_plan の satisfies"
status: 採用
issues:
  - "#237"
  - "#203"
origin: "epic #203 の grilling セッション（2026-08-08）で確定"
justifies:
  - "INF-659b139d-0cf8-4c65-b30d-93c5ee2dfc71"
revises:
  - "ADR-0048"
---
# ADR-0049: 型構成を 12 型へ改訂する — quality / test_plan の新設・infrastructure の再接続・design→test_plan の satisfies

- ステータス: 採用
- 日付: 2026-08-08
- 関連: ADR-0048, GitHub Issue #237, #203
- 改訂対象: **ADR-0048 §2 の型定義表だけ**（10 型 → 12 型）。§2 のそれ以外の決定（組み込みモデルの
  全面置換・落とした 4 型・改名表・型名の汎用性維持・`status` の日本語 enum と `status_note` 等）は
  不変で、§1 / §3 / §4 / §5 も全て不変
- 起点: epic #203 の grilling セッション（2026-08-08）で確定

## 背景

ADR-0048 §2 は 10 型（solution / use_case / requirement / design / infrastructure / adr /
risk / test_condition / test_case / defect）を定義した。移行が `docs/spec.md` の item 分割まで
進んだ段階で、この構成には受け皿の欠けがあることが判明した。

`sara check` は warning 12 件（orphan な ADR 8 件 + requirement 4 件）で pass していた。この orphan は
張り忘れではなく、**受け皿となる型が無い**ことに起因していた。テスト計画・品質方針にあたる文書が
requirement として置かれ、「ユーザーがどう使うか」を述べる use_case を親に持てずに宙に浮いていた。

同時に infrastructure が root（親を持たない型）として定義されていたため、`docs/infrastructure/` の
item は全件が構造上 orphan であり、warning が「接続漏れの信号」として機能していなかった。

orphan 判定の仕様は sara v0.9.4 のソースで確認済みである。判定は **`direction: upstream` の
relation を実際に 1 本でも張っているか**だけで決まり（`graph/knowledge_graph.rs` の `orphans()`、
`model/item.rs` の `has_upstream()` / `ItemType::is_root()`）、型単位・item 単位で warning を
抑止する設定は存在しない（`validation/rules/orphans.rs` の `validate()` は config を受け取るが
参照していない）。黙らせる唯一の方法は upstream relation を実際に張ることである。

## 決定

### 1. 型を 12 型へ拡張し、root を solution と adr のみに絞る

ADR-0048 §2 の表を次で置き換える。

| 型 | prefix | 親 |
|---|---|---|
| solution | SOL | なし（根） |
| use_case | UC | solution |
| requirement | REQ | use_case |
| design | DSG | requirement / test_plan |
| quality | QA | solution |
| test_plan | TP | solution |
| infrastructure | INF | quality / design |
| adr | ADR | なし（分離型） |
| risk | RISK | requirement / design |
| test_condition | TC | risk |
| test_case | CASE | test_condition |
| defect | D | test_case |

これにより **「接続漏れ = orphan warning」という不変条件が全型で成立する**。root は solution と
adr のみで、他の型は upstream relation を 1 本以上持てる。

唯一の例外は defect で、`allowed_targets` が空のため upstream relation を張る手段が無い（親側の
test_case が downstream の `reveals` で接続する）。defect item は作った時点で必ず orphan になるため、
テスト工程に着手する段で `is_revealed_by` を defect の `allowed_targets` へ足すかを再検討する
（現時点で defect item は 0 件）。

本 ADR では **strict 化しない**（ADR-0048 §5 の方針を維持）。warning ゼロは実接続で達成する。
warning → error 昇格は `--strict` / `strict_mode` で可能だが**全 warning 一括の 0/1 スイッチ**で、
「orphan だけ CI ゲート」の粒度指定はできないため、移行途中で採ると移行そのものが止まる。

### 2. quality（QA）と test_plan（TP）を別の型として新設する

いずれも `solution` を親とし、upstream relation は `derives_from`、フィールドは requirement と同じ
`specification` / `specification_ja` を持つ。

- **quality**（`docs/quality/`）: 品質方針・規約・プロセス横断のガバナンス文書
- **test_plan**（`docs/test-plan/`）: ISTQB のテスト計画活動の成果物（テストスコープ・スコープ外
  宣言・テストレベル・テストアプローチ・テスト容易性の要求）

既存のテスト系 4 型（risk / test_condition / test_case / defect）は ISTQB のテスト分析以降に対応し、
**テスト計画に対応する型が空いていた**。移設対象の文書はいずれもテスト計画の成果物であるため、
test_plan がこれを受ける。

一方、品質方針・規約はテストプロセスの外にあるプロセス横断の関心事なので quality として分離する。
**両者を 1 型に混ぜない**のは、infrastructure が抱えていた「品質系と基盤系の混在」を新型内で
再発させるため。

`risk` の `parent_types` には **test_plan を追加しない**。risk の親は「リスクが何に対して発生して
いるか」を示す構造であり、requirement / design が正しい対応先である。テスト計画がどの risk を扱うかは
下流の test_condition → `mitigates` → risk で追跡できる。テスト工程が動いてから不足が判明した場合に
peer relation の追加を再検討する。

sara の RFC2119 キーワード検証は **requirement 型にハードコード**されており（`validation/rules/metadata.rs`）、
quality / test_plan の `specification` には効かない。英語の規範文で書く規約自体は `docs/requirements/`
と同じだが、機械検証されないためレビューで守る。

### 3. infrastructure を再定義し、root から外す

定義を「プロダクトを**開発・提供・稼働**させる技術基盤」へ拡張する。CI/CD・リリース・開発環境に
加え、配信・ホスティング・クラウド（Cloudflare / AWS / Vercel 等）を含む。web アプリケーションを
想定範囲に含めた横展開を見据えた拡張である。

`allowed_targets` へ upstream relation `satisfies`（対象: `quality` / `design`）を追加し、root から
外す。**dev 基盤は quality を、runtime 基盤は design を satisfy する**。

**型名・prefix は変更しない**。`INF-<UUID>` は ADR の `justifies` から参照されており、改名は
ID 変更 = 全参照の張り替えを伴う。違和感の主因だった品質系との混在は quality 型の分離で解消した。

### 4. design の `satisfies` 対象へ test_plan を追加する

design の `satisfies` は ADR-0048 では requirement のみを対象としていた。テスト計画系 requirement の
移設対象 4 件のうち 3 件は design 側から `satisfies` で参照されており、参照元は design item 6 件・
辺の総数は 6 本である。しかも DSG-2947b4a5 と DSG-901351ea は **`satisfies` がその 1 本だけ**である。
対象を拡張しないまま移設すると、今度は design 側が orphan 化して epic #203 の完了条件（warning 0）が
成立しない。

意味づけとしても「テストハーネスの実装形を表す design は、requirement ではなくテスト計画を満たす
実体」であり、`satisfies` の語義と整合する。移設側は 6 辺を新 TP-ID へ張り替えるだけで済む。

### 5. ADR の `justifies` 対象型へ quality / test_plan を追加する

ADR は requirement / design / infrastructure / quality / test_plan へ任意に接続する。ADR-0048 §2 が
定めた model.yaml では前 3 型のみが対象だった（§2 の本文は `justifies` の対象型に触れていない。
対象型は model.yaml の `adr` 型の `allowed_targets` が持つ）。

## 根拠

### なぜ「orphan warning を抑止する」ではなく「型を増やす」で解いたのか

sara には型単位・item 単位で orphan warning を抑止する設定が無い（`validation/rules/orphans.rs`
実装確認済み）。仮にあったとしても採らない。orphan warning は「接続漏れ」の唯一の信号であり、
受け皿の無さを抑止設定で隠すと、本物の張り忘れも同じ静けさの中に埋もれる。

型を増やす側の代償は「型が増える」ことだけで、これは受け皿が実在しないという事実の反映である。
テスト計画・品質方針の文書は既に存在しており、型が無いのは構造の欠落であって単純化ではない。

### なぜ quality と test_plan を分けたのか

1 型に混ぜれば型数は 11 で済むが、それは infrastructure が抱えていた混在をそのまま新型へ移すだけで
ある。infrastructure の違和感の主因は「品質系（DoD・トレーサビリティ検証）と基盤系（CI/CD・配信）が
同じ型に同居していた」ことで、この分離こそが §3 で infrastructure を再定義できた前提になっている。
同じ間違いを新設時点で繰り返す理由が無い。

分離の切り口は ISTQB のテストプロセスに置いた。test_plan は「テスト計画活動の成果物」という工程上の
位置で定義でき、既存のテスト系 4 型（テスト分析以降）と連続する。quality はその工程の外にある
プロセス横断の関心事で、工程上の位置を持たない。

### なぜ infrastructure を root から外せるようになったのか

ADR-0048 の時点では infrastructure が satisfy できる相手が design しか無く、dev 基盤（CI・トレーサ
ビリティ検証）は design ツリーに属さないため root にせざるを得なかった。quality を新設したことで
dev 基盤の接続先ができ、2 経路（quality / design）で solution ツリーへ届くようになった。

つまり §2 の型分離が §3 の前提であり、逆順では成立しない。

### なぜ design の orphan 化を許容しなかったのか

design 側を orphan のまま放置する案・`satisfies` とは別の専用 relation を新設する案の 2 つを検討した。
前者は epic #203 の完了条件（warning 0）と両立しない。後者は「satisfies と意味が重なる関係を 2 本持つ」
ことになり、どちらを張るかの判断コストを毎回払うことになる。

## 影響

本 ADR は #237（型構成の改訂本体）のマージ後に、その決定を記録するため後追いで起票した。したがって
影響のうち型定義側は既に適用済みで、item 側の移設が残っている。

適用済み:

- `docs/model.yaml` が 12 型構成になり、`dev/flake.nix` の `sara-id` prefix マップへ
  `quality | qa) prefix=QA` / `test_plan | test-plan | tp) prefix=TP` が追加された。`CLAUDE.md` の
  ディレクトリ表・`docs/model.yaml` 冒頭コメント・epic #203 本文の mermaid モデル図も 12 型構成へ
  更新済み
- `docs/test-plan/` が新設され、テスト計画にあたる requirement 4 件が TP item として移設された。
  §4 で述べた design → test_plan の 6 辺も新 TP-ID へ張り替え済み

未了:

- **`docs/quality/` は未作成で、`type: quality` の item は 0 件**。品質方針にあたる requirement の
  移設は epic #203 の後続 issue が担当する
- **infrastructure の 6 件は orphan warning を出し続けている**。root から外れた一方で `satisfies` の
  張り替えが未了なためで、これは意図した中間状態である。runtime 基盤にあたるものは design へ即座に
  張れるが、dev 基盤にあたるものは張り替え先の quality item がまだ無く、上の `docs/quality/` 新設を
  待つ。いずれも解消は後続 issue が担当する
- `CLAUDE.md` の散文（ディレクトリ表の直後にある「quality / test_plan は item もディレクトリも
  まだ無い」の記述）が表の更新に追随していない。test_plan は既に実在するため未着手なのは quality
  だけである旨へ直す必要がある

## 棄却した代替案

### quality と test_plan を 1 型（例: `policy`）にまとめる

型数は 11 で済むが、混在の解消が §3 の前提になっているため同じ変更の中で矛盾する。
→ 根拠「なぜ quality と test_plan を分けたのか」

### infrastructure を root のまま残す

`docs/infrastructure/` の item が構造上ずっと orphan であり続け、「接続漏れ = orphan warning」の
不変条件が全型では成立しない。orphan warning に 1 件でも構造的な常在ノイズがあると、本物の
張り忘れがその中に埋もれる。→ 根拠「なぜ infrastructure を root から外せるようになったのか」

### infrastructure を改名する（例: `platform`）

品質系との混在という違和感の主因は quality 型の分離で解消済みで、改名は残った表記の問題だけを
扱う。一方で `INF-<UUID>` は ADR の `justifies` から参照されており、改名は ID 変更 =
全参照の張り替えを伴う。得られるものに対して壊す範囲が大きい。

### risk の `parent_types` へ test_plan を追加する

risk の親は「リスクが何に対して発生しているか」を示す構造であり、テスト計画はリスクの発生元では
なくリスクへの対処側である。テスト計画がどの risk を扱うかは下流の test_condition →
`mitigates` → risk で既に追跡できる。→ §2

### design → test_plan に専用 relation を新設する

`satisfies` と意味が重なる関係を 2 本持つことになり、design item を書くたびにどちらを張るかの
判断コストが発生する。「テストハーネスの実装形はテスト計画を満たす」は `satisfies` の語義に
収まる。→ §4

## 保留

- **strict 化・必須チェック化**（→ ADR-0048 §5）。warning ゼロを実接続で達成したあとに判断する
- **defect の `is_revealed_by`**（§1）。テスト工程に着手する段で再検討する
- **risk への peer relation 追加**（§2）。テスト工程が動いてからテスト計画との紐付けに不足が
  判明した場合に再検討する
