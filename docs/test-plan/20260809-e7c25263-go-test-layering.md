---
id: "TP-e7c25263-6d2d-4a37-8275-26906889d912"
type: test_plan
name: "エンジンの Go テストは nix を介さない実 FS 統合テストを主戦力とする"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  The engine SHALL be verified primarily by Go tests that run against a real filesystem
  under a temporary directory, with the source side supplied as an ordinary directory and
  without invoking nix at any point, so that the layer under test is the placement logic
  itself rather than the nix evaluation that produces its input. The filesystem SHALL NOT
  be abstracted behind an interface for the sake of these tests: the operations under test
  are themselves filesystem semantics (symlink, rename, unlink, permission failure), and a
  double would verify the double rather than the behaviour. The boundary the engine tests
  SHALL respect is the manifest: everything upstream of `manifest.json` belongs to the
  evaluation tests and everything downstream of the flake entrypoint belongs to the E2E
  harness. Safety-critical logic — conservative stale removal above all — SHALL be covered
  table-driven at the unit level in addition to whatever integration coverage reaches it.
specification_ja: |
  エンジンは、一時ディレクトリ下の実ファイルシステムに対して動く Go テストを主戦力として
  検証しなければならない。配置元は普通のディレクトリで与え、経路のどこでも nix を起動して
  はならない（テスト対象を、入力を作る nix 評価ではなく配置ロジックそのものに保つため）。
  これらのテストのためにファイルシステムをインターフェースで抽象化してはならない。テスト
  対象そのものがファイルシステムの意味論（symlink / rename / unlink / 権限失敗）であり、
  test double を挟めば double を検証することになるからである。エンジンのテストが守るべき
  境界は manifest であり、`manifest.json` より上流は評価テスト、flake entrypoint より
  下流は E2E ハーネスの担当としなければならない。安全に直結するロジック——とりわけ保守的
  stale 除去——は、統合テストの被覆とは別に、ユニットレベルで table-driven に覆わなければ
  ならない。
---
# TP-e7c25263: エンジンの Go テストは nix を介さない実 FS 統合テストを主戦力とする

## 仕様

エンジン（Go）のテストは、`t.TempDir()` 下の実ファイルシステムに対して動く統合テストを
主戦力とし、nix を起動しない。配置元は普通のディレクトリ（偽 source）で与える。

テスト層の境界は 2 点で切る。

| 境界 | 上流 / 下流の担当 |
|---|---|
| `manifest.json` | 上流（`mkManifest` の評価）は nix-unit / namaka（→ TP-36e90d5d）|
| flake entrypoint | 下流（`nix build` / `nix eval` / `nix-env --set` を含む実経路）は E2E（→ TP-229b69c0）|

ファイルシステムをインターフェースで抽象化しないのは、テスト対象そのものが FS の意味論
（symlink / rename / unlink / 権限失敗）だから。抽象化すると double の挙動を検証すること
になり、実 FS でしか出ない失敗（EACCES / ENOTDIR / EISDIR / EINVAL）が観測できない。故障を
実 FS の条件で誘発する具体的な手法は TP-deb05610 が定める。

保守的 stale 除去は、統合テストの被覆とは別にユニットレベルで table-driven に覆う。誤除去は
ユーザーのファイルを消す不可逆な失敗であり、正常系の統合テストでは網羅できない入力空間を
持つため。

## 出典

ADR-0006「エンジンを固定の Go バイナリにし、lib はデータ生成に徹する」の「テスト戦略」節。
同節が「エンジン（Go）: ユニットテスト（特に**保守的 stale 除去の安全不変条件**を
table-driven）+ tmpdir 統合テスト（実 FS・偽 source・nix 不使用）」と定めている。

層の切り分けは `tests/e2e/README.md` の冒頭が実態として記述している（「`lib`（nix-unit /
namaka の評価テスト）や配置エンジン（Go の tmpdir 統合テスト）が**nix を使わずに**純ロジック /
FS 操作を検証するのに対し、ここでは flake entrypoint からの実経路を一気通貫で回す」）。

> `internal/` + `cmd/nput/` の Go テストは本 item の適用対象で、テスト資産全体の中で最大の
> 規模を占める。本 item はその階層戦略のみを規範とし、個々のテストが何を検証するかは
> test_condition / test_case の担当。
