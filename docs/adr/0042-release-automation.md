---
id: "ADR-0042"
type: adr
name: "リリースを bump PR 起点で自動化する（VERSION ファイル + 自動タグ + 自動リリースノート）"
status: 採用
origin: "次期マイルストーン追加計画の grilling（2026-07-04）。参照実装として同作者の cryoflow リポジトリのリリース自動化（bump workflow + release workflow）を採用する"
justifies:
  - "INF-9878e9f5-1ec0-4ef2-879b-859ea538cc43"
  - "REQ-9ed6b500-a11f-414e-a763-adb47c89f7d4"
  - "REQ-a5053191-1c6a-449b-9c5e-5ff49dc5aead"
references:
  - "ADR-0027"
  - "ADR-0028"
  - "ADR-0030"
---
# ADR-0042: リリースを bump PR 起点で自動化する（VERSION ファイル + 自動タグ + 自動リリースノート）

- ステータス: 採用
- 日付: 2026-07-04
- 関連: ADR-0027, ADR-0028, ADR-0030, `.github/workflows/`
- 改訂対象: なし（新規領域）
- 起点: 次期マイルストーン追加計画の grilling（2026-07-04）。参照実装として同作者の cryoflow リポジトリのリリース自動化（bump workflow + release workflow）を採用する

## 背景

現状リリースの機構が無く、利用者は flake ref（ブランチ / rev）で pin するしかない。バージョン付き GitHub Release があれば、変更履歴の共有（自動生成ノート）と安定 ref（タグ）を提供できる。

参照した cryoflow の方式は 2 workflow 構成: ① `bump-version.yml`（workflow_dispatch・非 main ブランチ上でバージョンファイルを書き換えてコミット → 通常の PR フローでマージ）、② `release.yml`（main への push・バージョンファイルの paths フィルタ → バージョンを読み取り `softprops/action-gh-release` の `generate_release_notes: true` でタグ + Release 作成）。「bump PR のマージがリリースを駆動する」ため main 直コミット禁止（ADR-0025）と整合し、手動のタグ打ちが要らない。

nput 用に決めるのはバージョンの置き場所（Python の pyproject.toml に相当するものが無い）と成果物の扱い。

## 決定

### 1. バージョンの一次情報 = リポジトリ直下の `VERSION` ファイル

- プレーンテキスト 1 行（例 `0.1.0`）。semver に従う。
- `flake.nix` は `builtins.readFile ./VERSION` で読み、`packages.nput` の `version` に反映する。
- Go バイナリへは nix build の `ldflags`（`-X`）で埋め込み、**`nput --version`（または `nput version`）を新設**して表示する。単一の一次情報から flake / バイナリの両方が導出され、二重管理しない。

### 2. `bump-version.yml`（workflow_dispatch）

- バージョン文字列を入力に取り、**非 main ブランチ上で** `VERSION` を書き換えて `chore: bump version to X` をコミット・push する（cryoflow 同型）。マージは通常の PR フロー（required checks・ADR-0030 の対象）。
- 手元で `VERSION` を編集して PR を作る手動経路も同格に有効（workflow は省力化であって唯一の経路ではない）。

### 3. `release.yml`（main push + `VERSION` paths フィルタ）

- main への push のうち `VERSION` が変更されたものだけで発火。`VERSION` を読み取り、`softprops/action-gh-release`（バージョン pin）で **タグ `vX.Y.Z` 作成 + リリースノート自動生成 + GitHub Release 作成**を行う。
- リリースノートは GitHub 標準の自動生成（Conventional Commits 履歴がそのまま読みやすい）。CHANGELOG.md はコミットしない（生成物非コミットの方針・ADR-0037 と同じ姿勢）。

### 4. 成果物は添付しない

- nput は実行時に `nix` / `git` を叩く nix 前提ツールで、配布の正規経路は flake ref（タグで pin 可能になる）。バイナリ添付は nix の無い環境で動かないため価値が薄い。将来需要が出たら release.yml にステップを足せる seam だけ残す（本 ADR は禁止しない）。

## 根拠

- **bump PR 駆動**: タグ打ち忘れ・タグと内容の不一致が構造的に起きない。リリースの意思決定が PR レビューという既存ゲートを通る。main 直コミット禁止とも矛盾しない（bump コミットは branch 上で bot が積み、マージは人間の PR 操作）。
- **`VERSION` ファイル**: paths フィルタが 1 ファイルで済み、言語非依存で flake / Go / CI のどこからも読める。`flake.nix` 内に埋めると paths フィルタが flake.nix 全変更に反応してしまう。
- **自動生成ノート**: このリポジトリは Conventional Commits を厳格運用しており（commit-flow）、標準の自動生成で十分な品質のノートになる。git-cliff / release-please は CHANGELOG コミットと bot PR の運用を増やす割に得るものが少ない。

## 影響

- **新規**: `VERSION`・`.github/workflows/bump-version.yml`・`.github/workflows/release.yml`。
- **`flake.nix`**: `packages.nput` の version を `VERSION` から読む・ldflags 埋め込み。
- **`cmd/nput`**: `--version` 表示の新設（cobra の Version フィールド）。
- **`docs/design.md`**: CI / リリース節に本方式を追記。
- **ADR-0030 との関係**: release.yml は required checks の対象外（main push 後のリリース作業であり merge gate ではない）。

## 棄却した代替案

- **タグ push トリガー（手動タグ打ち）**: 最小構成だが、タグ打ち忘れ・タグ位置の誤りが人間の注意力頼みになる。bump PR 駆動の方が既存のレビュー動線に乗る。
- **release-please / git-cliff（CHANGELOG.md コミット方式）**: 生成物コミットと bot PR 運用が増える。標準の自動生成ノートで足りる。
- **バイナリ成果物の添付**: nix 前提ツールに単体バイナリの需要が薄い。必要になった時点で release.yml に追加。
- **`flake.nix` 内にバージョンを直書き**: paths フィルタの精度が落ち、リリース意図のない flake.nix 変更で release.yml の発火判定が複雑になる。
