# ADR-0028: cachix push を main push の os マトリクスで実装する（ADR-0012 §4 の実装補足）

- ステータス: 採用
- 日付: 2026-06-20
- 関連: ADR-0012（§4 キャッシュ投入）, ADR-0027（CI 実装補足）, ADR-0011（stdlib-only Go）
- 参照: `~/src/github.com/yasunori0418/nur-packages` の `.github/workflows/cachix-push.yml`

## 背景

ADR-0012 §4 は「tag push 時に matrix（x86_64-linux / aarch64-linux / aarch64-darwin）で `nix build .#nput` し cachix `yasunori0418` に投入する」と決めていたが、実装フェーズで2点を見直した。

- **トリガ**: nput は flake input として消費される CLI で、consumer の CI / ローカルが「最新 main の `nput` を実ビルドせずキャッシュから引く」恩恵が大きい。tag push のみだとリリース間の main 変更がキャッシュされない。
- **ワークフロー構成**: 参照元 nur-packages は多数パッケージのため「未キャッシュのものだけ選別する `ci-matrix` app + plan / collect / build-and-push の3段」を組むが、nput は**実質1パッケージ**なので過剰。

## 決定

### 1. トリガは push:main + `paths` + `workflow_dispatch`

- main への push のうち、nput バイナリの内容を決める入力（`**.nix` / `**.go` / `go.mod` / `go.sum` / `flake.lock` / `dev/flake.lock`）が変わったときに投入する。docs 等だけの変更では走らせない。
- 手動投入は `workflow_dispatch`。
- ADR-0012 §4 の「tag push」からの変更。tag リリースに紐づけず、main の最新を常時キャッシュする方針に改める。

### 2. 構成は test.yml と同じ os×system マトリクスの単純ビルド

- `ubuntu-latest`=x86_64-linux / `ubuntu-24.04-arm`=aarch64-linux / `macos-latest`=aarch64-darwin の3環境で `nix build .#packages.<system>.nput` をネイティブ実行する。
- 投入は `.github/actions/setup-nix`（cachix `yasunori0418`）の `cachix-action` が authToken 指定でビルド生成パスを自動 push する経路に任せる（明示 push ステップは置かない）。
- nur-packages の `ci-matrix` 選別機構（plan / collect）は採らない。単一パッケージで選別の利得が無く、複雑性に見合わない。

## 根拠

- **main push 投入**: nput は配置エンジンの CLI で consumer 側 activation / CI が実体を要する。最新 main を都度キャッシュすれば consumer のビルド時間と再現性が改善する。
- **単純マトリクス**: 1パッケージでは「未キャッシュ選別」のジョブ往復コストの方が高い。`cachix-action` の重複 push スキップ（store パスのハッシュ一致分はスキップ）で十分。
- **fork PR の secrets**: 本ワークフローは push:main / workflow_dispatch 起動で fork PR からは走らないため、`CACHIX_AUTH_TOKEN` 不在問題は生じない（投入は本リポジトリ権限下のみ）。

## 影響

- `.github/workflows/cachix-push.yml` を追加。
- `docs/design.md`「CI 実行」に cachix push の段落を補足。
- ADR-0012 §4 は supersede しない（投入対象とキャッシュ名は維持し、トリガと構成のみ実装で具体化）。

## 棄却した代替案

- **tag push のみ（ADR-0012 §4 原案）**: リリース間の main 変更がキャッシュされず、開発中の consumer が実ビルドを強いられる。
- **nur-packages の ci-matrix 選別を踏襲**: 多パッケージ前提の機構で、単一パッケージの nput には過剰。
- **`nix-access-token` を setup-nix に追加**: nput は public repo / public flake input で `github_access_token` で足り、現時点で不要。private fetch が要る段で別途追加する。
