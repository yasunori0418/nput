# ADR-0048: ドキュメントは sara でグラフ構造化する — model.yaml の全面置換・UUIDv4 二層 ID・CI 非必須開始

- ステータス: 採用
- 日付: 2026-08-02
- 関連: ADR-0030, ADR-0037, `docs/spec.md`, `docs/design.md`, `docs/concept.md`, `docs/adr/README.md`, GitHub Issue #203, #206, #207
- 改訂対象: なし（新領域。quality-observability の実現手段を差し替えるが、既存 ADR の決定を反転しない）
- 起点: epic #203 の grilling セッション（2026-07-31・2026-08-01）で確定

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

型名・フィールド名は**汎用のまま保つ**（nput 固有の語彙を混ぜない）。当面 nput 内に置くが、型が安定したら
skills リポジトリか専用リポジトリへ切り出して横展開する前提のため。切り出しに備えて設計意図を
`docs/model.yaml` のコメントに残す。

sara の item にできない外部参照は text フィールドへ逃がす（adr の `issues: !list text` / `origin: text`）。

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
- **ADR のみ連番を維持する**（`ADR-0044`）。47 本の相互参照・`docs/adr/README.md` の運用・Issue 言及を
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

`changes` job の gate には掛けない。あの filter は nix / Go ソースを対象にしており、`sara check` が最も
効くのは docs-only PR であるため。

## 影響

- `docs/model.yaml` / `sara.toml` が追加され、`docs/` 配下が sara のスキャン対象になる。frontmatter の
  無い `.md` は `ParseResult::Skipped` で静かに無視される（`sara-core/src/repository/scanner.rs` 確認済み）
  ため、README / 概要文書 / test-plan 等への除外設定は不要
- devShell に `sara-id` と util-linux が加わる。CI 用に `devShells.sara`（sara 単体）を分けており、
  docs-only PR で nput のビルドと dogfood の shellHook を走らせない
- `docs/adr/README.md` の「ADR は **supersede しない**」という断定表記は、本 ADR の §3（当面使わない・
  恒久排除ではない）と趣旨を揃えるため緩和が必要。この改訂は #208 のスコープで行う
- 移行は epic #203 の 7 段階（#207 → #213）で進める。段階 2（#208・ADR 47 本への frontmatter 追加）が
  単独で最大の即効性を持つ
- skills リポジトリ側の改修（テスト成果物の恒久化・全 ID のファイル分割・UUIDv4 採番手順など）は
  **nput 先行・skills は後追い**とし、移行の実地で必要な改修を確定させてから着手する。本 ADR のスコープ外

## 保留

- **glossary（`CONTEXT.md` 約 25 用語）の item 化**。用語間参照と用語 → ADR の出典を機械検証できる展望は
  あるが、分割すると「一覧して語彙を身につける」用語集の機能を失う。用語は最も安定した資産で後から移行しても
  失うものが少なく、100 item を作る過程で判断材料が得られる
- **strict 化・必須チェック化**（§5）
- **`model.yaml` の切り出し**（§2）
- **`supersedes` の追加**（§3）
