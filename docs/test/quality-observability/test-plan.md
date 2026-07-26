# テスト計画: quality-observability

- 対象: 公開契約の台帳生成・検証基盤（`apps.qa-tracer` / `docs/quality/` / CI 検証 check）。
  `docs/dev/quality-observability/spec.md` の REQ-01〜REQ-10・NFR-01〜NFR-05
- 作成日: 2026-07-26
- 前提資料: `docs/dev/quality-observability/spec.md`（2026-07-26 改訂版）/
  `docs/dev/definition-of-done.md` / `docs/design.md`（「テスト戦略」節）/
  ADR-0027 / ADR-0030 / ADR-0031

## 1. 目的とスコープ

- 目的: 台帳が公開契約の母集団を正しく反映し、**腐敗（対応関係の欠落・参照切れ・生成物と
  コミット内容の乖離）を CI が確実に検出して main へのマージを止める**ことを確認する。
  台帳の内容が「テストの十分性」を表すかは検証対象としない（仕様のスコープ外）。
- スコープ内:
  - `apps.qa-tracer` の生成モード・検証モード（REQ-01〜REQ-08）
  - 抽出器 3 系統（cobra `Use:` / `lib/default.nix` の公開属性 / `modules/*.nix` の
    `mkOption`・`mkEnableOption`。数え方の規則は REQ-02 が定める）
  - **CI ゲート配線**（REQ-09）— `.github/actions/changes` への第 2 出力 `quality` の追加、
    `.github/workflows/test.yml` の job 追加、`main branch protection` ruleset への
    required status check 登録。別ファイル・別管轄だが検証の実行契機そのものであり、
    スコープに含める
  - **`docs/spec.md`**（変更検出の対象として。REQ-05 の見出し実在検証が見出し文字列に
    依存し、見出しの改名で検証が壊れるため。ただしアンカー ID の付与作業自体はスコープ外）
  - `docs/dev/definition-of-done.md` への DOD-05 追加（REQ-10）
  - `docs/design.md`「テスト戦略」節への追記（NFR-04）
- スコープ外:
  - 台帳が指すテストが当該機能を実際に検証しているかの判定（仕様のスコープ外。REQ-03 が
    明示する限界そのもの）
  - 既存の Go テスト 254 件・nix-unit 58 件・e2e 7 シナリオ・namaka スナップショット自体の
    再検証（今回の変更で触れない既実装）
  - `docs/spec.md` のアンカー ID 付与・全面 REQ 化（仕様のスコープ外。別 epic）
  - **テスト一覧（REQ-04 の出力）の鮮度維持の検証**。検証（REQ-05〜REQ-07）が対象とするのは
    台帳 `traceability.md` のみで、一覧は仕様上の検証対象ではない。`quality` フィルタは
    e2e シナリオの実体（`tests/e2e/scenarios/*.sh`）に反応しないため一覧は乖離しうるが、
    一覧の目的はテストの全体像の俯瞰であり、台帳と違いマージ可否のゲートに使わないため、
    鮮度の機械的な担保は行わない（生成時点の内容の正しさは AC-06 で検証する）

## 2. プロダクトリスク評価

プロダクトリスク（product risk）を発生可能性（likelihood）× 影響度（impact）で評価する。

| # | リスク項目 | 発生可能性 | 影響度 | リスクレベル | 根拠 |
|---|---|---|---|---|---|
| R1 | **台帳のみを変更した PR で検証 check がスキップされ、腐敗を検出できない** | 中 | 高 | **高** | 仕様は第 2 出力 `quality` によるゲート方式へ是正済み（REQ-09）だが、誤配線の余地が残る。既存 job のゲート方式は 2 通りあり（**`e2e` / `go-coverage` は job-level `if: needs.changes.outputs.run == 'true'`**〔`.github/workflows/test.yml:90` / `:63`〕、**`flake-check` は step-level `if`**〔`Setup Nix` :52 / `Run nix flake check` :58。job と matrix は常に走り常に報告する〕）、新設 job が誤って既存の `run` 出力で job-level ゲートすると、台帳のみを変更した PR（3 章 (a) のパターン ① 相当）で job ごと skip される。skip は required status check 上「成功扱い」（→ ADR-0030）のため、**誤配線が静かに成立し腐敗が素通りする**。変更検出の挙動は実 PR でしか確認できず、発見が遅れやすい |
| R2 | `quality` 出力の誤配線により、台帳のみを変更した PR で既存の nix job が余計に実走する | 中 | 中 | 中 | 仕様は「`relevant` 無改変 + 第 2 出力 `quality`」へ是正済み（REQ-09）。残るのは実装が誤って `relevant` 側へパスを足す・フィルタ対象を取り違える配線ミス。誤配線は台帳のみを変更した PR（3 章 (a) のパターン ① 相当）での **`e2e` / `go-coverage` の job 起動**、または **`flake-check` の `Run nix flake check` ステップ実走**として現れる（`flake-check` は job が常時起動するため、job の実行有無では観測できない）。ADR-0027 の「docs のみの変更で nix を実走させない最適化」の退行に当たる |
| R3 | 抽出母集団の件数が実装の規則逸脱により台帳と食い違う | 中 | 中 | 中 | 数え方の規則は REQ-02 へ明記済み（宣言のみ数える・submodule の親と葉は各 1 件・`modules/flake-parts.nix` の transposed オプションは除外）。残るのは抽出器の実装が規則から逸脱するリスクで、`inherit (lib) mkOption` の取り込み行や transposed オプションは素朴な走査で誤計上しやすい（`modules/*.nix` の素朴な走査は 7 ヒットし、規則適用後の正解 5 件と一致しない） |
| R4 | **生成の非決定性により REQ-07 の差分検証が恒常的に失敗する** | 中 | 高 | **高** | NFR-03 が REQ-07 成立の前提。抽出は 3 系統のファイル走査で、**glob の列挙順・連想配列の反復順**（実装言語が Go なら map の反復順。言語は基本設計で確定）のいずれかがソートされていないと出力が揺れる（Nix の attrset は評価時に属性名の辞書順へ正規化されるため要因に含めない）。揺れると**全 PR が赤**になり、検証基盤自体が信頼を失って無効化される（回復コストが高い） |
| R5 | 「仕様参照」列の解決不能により台帳の大半が `なし` になり、腐敗検出の実効性が下がる | 高 | 低 | 中 | `docs/spec.md` にはアンカー ID が存在しない。CLI 6 件は `### サブコマンド体系` のコードブロック内、モジュールオプション 5 件は `### 共通オプション（全モジュール）` に集約され、個別見出しが無い。lib のみ見出しで解決可能だが `### \`lib.projectRoot\` / \`lib.homeRoot\` / \`lib.systemRoot\`` が 3 API を 1 見出しに束ねる。**`なし` の多発は仕様どおりの正しい観測結果**（記述不足の可視化が本機能の目的）なので影響度は低いが、REQ-05 の見出し実在検証が実質 lib 系統にしか効かない点は認識が要る |
| R6 | namaka スナップショットテストがテスト一覧から漏れる | 低 | 中 | 低 | REQ-04 へ種別 `namaka` を明記済み（`tests/namaka/manifest-project/` が実在し root `flake.nix` の `checks.namaka` として配線）。残るのは実装が種別を取りこぼす通常の実装バグで、AC-06 の判定点で検出できる |
| R7 | 検証成功時に標準出力へ出力が出て ADR-0031「成功時は沈黙」に反する | 中 | 低 | 低 | NFR-01・AC-17 が明示。差分表示・進捗表示の実装で漏れやすいが、検出も修正も容易 |
| R8 | 生成モードが `nix eval` に依存し NFR-02 に反する | 低 | 中 | 低 | `lib/default.nix` の公開属性抽出は Nix 評価を使いたくなる誘引があるが、AC-04 で明示的に禁じられており設計段階で気づく。なお `dev/flake.nix` の `devShells.default` は Go を持つが jq / python を持たず、`devShells.ci` は jq を持つが Go を持たない。実装言語の選定制約として基本設計へ送る |

- 評価基準: リスクレベル = 発生可能性 × 影響度（高い順に優先）。R1〜R3・R6 は 2026-07-26 の
  仕様改訂（REQ-02 / REQ-04 / REQ-09）反映後の**残存リスク**として評価している。
- 計上前に該当経路のコード・設定ファイルを読み、発生し得ることを確認した。R1 / R2 は
  `.github/actions/changes/action.yaml` と `.github/workflows/test.yml` の実物（ゲート方式の
  行番号まで特定）、R3 は `modules/*.nix` の走査結果、R5 は `docs/spec.md` の見出し構成、
  R6 は `tests/namaka/` の実在と root `flake.nix` の `checks.namaka` 配線で裏取りした。

## 3. テストアプローチ

リスク評価を根拠に、どこを厚く/薄くテストするかを決める。

### 重点領域（高リスク: R1・R4）

- **CI ゲートの実効性（R1 / R2 / AC-14 / AC-15）** — 最重点。R1（高）と、その対をなす
  R2（中）を同一の観測で検証する。判定点を 2 系統に分ける。

  **(a) 実行 / skip の観測** — PR の変更パターン 4 分岐で、ゲート方式ごとに観測粒度を
  変える。パターンは次に固定する: **① `docs/quality/traceability.md` のみ変更**（AC-15
  準拠。`quality` フィルタに該当する docs のみの変更）/ **② nix・Go のみ変更** /
  **③ ①②の混在** / **④ `quality` フィルタに該当しない docs のみ変更**（例:
  `docs/concept.md`）。④ は ① と期待が逆で、新設 check が **skip される**のが正しい挙動
  であり、job が `quality` 出力でゲートされず常時実行される誤配線（REQ-09 違反）を
  検出するためのパターンとして置く:

  | 観測対象 | ゲート方式 | パターン ①（`docs/quality/traceability.md` のみ変更）で観測すること |
  |---|---|---|
  | 新設の検証 check | 第 2 出力 `quality`（REQ-09） | **job が実行される**（skip されない。AC-15） |
  | `e2e` / `go-coverage` | job-level `if`（`run` 出力） | **job が skip される**（起動しない） |
  | `flake-check` | step-level `if`（`run` 出力） | **`Run nix flake check` ステップが skip される**。job と matrix は常に起動し常に報告するため、job の実行有無では判定しない |

  パターン ②③ では、新設の検証 check・`e2e` / `go-coverage` の job・`flake-check` の
  `Run nix flake check` ステップの**すべてが実行される**ことを期待する。パターン ④ では、
  新設の検証 check の job が **skip され**、かつ `e2e` / `go-coverage` の job と
  `flake-check` の当該ステップも**実走しない**ことを期待する。

  テストレベルはシステム（実 PR 上での workflow 実行）。変更検出の挙動はローカル再現と
  乖離するため、実 PR での確認を主とする。

  **(b) マージブロックの成立（AC-14）** — (a) とは独立した判定点。台帳を壊した PR で
  ①当該 check が **failure として報告される**こと、②`main branch protection` ruleset の
  required status check により**マージが不可能になる**こと、の 2 点を分けて確認する。
  ①だけでは ruleset への登録漏れ（check は赤いがマージできてしまう状態）を検出できない。

- **生成の決定論性（R4）** — AC-18 の 2 回実行一致に加え、**入力の列挙順を撹乱した条件**でも
  同一出力になることを確認する。同一環境での単純な 2 連続実行は偶然一致しうるため、
  それだけでは NFR-03 の保証にならない。

### 標準領域（中リスク）

- **抽出母集団の規則適合（R3 / AC-01 / AC-03 / AC-05）** — REQ-02 の数え方の規則に対する
  同値分割・境界値。`modules/flake-parts.nix` の transposed オプション・
  `inherit (lib) mkOption` の取り込み行・入れ子 submodule 葉（`backup.enable` /
  `backup.suffix`）を**それぞれ独立した判定点**としてテストする。AC-03（新規サブコマンドの
  追加で行が現れる）は正例、**AC-05**（`rollback` の「テスト所在」が `なし`。
  `cmd/nput/rollback_test.go` は実在しない）は**負例**として、命名規約による紐づけが
  「存在しないものを存在すると言わない」ことを確認する。
- **検証モードの失敗検出（AC-07〜AC-10）** — 台帳を壊す 4 パターン（実在しないパスへの
  書き換え / 行の削除 / 母集団に無い行の追加 / 内容の手書き換え）ごとに、非ゼロ終了と
  標準エラー出力の内容を確認する。**どの機能が過不足しているかがエラー出力に現れること**
  （REQ-06）まで含める。
- **検証の非破壊性（AC-11）** — 検証モードの実行前後で台帳ファイルの内容が変化しないこと。
- **`apps.qa-tracer` の起動（AC-12）** — 生成モード・検証モードの両方が
  `nix run '.?dir=dev#qa-tracer'` で起動すること。現状 root / dev のどちらの flake にも
  `apps` 出力が無く全面的に新規追加となるため、統合レベルで押さえる。
- **テスト一覧の母集団（R6 / AC-06）** — Go テスト 254 件・nix-unit・namaka・e2e 7 シナリオが
  種別付きで出ること。特に namaka スナップショットテストが一覧から静かに落ちていないことを
  判定点にする（種別 4 値は REQ-04 で確定済み）。

### 軽量領域（低リスク）

- **NFR-01 の沈黙（R7 / AC-17）** — 標準出力が空であることの単純なアサート 1 件で足りる。
  実装ミスは自明に現れ、切り分けも容易。
- **flake outputs の非汚染（AC-13）** — root `flake.nix` に `apps` 出力が無いこと・
  `nput --help` に qa-tracer 相当のサブコマンドが現れないことの静的確認。現状 root / dev の
  どちらにも `apps` は無く、退行の余地が小さい。
- **ドキュメント追従（AC-16 / AC-19）** — `docs/dev/definition-of-done.md` に DOD-05 の行が
  存在し機械判定・人判定の合計が 5 件であること、`docs/design.md`「テスト戦略」節に
  `docs/quality/` への言及が存在することを確認する。**判定手段は本計画で用意するテスト
  （文字列一致の静的確認）とする**。

### 採用するテストレベル

コンポーネント（抽出器・台帳フォーマッタ・差分検証）／統合（生成 → 検証の往復）／
システム（`nix run` 経由の起動・CI 上の PR ゲート）。

### 採用するテストタイプ

機能テストを中心に据える。NFR-03 の決定論性は信頼性テスト、NFR-01 の沈黙は既存 ADR への
準拠テストとして扱う。性能・セキュリティ・ユーザビリティは対象外（開発者専用の内部ツールで
あり、規模の観点でも台帳 17 行・テスト一覧 320 行台〔Go 254 + nix-unit 58 + e2e 7 + namaka〕に
留まり、いずれも静的走査のみで生成できる〔NFR-02〕ため実行時間が問題にならない）。

## 4. 開始基準・完了基準

### 開始基準（entry criteria）

- `docs/dev/quality-observability/spec.md`（2026-07-26 改訂版）がレビューゲート通過済み
- 基本設計で以下が確定していること:
  1. `docs/quality/traceability.md` の出力形式
  2. REQ-04 のテスト一覧の出力形式（種別 4 値〔Go / nix-unit / namaka / e2e〕は仕様で確定済み）
  3. `.github/actions/changes` の第 2 出力 `quality` の具体配線（方式は REQ-09 で確定済み）
  4. 実装言語と実行環境（Go / jq / python の可用性制約。R8）
- `dev/flake.nix` に `apps.qa-tracer` が定義され、生成モード・検証モードの両方が起動できる

### 完了基準（exit criteria）

1. **高リスク項目（R1・R4）に対応するテストの消化率 100%、かつ全て合格**。分母は
   `test-case.md` で R1・R4 に紐づけられたケースの総数とする（後続の test-design で確定する）
2. AC-01〜AC-19 の 19 件すべてに対応するテストが存在し、実行済みで合格（未実行・未対応の
   受け入れ条件が 0 件）
3. CI ゲート検証（R1 / R2）について、3 章 (a) で固定した変更パターン ①〜④ すべてで
   期待どおりの結果（① は表、②〜④ は表直後の期待の記述に従う）が観測されている。
   観測粒度は 3 章 (a) の表に従い、**`flake-check` は `Run nix flake check`
   ステップの実行 / skip をもって判定する**（job の実行有無では判定しない）
4. 台帳を壊した PR で、当該 check が failure として報告されること、および main への
   マージがブロックされることの**両方**が確認済み（AC-14）
5. 台帳の「仕様参照」「テスト所在」列に**空欄が 0 件**であり、該当が無い欄はすべて `なし` と
   記載されている（AC-02）。`なし` の件数そのものは完了基準に含めない（R5 のとおり、
   `なし` の多発は記述不足を可視化した正しい観測結果であるため）
6. テスト一覧に namaka スナップショットテストが種別付きで現れている（R6 / AC-06）
7. 検証成功時の標準出力が空（AC-17）
8. 未解決の重大欠陥 0 件。**取得元は `test-execution-log.md` の欠陥候補欄**とし、そこに
   記録された欠陥候補のうち「台帳の腐敗を検出できない」「偽陽性で正常な PR が赤になる」の
   いずれかに該当するものが 0 件であること
9. `docs/dev/definition-of-done.md` の DOD-01 / DOD-02 が success、かつ DOD-05 として
   追加した本検証の check が success。**DOD-03 / DOD-04（人判定）は本基準に含めない** —
   完成の定義はプロジェクトのマージ可否の判定材料であり、テスト工程の完了判定とは別物の
   ため（DOD-04 の充足自体は AC-19 が部分的にカバーする）

※ 完了基準は後続の test-monitor / test-report が参照するため、測定可能な形で書いている。

## 5. 制約・前提

- **CI ゲートの検証には実 PR が要る**。R1 / R2 の性質上、変更検出の挙動はローカル再現と
  乖離する。加えて required status check の ruleset 登録にはリポジトリ管理権限を要し、
  登録の反映確認はマージ試行まで含む
- `dev/flake.nix` の `devShells.default` は Go / gopls を持つが jq・python を持たず、
  `devShells.ci` は jq を持つが Go を持たない。実装言語によっては CI シェルへの追加が要る
- 台帳の母集団は現時点で 17 件（CLI 6 / lib 6 / module 5）と小規模。テストデータは実
  リポジトリの作業ツリーそのもので足り、大規模なフィクスチャを用意する必要はない
- 未完了 epic が控えており母集団は今後増える。テストは件数をハードコードせず、**抽出結果と
  台帳の一致**を検証する形にする（AC-01 / AC-06 の具体的な件数は現時点のスナップショットと
  して扱う）

## 6. 成果物

- このテスト計画: `docs/test/quality-observability/test-plan.md`
- 初見者向けサマリ: `docs/test/quality-observability/mini-test-plan.md`
  （本計画のレビュー収束後に作成）
- 後続工程で作られる成果物（予定）: `test-analysis.md` / `test-design.md` / `test-case.md` /
  `test-procedures.md` / `test-execution-log.md` / `test-summary-report.md`

## 7. 次のステップ

- `/test-analyze quality-observability` を実行してテスト条件を識別する。
