---
id: "ADR-0038"
type: adr
name: "同一 entrypoint 内の cross-config target 衝突を `apply --all` 前段で検出し error 停止する"
status: 採用
origin: "次期マイルストーン追加計画の grilling（2026-07-04）。`apply --all` 並列化（→ ADR-0039）の検討中に「複数 config / profile / manifest 間の target 衝突をどう扱うか」が独立の前提問題として切り出された"
justifies:
  - "REQ-5c6b07da-3d06-414d-8770-4f438234b322"
  - "REQ-5923ac79-4a2d-43cd-b56c-2f1000c01b44"
  - "REQ-535b811d-dfc5-4eac-92db-737e70eb5415"
revises:
  - "ADR-0015"
  - "ADR-0024"
references:
  - "ADR-0013"
  - "ADR-0023"
  - "ADR-0035"
---
# ADR-0038: 同一 entrypoint 内の cross-config target 衝突を `apply --all` 前段で検出し error 停止する

- ステータス: 採用
- 日付: 2026-07-04
- 関連: ADR-0013, ADR-0023, ADR-0035, ADR-0039（本検査を前提に並列化）, ADR-0040, `docs/spec.md`
- 改訂対象: ADR-0015 §2 の「cross-config 同一 target は実行時後勝ち + foreign symlink warning」のうち、**同一 entrypoint 内**（`apply --all` の対象 config 間）の衝突を前段検出・error 停止へ改訂（別 entrypoint / 別ツールの衝突は実行時後勝ちのまま不変）。ADR-0024 §8 の `--all` 一括 eval を「rootKind に加え正規化後 target 一覧も取得」へ拡張
- 起点: 次期マイルストーン追加計画の grilling（2026-07-04）。`apply --all` 並列化（→ ADR-0039）の検討中に「複数 config / profile / manifest 間の target 衝突をどう扱うか」が独立の前提問題として切り出された

## 背景

別 config（= 別 manifest・別 profile）が同じ target を持つ場合、各 engine は自分の前世代 manifest としか diff しないため相手の symlink は foreign に見え、「warning + 後勝ち置換」になる（ADR-0015 §2）。`apply --all` は辞書順逐次のため後の config が決定的に勝つが、衝突は解消されず apply のたびに所有権が入れ替わるフリップフロップ状態が恒常化する。stale 除去の保守的不変条件（自分が記録した store パスを指すリンクしか消さない）は衝突下でも壊れないため実害は「どちらの symlink が残るか」に限られるが、これは誤設定であり、通しても直らない。

一方で `apply --all` の対象 config は**全て同一 entrypoint に載っている**。HM モジュールでは同じ構造（全 config が 1 つの eval に載る）を根拠に eval 時衝突検出を決定済み（ADR-0035）であり、standalone の `--all` にも同じ「eval で分かる衝突は前段で止める」原則を適用できる。加えて `--all` の並列化（ADR-0039）を入れると後勝ち順が非決定化するため、検出可能な衝突を前段で排除しておくことが並列化の前提になる。

## 決定

### 1. 検出対象 = `apply --all` で選択された config 集合内の、正規化後 target の重複

- 比較する target は正規化後（属性キー既定値・明示 `target` 上書き・`subdir` 前置〔→ ADR-0040〕を解決した後の文字列）。
- 突き合わせは **rootKind バケット単位**で行う: project 同士 / home 同士 / system 同士は同一バケット、fixed root は root 文字列値ごとに別バケット。`--root` 一律上書き（ADR-0017）が指定された場合は全 config が同一 root に解決されるため全体を 1 バケットとして検査する。
- 検出したら **build にも配置にも入らず error 停止**する（メッセージに衝突 target と両 config 名を含める）。終了コードは一般エラー（1）。

### 2. 実装層 = データ算出は nix（lib）・検出判定は Go CLI（cmd/nput）・engine は不変

- **nix（lib）**: `mkManifest` が `rootKind` を passthru している方式（ADR-0023）と同じく、**正規化後 target 一覧を passthru に追加**する。`--all` の一括 eval（ADR-0024 §8 の `--apply` 式）を拡張し、config 名 → `{ rootKind, targets }` を **1 回の安価 eval（build なし）**で取得する。
- **Go CLI（`cmd/nput`）**: 取得したマップをバケット分けして重複判定し、error 停止する。検査対象は `--all` のフィルタ（`--project-root` / `--home-root` / `--system-root`・ADR-0017）で**選択された config 集合**であり、選択ロジックは CLI にしかないため判定は CLI 層に置く。
- **Go engine（`internal/engine`）**: **変更しない**。engine は 1 起動 = 1 manifest の設計（ADR-0003 の層分離）で cross-config を構造的に見ない。検出不能な衝突への実行時後勝ち + foreign symlink warning（ADR-0015）も engine の挙動として不変。
- HM モジュールは例外的に nix 側（module eval の assertion・ADR-0035）で検出する。全 config が 1 つの module eval に載るため nix で完結できる、という CLI 検査との対称。

### 3. 検査範囲の限界を仕様として明記する

- **単一 `apply <name>` は検査しない**（対象 config しか eval しない・従来挙動不変）。sibling config との衝突は実行時 warning で観測される。
- **検出不能な衝突は従来のまま**: 別 entrypoint・別ツール・手動配置・project と fixed が偶然同じ root に解決するケース（fixed の root 値と git toplevel の一致は eval では判定できない）は実行時後勝ち + foreign symlink warning。これらは元々レースであり本 ADR のスコープ外。

## 根拠

- **error にする理由**: フリップフロップは誤設定で、warning で通しても直らない。HM（ADR-0035）と同じ error 停止に揃えることで「同一 eval に載る衝突は止まる」という一貫した期待をユーザーに与える。
- **CLI 層に判定を置く理由**: 検査集合が CLI のフィルタ選択に依存する。nix 側（`--apply` 式内）で throw する案は、フィルタを nix 式へ持ち込む複雑化になり、エラーメッセージの組み立ても CLI の方が制御しやすい。
- **passthru で target を取る理由**: rootKind と同じ「build せず安価 eval で先取りする」既存動線（ADR-0023）に乗るため、eval 回数は増えない（1 回の一括 eval の返り値が増えるだけ）。

## 影響

- **`lib/manifest.nix`**: `mkManifest` の passthru に正規化後 target 一覧を追加。
- **`cmd/nput/nix.go`**: `--all` 一括 eval の `--apply` 式を `{ rootKind, targets }` 取得へ拡張（flake / legacy `-f` の両経路）。
- **`cmd/nput/apply.go`**: build 前のバケット検査・error 停止。
- **`docs/spec.md`**: `--all` 節に前段検査（対象・バケット規則・限界）を追記。エラー仕様表に cross-config 衝突 error を追加。
- **ADR-0015 / ADR-0024**: 各改訂対象への改訂注記を同一 PR で追記。
- **テスト**: バケット判定（project/home/system/fixed 値別・`--root` 一律時の単一バケット化）・フィルタ選択との組合せ・単一 apply の非検査を go test で。e2e に衝突 fixture の error 停止を追加。

## 棄却した代替案

- **warning で続行**: フリップフロップ誤設定を温存し、並列化（ADR-0039）後は挙動が非決定になる。ADR-0035 と非対称。
- **engine 層で検出**: engine は 1 manifest しか見ない層分離（ADR-0003）を破り、複数 manifest を engine に渡す界面変更（ADR-0035 で棄却済みの方向）が要る。
- **nix 側（--apply 式内）で throw**: フィルタ選択の nix 式への持ち込みとエラーメッセージ制御の難しさに見合わない。データ算出と判定の分離が層として素直。
- **単一 `apply <name>` でも sibling を検査**: 名指し apply のたびに全 config eval が走り、単一 config の軽さ（ADR-0024 §8 の逆方向）を失う。--all のみで足りる。
