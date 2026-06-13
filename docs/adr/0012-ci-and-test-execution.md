# ADR-0012: CI・テスト実行基盤を cryoflow 構成踏襲で確定する（flake check と E2E ジョブの分離）

- ステータス: 採用
- 日付: 2026-06-13
- 関連: ADR-0006, ADR-0011, `docs/spec.md`
- 参照: `~/src/github.com/yasunori0418/cryoflow` の CI 構成（同一メンテナの実績構成）

## 背景

ADR-0006 はテスト戦略（lib = nix-unit / namaka、engine = Go ユニット + tmpdir 統合、E2E = 非 NixOS + nix コンテナ）を
決めていたが、**それを CI でどう実行するか**（インストーラ・実行場所・キャッシュ）が未確定だった。

同一メンテナの cryoflow プロジェクトが既に Nix ベースの CI を運用しており、これを参照基準とする。
cryoflow の構成要素は次の通り。

- `.github/actions/setup-nix` composite action: `cachix/install-nix-action` + `cachix/cachix-action`（キャッシュ名 `yasunori0418`）。各 action は SHA pin。
- `dev/flake.nix` が `ci` devShell（dev 専用ツールを抜いた最小構成）を公開し、`nix develop '.?dir=dev#ci' -c <cmd>` で叩く。
- 軽いテスト（pytest）は devShell ジョブ、ビルドは matrix（x86_64-linux / aarch64-linux / aarch64-darwin）の別ジョブ。

## 決定

### 1. nix インストーラと共通化 = cryoflow 踏襲

- nix インストーラは **`cachix/install-nix-action`**（SHA pin）。ADR-0006 の「非 NixOS + nix」E2E はこの経路で実現する。
- `.github/actions/setup-nix` composite action に install + cachix（`yasunori0418`）をまとめ、全 workflow から再利用する。

### 2. CI シェルを `dev/flake.nix` に公開

- `dev/flake.nix` に `ci` devShell を置き、ビルド / テストツールチェイン（go・nix-unit・namaka 等）を最小構成で提供する。
- CI は `nix develop '.?dir=dev#ci' -c <cmd>` で実行する。

### 3. テスト層を「sandbox 可能 = flake check / E2E = 別ジョブ」で分ける

- **lib（nix-unit / namaka）と engine Go（ユニット + tmpdir 統合）は `nix flake check`** に集約し PR で常時実行する。
  いずれも純評価・偽 source・nix 不使用でサンドボックスと相性が良い。
- **E2E は ubuntu-latest の別ジョブ**。`cachix/install-nix-action` で nix を入れた後、フィクスチャに対し `nput apply` を
  実行し FS / nix profile / rollback をアサートする。profile 書込・実 FS 変更・特権を伴い `nix flake check` の
  サンドボックスでは成立しないため、check 外のジョブに出す。cryoflow の「軽い = check 相当 / 重い = devShell ジョブ」分離と同型。

### 4. キャッシュ投入 = matrix ビルド

- tag push 時に matrix（x86_64-linux / aarch64-linux / aarch64-darwin）で `nix build .#nput` し cachix `yasunori0418` に投入する。

## 根拠

- **同一メンテナの実績構成**を踏襲することで CI の保守コストと再現性が揃う。
- **E2E をサンドボックス外に出す**のは技術的必然。nput の E2E は nix profile を実際に書き換え rollback を検証するため、
  ネットワーク遮断・特権制限のある `nix flake check` 内では成立しない。検証対象（非 NixOS + nix で nput が動く）と
  install-nix-action のジョブが直接一致する。
- **dockerTools 自作イメージは過剰**。「nix 入りイメージ」を dockerTools で機能させるには store / daemon / sandbox の配管が要り、
  `cachix/install-nix-action` が既に提供する機能の再発明になる。検証対象は image 構築の再現性ではない。

## 影響

- 実装フェーズで `.github/actions/setup-nix/`・`.github/workflows/`（flake check / E2E / cachix-push）・`dev/flake.nix` の
  `ci` devShell を追加する。
- `docs/spec.md` のテスト記述に CI 実行基盤（flake check / E2E ジョブ分離）を補足する。

## 棄却した代替案

- **pkgs.dockerTools でテストイメージを自作**: nix を image 内で機能させる配管が重く、`cachix/install-nix-action` の再発明。
  結局 CI ジョブで動かす必要があり、実行基盤の決定として image 構築しか答えていない。
- **全層を `ci` devShell の 1 ジョブで実行**: 純評価テスト（lib/engine）まで `nix flake check` 外に出てしまい、
  PR 常時チェックの恩恵を削る。
- **全層を `nix flake check` に詰める**: E2E の profile 書込・特権がサンドボックスと衝突し成立困難。
- **DeterminateSystems/nix-installer を使う**: 動作するが cryoflow が `cachix/install-nix-action` で揃っており、
  メンテナのツールチェインを統一する方が保守上有利。
