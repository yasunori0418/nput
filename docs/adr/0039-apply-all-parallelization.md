---
id: "ADR-0039"
type: adr
name: "`apply --all` を build・配置の両段階で並列化する（ADR-0038 の前段衝突検査を前提）"
status: 採用
origin: "次期マイルストーン追加計画の grilling（2026-07-04）。複数 profile / manifest 処理の高速化要望"
revises:
  - "ADR-0016"
references:
  - "ADR-0013"
  - "ADR-0033"
  - "ADR-0035"
  - "ADR-0038"
---
# ADR-0039: `apply --all` を build・配置の両段階で並列化する（ADR-0038 の前段衝突検査を前提）

- ステータス: 採用
- 日付: 2026-07-04
- 関連: ADR-0013, ADR-0033, ADR-0035, ADR-0038（前提: 前段衝突検査）, `docs/spec.md`, `cmd/nput/apply.go`
- 改訂対象: ADR-0016 §2 の「`apply --all` の適用順は辞書順（キーソート・決定的）」を並列適用（実行順非決定・完了後の集約表示は辞書順）へ改訂
- 起点: 次期マイルストーン追加計画の grilling（2026-07-04）。複数 profile / manifest 処理の高速化要望

## 背景

`apply --all` は一括 eval の後、config ごとに `nix build` → engine 配置を辞書順逐次（ADR-0016 §2）で回す（`cmd/nput/apply.go` の `runApplyAll` / `aggregateApply`）。支配的コストは config ごとの `nix build` で config 数に比例して直列に伸び、`method = "copy"` の大きなツリーでは配置段階も無視できない。

並列化の安全性を 3 点で精査した。

1. **flock 競合は構造的に起きない**: ロックは解決後 profileDir 単位（ADR-0013）で、`--all` の各 config は name が attrset キーとして一意 → profileDir も一意。同一プロセス内の goroutine 同士が同じロックを取り合う構造は存在しない。外部プロセスとの競合は従来通り flock が直列化する（try-lock / blocking wait の使い分けも不変）。
2. **`nix build` の並列は無条件に安全**: target 衝突の検査は build 段階には存在しない（同一 manifest 内 = eval 時・ADR-0024 §5、cross-config と祖先 symlink = 配置時）。build は link-farm derivation を store に実現するだけで配置先に触れず、store の整合は nix daemon が守る。
3. **配置並列で変わり得たのは cross-config 後勝ちの順序だけ**: これは前提 ADR-0038 が同一 entrypoint 内の衝突を前段 error で排除したため、並列化しても意味論の後退は残らない（残る非決定は別 entrypoint 等、元々レースだった領域のみ）。stale 除去の保守的不変条件は衝突下でも壊れない。

## 決定

### 1. build 段階を worker pool で並列化する

- config ごとの `nix build` を goroutine worker pool で並列実行する。並列度の既定は論理 CPU 数（`runtime.NumCPU()`）とし、上限フラグ（`--jobs` 等の名称・既定値の詳細）は実装時に確定する。
- nix daemon 側のビルド並列度とは独立の「nput が同時に投げる build プロセス数」の制御であり、二重制御にはならない。

### 2. 配置段階も config 単位の goroutine で並列化する

- ADR-0038 の前段検査を通過した config 群は、配置（engine 呼び出し）も config 単位で並列化する。flock は profileDir 単位で自然に分離される。engine 内部（1 manifest の配置）は逐次のまま変えない。
- `Warnf` / 結果集約は並行安全にする（channel または mutex）。**ユーザー向けの結果表示・`--json`（ADR-0033）の配列順は完了後に辞書順で集約**し、表示の決定性を保つ（ADR-0016 §2 の「順序の意味は表示・集約に限られる」を並列時代の形で維持）。
- warning のストリーム出力（foreign symlink 等）は発生順（非決定）で流れることを spec に注記する。
- 部分失敗の扱い・終了コード優先度（error > conflict > 0・`applyAllExitCode`）は不変。

### 3. 対象は `apply --all` に限定する

- 「複数 manifest を 1 コマンドに渡す」CLI 拡張は行わない（ADR-0035 で棄却済み・世代単位の曖昧化）。HM activation の複数 profile 起動（ADR-0035）の並列化はモジュール配線側の将来論点としてスコープ外。
- 単一 config 内の entry 単位の並列化も行わない（stale 除去・祖先 symlink 検査の順序意味論に踏み込む割に、symlink 主体では利益がない）。

## 根拠

- **両段階とも並列化する理由**: build は支配項として当然の対象。配置も、flock・stale 除去の不変条件が config 間で完全に独立しており、ADR-0038 により意味論コストが消えた以上、`method = "copy"` 大ツリーでの実益を捨てる理由がない。動機（配置処理の高速化）にも合致する。
- **engine 内部を並列化しない理由**: engine の保守的不変条件（祖先 walk → 配置 → stale 除去の順序）は 1 manifest 内の逐次性に依存しており、崩す利益が無い。並列化は「config 間」というロック境界と一致する粒度でだけ行う。
- **表示を辞書順集約にする理由**: 各 config が独立 atomic である以上、実行順に意味は無い（ADR-0016 §2 の根拠のまま）。ユーザーが読む集約結果だけ決定的であればよい。

## 影響

- **`cmd/nput/apply.go`**: `runApplyAll` / `aggregateApply` / `aggregateDryRun` の worker pool 化・集約の並行安全化・辞書順の最終ソート。
- **`docs/spec.md`**: `--all` 節に並列実行・並列度フラグ・表示の辞書順集約・warning 順序の注記。
- **ADR-0016**: 改訂対象への改訂注記を同一 PR で追記。
- **テスト**: race detector（`go test -race`）での並行テスト・部分失敗集約・e2e `--all` の結果集約順。CI への `-race` 組込は実装時に判断。

## 棄却した代替案

- **build のみ並列・配置は辞書順直列**: 安全側の折衷だが、ADR-0038 により配置並列の意味論コストが消えたため、copy 大ツリーの実益を捨てる理由がなくなった。
- **パイプライン並列（config ごと build→配置を 1 goroutine で通す）**: 実装は同等に可能だが、ADR-0038 の前段検査で build 開始前にバリアが 1 つある本設計では実質同じ重なりになり、段階を分ける方が集約・進捗表示が単純。
- **配置の goroutine を entry 単位まで細分化**: engine の順序意味論に踏み込み、保守的不変条件の証明が難しくなる。config 単位で十分。
- **並列度を無制限にする**: config 数だけ nix build プロセスが立ち上がり、daemon キューとメモリを圧迫する。CPU 数既定 + フラグ制御が妥当。
