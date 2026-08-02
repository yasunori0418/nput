---
id: "DSG-836aa5cb-0389-4adb-990b-144fe5aeffe3"
type: design
name: "engine は nix を使わない tmpdir 統合テストで検証し、stale 除去の安全不変条件を table-driven に重点配分する"
satisfies:
  - "REQ-16aef46b-7bb8-4ca1-b962-e9f3ed1fd1d2"
  - "REQ-b74a118a-1272-44eb-944c-7725163211c6"
  - "REQ-5e75aabc-0e8f-4a6c-92bd-a712dc68a940"
  - "REQ-d2277c7a-7992-49af-a9dc-4cc73843a6f9"
---
# DSG-836aa5cb: engine は nix を使わない tmpdir 統合テストで検証し、stale 除去の安全不変条件を table-driven に重点配分する

## 設計

engine（`internal/`）は **Go ユニットテスト + tmpdir 統合テスト**で検証する。
統合テストは実 FS を使うが、**source は偽物・nix は使わない**。

重点は **保守的 stale 除去の安全不変条件（誤削除防止）を table-driven で網羅する**ことに置く。

この形を選ぶ理由。

- **実 FS を使うが nix を使わない**という中間の位置取りが、engine の責務範囲と
  一致する。REQ-b74a118a の通り engine は stdlib-only で、その仕事は
  `manifest.json` を入力に FS を操作することに尽きる。nix を実際に走らせなくても
  engine の全経路を踏める（実 nix の一気通貫は DSG-2947b4a5 の E2E が担う）。
  FS をモックしないのは、symlink・パーミッション・親ディレクトリといった
  検証対象がモックでは再現しきれないため
- **stale 除去に重点を置く**のは、そこが**壊れたときの被害が非対称**だからである。
  REQ-16aef46b が定める不変条件（前世代の記録通りを指す symlink のみ消し、通常ファイル・
  非管理 link には触れず、copy target は消さない）が破れると**ユーザーの実ファイルが
  消える**。配置側の失敗は「置かれない」で済むが、除去側の失敗は復旧できない
- **table-driven にする**のは、この不変条件が「前世代の記録 × 現状の実体」の
  組み合わせに対する判定表そのものだからである。記録あり/なし、実体が
  symlink/通常ファイル/ディレクトリ/不在、リンク先が記録通り/別物 — といった
  組み合わせを網羅する形式として、ケース表がそのままテストになる

同じ tmpdir 統合テストが、REQ-5e75aabc の undo ジャーナル（途中失敗時の巻き戻し）と
REQ-d2277c7a の copy place-once（target 不在時のみマテリアライズ）の検証も担う。
どちらも「途中で失敗させる」「target を先に作っておく」といった FS 状態の作り込みが
必要で、実 FS 上でないと組めない。

## 出典

`docs/design.md`「テスト戦略」のテーブル 2 行目。
