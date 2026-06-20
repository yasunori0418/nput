# ADR-0027: flake check を CI で os×system マトリクス実行する（ADR-0012 の実装補足）

- ステータス: 採用
- 日付: 2026-06-20
- 関連: ADR-0012, ADR-0011（cgo 未使用 / stdlib-only）, ADR-0006
- 参照: PR #40, `~/src/github.com/yasunori0418/cryoflow` の CI 構成

## 背景

ADR-0012 §3 は「lib（nix-unit / namaka）と engine Go を `nix flake check` に集約し PR で常時実行する」と決めたが、
実行基盤の具体は実装フェーズに送られ、次の3点が未確定だった。

- flake check を **どの system 上で**走らせるか（`perSystem` は 4 system 定義だが、CI ランナーは限られる）。
- workflow の **トリガ**（初期計画では `pull_request` + `push`(branches-ignore: main) を想定）。
- engine の Go check（go-vet / golangci-lint）の **サンドボックス条件**。

PR #40 の実装で確定したため、ADR-0012 を **supersede せず補足**する（テスト層分離と cryoflow 踏襲という基本方針は維持）。

## 決定

### 1. flake check を os×system の3環境マトリクスで実行

- `ubuntu-latest`=x86_64-linux / `ubuntu-24.04-arm`=aarch64-linux / `macos-latest`=aarch64-darwin の3環境で `nix flake check -L` を回す。
- **x86_64-darwin は CI 対象外**。GitHub ホストの標準 x86_64 macOS ランナーが乏しいため。`perSystem` の4 system 定義は残すが、CI で評価・ビルド検証されるのは上記3環境。

### 2. トリガは `pull_request` + `workflow_dispatch`

- 初期計画の `push`(branches-ignore: main) は採らない。PR 経由運用が前提（main へ直 push しない・保護ブランチ）で push トリガは PR チェックと重複するため。
- 手動再実行は `workflow_dispatch` で担保する。

### 3. Go check は `CGO_ENABLED=0`

- go-vet / golangci-lint の check derivation を `CGO_ENABLED=0` のピュア Go で実行する。nput は cgo 未使用（ADR-0011 stdlib-only）で、サンドボックスに C コンパイラを持ち込まずに検査できる。

## 根拠

- **3環境マトリクス**: 「非 NixOS でも nix さえあれば動く」（ADR-0012）という主張は Linux/macOS・x86/arm の差で破れ得る。flake check 自体を実サポート3環境で回すのが、評価・ビルド差の回帰に対する最小の保険。単一ランナーでは aarch64 / darwin 固有差を取りこぼす。
- **push トリガ削除**: PR ゲートで十分。main は merge 済みで再チェック不要、二重実行の CI 時間を削減できる。
- **`CGO_ENABLED=0`**: C ツールチェイン非依存で決定論とサンドボックス整合性を担保する。

## 影響

- `docs/design.md` テスト戦略に CI 実行（flake check マトリクス / トリガ / setup-nix）を補足した。
- cachix push（ADR-0012 §4）は別 issue のまま未着手（本 ADR の対象外）。
- E2E ジョブ（ADR-0012 §3）は #19 フェーズ2 で別途追加する（本 ADR の対象外）。

## 棄却した代替案

- **単一 `ubuntu-latest` のみ**: aarch64 / darwin 固有の評価・ビルド回帰を検出できない。
- **4 system 全部（x86_64-darwin 含む）**: GitHub ホストの標準 x86_64 macOS ランナーが乏しく、安定運用しにくい。
- **`push` トリガ併用**: PR ゲートと重複し CI 時間を浪費する。
