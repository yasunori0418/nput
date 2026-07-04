# ADR-0037: ドキュメントサイトは Astro Starlight + 同一リポジトリ + Cloudflare Pages（ビルド時生成・英語ルート i18n）で構築する

- ステータス: 採用
- 日付: 2026-07-04
- 関連: ADR-0027, ADR-0028, `docs/publication-roadmap.md` §4-A, `docs/glossary.md`
- 改訂対象: なし（新領域。publication-roadmap §4-A の計画を具体化する）
- 起点 Issue: #57

## 背景

publication-roadmap §4-A は SSG ドキュメントサイトの構想（Astro / Starlight 想定・英日 bilingual・nixdoc + gomarkdoc + 手書きガイドの 3 ソース集約）を挙げたが、置き場所・ホスティング・生成物の扱い・i18n 構成が未決だった。#57 の論点（Starlight の i18n と doc 抽出パイプラインの取り回し・Nix lib への doc-comment 付与）を解消し、実装に入れる状態にする。

## 決定

### 1. サイトソースは同一リポジトリの `site/` に置く

- Astro（Starlight）プロジェクトを本リポジトリの `site/` に置く。別リポジトリには分離しない。
- リファレンスの抽出元（`lib/` の doc-comment・Go ソース）とサイトが同一リポジトリにあることで、**API 変更とドキュメント更新が同一 PR で原子的に進む**。
- サイトの内容は **usage ガイド + API リファレンス（英日）**。設計根拠ドキュメント（ADR / CONTEXT.md / spec.md / design.md・日本語）は公開方針（roadmap §1「ソース＝英語、設計根拠＝日本語」）通りリポジトリ内に留め、サイトには載せない。

### 2. ホスティングは Cloudflare Pages・デプロイは GitHub Actions から

- ホスティングは **Cloudflare Pages**。PR ごとのプレビューデプロイ（branch deploy）でドキュメントレビューを URL で回せる。
- デプロイは Cloudflare の Git 連携ビルドではなく **GitHub Actions から wrangler で行う**。ビルドに nix（nixdoc / gomarkdoc の実行）が必要で、Cloudflare 側のビルド環境では賄えないため。main への push で本番デプロイ、PR で preview デプロイ。
- CI（ADR-0027 の check 群）に site のビルド検証を加え、リファレンス抽出の破損（doc-comment の構文エラー等）を merge 前に検出する。

### 3. リファレンスはビルド時生成・生成物はコミットしない

- **① Nix lib**: 公開 API（`mkManifest` / `mkOutOfStoreSymlink` / マーカー群）に RFC-145 形式の doc-comment（`/** */`）を付与し、**nixdoc** で CommonMark を抽出する。既存の `#` 行コメント（実装注記）とは役割分担する（`/** */` = 公開 API 契約、`#` = 実装注記）。
- **② Go**: **gomarkdoc** で godoc コメント（公開ブロッカー④で英語化済み）から markdown を生成する。`internal/` は公開 import 面ではない（ADR-0006）ため、リファレンスの主対象は CLI（`cmd/nput` のコマンド体系）とし、internal パッケージの掲載範囲は実装時に絞る。
- **③ 手書き usage ガイド**: `site/` 配下に英日で執筆する。
- ①②の生成 markdown は**リポジトリにコミットしない**。CI / ローカルのビルドステップで毎回生成し、ソースとリファレンスの乖離を構造的に排除する。

### 4. i18n は英語ルート + `/ja/`

- Starlight の i18n 構成で **root ロケール = 英語（`/`）、日本語 = `/ja/`** とする。README を英語 canonical とした公開方針（roadmap §1）と一貫させる。
- 未翻訳ページは Starlight の標準機構で root ロケール（英語）へフォールバック表示し、日本語訳の遅れがリンク切れにならない状態を保つ。
- 用語は `docs/glossary.md`（英語 canonical）/ `docs/glossary.ja.md` に準拠する。

## 根拠

- **同一リポジトリ**: 別リポジトリだと API 変更 → 抽出結果の更新が非原子的になり、サイトが古いリファレンスを配信し続ける事故が構造化する。リポジトリの肥大は `site/` の隔離とビルド分離（flake の devShell / CI ジョブ分割）で抑えられる。
- **Actions + wrangler**: 生成物非コミット方針を採る以上、ビルドに nix が要り、デプロイ主体は Actions 一択。wrangler は preview / 本番の両デプロイを同一機構で扱える。
- **ビルド時生成**: 生成物コミットは「生成し忘れ」「手編集の混入」を許す。CI 生成なら乖離ゼロが機械的に保証される。
- **英語ルート**: 外部利用者の第一言語面を英語に揃える公開方針の一貫。作者の主言語（日本語）は `/ja/` で完全提供する。

## 影響

- **リポジトリ構成**: `site/`（Astro Starlight・pnpm 等の JS ツールチェーンは devShell に追加）。
- **`lib/`**: 公開 API への RFC-145 doc-comment 付与（既存 `#` コメントとの役割分担）。
- **CI（`.github/`）**: site ビルド検証ジョブ・wrangler デプロイワークフロー（main = 本番 / PR = preview）。Cloudflare の API token は repo secrets で管理。
- **`docs/publication-roadmap.md`**: §4-A の具体化として本 ADR を参照。
- **#57**: 本 ADR を設計確定として sub-issue（doc-comment 付与 / gomarkdoc 組込 / サイト骨格 + i18n / 抽出パイプライン / CI・デプロイ / usage ガイド執筆）に分割する。
- **優先度**: 本マイルストーン内では機能 4 本（→ ADR-0033〜0036）の**後**に着手する（grilling で確定）。

## 棄却した代替案

- **別リポジトリ（`nput-docs`）**: API 変更との同期が非原子的。本体リポジトリを軽く保つ利益より乖離リスクが大きい。
- **GitHub Pages**: リポジトリ完結で簡便だが、PR プレビューデプロイが標準では無く、ドキュメントレビューの UX で Cloudflare Pages に劣る（grilling で Cloudflare Pages を選択）。
- **生成 markdown をコミットして Cloudflare Git 連携ビルドに乗せる**: デプロイ設定は簡単になるが、生成物とソースの同期を CI チェックと人間の運用で守り続けるコストが恒常化する。
- **日本語ルート + `/en/`**: 公開方針（英語が外向きの一次言語）と逆転する。
- **設計ドキュメント（ADR / spec 等）もサイトに掲載**: 日本語のままの掲載は bilingual 方針と不整合、翻訳は維持コストが過大。設計根拠は「リポジトリを読む人」向けであり、サイトの対象読者（利用者）と異なる。
