---
id: "ADR-0048"
type: adr
name: "ドキュメントは sara でグラフ構造化する — model.yaml の全面置換・UUIDv4 二層 ID・CI 非必須開始"
status: 採用
issues:
  - "#203"
  - "#206"
  - "#207"
origin: "epic #203 の grilling セッション（2026-07-31・2026-08-01）で確定"
justifies:
  - "INF-659b139d-0cf8-4c65-b30d-93c5ee2dfc71"
  - "TP-d7da4065-ce7c-4a0b-be49-5108256e177a"
references:
  - "ADR-0030"
  - "ADR-0037"
---
# ADR-0048: ドキュメントは sara でグラフ構造化する — model.yaml の全面置換・UUIDv4 二層 ID・CI 非必須開始

- ステータス: 採用
- 日付: 2026-08-02
- 関連: ADR-0030, ADR-0037, ADR-0049, ADR-0050, `docs/spec.md`, `docs/design.md`, `docs/concept.md`, `docs/adr/README.md`, GitHub Issue #203, #206, #207
- 改訂対象: なし（新領域。「仕様とテストの全体像把握」= quality-observability（→ Issue #203）の実現手段を
  差し替えるが、既存 ADR の決定を反転しない）
- 起点: epic #203 の grilling セッション（2026-07-31・2026-08-01）で確定

> **2026-08-08 改訂注記（ADR-0049）**: 改訂対象は本 ADR **§2 の型定義表（10 型）だけ**。§2 のそれ以外の
> 決定（組み込みモデルの全面置換・`hardware_requirement` 等 4 型を定義しない判断・組み込み型の改名表・
> 型名とフィールド名を汎用に保つ方針・`issues` / `origin` の text 逃がし・`status` の日本語 enum と
> `status_note`・「廃止」を当面持たない判断）は不変であり、§1 / §3 / §4 / §5 も全て不変。変わったのは
> 型の内訳と接続で、`quality`（QA）/ `test_plan`（TP）を新設して 12 型となり、`infrastructure` は root
> ではなく quality / design を `satisfies` する型へ、`design` の親は requirement / test_plan へ、
> ADR の `justifies` 対象は requirement / design / infrastructure / quality / test_plan へそれぞれ
> 拡張された。§2 の表をそのまま実装に使わないこと（→ ADR-0049）。

> **2026-08-09 改訂注記（ADR-0050）**: 改訂対象は本 ADR **§5 の `strict_mode = false` の決定だけ**。
> epic #203 の移行完了（warning 0 到達）により保留を解き、`strict_mode = true` へ変更した。
> §5 のそれ以外の決定（required status check に登録しない・DoD に追加しない・CI job のローカル
> filter）は不変で、§1〜§4 も全て不変（→ ADR-0050）。

> **2026-08-09 改訂注記（ADR-0051）**: 改訂対象は本 ADR **§2 の型定義表のうち defect（D）の行だけ**。
> defect 型は廃止され、発生した欠陥は GitHub Issues（`bug` label）で管理する。テスト系統のグラフは
> `risk → test_condition → test_case` で終端し、relation `reveals` / `is_revealed_by` も削除された。
> §2 のそれ以外の決定と §1 / §3 / §4 / §5 は不変（→ ADR-0051）。

## 背景

nput の仕様・設計・テストは `docs/spec.md`（1200 行超）/ `docs/design.md`（550 行）/ `docs/concept.md`
（400 行）と 47 本の ADR に散文で書かれている。この形では「要求とリスクの紐付けが一貫しているか」を
機械検証できない。

先行して PR #206 が grep / awk による ID 突合（`tests/traceability/run.sh`）を試作したが close 済みで、
main には存在しない。#206 自身が挙げた最大の穴は **参照先 ID の実在を検証しないこと**である。下流成果物への
`grep` で判定するため、存在しない `TC-999` を書いても検出されずカバー率に計上される。

#206 は sara（https://github.com/cledouarec/sara）を実機検証し、テスト工程の文書管理としては `run.sh` より
強いことを確認していた（壊れた参照 `CASE-002 → TC-999` と階層違反をいずれも exit=1 で検出）。一方でこう
結論していた。

> ID 総数 120 前後をすべて個別ファイルへ分解する必要があり、現在の散文中心の成果物構造とは相性が悪い。
> sara の素直な適用先は ADR 47 本。

## 決定

### 1. sara を採用し、#206 の「相性が悪い」評価を意図的に覆す

ドキュメントを sara のナレッジグラフとして構造化する。#206 の評価は *現在の散文構造を維持する前提* での
ものであり、本決定は**その構造自体を変える**（`docs/spec.md` 等を item へ分割する）。分解コストは承知の
うえで受け入れる。

理由: 主目的である「要求とリスクの紐付けの一貫性管理」はファイル粒度では達成できない。ファイル単位の item
では「`spec.md` にテストがある」までしか言えず、「REQ-07 のテストが無い」を検出できない。

これに伴い、#206 が保留した判断（「`run.sh` に参照先実在チェックを足して独立に切り出すか否か」）は
**切り出さない**で決着する。sara が同等以上を提供するため。

元の 3 文書は捨てず、「全体像 + item へのリンク集」へ縮退させる（各 100 行程度）。通読の入口を残し、
DOD-04（3 文書が実装に追従）をそのまま維持するため。50 ファイルを `sara query` で辿るのは通読の代替に
ならない。

### 2. `docs/model.yaml` で組み込みモデルを全面置換する

sara のカスタムスキーマは組み込みモデルを完全置換する仕様で、部分マージはされない。`sara schema -o` で
吐いた組み込みモデルを起点に、nput の文書構造へ合わせて 10 型を定義する。

| 型 | prefix | 親 |
|---|---|---|
| solution | SOL | なし（根） |
| use_case | UC | solution |
| requirement | REQ | use_case |
| design | DSG | requirement |
| infrastructure | INF | なし（根） |
| adr | ADR | なし（分離型） |
| risk | RISK | requirement / design |
| test_condition | TC | risk |
| test_case | CASE | test_condition |
| defect | D | test_case |

組み込みの `hardware_requirement` / `hardware_detailed_design` / `scenario` / `system_architecture` は
**定義しない**。nput にハードウェアは無く、Scenario / SystemArchitecture に対応する実体（use_case と
requirement の中間層・requirement と design の中間層）も持たない。

残る組み込み型は汎用名へ改名して引き継ぐ。

| 組み込み | 本モデル |
|---|---|
| `system_requirement` / `software_requirement` | `requirement` |
| `software_detailed_design` | `design` |
| `architecture_decision_record` | `adr` |

nput にはハードウェア / ソフトウェアの区別が無く、system / software の 2 段も持たないため、
1 つの `requirement` / `design` に統合する。

型名・フィールド名は**汎用のまま保つ**（nput 固有の語彙を混ぜない）。当面 nput 内に置くが、型が安定したら
skills リポジトリか専用リポジトリへ切り出して横展開する前提のため。切り出しに備えて設計意図を
`docs/model.yaml` のコメントに残す。

sara の item にできない外部参照は text フィールドへ逃がす（adr の `issues: !list text` / `origin: text`）。

adr の `status` は実データの表記に合わせ、`!enum {提案, 採用}` の**日本語値**とする。本 ADR を含む
全 48 本が `採用` で、`accepted` 等の英語表記は 1 本も無い。組み込みの `proposed` / `accepted` /
`deprecated` をそのまま使うと #208（既存 47 本への frontmatter 追加）が全件翻訳になるため採らない。

カッコ書きの改訂注記（例: 「採用（2026-06-14 改訂: … → ADR-0015）」）は enum 値に収まらないので、
`status_note: text` を受け皿として別に持つ。注記を持つのは #208 が扱う既存 47 本のうち 10 本。この注記は
`docs/adr/README.md` が「正しい運用例」として明示的に保護している情報であり、enum 化で捨ててはならない。

`deprecated` 相当の「廃止」は**当面値に持たない**。§3 の通り `supersedes` を定義していない現状で廃止
ステータスだけを先に置くと、「旧 ADR は読まなくてよい」という誤読を招く。`supersedes` の追加を再検討する
時点で「廃止」も併せて足す。

### 3. `supersedes` は当面定義せず、部分改訂を表す `revises` を独自に定義する

組み込みの `supersedes` / `superseded_by` は model.yaml から落とす。**恒久的な排除ではない。**

`supersedes` は「文書丸ごとの失効」を意味するが、nput の ADR 改訂は全て**節単位の部分改訂**で、丸ごと
失効した実例が無い（改訂実態の調査 2026-08-01: 最も近い ADR-0033 ← ADR-0043 でも、タイトルになった
決定自体は存続している）。実例の無い関係を先に定義すると、部分改訂に `supersedes` を誤用して「旧 ADR は
読まなくてよい」と読者に誤認させる。これは `docs/adr/README.md` が防ごうとしている失敗そのものである。

代わりに、ADR ヘッダの「改訂対象:」に対応する `revises` / `is_revised_by`（peer）と、「関連:」に対応する
`references` / `is_referenced_by`（peer）を独自に定義する。前者は旧 ADR 側の blockquote 注記と対になり、
`docs/adr/README.md` が記録した注記漏れ問題（ADR-0023 / 0024 / 0025 / 0028 / 0031 と対象旧 ADR の間）を
機械検証できるようにする。

**全節が実質失効した ADR が現れた時点で、`supersedes` +「廃止」ステータスの追加を再検討する。**

リスクとテストの系統には `threatens` / `mitigates` / `covers` / `reveals`（いずれも逆方向つき）を独自定義する。
risk は requirement と design の**両方**を脅かせる。使い分けは「その設計を別案に置き換えれば消えるリスクのみ
design 側。要求が満たされないこと自体への懸念は常に requirement 側」とする。

### 4. ID は UUIDv4 の二層構成とする（ADR のみ連番維持）

sara が扱う正式 ID はフル UUIDv4、人間が触る面は前方 8 文字の省略形を使う。

| 用途 | 形式 | 例 |
|---|---|---|
| 正式 ID（frontmatter `id:`・relation リスト） | `<PREFIX>-<フル UUIDv4>` | `REQ-8521e8f8-99cc-47f0-b7d0-70c24c837612` |
| ファイル名 | `<YYYYMMDD>-<前方8文字>-<slug>.md` | `20260801-8521e8f8-lock-ordering.md` |
| 散文中の参照 | `<PREFIX>-<前方8文字>` | `REQ-8521e8f8` |

- 乱数 ID により**並列レーン（parallel-worktree）での採番衝突を構造的に回避する**。連番は複数レーンが
  同時に「次の番号」を取ると必ず衝突するが、フル UUIDv4（乱数 122bit）は事実上衝突ゼロ
- 省略形は正式 ID の前方一致なので、8 文字で grep すれば宣言側・参照側の両方にヒットする
- **ULID / UUIDv7（時刻順序付き）を採用しない**。先頭が時刻部のため、(1) 8 文字省略形が同一秒のバッチ生成で
  全て同じ文字列になり衝突する、(2) フル形でも先頭が似通い diff レビューでの目視比較が効かない。v4 は
  先頭から乱数なので両方成立する。時系列の並びは ID ではなくファイル名の日付プレフィックスが担う
- **ADR のみ連番を維持する**（`ADR-NNNN`）。既存 47 本の相互参照・`docs/adr/README.md` の運用・Issue 言及を
  壊さないため
- 8 文字 prefix の偶然重複（120 item で約 10⁻⁶）は採番コマンドの重複チェックが再生成で潰す

sara の ID 検証は「非空」「英数字 / ハイフン / アンダースコア」のみで、prefix 一致も `id_format` も
検証しない（`sara-core/src/model/item.rs` 実装確認済み）ため、フル UUID のハイフンはそのまま通る。
代償として `sara init` の自動採番（`suggest_next_id`）は使えない（連番前提のため）。devShell 同梱の
`sara-id` コマンドがこれを代替し、`uuidgen -r`（util-linux）で生成して 8 文字 prefix の重複を `docs/` の
1 回走査で検出し、既出なら生成し直す。

### 5. CI は非必須チェックから開始する

`sara check` を CI に追加するが、**required status check には登録せず、DoD にも追加しない**。
`strict_mode = false` とし、orphan は警告に留める。検出対象は broken refs / duplicate ID / cycles。

- 移行中は「テスト未着手の要求」「まだ親を持たない item」が正常に存在するため、orphan をエラーにすると
  移行そのものが進まない
- 閾値ゲートを持たない点は `go-coverage` と同方針（→ ADR-0030）。他 PR とのマージ順依存を避ける
- DoD の残り枠は温存する。strict 化・必須チェック化は移行完了後に判断する

既存の `changes` job は再利用しない。あの filter は nix / Go ソースを対象にしており、`sara check` が最も
効くのは docs-only PR であるため。代わりに job 内へローカルな filter（`docs/**` / `sara.toml` / `dev/**` /
`test.yml`）を置き、無関係な PR では nix のセットアップごと skip する。

同じ job で `dev/tests/sara-id.sh`（採番契約のテスト）も実行する。`checks.sara-id` として
`nix flake check ./dev` にも載せているが、CI の `flake-check` job はルート flake を対象にするため
dev の checks は CI に載らない。テストが「人が思い出したときだけ動く」状態を避けるための明示ステップ。

## 根拠

### なぜ item 分割まで踏み込むのか

#206 の「散文構造と相性が悪い」は正しい観測だが、それは *現在の構造を維持する前提* の話である。本 ADR が
主目的に置く「要求とリスクの紐付けの一貫性管理」はファイル粒度では原理的に達成できない。ファイル単位の
item では「`spec.md` にテストがある」までしか言えず、「REQ-07 のテストが無い」を検出できない。検出できない
なら導入する意味が無く、逆に検出したいなら構造を変えるしかない。分解コストはこの帰結として受け入れる。

### なぜ `id_format` を書くのか（検証されないのに）

sara は `id_format` を検証しない（`sara-core/src/model/item.rs` 実装確認済み）。それでも各型に書くのは、
model.yaml 単体が ID 規約のドキュメントとして読まれるため。§2 が前提とする他プロジェクトへの切り出し後は、
この ADR が付いてこない。実効性が無いことはファイル冒頭と `solution` 型のコメントで明示する。

### なぜ CI を非必須で始めるのか

必須にすると、移行の途中段階（親を持たない item・テスト未着手の要求が正常に存在する状態）でマージが
止まる。移行そのものを止める仕組みは移行の役に立たない。`go-coverage` が閾値ゲートを持たない理由
（→ ADR-0030）と同根で、他 PR とのマージ順依存も避けられる。

## 影響

- `docs/model.yaml` / `sara.toml` が追加され、`docs/` 配下が sara のスキャン対象になる。frontmatter の
  無い `.md` は `ParseResult::Skipped` で静かに無視される（`sara-core/src/repository/scanner.rs` 確認済み）
  ため、README / 概要文書 / test-plan 等への除外設定は不要
- devShell に `sara-id` が加わる（`uuidgen -r` を供給する util-linux は `sara-id` の `runtimeInputs`
  として wrapper の PATH に前置されるため、devShell 側には載せない）。CI 用に `devShells.sara` を分けており、
  docs-only PR で nput のビルドと dogfood の shellHook を走らせない
- `docs/adr/README.md` の「ADR は **supersede しない**」という断定表記は、本 ADR の §3（当面使わない・
  恒久排除ではない）と趣旨を揃えるため緩和が必要。この改訂は #208 のスコープで行う
- 移行は epic #203 の 7 段階（#207 → #213）で進める。段階 2（#208・ADR 47 本への frontmatter 追加）が
  単独で最大の即効性を持つ
- skills リポジトリ側の改修（テスト成果物の恒久化・全 ID のファイル分割・UUIDv4 採番手順など）は
  **nput 先行・skills は後追い**とし、移行の実地で必要な改修を確定させてから着手する。本 ADR のスコープ外

## 棄却した代替案

### `run.sh`（grep / awk 突合）に参照先実在チェックを足して独立に切り出す

#206 が保留していた案。sara が同等以上（broken refs に加えて duplicate ID / cycles / 階層違反）を提供し、
自前実装の保守を抱えずに済むため採らない。#206 自身が sara を実機検証して「テスト工程の文書管理としては
`run.sh` より強い」と確認している。

### 組み込みモデルをそのまま使う

`hardware_requirement` / `hardware_detailed_design` は nput に対応する実体が無く、`scenario` /
`system_architecture` も中間層として実在しない。空の型が並ぶと、後任が「ここを埋めるべきか」を毎回判断する
コストを負う。カスタムスキーマは組み込みを完全置換する仕様なので、部分的に残す選択肢も無い。

### `supersedes` を最初から定義しておく

改訂実態の調査（2026-08-01）で、文書丸ごとの失効に該当する ADR は 1 本も無かった。実例の無い関係を先に
定義すると、部分改訂に誤用されて「旧 ADR は読まなくてよい」という誤読を生む。これは `docs/adr/README.md`
が防ごうとしている失敗そのもの。必要になった時点で足すほうが安全（→ §3）。

### ID に ULID / UUIDv7（時刻順序付き）を使う

先頭が時刻部になるため、(1) 8 文字省略形が同一秒のバッチ生成で全て同じ文字列になり衝突する、
(2) フル形でも先頭が似通い diff レビューでの目視比較が効かない。v4 は先頭から乱数なので両方成立する。
時系列の並びは ID ではなくファイル名の日付プレフィックスが担えば足りる（→ §4）。

### ADR も UUID へ移行する

既存 47 本の相互参照・`docs/adr/README.md` のセルフチェック運用・Issue 本文での言及がすべて連番前提で、
移行すると壊れる範囲がリポジトリ外（Issue / PR）にまで及ぶ。連番の弱点（並列レーンでの採番衝突）は
ADR の作成頻度では実害にならない（→ §4）。

### `status` に組み込みの英語 enum（`proposed` / `accepted` / `deprecated`）を使う

実データは全 48 本が `採用` で英語表記は 1 本も無く、採ると #208 が全件翻訳になる。さらに既存 47 本のうち
10 本が持つカッコ書きの改訂注記が enum 値に収まらず、退避先を各ファイルで判断することになる（→ §2）。

## 保留

- **glossary の item 化**。対象は 2 系統ある。`docs/glossary.md`（22 用語・英語の正本）と対訳の
  `docs/glossary.ja.md`（22 用語）は `sara.toml` のスキャン範囲内にあり、frontmatter を持たないため
  現状は `Skipped` で無視される。`CONTEXT.md`（21 用語）はリポジトリルートにありスキャン範囲外。
  用語間参照と用語 → ADR の出典を機械検証できる展望はあるが、分割すると「一覧して語彙を身につける」
  用語集の機能を失う。用語は最も安定した資産で後から移行しても失うものが少なく、100 item を作る過程で
  判断材料が得られる
- **strict 化・必須チェック化**（§5）
- **`model.yaml` の切り出し**（§2）
- **`supersedes` の追加**（§3）

## 後日の追記: sara 0.10.0 で `id_format` が検証されるようになった（2026-08-16）

§「なぜ `id_format` を書くのか（検証されないのに）」の前提は sara 0.10.0 で失効した。
breaking change `feat!: drive identifier generation from id_format templates` により
`id_format` は採番・補完・検証を駆動する SSOT になり、パースできないテンプレートは
schema のロード時に拒否される。

このリポジトリでは `{prefix}-{uuid}` が未知のプレースホルダとして弾かれ、schema 全体の
ロードが失敗 → 組み込みモデルへフォールバック → カスタム型（test_condition / test_case 等）が
全て `unknown item type` になり 376 パスが skip される、という形で顕在化した。
`{uuid}` → `{uuid4}` の置換で解消（ADR の `{prefix}-{seq:04}` は 0.10.0 でも有効）。

本文の「検証しない」という記述は 0.9.x 時点の事実として残す。現在の規範は
`docs/model.yaml` 冒頭の「ID 形式」節が持つ。
