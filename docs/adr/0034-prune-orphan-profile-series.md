# ADR-0034: `nput prune` を孤児 profile 系列の削除に限定して実装する

- ステータス: 採用
- 日付: 2026-07-04
- 関連: ADR-0013 §7, ADR-0021, ADR-0025 §4, ADR-0033, `docs/spec.md`
- 改訂対象: ADR-0024 §11 の「cleanup コマンドは MVP 非対応・将来 `nput prune` の seam を残す（消費側の要求が出た時点で追加）」を、本 ADR で実装決定へ進める。seam の設計（backref `.root` による逆引き）自体は不変
- 起点: 次期マイルストーン計画の grilling（2026-07-04）。公開後の実運用（プロジェクトのクローン削除・`--root` 使い分け）で孤児系列が実際に蓄積し始めたことを「消費側の要求」と判断した

## 背景

ADR-0024 §11 は orphan profile dir（クローン削除で残る `<roothash>/<name>`）の cleanup を MVP 非対応とし、backref `.root`（ADR-0013 §7）による逆引きを将来 `nput prune` の seam として残した。profile 専用ディレクトリレイアウト（ADR-0025 §4）で backref は `<roothash>` 階層に置かれ、複数 `<name>` で共有される。

孤児系列は store を占有しない（store は `nix-collect-garbage` で解放される）が、`<state>/nix/profiles/nput/` 配下に世代 symlink dir として残り続け、手動削除には backref の中身を一つずつ確認する手間がかかる。

## 決定

### 1. 削除対象 = 「root パスが実在しない」roothash 系列のみ

- `nput prune` は `<state>/nix/profiles/nput/` 配下の **`<roothash>` 階層**（backref `.root` を持つ系列 = project mode / fixed root / `--root` 上書き）を走査し、`.root` が記録する root 絶対パスが **FS 上に実在しない**系列だけを、`<roothash>` ディレクトリごと（配下の全 `<name>` profile・世代リンク・`.pending`・`.root`）削除する。
- **判定は「root パスの実在」のみ**。root が実在する系列は、entrypoint の消失など「もう使っていない」兆候があっても対象にしない（機械判定できず誤削除リスクが高い）。
- `<name>` 直キーの系列（home mode: `<state>/nix/profiles/nput/<name>`・→ ADR-0024 §2）は root = `$HOME` が常に実在するため、構造的に prune の対象外。system mode の系列（→ ADR-0036）も root = `/` が常に実在するため同様に対象外。
- **配置物には一切触れない**。root が実在しない以上、配置先も存在せず、engine の保守的 stale 除去の不変条件（記録通りの nput 管理 symlink だけ削除）を拡張しない。
- store の回収は従来通り `nix-collect-garbage` に委ねる（prune は世代 symlink dir を消すことで gcroot を外し、次回 GC で回収可能にする）。

### 2. 安全機構 = dryrun・確認プロンプト・flock・アンマウント caveat

- **`--dryrun`**: 削除予定系列（roothash / root パス / 配下 `<name>` 一覧）を stdout に出力して終了（削除しない）。stdout 機械可読専有の規律に乗る。
- **確認プロンプト**: 実削除の前に削除対象の **root パス一覧を必ず表示**し、`reset` と同型の確認プロンプト（→ ADR-0021）を出す。`--yes` でスキップ可。非 TTY では ADR-0025 の `reset` 非 TTY 規律に合わせる（確認できなければ中止・`--yes` で明示同意）。
- **flock**: 各系列の削除前に profileDir の flock を **try-lock** で取得し、取れない系列（実行中の engine がいる）は warning を出して skip する（削除は blocking wait しない。ロック中 = root が生きている強い兆候であり、待ってまで消す対象ではない）。
- **アンマウント caveat**: out-of-store な root（リムーバブルディスク・ネットワークマウント上のプロジェクト）は、アンマウント中「一時的に実在しない」誤判定があり得る。これが確認プロンプトで root パス一覧を必ず表示する理由であり、docs にも注記する。機械側での区別（マウントポイント判定等）は行わない。
- `--json`（→ ADR-0033）に新設時から対応する（削除（予定）系列の配列）。

### 3. コマンド形

```
nput prune [--dryrun] [--yes] [--json]
```

- 引数なし（名指し不要）。走査対象はユーザー state dir 全体で、entrypoint 発見・nix eval / build を行わない（純粋に profile 状態のみを見る数少ないコマンド）。
- `rollback` / `list-generations` のようなモード限定は持たない（対象が roothash 系列に構造的に限定されるため、フラグでの絞り込みは不要）。

## 根拠

- **孤児系列のみに限定する理由**: 「root パスが実在しない」は機械的に決まる安全側の条件で、誤削除の余地が backref の記録ミス（構造的に起きない）かアンマウント（プロンプトで防ぐ）しかない。世代間引き（`--older-than` 等）は `nix-env --profile <dir> --delete-generations` が既にやれる仕事で、二重化になるため含めない。
- **try-lock skip にする理由**: ロックが取れない = その系列で engine が動いている = root が実在して使われている、なので削除対象である可能性が極めて低い。blocking wait は「使用中の系列を消すために待つ」という倒錯した挙動になる。
- **配置物に触れない理由**: prune の対象は「root が消えた」系列であり、配置物は root ごと消えている。仮に部分的に残っていても（マウント境界等）、それは prune の仕事ではなく root が復活した時の `apply` / `reset` の仕事。

## 影響

- **`docs/spec.md`**: CLI 仕様に `prune` サブコマンドを追加（引数・フラグ・確認プロンプト・終了コード）。配置動作仕様の orphan profile 節の「手動削除 + docs 注記」を prune 参照へ更新。エラー仕様表に lock skip warning を追記。
- **`docs/design.md`**: CLI サーフェス・クイックリファレンスに `prune` を追加。
- **`CONTEXT.md` / `docs/glossary.md`**: `nput CLI` のサブコマンド列挙に `prune` を追加。
- **ADR-0024**: §11 への改訂注記（本 ADR で prune を実装決定）を同一 PR で追記。
- **実装（`internal/engine` / `cmd/nput`）**: backref 走査・root 実在判定（`os.Stat`）・try-lock・系列削除（`os.RemoveAll`）。stdlib-only 制約内。
- **テスト**: 孤児系列 fixture（backref が指す root を削除）での検出・削除、lock 競合 skip、`--dryrun` の非破壊性、アンマウント相当（root 一時不在）のプロンプト表示を go test / e2e で検証。

## 棄却した代替案

- **世代間引き（`--older-than` / `--keep N`）を prune に含める**: `nix-env --delete-generations` の二重化。生存系列の世代管理は nix profile 系コマンドの守備範囲（ADR-0002 の統一方針）。
- **root 実在でも entrypoint 消失等をヒューリスティック検出して候補提示**: 「使っていない」は機械判定できず、誤検出は削除系コマンドで最悪の UX。安全側の条件のみ扱う。
- **ロック中系列を blocking wait で削除**: 使用中系列は削除対象でない。skip + warning が正しい。
- **prune を `reset --orphans` 等の既存コマンド拡張にする**: `reset` は「名指しした config の配置物を無い状態へ戻す」（ADR-0020/0021）で、配置物に触れず profile 状態だけを消す prune とは不変条件が異なる。別コマンドに分離する方が誤解が少ない。
