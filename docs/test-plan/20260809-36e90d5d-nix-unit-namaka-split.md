---
id: "TP-36e90d5d-4524-4294-bc72-ee263bb02782"
type: test_plan
name: "評価テストは nix-unit で不変条件を、namaka で manifest 全体のスナップショットを見る"
derives_from:
  - "SOL-9fcd1d6e-6204-42e6-92bb-1faf966f0b3e"
depends_on:
  - "TP-403c55c7-d996-4951-8e6b-c3a7dddd387c"
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
  assembles them would otherwise resolve collisions silently by last-write-wins. That
  uniqueness SHALL NOT be left to review: the aggregator SHALL check for cross-file
  duplicate names before merging and SHALL fail the evaluation, naming the colliding test
  and every file that defines it.
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
  ファイル横断で一意でなければならない（そうしなければ組み立てのマージが衝突を後勝ちで黙って
  解決してしまうため）。この一意性をレビュー任せにしてはならない。アグリゲータはマージ前に
  ファイル横断の重複名を検査し、衝突したテスト名とそれを定義している全ファイルを示して評価を
  失敗させなければならない。
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
帰結として、テスト名はファイル横断で一意でなければならない（`//` マージ自体は衝突を後勝ちで
黙って解決するので、放置すれば片方のアサートが実行されないまま緑になる）。

**一意性はレビュー任せにしない**。アグリゲータはマージ前にファイル横断の重複名を検査し、
衝突があれば衝突したテスト名と定義元ファイル名を示して評価を落とす（→ Issue #287）。検査
自身がダミー入力で検証されている（→ Issue #308、CASE-276945a3）ため、検査が常に「衝突なし」を
返す退行も捕まる。ファイル固有の接頭辞を付ける慣行は、検査に頼る前の第一手として残る。

評価テストが内部関数を叩くための seam は TP-403c55c7、store hash の揺れを避ける test double
のイディオムは TP-d3d06fe4 の担当。

## 出典

ADR-0006「エンジンを固定の Go バイナリにし、lib はデータ生成に徹する」の「テスト戦略」節が
「lib（純データ）: nix-unit で `mkManifest` の不変条件、namaka で `manifest.json` 全体の
スナップショット回帰」と定めている。アグリゲータ規約は ADR-0010 系の実装（`tests/nix-unit.nix`）が
根拠で、同ファイルのヘッダコメントが列挙方式と名前一意の要請を述べている。同コメントは
Issue #287 以降「一意性は規約に留めず、マージ前の重複検査で機械的に強制する」も述べており、
要請の記述に留まらず強制の宣言も持つ（検査の実体は `tests/nix-unit-lib.nix` の `mergeTests`。
→ Issue #308）。

この分担を実装しているテスト資産は `tests/nix-unit/` と `tests/namaka/` 配下の全テスト
（manifest-eval 区分）。件数と leaf 名はここに書き下さない——leaf を足すたびにずれる
スナップショットになるため（→ Issue #321 / #324）。現時点の一覧は列挙で得る
（`dev/scripts/test-inventory.sh --static`）。
