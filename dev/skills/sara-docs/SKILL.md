---
name: sara-docs
description: sara ドキュメントグラフ（docs/ 配下の 1 ファイル 1 item・frontmatter relation・sara CLI による機械検証）の基盤知識スキル。sara 管理下の docs/ で item（solution / use_case / requirement / design / quality / test_plan / infrastructure / adr / risk / test_condition / test_case）を新規作成・編集・削除する、relation を張る・張り替える、グラフを辿る・検証する、ID を採番する、といった操作の前に必ず参照する。工程スキル（/test-plan・/test-analyze 等）が前提とする型・relation・ID・specification の規約はここが正本。sara を使わないリポジトリの Markdown ドキュメント一般は対象外。
license: MIT
disable-model-invocation: false
user-invocable: true
---

# sara-docs: sara ドキュメントグラフの基盤規約

sara（Requirements Knowledge Graph CLI）で管理する `docs/` の読み書きに共通する規約を
1 箇所に集めた基盤スキル。工程スキル（/test-plan・/test-analyze 等）はこの規約を前提に
書かれており、ここと矛盾する出力をしない。

## 前提

- **sara は必須**。このスキルの対象リポジトリは `sara.toml` + `docs/model.yaml` を持ち、
  `sara check` が通ることを常時の不変条件とする（strict 有効なら orphan も error）。
  item を書き込んだら必ず `sara check` を回して green を確認してから完了とする。
- **1 ファイル 1 item**。工程単位の固定文書（`test-plan.md` のような 1 本もの）は作らない。
  全体像は文書ではなく `sara query` / `sara report matrix` で辿る。
- モデルは次の **11 型** に固定する（model.yaml がこの構造であることを前提にする）。

## 型セット

| 型 | prefix | ディレクトリ | 親（upstream relation）|
|---|---|---|---|
| solution | `SOL` | `docs/solution/` | なし（根）|
| use_case | `UC` | `docs/use-cases/` | solution（`refines`）|
| requirement | `REQ` | `docs/requirements/` | use_case（`derives_from`）|
| design | `DSG` | `docs/design/` | requirement / test_plan（`satisfies`）|
| quality | `QA` | `docs/quality/` | solution（`derives_from`）|
| test_plan | `TP` | `docs/test-plan/` | solution（`derives_from`）|
| infrastructure | `INF` | `docs/infrastructure/` | quality / design（`satisfies`）|
| adr | `ADR` | `docs/adr/` | なし（分離型。`justifies` で 1 本以上接続）|
| risk | `RISK` | `docs/risks/` | requirement / design（`threatens`）|
| test_condition | `TC` | `docs/test/…` | risk（`mitigates`）|
| test_case | `CASE` | `docs/test/…` | test_condition（`covers`）|

テスト系 item（TC / CASE）を `docs/test/` 配下でどう区分けするか（サブディレクトリの
切り方）と、CASE の `target`（テスト資産の正準表記）の規則はプロジェクト規約
（CLAUDE.md 等）に委ねる。

親を持たない根は solution と adr のみ。他の型は upstream relation を 1 本以上張る
（張り忘れは strict check の orphan error になる）。frontmatter は手書きせず
**`sara init` で生成する**（relation・型別フィールドはオプションで渡す。手順は
[references/sara-cli.md](references/sara-cli.md)）。型・フィールドの正本は各リポジトリの
`docs/model.yaml`（`sara schema` で確認できる）。

## forward の張り方（規範）

item は**上流から下流へ、親が先に在る順**で起こす。テスト系の系統なら:

1. REQ / DSG（仕様・設計）が在る
2. それを脅かす risk を起こし `threatens` を張る
3. risk を潰すテーマとして test_condition を起こし `mitigates` を張る
4. テーマを検証する実体として test_case を起こし `covers` を張る

relation は**起こした側（下流）の frontmatter に宣言する**（`sara check` の JSON も
宣言辺だけを持つ）。前工程の item が無いまま下流を起こさない — 無ければ前工程の実行を
提案して停止するのが工程スキルの原則。既存資産から下流を先に起こす逆算は規範外の
移行専用手順で、[references/reverse-import.md](references/reverse-import.md) に隔離する。

## ID 規約（UUIDv4 二層構成）

| 用途 | 形式 | 例 |
|---|---|---|
| 正式 ID（frontmatter `id:`・relation リスト）| `<PREFIX>-<フル UUIDv4>` | `REQ-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4` |
| ファイル名 | `<YYYYMMDD>-<前方8文字>-<slug>.md` | `20260802-2b0c2bb8-mkmanifest-pure-function.md` |
| 散文中の参照 | `<PREFIX>-<前方8文字>` | `REQ-2b0c2bb8` |

- 採番は **ADR を除き `sara-id` コマンド**（8 文字 prefix の重複チェック込み）。連番を手で
  振らない（並列レーンでの採番衝突を構造的に避けるため）。`sara-id <型名> [slug]` が
  `id:` / `filename:` / `ref:` の 3 行を返す。採番した正式 ID を `sara init` の `--id` へ
  渡して item を生成する（init 自身の自動採番は連番前提なので使わない）。
- **ADR のみ連番**（`ADR-NNNN`）を維持する。`sara-id ADR` は exit 2 で拒否する仕様なので、
  `docs/adr/` の最大値 + 1 を手で採る。
- relation リストにはフル ID を書く。散文では省略形を使う（省略形は正式 ID の前方一致
  なので、8 文字で grep すれば宣言側・参照側の両方に当たる）。

## risk の評価と優先度の導出

- **threatens の張り先は requirement が既定**。設計を差し替えたら消える懸念（設計選択に
  固有のリスク）だけを design へ張る。判断は risk 単位でなく edge ごとに行う。
- **risk の `level` は likelihood × impact から導出する**（手で判断しない）:

  | likelihood \ impact | high | medium | low |
  |---|---|---|---|
  | **high** | high | high | medium |
  | **medium** | high | medium | low |
  | **low** | medium | low | low |

  プロジェクトが独自の導出規約・契約テストを持つ場合はそちらが正本。
- **test_condition の優先度はフィールドにしない**。mitigates 先 risk の `level` から
  導出する（複数なら最高 level。例外は item 本文に根拠付きで記載）。

## specification 規約

`specification` フィールドを持つ型（requirement / quality / test_plan）は:

- **`specification` は英語**で書く。sara が RFC2119 キーワードの存在を検証するため。
  綴りは **SHALL 系に固定**（SHALL / SHALL NOT。MUST / MUST NOT は書かない。
  SHOULD / MAY はそのまま）。
- 対になる日本語の規範文を **`specification_ja` に併記**する。規範を述べる文は平叙形でなく
  規範助動詞で終える（SHALL = 〜しなければならない / SHALL NOT = 〜してはならない /
  SHOULD = 〜すべきである / MAY = 〜してもよい）。強度を 2 フィールド間でずらさない。

## 機械的検出の使い分け

- **`sara check`** — グラフの整合（broken reference・duplicate ID・cycle・orphan）。
  書き込みのたびに回す。green が完了条件。
- **`sara-gap`** — valid なグラフの**未カバー 3 段**（threatens されていない REQ/DSG・
  mitigates されていない risk・covers されていない TC）。工程の着手点リストとして使う
  （unthreatened = リスク識別の未着手、unmitigated = テスト分析の未着手、
  uncovered = テスト実装の未着手）。exit 0 = ギャップなし / 1 = あり / 2 = check 失敗。

CLI の詳しい使い方（query のフル ID 制約と省略形の解決手順を含む）は
[references/sara-cli.md](references/sara-cli.md)。
