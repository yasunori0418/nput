# nput

フェッチ済みの git リポジトリをユーザー環境の任意パスへ symlink / copy で配置する Nix ライブラリ・モジュール群。設定生成は行わない。

## ドキュメント

設計・仕様・コンセプトは作業の文脈に応じて以下を参照する。

- `docs/concept.md` — コンセプト・設計の哲学・既存ツールとの比較
- `docs/design.md` — 設計書（レイヤー構成・flake outputs・モジュール設計・使用パターン）
- `docs/spec.md` — 仕様書（lib API・entries スキーマ・配置動作・エラー仕様）

## 開発コマンド

```bash
nix flake check          # 評価エラー・型チェック
nix build .#<package>    # パッケージビルド
nix run .#<script>       # activation スクリプト実行
```

## 規約

- **実装フェーズ**。main ブランチへの直接コミットは禁止。作業は必ずブランチを切り、PR 経由でマージする。
- `flake.nix` は `flake-parts.lib.mkFlake` ベース
- `lib/` は nixpkgs のみに依存する。home-manager / NixOS / nix-darwin への依存を持ち込まない
- ユーザーに確認・質問する際は、テキストで質問を投げず **AskUserQuestion ツールを積極的に使う**。設計判断の確認・曖昧な依頼の解釈確認・代替案の選択などで使い、各質問は推奨オプションを先頭に置く

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`gh` CLI). See `docs/agents/issue-tracker.md`.

### Triage labels

Default canonical labels (needs-triage / needs-info / ready-for-agent / ready-for-human / wontfix). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context — one `CONTEXT.md` + `docs/adr/` at repo root. See `docs/agents/domain.md`.
