---
id: "TP-36e90d5d-4524-4294-bc72-ee263bb02782"
type: test_plan
name: "評価テストは nix-unit で不変条件を、namaka で manifest 全体のスナップショットを見る"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
specification: |
  The evaluation layer SHALL be verified by two tools with disjoint roles: nix-unit SHALL
  assert named invariants of the manifest-producing functions, one assertion per property
  that is stated somewhere as normative, and namaka SHALL take a snapshot of the whole
  emitted `manifest.json` so that any unintended change to the document — including changes
  to fields no invariant names — surfaces as a diff. A property that is normative SHALL be
  asserted by nix-unit and SHALL NOT be left to the snapshot alone, because a snapshot
  records what the code does rather than what it must do and is re-accepted wholesale when
  it changes. The nix-unit suite SHALL be assembled by enumerating the test directory, so
  that adding a leaf file is sufficient to run it and no aggregator or flake edit is
  required; consequently test names SHALL be unique across files, since the merge that
  assembles them resolves collisions silently by last-write-wins.
specification_ja: |
  評価層は、役割の重ならない 2 つのツールで検証しなければならない。nix-unit は manifest を
  生成する関数群の名前の付いた不変条件をアサートし（規範として述べられている性質 1 つに
  つきアサート 1 つ）、namaka は emit される `manifest.json` 全体のスナップショットを取り、
  文書への意図しない変更——どの不変条件も名指ししていないフィールドの変更を含む——が diff
  として現れるようにしなければならない。規範である性質は nix-unit でアサートしなければ
  ならず、スナップショットだけに委ねてはならない（スナップショットはあるべき姿ではなく現に
  そうなっている姿を記録するもので、変わったときは丸ごと再承認されるため）。nix-unit の
  スイートはテストディレクトリの列挙で組み立てなければならない（leaf ファイルを足すだけで
  実行され、アグリゲータや flake の編集を要さないようにするため）。その帰結として、テスト名は
  ファイル横断で一意でなければならない（組み立てのマージは衝突を後勝ちで黙って解決するため）。
---
# TP-36e90d5d: 評価テストは nix-unit で不変条件を、namaka で manifest 全体のスナップショットを見る

## 仕様

評価層（lib）の検証は 2 ツールで役割を分ける。

| ツール | 役割 |
|---|---|
| nix-unit | manifest 生成関数の**名前の付いた不変条件**を 1 性質 1 アサートで検証する |
| namaka | emit される `manifest.json` **全体**のスナップショット回帰。どの不変条件も名指ししていないフィールドの変化も diff で出す |

規範である性質はスナップショット任せにしない。スナップショットは「あるべき姿」ではなく
「現にそうなっている姿」の記録で、変わったときは丸ごと再承認される（退行と意図した変更を
区別しない）ため。

**アグリゲータ規約**: nix-unit のスイートはテストディレクトリの列挙で組み立てる。leaf な
テストファイルを足すだけで自動的に載り、集約ファイルや `flake.nix` の編集を要さない。
帰結として、テスト名はファイル横断で一意でなければならない（マージが後勝ちで、衝突しても
エラーにならず片方が黙って消えるため）。

評価テストが内部関数を叩くための seam は TP-403c55c7、store hash の揺れを避ける test double
のイディオムは TP-d3d06fe4 の担当。

## 出典

ADR-0006「エンジンを固定の Go バイナリにし、lib はデータ生成に徹する」の「テスト戦略」節が
「lib（純データ）: nix-unit で `mkManifest` の不変条件、namaka で `manifest.json` 全体の
スナップショット回帰」と定めている。アグリゲータ規約は ADR-0010 系の実装（`tests/nix-unit.nix`）が
根拠で、同ファイルのヘッダコメントが列挙方式と名前一意の要請を述べている。

現況は nix-unit 7 ファイル（structure / defaults / gates / escapes-base / anchor-name /
resolve-marker / farm-entries）+ namaka スナップショット 1 件（manifest-project）。
