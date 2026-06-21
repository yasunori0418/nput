# ADR-0030: テスト成功を main マージの必須条件にする（ruleset で required status check）

- ステータス: 採用（public 化を前提とする）
- 日付: 2026-06-21
- 関連: ADR-0027（flake check CI マトリクス / トリガ・本 ADR が §2 を補足訂正）, ADR-0012（CI とテスト実行）, ADR-0028（cachix push）
- 参照: 現行 `.github/workflows/test.yml`, `.github/actions/setup-nix`

## 背景

PR で CI（`nix flake check` マトリクス + E2E）は走るが、**テストが失敗していても `main` へマージできてしまう**。
CLAUDE.md は「main 直接コミット禁止・PR 経由マージ」を運用ルールとして定めるが、**テスト成功の強制は技術的に担保されていない**。

「テストが通らなければマージ不可」を実現する正攻法は、GitHub の **ruleset で required status check を課す**こと。

### 前提: public 化

ruleset / branch protection は **個人アカウント所有の private + free プランでは API・UI ともにロック**されている
（`gh api .../rulesets` → `403: Upgrade to GitHub Pro or make this repository public`）。

本リポジトリは Nix ライブラリ・モジュール群（`docs/concept.md` が既存ツール比較を含む OSS 前提）であり、**public 化**で機能解放する方針を採る（GitHub Pro 課金は採らない）。
public 化は **アカウント/リポジトリ側の人手操作**であり、ruleset 適用の前提ステップとなる。

## 決定

### 1. `main` に required status check を課す ruleset を作る

`main`（`~DEFAULT_BRANCH`）への PR マージ条件に、PR で実際に出る 4 つの check 名を必須化する:

- `e2e`
- `flake-check (ubuntu-latest, x86_64-linux)`
- `flake-check (ubuntu-24.04-arm, aarch64-linux)`
- `flake-check (macos-latest, aarch64-darwin)`

> matrix job の check 名は `(os, system)` 付き。job 名 / matrix を変えると check 名も変わるため ruleset 側の追従が要る。

ruleset の縛り:

- **`pull_request` ルールで `main` への直接 push を禁止**し、CLAUDE.md「main 直接コミット禁止」を技術的に担保する。
- **bypass を許さない**（`bypass_actors` 未指定）。owner でも PR + テストを通さないとマージ不可（owner は ruleset 自体の編集は可能）。
- **up-to-date は強制しない**（`strict_required_status_checks_policy: false`）。個人運用で PR が多くないうち再走コストを避ける。
- **レビュー承認は不要**（`required_approving_review_count: 0`）。テスト成功のみをマージ条件にする。

### 2. `paths` 非互換を「単一 workflow + if スキップ」で解消する（ADR-0027 §2 を補足訂正）

required status check は **走らない check を `Expected / 待機中` として扱いマージをブロックし続ける**。
現行 `test.yml` は `pull_request` トリガ段で `paths` フィルタを持つため、**docs-only PR では workflow が起動せず必須 check が報告されずマージ不能**になる。

一方、**job が生成された上で `if:` でスキップされた check は "Skipped" = 成功扱い**になる。これを使い、ADR-0027 §2 の「docs-only では回さない」最適化を保ったまま required check を成立させる:

- **トリガ段の `paths` フィルタは撤去**し、workflow は全 PR で起動する。
- **変更検出を専用 composite action 化**する（`.github/actions/<filter>` を `setup-nix` と同じ流儀で新設。`dorny/paths-filter` 等を SHA pin で内包し、nix / Go / CI 定義の変更フラグを outputs で返す）。各 CI workflow はこの action を参照する。
- `flake-check` / `e2e` は変更検出 job を `needs` し、`if:` で対象外なら **skip → "Skipped"（成功扱い）**。nix は走らず check 名だけ出る。

> 専用 action 化は「検出ロジックを workflow 間で重複させず一箇所に閉じる」ため。`setup-nix` と同じく `.github/actions/` 配下に composite で置き、各 CI は `uses: ./.github/actions/<filter>` で参照する。

これにより ADR-0027 §2 の「`pull_request` に `paths` フィルタを付ける／docs-only は `workflow_dispatch`」という記述は陳腐化する。本 ADR が §2 を **supersede せず補足訂正**する（最適化の意図＝docs-only で nix を回さないは維持し、実現手段を trigger-paths から action 検出 + if スキップへ移す）。

## 適用コマンド（public 化後）

```bash
gh api -X POST repos/yasunori0418/nput/rulesets --input - <<'JSON'
{
  "name": "main-required-tests",
  "target": "branch",
  "enforcement": "active",
  "conditions": { "ref_name": { "include": ["~DEFAULT_BRANCH"], "exclude": [] } },
  "rules": [
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 0,
        "dismiss_stale_reviews_on_push": false,
        "require_code_owner_review": false,
        "require_last_push_approval": false,
        "required_review_thread_resolution": false
      }
    },
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": false,
        "do_not_enforce_on_create": false,
        "required_status_checks": [
          { "context": "e2e" },
          { "context": "flake-check (ubuntu-latest, x86_64-linux)" },
          { "context": "flake-check (ubuntu-24.04-arm, aarch64-linux)" },
          { "context": "flake-check (macos-latest, aarch64-darwin)" }
        ]
      }
    }
  ]
}
JSON
```

## 根拠

- 運用ルール（PR 経由・テスト前提）を **技術的ゲート**に落とし、誤マージを構造的に防ぐ。
- 専用 composite action で変更検出を一箇所に集約し、複数 workflow（test / 将来追加）から再利用する。`setup-nix` の前例に倣う。
- `if` スキップ方式は「skipped 必須 check は成功扱い」という GitHub の挙動に依拠し、docs-only PR を止めずに最適化を保つ。
- check 名は実 PR（`gh pr checks`）から確定したものを使い、job 名変更時の追従ポイントを ADR に明記する。

## 影響

- **public 化が前提**。これは ruleset 適用前の人手操作で、本 ADR のスコープ外の作業として issue のチェックリストに載せる。
- `test.yml` の `paths` トリガ撤去 + 変更検出 composite action 新設 + `needs`/`if` 改修が必要（実装は別途・issue 管理）。
- ADR-0027 §2 の `paths` フィルタ記述を補足訂正（本 ADR 参照を追記）。`docs/design.md` の CI 記述も追従更新。
- ruleset 適用後、ダミー PR で「失敗時マージ不可 / 成功時マージ可 / docs-only はスキップで通る」を実地確認する。

## 棄却した代替案

- **GitHub Pro 加入で private 維持**: 月額課金が発生。OSS 前提の本リポジトリでは public 化が自然で無料。
- **トリガ段 `paths` 撤去（全 PR で nix を実走）**: docs-only PR でも flake-check×3 + e2e が実走しコスト増。ADR-0027 §2 の最適化意図を捨てることになる。
- **ツイン workflow（paths / paths-ignore で同名 check を二重定義）**: matrix を skip 用にも重複定義する必要があり保守が二重化。単一 workflow + if スキップの方が DRY。
- **集約 success job を単一必須化**: 必須 check 名が 1 つで済むが集約 job 実装が要り、matrix 個別の成否が ruleset から見えにくくなる。
- **branch protection（旧 API）**: ruleset が後継で表現力が高く GitHub も推奨。free private で使えない制約は同じ。
- **運用ルールのみ（現状維持）**: 強制力がなく誤マージを防げない。本 ADR の動機そのもの。
