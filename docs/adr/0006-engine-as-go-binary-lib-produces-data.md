# ADR-0006: エンジンを固定の Go バイナリにし、lib はデータ生成に徹する（生成 bash を廃する）

- ステータス: 採用（2026-06-13 一部改訂: 「nput の露出と環境セットアップ」節・実行フロー・該当棄却案を ADR-0007 が反転）
- 日付: 2026-06-11
- 関連: ADR-0002, ADR-0003, ADR-0005, ADR-0007, `docs/concept.md`, `docs/design.md`, `docs/spec.md`

> **2026-06-13 改訂（ADR-0007）**: 本 ADR の以下の決定は ADR-0007 で反転した。
> - 「エンジン `nput` は PATH に常駐せず per-config ラッパー（`nix run .#x`）経由」→ **汎用 `nput` CLI を PATH に常駐させ一次 UX に**。`mkActivationScript` ラッパーは廃止（`mkManifest` は存続）。
> - 「グローバル `nput` に config 発見機構を足さない」「`nput init` スキャフォルドは採らない」（棄却案）→ **entrypoint ファイル発見（CWD / `-f`）と `nput init`（`nix flake init -t` ラッパー）を採用**。config を Nix で書き `nix build` で評価する thesis は維持。
> - エンジンは Go ライブラリ化し、`cmd/nput` CLI が import する（`manifest.json` in 契約はライブラリ API として温存）。
> 固定 Go エンジン・`manifest.json` 契約・profile / 世代 / 保守的 stale 除去・ネイティブ FS という他の決定は不変。

## 背景

初期設計（ADR-0002）では `mkActivationScript` が `pkgs.writeShellApplication` で **config ごとに entries を埋め込んだ
bash スクリプトを生成**し、`result/bin/nput` として実行する形だった。配置ロジック（`ln` / `rsync` / stale 除去 / profile swap）は
この生成 bash の中にあった。

しかしこれは concept.md の 2 つの主張と矛盾する。

- **「テスト可能な純粋関数群」**: 実際の配置ロジックが生成 bash 内にあり、最もテストしづらい成果物になる。純粋なのは
  「bash 文字列と link farm を組み立てる部分」だけ。
- **「engine = 全層で配置を一手に所有する*単一の*コア機構」（ADR-0003）**: 「単一」と言いつつ実体は config ごとに再生成される
  N 個の bash スクリプトであり、単一のコアではない。

つまり nput は実質「shell script 生成ツール」になっていた。実装前にこのギャップを正す。

## 決定

- **エンジン = 単一の固定 Go バイナリ**（`packages.nput`、`buildGoModule`）。配置ロジックは 1 箇所だけに存在する。
- **lib は純データ生成器に徹する。** `lib.mkManifest { entries, root } -> derivation` が **`manifest.json` + symlink farm を含む
  derivation（データ）だけ**を生成する純粋関数。配置ロジックは持たない。
- **`mkActivationScript` は両者を配線する薄いラッパー**（`writeShellApplication` で `${nput}/bin/nput apply ${manifest}` を呼ぶ）。
  これにより `nix run .#vim-plugins` の per-config 独立適用 UX を保ちつつ、ロジックは固定エンジン 1 箇所に閉じる（→ ADR-0005 のエンジン形 C）。
- **Nix↔Go の契約 = `manifest.json`**。前方互換のため `schemaVersion` を持ち、エンジンが対応版より新しい manifest を拒否する。
- **GC アンカー = `manifest.json` + symlink farm**。配置元への明示 symlink ツリーで GC 参照を堅牢に張る（文字列スキャン依存にしない）。
  profile はこの derivation を指す。
- **FS 操作（symlink / copy）はネイティブ Go**（`os.Symlink` / ネイティブ再帰コピー）。`nix`（profile）と `git`（toplevel）のみ
  サブプロセスで叩く。`rsync` / `ln` の runtime 依存を落とす。
- **profile 操作 = `nix-env --profile <dir> --set <link-farm>`**。「1 世代 = 1 link-farm の atomic 置換」に唯一きれいに対応する
  primitive（ADR-0002）。各世代が GC root になるのは `--set` の副作用で、自前 gcroot 登録は不要。
- **並行実行 = `profileDir` 単位の flock + try-lock**（短い待機後、保持中ならスキップして通知）。devShell の高頻度キックで
  待機ハングもキュー積み上がりも避ける。保持者が同一 manifest を適用して収束させるためスキップしても冪等性で安全。
- **部分失敗 = コミット最後**。配置（新規 / 既存 nput symlink の張替 / stale 除去）を全て成功させてから `--set` する。
  途中失敗は非ゼロ中断・`--set` せず前世代を記録状態に保つ。FS は一時的に部分適用されうるが冪等な再実行で収束し、
  **積まれる世代は常に完全適用済み**になる。配置順は「新規 / 張替を先に置き、最後に stale 除去」（現状態を壊す前に新状態を揃える）。

### CLI 構造（サブコマンド体系）

ADR-0002 のフラグ式 CLI（`nput --rollback` 等）を、独立操作は **サブコマンド体系**に置き換える。
dry-run だけは apply の挙動を副作用なしで見る**モード**なので、独立操作ではなく `--dryrun` フラグとする。

```
nput                   # = apply（サブコマンド省略時のデフォルト）
nput apply             # 新世代を作って配置（明示）
nput apply --dryrun    # dry-run（副作用ゼロ。下記）。bare の nput --dryrun も同義
nput rollback          # 前世代へ（home mode 限定）
nput list-generations  # 世代一覧（home mode 限定）
nput gitignore         # 配置 target を .gitignore 向けに stdout 出力（書き込みなし、→ ADR-0005）
nput <sub> --root <p>  # project mode の root 上書き（global flag）
```

per-config ラッパーは `exec ${nput}/bin/nput --manifest ${manifest} "$@"` の形でサブコマンド / フラグを転送する。これにより
`nix run .#x`（= apply）/ `nix run .#x -- --dryrun` / `nix run .#x -- rollback` という UX になる。サブコマンド省略時は apply に
フォールバックして `nix run .#x` の素直な適用 UX を保つ。

### dry-run（`--dryrun` フラグ）

apply 前に「何が起きるか」を表示する**副作用ゼロ**のモード（apply への `--dryrun` フラグ）。Go エンジンだからこそ素直に実装でき、nput の
「ユーザーが配置を明示的に握る」哲学に沿う。サブコマンド化（`plan`）は棄却し、apply の修飾フラグとした。

- 前世代 manifest との diff を人間可読で表示する: `place`（新規）/ `replace`（既存 nput symlink の旧→新）/ `remove`（stale 除去）/
  `conflict`（nput 非管理の実ファイルがあり apply ならエラー停止する箇所）/ `no-op`（project mode の世代スキップ該当）。
- **FS 書込なし・`--set` なし・flock も取らない**（読み取り専用。別 apply と競っても plan が少し古くなるだけ）。root 解決は読み取りのみ行う。
- 終了コード: `conflict` があれば**非ゼロ**、無ければ 0。CI の事前チェック（apply 可能性の gate）に使える。
- 機械可読出力（`--json` 等）は将来拡張に留める。

### nput の露出と環境セットアップ

- **エンジン `nput` はユーザーの PATH に常駐しない。** per-config ラッパー（`nix run .#x`）経由で起動され、ラッパーが
  `--manifest` を注入する。`rollback` / `gitignore` / `--dryrun` 等は per-config（profile / manifest 単位）の操作であり、
  「どの配置を対象にするか」は manifest コンテキストで決まる。裸のグローバル `nput` には manifest コンテキストが無い。
- **グローバル `nput` に「CWD / 設定ファイルから config を発見する」機構は足さない。** config は Nix 評価が確定する（flake で書く）モデルを保つ。
- **環境セットアップは Nix `templates` flake output で提供する。** `nix flake init -t <nput>#standalone` / `#project` で starter flake を展開する
  （`#project` は devShell shellHook 配線 + `.gitignore` ガイド入り）。nput 自身はファイルを 1 文字も生成せず、「設定を生成しない」thesis を保つ。

### 実行フロー

```
ラッパー (nix run .#x) → nput apply --manifest <link-farm path> [--root projectRoot]
  1. flock(profileDir) を try-lock（保持中ならスキップ）
  2. root 解決（home: $HOME / project: git rev-parse、--root 上書き）
  3. profileDir の前世代 manifest.json を読む（無ければ初回 = 削除対象ゼロ）
  4. project mode かつ新 link-farm が前世代と同一なら no-op で終了（世代スキップ）
  5. manifest.json を読み新旧 diff → 新規 / 張替を配置 → 保守的 stale 除去（ネイティブ FS）
  6. nix-env --profile <profileDir> --set <link-farm path>（サブプロセス・コミット点）
```

### テスト戦略

- **lib（純データ）**: nix-unit で `mkManifest` の不変条件、namaka で `manifest.json` 全体のスナップショット回帰。
- **エンジン（Go）**: ユニットテスト（特に**保守的 stale 除去の安全不変条件**を table-driven）+ tmpdir 統合テスト（実 FS・偽 source・nix 不使用）。
- **E2E**: 非 NixOS + nix のコンテナで `nix run .#x` → FS / profile / rollback をアサート（nput の「非 NixOS でも nix さえあれば動く」主張を検証）。
  NixOS VM テスト（`runNixOSTest`）はモジュール経路の実装段で追加。

### ソース配置

- `cmd/nput/`（エントリポイント）+ `internal/`（配置 / diff / stale 除去の純ロジック）。`packages.nput` = `buildGoModule`。

## 根拠

- 「単一エンジン・テスト可能」を文字通り満たす。lib は Nix で、エンジンは Go でテストでき、Nix↔Go の契約が `manifest.json` 1 点に閉じる。
- **Go**: 安全クリティカルな保守的 stale 除去をユニットテストできる。単一静的バイナリ・`buildGoModule` で nixpkgs ビルドが容易で、
  サブプロセス / FS 操作が素直。エンジンは結局 `nix-env` / `git` を叩くオーケストレータなので Go が実務的。
- **manifest + symlink farm**: GC 参照を store パス文字列の内容スキャンに頼らず明示 symlink で張ることで、rollback の安全性
  （ADR-0002 の核心要求）に堅牢。
- **`nix-env --set`**: 「1 世代 = 1 link-farm の atomic 置換」に唯一きれいに対応。非推奨扱いなのは `nix-env -i` による
  ユーザー環境の命令的管理であって、`--profile --set` の低レベル profile primitive は HM / NixOS も依存している現役経路。

## 影響

- **ADR-0002 を実装機構の面で置換**: 「`writeShellApplication` で生成」「activation スクリプト」は固定 Go エンジンに置き換わる。
  ただし profile / 世代 / store マニフェスト / 保守的 stale 除去という **decision 自体は不変**（機構の担い手が生成 bash → Go エンジンに変わるだけ）。
  ADR-0002 のフラグ式 CLI（`nput --rollback` / `--list-generations`）は本 ADR のサブコマンド体系（`nput rollback` / `list-generations` / `gitignore`）に置き換わる。dry-run は `--dryrun` フラグ。
- **ADR-0003 を refine**: 「nput 自身の `ln` / `rsync`」はネイティブ Go FS 操作に置き換え、`rsync` 依存を廃止する。
  「単一エンジンが全層で配置を所有」という決定はむしろ強化される（生成 bash N 個 → 固定バイナリ 1 つ）。
- `docs/spec.md` / `docs/design.md`: `lib.mkManifest` 追加、`mkActivationScript` をラッパーとして再定義、依存表から `rsync` 削除・
  Go エンジン追加、CLI フロー、locking / 部分失敗 / `schemaVersion` を反映。
- `CONTEXT.md`: `engine` 定義を「固定 Go バイナリ・ネイティブ FS」に更新し、`manifest`（Nix↔Go 契約）を追記。
- `flake.nix`: `packages.<system>.nput`（`buildGoModule`）の追加は実装フェーズの作業。

## 棄却した代替案

- **config ごとの生成 bash（現状の暗黙設計）**: 単一エンジン・テスト可能性に反する。本 ADR が正す対象そのもの。
- **固定 bash 1 スクリプト**: コンパイル不要で純 Nix だが、安全クリティカルな保守的 stale 除去のロジックを bash で正しく書く / テストするのが難しい。
- **Rust**: 同等に堅牢で妥当だが反復コストが重め。CLI / サブプロセス指向の素直さと nixpkgs ビルド容易性で Go を採用。
- **`nix profile`（新 CLI）**: 「単一 store パスへの atomic set」操作が無く、「1 世代 = 1 link-farm」モデルと噛み合わない。
  remove+install の 2 手は atomic 性が落ち中間世代が濁る。
- **エンジンが世代 symlink を直接管理**: CLI 非依存で完全制御だが、GC root 登録・世代採番の正しさを自前で背負う。`nix-env --set` に委譲する方が安全。
- **`rsync` / `ln` サブプロセス**: runtime 依存が残り、FS ロジックのユニットテストが難しい。ネイティブ Go FS を採用。
- **`manifest.json` のみ（symlink farm 無し）**: GC 参照が store パス文字列の内容スキャン依存で、パス表現を変えた瞬間に静かに参照が消える脆さがある。
- **dry-run を `nput plan` サブコマンドにする**: terraform 風で読みやすいが、dry-run は apply の挙動を副作用なく見る**モード**であって独立操作ではない。
  apply の修飾フラグ `--dryrun` の方が意味的に正しく、サブコマンド表面積も増やさない。
- **グローバル `nput` に config 発見機構を足す / `nput init` でファイルをスキャフォルドする**: config を Nix 評価でなく CWD / 設定ファイルから
  発見するのは現モデルと衝突する。環境セットアップは Nix `templates`（`nix flake init -t`）で賄え、nput がファイルを生成すれば「設定を生成しない」thesis を傷つける。
