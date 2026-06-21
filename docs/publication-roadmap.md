# nput 公開ロードマップ

リポジトリを public へ公開するための計画。grilling で確定した方針・MVP 検証結果・公開ブロッカー・公開後ワークストリームを 1 箇所に集約する。本書は計画文書であり、確定済みの設計判断は ADR / CONTEXT / spec / design を一次情報とする。

最終更新: 2026-06-22

---

## 1. 方針（grilling 決定台帳）

公開に向けた grilling で確定した判断。

### 言語・ドキュメント
- **英語化する面**: README / ユーザー向け出力 / `--help` / **全ソースコメント**（Go は godoc 形式）
- **日本語のまま維持**: 設計ドキュメント（ADR / CONTEXT.md / spec.md / design.md / 本書）
- コメント中の `→ ADR-NNNN` ポインタは言語中立なので残す
- 境界の原則: **ソース＝英語、設計根拠ドキュメント＝日本語**

### CLI（UNIX 哲学「沈黙は金」）
- 成功時は**デフォルト沈黙**（warning / error のみ stderr）
- `-v` / `--verbose` = **配置レポート（サマリ + per-target 行）の opt-in** に再定義
- 内部 nix コマンド開示は `--debug` に分離
- `--quiet` は**削除**（デフォルト沈黙で不要・未公開のため互換配慮不要）
- 不変: stdout = 機械可読専有（`gitignore` / `--dryrun`）、foreign symlink warning は常時表示、`reset` 確認プロンプトは存続

### 公開 API
- **`listFilesInSrc` を lib から除去**し、「動的 entry 生成は既 realise の store パス / flake input を `builtins.readDir` せよ（IFD 回避）」を docs の応用 idiom に降格する
  - 理由: 実体は `builtins.readDir` の薄い wrapper で、実運用ゼロ（dogfood は手動列挙、templates はコメント例のみ、テストのみが叩く）。nput の一仕事は「配置」でありディレクトリ列挙は別仕事。公開後の関数除去は破壊的変更になるため公開前に決定した

### ドキュメント
- **README**: 包括的単体（公開時点で唯一の英語 usage 面のため自己完結させる）
- **`docs/glossary.md`（英語）**: CONTEXT.md の括弧英語訳から canonical 用語集を抽出し、README / コメント / 出力の表記基準にする

---

## 2. MVP 検証結果（2026-06-22）

公開ブロッカー着手の前提条件として MVP 完成度を再確認した。

| 検証 | 結果 |
|---|---|
| `nix flake check`（nix-unit / namaka / go-vet / golangci-lint / treefmt / hm-module） | all passed |
| `go test ./...` | 全パス |
| e2e ハーネス（`tests/e2e/run.sh` 01-06、実 nix build/eval/`nix-env --set`） | 6/6 PASS |
| spec ↔ 実装 乖離 | 0 件 |

**結論: MVP（standalone CLI + project mode + home mode）は完成している。** system mode / NixOS / nix-darwin は設計通り MVP 外（将来拡張）。

### Go テストカバレッジ（参考値）
`go test ./... -cover` 実測:

| パッケージ | カバレッジ |
|---|---|
| cmd/nput | 14.4% |
| internal/engine | 69.9% |
| internal/gitutil | 86.7% |
| internal/lock | 84.2% |
| internal/manifest | 89.5% |
| internal/paths | 58.8% |
| internal/planner | 86.3% |

cmd/nput が低いのは nix eval/build + cobra が一体で単体テストしにくい構造的理由による（→ 公開後ワークストリーム）。

---

## 3. 公開ブロッカー（public 化の瞬間まで必須）

依存順に並べる。`!` は破壊的変更（Conventional Commits）。main 直コミット禁止のため各ブロッカーはブランチ + PR で進める。

**順序の根拠**: API / 挙動を先に確定（①②）→ 確定した文字列を英語化（③④）→ 用語集を土台に README（⑤⑥）。後工程の英語化対象が前工程で確定する。

### ① listFilesInSrc 除去（公開 API 変更）
- `lib/list-files.nix` 削除 / `lib/default.nix` の export 削除
- `tests/nix-unit.nix` の関連テスト（5 件）削除
- docs 整理（日本語維持）: spec.md（§132 / §478 応用節 / エラー仕様表）・design.md・CONTEXT.md（subpath 項）・templates コメント・`dev/nput.nix` の stale コメント → 「動的 entry 生成は既 realise store パス / flake input を `builtins.readDir`」idiom 注記へ置換
- コミット: `refactor(lib)!: drop listFilesInSrc, demote to readDir idiom`

### ② UNIX 出力再設計（CLI 挙動変更）
- `cmd/nput/main.go`: `--quiet` 削除、`-v` / `--verbose` を「配置レポート opt-in」に再定義、`--debug`（nix コマンド開示）追加
- `cmd/nput/apply.go` / `reset.go` / `rollback.go`: 成功レポートをデフォルト沈黙化し `-v` で gating
- `cmd/nput/nix.go`: nix コマンド開示を `--debug` gating
- spec.md / design.md の出力規律更新（+ 新 ADR か既存改訂）、e2e の該当箇所更新
- コミット: `feat(cli)!: silent on success, opt-in report via -v` / `feat(cli): move nix command disclosure to --debug` / `refactor(cli)!: remove --quiet`

### ③ 出力 / help 英語化
- 全 stderr / stdout 文字列、cobra Short / Long / flag usage を英語化（②で確定した文字列が対象）
- コミット: `i18n(cli): translate user-facing output and help to English`

### ④ 全ソースコメント英語化（Go = godoc 形式）
- cmd/ internal/ lib/ modules/ templates/ の全コメント。Go の exported 宣言は識別子名始まりの doc comment にする
- パッケージ単位でコミット分割: `docs(cmd)` / `docs(engine)` / `docs(planner|paths|...)` / `docs(lib)` / `docs(modules)` / `docs(templates)`

### ⑤ docs/glossary.md（英語）
- CONTEXT.md 括弧英語訳から canonical 用語集を抽出
- コミット: `docs: add English glossary`

### ⑥ README（英語・包括的単体）
- concept / install / quickstart（project + home）/ entries スキーマ / コマンドリファレンス / 他ツール比較 / MVP ステータス。⑤の用語に準拠
- コミット: `docs: add public English README`

---

## 4. 公開後ワークストリーム（継続改善）

公開ブロッカーではない。公開後に順次着手する。

### A. SSG ドキュメントサイト
- **Astro**（Starlight 想定）で構築
- **英日 bilingual i18n 対応**
- ソース: ① Nix lib を `/** */`（RFC-145 doc-comment）化 → **nixdoc** で CommonMark 抽出、② Go を **gomarkdoc** で markdown 化、③ 手書き usage ガイド
- 3 ソースを 1 サイトへ集約

### B. Nix private 関数テスト充実
- 現状 `normalizeManifest` / `listFilesInSrc`（公開寄り）のみ。`manifest.nix` 内 private ヘルパ（`escapesBase` の `..` 深さ判定 / `anchorName` の sha256 / marker → enum 変換）は `let` 束縛で未露出
- 論点: テストのため内部を露出するか、公開面経由で振る舞いテストするか（公開後に決定）

### C. Go テスト充実
- 最大効果: cmd/nput の orchestration ロジック（flag 検証 / `--all` 集約 / exit code 判定）を nix 呼び出しから分離し、nix インジェクション seam を拡張して単体テスト可能にする（engine は既に `Commit` / `Git` / `Build` を注入できる設計）
- 些末な穴: internal/paths の `StateDir()` / `GenerationLink()`（0%・数行で追加可能）
- エラー経路の穴: engine の out-of-store Lstat 非 ENOENT / copy mkdir・chmod 失敗 / resolveRoot の project・fixed 分岐 / cleanupPending 再 flock 失敗
- CI / devShell へのカバレッジ計測組み込み（現状なし）

### D. Go パフォーマンス向上 / リファクタリング（backlog）
すべて stdlib-only 制約に整合。

**高優先:**
- `reverifyStale` の plan 時 Lstat 結果キャッシュ化（重複 I/O 削減）
- Apply / Rollback / Reset の「lock → 操作 → cleanup」共通化（`executeWithLock` 抽出）
- copy / symlink の `Executor` interface dispatch 統一（place.go / copy.go / drift.go の責務分散解消・テスト容易性向上）

**中〜低:** `byTarget` map キャッシュ / FS walk ロジック（ancestorSymlink / copyTree / reset）統一 / root override × profile layout の分岐局所化 / copyTree の重複 Lstat 削減 / error wrapping に context 付与。

---

## 5. 技術メモ（grilling 中の Q&A）

- **IFD と src**: `entries.<name>.src` は engine 実行時解決なので IFD を起こさない（`toString` して symlink するだけ）。IFD を誘発するのは eval 時に `readDir` する `listFilesInSrc` のみ（→ ブロッカー① の除去で論点ごと消える）
- **`pkgs.hello` を src に渡せるか**: 可能。`srcType` が derivation を許容するため `src = pkgs.hello` は型を通り、`subpath` で内部ファイル（例 `bin/hello`）も指定できる。ただし `listFilesInSrc { src = pkgs.hello; }` は IFD 回避で不可（除去後は idiom 側の注意書きに移る）
