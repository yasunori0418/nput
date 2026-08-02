# nput 設計書

要求（requirement）を「どう実現するか」の全体像と、個別設計（design item）への索引。

設計判断は **すべて `docs/design/` の item が持ち**、その決定と根拠は `docs/adr/` の ADR が
`justifies` で裏づける。本文書は通読の入口として全体像だけを述べ、詳細は item へのリンクで
示す（README → 本文書 → item の 3 層構造）。

> **この文書の書き方（規約）**
>
> - 散文は **見出し 1 つ（h2 / h3 の最も内側）につき 10 行以内**。表・リンク列挙・コード
>   ブロックはこの制限に含めない
> - **詳細は必ず item へリンクし、本文に書き下さない**。設計の断片をここへ写すと item と
>   二重管理になり、改訂時に片方だけ古くなる。**例外は「設計目標」の表**で、これは設計判断を
>   評価する軸の横断ビューであり、対応する item を持たないため本文に置く。各目標がどう
>   満たされるかは requirement / design item が持ち、表には書かない
> - 「何を満たすべきか」は requirement の領分で本文書は扱わない（→ `docs/spec.md`）
> - item を横断して検索・追跡するには `sara query`（→ `docs/agents/domain.md`）を使う
> - item の `## 出典` は本文書の現行章立てを指さない。原文の読み方を含め `docs/spec.md` の
>   同項が正（同項は 3 文書共通の手順として書いてある）

---

## 概要

nix store のパス（リポジトリ全体・サブディレクトリ・単一ファイル）を、root 相対の任意パスへ
symlink または copy で配置する Nix ライブラリ・モジュール群。配置ロジックはテスト可能な純粋関数 +
単一の配置エンジンとして実装し、ユーザーが配置を明示的に握る。home-manager に依存せず単体で
動作しつつ、HM / NixOS / nix-darwin のモジュールとも統合できる。ただし統合層は配置ロジックを
持たず、エンジンを起動する薄い配線に徹する（→ ADR-0003）。

nput が何であるか・何のために作るかは solution / use_case の領分。→ `docs/concept.md`。

---

## 設計目標

設計判断を評価する軸の一覧。**各軸が何を要求するか（規範）は requirement が持ち、どう実現するかは
design item が持つ**。ここは軸の名前と着眼点だけを並べ、満たし方は書かない。

| 目標 | 着眼点 |
|---|---|
| 純粋性・テスト可能性 | 配置ロジックをモジュールに依存せずテストできるか |
| 独立性 | home-manager 不在の環境で単体で動くか |
| 統合性 | 既存のモジュールシステムから起動できるか |
| 柔軟性 | モジュールシステムを介さずに使えるか |
| 取得手段非依存 | 取得方法の変化が nput の変更を要求しないか |
| 粒度 | リポジトリ全体・サブディレクトリ・単一ファイルを同じ扱いにできるか |
| 更新の独立性 | 1 つの更新が他の配置単位へ波及しないか |
| 非生成 | ファイルの内容に関与していないか |
| 世代管理 | 適用を取り消せるか |
| root 明示 | 配置先が暗黙のデフォルトに落ちていないか |

---

## プロジェクト構成

リポジトリはトップレベル 5 ディレクトリ（`lib/` / `cmd/` / `internal/` / `templates/` /
`modules/`）に分ける。配置ロジックは Go エンジンが単一の源として所有し、`lib/` はデータ生成に
徹する（→ ADR-0006, ADR-0007）。

- [DSG-1361df1a](design/20260802-1361df1a-repo-layout.md) — リポジトリを lib / cmd / internal / templates / modules のトップレベル 5 ディレクトリへ分ける
- [DSG-e4d5db6b](design/20260802-e4d5db6b-lib-module-split.md) — lib は公開 API・型定義・manifest 生成・マーカー構築子の 4 ファイルへ分割する
- [DSG-7d354fe0](design/20260802-7d354fe0-go-package-split.md) — Go 側は cmd/nput に CLI 面を、internal に配置ロジックを置く 2 パッケージ構成にする
- [DSG-aeb5e219](design/20260802-aeb5e219-modules-file-split.md) — modules は common.nix に共通オプションを集約し、統合層ごとに 1 ファイルへ分ける
- [DSG-4a84f282](design/20260802-4a84f282-implementation-scope.md) — 実装スコープを standalone CLI + project mode + home mode に限り、system mode とモジュール 2 層は将来拡張に置く

---

## レイヤー構成

CLI / engine / lib / `common.nix` / 統合層の 5 段に積み、依存は呼ぶ側から呼ばれる側への一方向に
限る。配置の振る舞いは全層でエンジンが単一の源であり、各層はネイティブ機構へ翻訳しない
（→ ADR-0003, ADR-0006）。

- [DSG-17db0831](design/20260802-17db0831-layer-stack-dependency.md) — 層を CLI / engine / lib / common.nix / 統合層の 5 段に積み、依存を呼ぶ側から呼ばれる側への一方向に限る

---

## flake.nix outputs 設計

nput 自身の flake outputs は packages / templates / 各モジュール / flakeModules / lib の 5 系統。
ユーザーの entrypoint 側は `nput.<name>` に named manifest を公開し、直書きと flake-parts の
2 経路が同一の derivation を生む（→ ADR-0007, ADR-0029, ADR-0032）。

- [DSG-16373ec2](design/20260802-16373ec2-flake-outputs-attrs.md) — nput 自身の flake outputs は packages / templates / 各モジュール / flakeModules / lib の 5 系統で構成する
- [DSG-0e186e89](design/20260802-0e186e89-module-stubs.md) — NixOS / nix-darwin モジュールは中身を将来拡張としたままスタブとして公開する
- [DSG-d2e17f4f](design/20260802-d2e17f4f-lib-no-dynamic-entries.md) — 動的 entry 生成のヘルパを lib に置かず、readDir する idiom をドキュメントで示す
- [DSG-92f54490](design/20260802-92f54490-legacy-nix-invocation-reuse.md) — legacy entrypoint の分岐は attr path 組み立てに閉じ、nix 呼び出しヘルパを共通で再利用する

---

## コアロジック設計（lib データ生成 + Go エンジン）

`lib.mkManifest` が link farm derivation（`manifest.json` + symlink ツリー）を生成し、エンジンが
実行時に前世代 manifest と diff して配置・stale 除去・profile swap を行う（→ ADR-0002, ADR-0006）。

- [DSG-0b94308c](design/20260802-0b94308c-cleanup-from-home-manager.md) — 配置と cleanup のアルゴリズムは home-manager の linkGeneration / cleanup を範として Go で再実装する

`lib` の API・`entries` スキーマ・`manifest.json` の内容・CLI の実行モデルは仕様（requirement）の
領分（→ `docs/spec.md`「lib API」「entries スキーマ仕様」「manifest.json スキーマ」「CLI 仕様」）。

---

## モジュール統合設計

全モジュールと devShell は root と activation タイミングを供給してエンジンを起動する薄い配線。
起動の差は link-farm の取得方法だけに閉じる（→ ADR-0003, ADR-0026）。

- [DSG-98d7fa5d](design/20260802-98d7fa5d-integration-layer-kick-classes.md) — engine の起動を entrypoint 駆動とビルド済み manifest の 2 クラスに束ね、統合層の差を link-farm の取得方法だけに閉じる

各モジュールが公開するオプションと動作は仕様の領分
（→ `docs/spec.md`「モジュールオプション仕様・モジュール別動作仕様」）。

---

## 使用パターン

→ `docs/concept.md`「想定する使われ方」（use_case 7 件）。個々の設定の書き方はテンプレート
（`nput init`）と `templates/` の実物を参照する。

---

## テスト戦略

lib は Nix 評価テスト、エンジンは実 FS の tmpdir 統合テスト、非 NixOS で動く主張は実 nix の
E2E で検証する（→ ADR-0006, ADR-0007, ADR-0012）。

- [DSG-fb49e36c](design/20260802-fb49e36c-lib-test-nix-unit-namaka.md) — lib は nix-unit の評価テストと namaka のスナップショットの 2 手段で検証する
- [DSG-836aa5cb](design/20260802-836aa5cb-engine-test-go-unit-tmpdir.md) — engine は nix を使わない tmpdir 統合テストで検証し、stale 除去の安全不変条件を table-driven に重点配分する
- [DSG-2947b4a5](design/20260802-2947b4a5-e2e-harness-shape.md) — E2E は tests/e2e の bash ハーネスに置き、シナリオの詳細は同ディレクトリの README へ委譲する
- [DSG-bb6e03fd](design/20260802-bb6e03fd-e2e-legacy-entrypoint.md) — E2E に legacy entrypoint シナリオを置き、NIX_PATH を flake.lock の nixpkgs に pin して検証する
- [DSG-901351ea](design/20260802-901351ea-nixos-vm-test-position.md) — NixOS VM テストは runNixOSTest でモジュール経路の実装時に追加し、E2E ハーネスとは別系統に置く

CI での実行基盤・キャッシュ・リリースは infrastructure item の領分。

- [INF-d1230e1f](infrastructure/20260802-d1230e1f-ci-pipeline.md) — CI パイプライン（flake check の os×system マトリクスと E2E ジョブ）
- [INF-af33c5a1](infrastructure/20260802-af33c5a1-binary-cache.md) — バイナリキャッシュ（main push 時の cachix 投入）
- [INF-9878e9f5](infrastructure/20260802-9878e9f5-release-automation.md) — リリース自動化（VERSION ファイル起点の bump PR・自動タグ・自動リリースノート）
- [INF-8b97573f](infrastructure/20260802-8b97573f-merge-gate.md) — マージゲート（main の ruleset と required status checks）
- [INF-659b139d](infrastructure/20260802-659b139d-traceability-verification.md) — トレーサビリティ検証基盤（sara によるドキュメントグラフの機械検証）
- [INF-0865477b](infrastructure/20260802-0865477b-docs-site.md) — ドキュメントサイト（Astro Starlight + Cloudflare Pages・ビルド時リファレンス生成）

---

## 設計判断の所在

本節はリンク集ではなく、決定がどこにあるかの案内。主要な設計判断の決定と根拠は **ADR が持つ**
（`docs/adr/`）。ADR は `justifies` で requirement / design / infrastructure へ接続しているため、
ある item を裏づける決定は本文書の一覧ではなく `sara query <フル ID> -u` で辿る。全 ADR を
ここへ並べても改訂のたびに古くなるだけなので置かない。

---

## 関連文書

- `README.md` / `README.ja.md` — 3 層構造の最上段（導入と使い方）
- `docs/concept.md` — コンセプト（solution / use_case への索引）
- `docs/spec.md` — 仕様（requirement item への索引）
- `docs/adr/` — 意思決定の記録
- `docs/model.yaml` — sara の型定義（item の型・関係・ID 形式）
