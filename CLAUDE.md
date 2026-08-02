# nput

フェッチ済みの git リポジトリをユーザー環境の任意パスへ symlink / copy で配置する Nix ライブラリ・モジュール群。設定生成は行わない。

## ドキュメント

`docs/` は **README → 概要文書 → item の 3 層構造**。規範的な内容は item（1 ファイル 1 主張の
Markdown + YAML frontmatter）が持ち、概要文書は通読の入口として全体像と item への索引を担う。

### 概要文書（通読の入口）

- `docs/concept.md` — コンセプト（solution / use_case への索引）+ 設計の哲学・既存ツールとの比較・north-star・設計の変遷
- `docs/design.md` — 設計（design item への索引）
- `docs/spec.md` — 仕様（requirement item への索引）

**これらは概要に縮退済み**。各文書の冒頭に書き方の規約（セクションあたりの散文の行数上限・
詳細は item へリンクし本文に書き下さない）があり、それに従う。仕様・設計の内容を足すときは
item を新設し、概要文書にはリンクを 1 行足すに留める。

### item 群（規範的な内容の所在）

| ディレクトリ | 型 | prefix | 親 |
|---|---|---|---|
| `docs/solution/` | solution | `SOL` | なし（根）|
| `docs/use-cases/` | use_case | `UC` | solution |
| `docs/requirements/` | requirement | `REQ` | use_case |
| `docs/design/` | design | `DSG` | requirement |
| `docs/infrastructure/` | infrastructure | `INF` | なし（根）|
| `docs/adr/` | adr | `ADR` | なし（分離型）|
| `docs/risks/`（未作成）| risk | `RISK` | requirement or design |
| `docs/test/<対象>/`（未作成）| test_condition / test_case / defect | `TC` / `CASE` / `D` | 上位から順に |

型・関係・フィールドの定義は `docs/model.yaml` が持つ。「未作成」の 2 つは model.yaml が
定義済みで、テスト工程に着手する時点でこの配置に従って作る。risk を `requirement` と `design`
のどちらに張るかの使い分けは `docs/agents/sara-graph.md`。

### ID 規約（UUIDv4 二層構成）

| 用途 | 形式 | 例 |
|---|---|---|
| 正式 ID（frontmatter `id:`・relation リスト）| `<PREFIX>-<フル UUIDv4>` | `REQ-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4` |
| ファイル名 | `<YYYYMMDD>-<前方8文字>-<slug>.md` | `20260802-2b0c2bb8-mkmanifest-pure-function.md` |
| 散文中の参照 | `<PREFIX>-<前方8文字>` | `REQ-2b0c2bb8` |

- 採番は devShell 同梱の `sara-id` コマンドで行う（8 文字 prefix の重複チェック込み）。
  連番を手で振らない（並列レーンでの採番衝突を構造的に避けるため）
- **ADR のみ連番を維持する**（`ADR-0049`）。既存 48 本の相互参照・`docs/adr/README.md` の運用・
  Issue 言及を壊さないため
- requirement の `specification` フィールドは **英語で書く**。sara が RFC2119 キーワード
  （MUST / SHALL / SHOULD …）の存在をハードコードで検証するため。対になる日本語の規範文は
  `specification_ja` に併記する

### item を辿る（`sara query`）

`sara` は devShell 経由で使う（`nix develop ./dev --command sara ...`）。

```bash
sara check                                 # グラフ全体の検証（broken reference / duplicate ID / cycles）
sara query <フル ID> -u                    # 上流を辿る（requirement → use_case → solution）
sara query <フル ID> -d                    # 下流を辿る（requirement → design → risk → test）
sara report coverage                       # カバー率
sara report matrix                         # トレーサビリティマトリクス
```

**`sara query` はフル ID しか受け付けない**（省略形 `REQ-2b0c2bb8` は "Item not found"）。
散文中の省略形からフル ID を得るには 8 文字で grep する。省略形は正式 ID の前方一致なので、
宣言側・参照側の両方がヒットする。

```bash
rg -l 2b0c2bb8 docs/                       # 宣言している item ファイルと参照元を列挙
rg -o 'REQ-2b0c2bb8[0-9a-f-]*' docs/requirements/20260802-2b0c2bb8-*.md | head -1
```

## 開発コマンド

```bash
nix flake check          # 評価エラー・型チェック
nix build .#<package>    # パッケージビルド
nix run .#<script>       # activation スクリプト実行
nix develop ./dev --command sara check   # ドキュメントグラフの検証
```

## 規約

- **実装フェーズ**。main ブランチへの直接コミットは禁止。作業は必ずブランチを切り、PR 経由でマージする。
- `flake.nix` は `flake-parts.lib.mkFlake` ベース
- `lib/` は nixpkgs のみに依存する。home-manager / NixOS / nix-darwin への依存を持ち込まない
- **ドキュメントの配置ルール・ID 規約はこのプロジェクトの規約が優先する**（→「ドキュメント」節）。
  要求は `docs/requirements/`、リスクは `docs/risks/`、テスト成果物は `docs/test/<対象>/` へ
  1 ファイル 1 item で置き、ID は `sara-id` で採番する。スキル既定の出力先・採番方式（散文への
  ID 直書き・連番）がこれと食い違う場合はプロジェクト規約に従う
- ユーザーに確認・質問する際は、テキストで質問を投げず **AskUserQuestion ツールを積極的に使う**。設計判断の確認・曖昧な依頼の解釈確認・代替案の選択などで使い、各質問は推奨オプションを先頭に置く

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`gh` CLI). See `docs/agents/issue-tracker.md`.

### Triage labels

Default canonical labels (needs-triage / needs-info / ready-for-agent / ready-for-human / wontfix). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context — one `CONTEXT.md` + `docs/adr/` at repo root. See `docs/agents/domain.md`.

### Sara document graph

`docs/` は sara のナレッジグラフ（`docs/model.yaml` + `sara check`）。risk を
`requirement` と `design` のどちらに張るかの使い分け規約は `docs/agents/sara-graph.md`。
