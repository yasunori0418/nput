# ADR-0011: engine / CLI の技術スタックを確定する（依存方針・cobra・store path 取得・vendorHash・ロック）

- ステータス: 採用
- 日付: 2026-06-13
- 関連: ADR-0006, ADR-0007, ADR-0010, `docs/spec.md`, `docs/concept.md`
- 改訂対象: ADR-0007「store path 取得」を out-link 方式に具体化。ADR-0006 / ADR-0007 の「Go ライブラリ」語を `internal/` 層分離と明文化

> **2026-06-14 改訂注記（ADR-0016）**: 本 ADR の以下を拡張・具体化した。
> - **再帰コピーの「mode 保存」→「mode 保存 + owner-write 付与」**。store の read-only mode（0444 / 0555）をそのまま保存するとコピー先が編集できず copy の用途に反するため、コピー後に owner-write を付与する（perm 相対構造は保持）。`os.CopyFS` 棄却理由（perm 非保存・symlink 非対応）は不変。
> - **src ツリー内 symlink は symlink のまま複製**（deref しない）。store 内への絶対 symlink は store 依存が残る点を docs に明記。
> - **symlink farm の GC アンカー名は `target` のハッシュ**（サニタイズだと別 target が同名衝突する）。

## 背景

ADR-0006（言語 = Go・lib はデータ生成・固定エンジン）と ADR-0007（CLI を一次 UX に・engine をライブラリ化）で
大枠の技術方針は決まっていた。しかしその**下位の具体技術**が未確定だった。

- CLI フレームワーク（cobra / kong / stdlib flag …）
- 第三者ライブラリをどこまで許すか（stdlib-only か）
- CLI が `nix build` の link-farm store path を取得する手段
- `buildGoModule` の vendor 管理戦略（cobra 依存が入ったため vendorHash が必要）
- `profileDir` ロックの実装手段
- 再帰コピー / JSON / エラー処理の実装基盤
- Go toolchain のバージョン pin 方法（Nix サンドボックスはネットワーク遮断）

ADR-0006 の「ソース配置 = `cmd/nput/` + `internal/`」と ADR-0007 の「engine を Go ライブラリとして実装」も、
**「ライブラリ」が公開 import 可能な再利用モジュールを意味するのか、バイナリ内部の層分離なのか**が曖昧だった。

## 決定

### 1. 依存ポリシーを層で分ける

- **engine（`internal/`）= stdlib-only 厳守**。配置 / diff / 保守的 stale 除去という安全クリティカルなロジックを
  外部依存ゼロでユニットテストできる状態に保つ。
- **CLI 層 = 最小依存を許可**（cobra + fatih/color の 2 個まで）。人間向け UX（サブコマンド / help / 色）に限る。

### 2. CLI フレームワーク = cobra、dryrun 色付け = fatih/color

- **cobra**: Go エコシステムの事実上の標準。nixpkgs で well-packaged、サブコマンド / フラグ / シェル補完 / help 整形が成熟。
  ADR-0007 の「`--help` で内部実行する nix コマンドを開示する透明性」要件にも help 整形の質で寄与する。
- **fatih/color**: dryrun の `place` / `replace` / `remove` / `conflict` / `no-op` を色分け表示する（CLI 層の関心）。
  TTY 判定・`NO_COLOR` 尊重を備える。

### 3. engine は `internal/` の層分離に留める（公開モジュール化しない）

- engine は `internal/` に置き**外部 import 不可**とする。ADR-0006 / ADR-0007 の「ライブラリ」は
  **テスト可能性のためのバイナリ内層分離**を意味し、公開 API の semver 互換負債は背負わない。
- 安定面は `manifest.json` 入力契約（→ ADR-0006, ADR-0010）の 1 点に閉じる。

### 4. store path 取得 = out-link を profileDir 内に作る（GC 窓を塞ぐ）

- CLI は link-farm の store path を **`nix build <ep>#nput.<system>.<name> --out-link <profileDir>/.pending-<name>`**
  （legacy は `nix-build <ep> -A nput.<name> --out-link <profileDir>/.pending-<name>`）で得る。`os.Readlink` で store path を読む。
- これにより **取得と indirect gcroot 登録を一手**に行い、`nix build` 完了〜`nix-env --set` の間に並行
  `nix-collect-garbage` が link-farm を回収して**配置中の symlink が dangling 化する窓**を塞ぐ。
- `.pending-<name>` は**固定パス**にする。`--out-link` は既存パスを上書きするため、apply ごとに前回 pending root が
  置き換わり orphan は config あたり最大 1。`nix-env --set` 成功後に削除する（世代リンクが root を引き継ぐ）。
- **dryrun は link-farm を build するが配置しない**読み取り専用なので dangling 量産の窓がなく、pending gcroot を張らない。

### 5. nixpkgs ビルド = `buildGoModule` + vendorHash 文字列

- `buildGoModule { vendorHash = "sha256-..."; }` を pin し、依存変更時に `nix-update` 等で更新する nixpkgs イディオム。
- flake input を増やさず、cobra 程度の依存規模では hash 更新負荷も軽い。

### 6. `profileDir` ロック = `syscall.Flock`

- stdlib の `syscall.Flock(fd, LOCK_EX|LOCK_NB)`。advisory ロックでプロセス終了時に OS が自動解放するため
  クラッシュしても stale lock が残らない。linux / darwin 両対応。try-lock は `LOCK_NB` でそのまま表現する（→ ADR-0006）。

### 7. 再帰コピー / JSON / エラー処理（stdlib）

- **再帰コピー**: `filepath.WalkDir` + `io.Copy` + `os.Chmod` の手書き。spec の「mode 保存」要件のため
  `os.CopyFS`（Go 1.23+・perm 非保存・symlink 非対応）は使わない。
- **JSON**: `encoding/json`。`schemaVersion` は engine 対応版より新しい manifest を拒否する（→ ADR-0006）。
- **エラー**: stdlib `fmt.Errorf` + `%w` ラップ。`errors.Is` / `errors.As` で判別する。

### 8. Go toolchain = nixpkgs の go に pin、`toolchain` ディレクティブ不使用

- `buildGoModule` が使う nixpkgs の go を唯一の真実にし、`go.mod` は `go 1.xx`（ランゲージ表記）のみ。
  `toolchain` ディレクティブは置かず、`GOTOOLCHAIN=local` 相当でサンドボックス内の Go 自動ダウンロードを防ぐ。
- nixpkgs 更新で go も追従する。

## 根拠

- **層で依存を分ける**ことで「安全クリティカルなコアは stdlib-only でテストする」（ADR-0006）を厳密化しつつ、
  CLI の人間向けエルゴノミクス（cobra / 色）も得る。
- **out-link を profileDir 内に**置く案は、当初検討した「`--print-out-paths` で stdout パース」が GC 窓を塞げない
  欠点を解消する。配置ステップは link-farm を指す symlink を張るため、build〜`--set` 間の GC は単なる
  `--set` 失敗では済まず **dangling symlink を量産**しうる。登録済み gcroot を `--set` 前に張る価値は大きい。
- **bare symlink は GC root にならない**。`os.Symlink` で profileDir にリンクを置いても GC root にはならず、
  `nix build --out-link`（または `nix-store --add-root --indirect`）が作る indirect root のみが窓を塞ぐ。
- **`buildGoModule` + vendorHash** は flake input を増やさず nixpkgs 標準経路に乗る。gomod2nix も `gomod2nix.toml`
  再生成の手間が残り、flake input が増える点で「lib は nixpkgs のみ依存」の思想と緊張する。

## 影響

- **`docs/spec.md`**: 実行フローの「store path 取得」を out-link-in-profileDir に更新。依存関係表に cobra / fatih/color・
  vendorHash・stdlib 機構を反映。
- **`CONTEXT.md`**: `engine` 定義に「`internal/` 層分離（公開モジュールではない）」「out-link で GC 窓を塞ぐ」を追記。
- **`flake.nix`**: `packages.<system>.nput`（`buildGoModule` + vendorHash）の追加は実装フェーズ。
- ADR-0007 の「store path 取得」記述（`nix build … → store path`）を本 ADR が out-link 方式に具体化する。

## 棄却した代替案

- **CLI も含め全層 stdlib-only（stdlib flag + 手動 dispatch）**: 依存ゼロだが help / 補完を自作する負荷が高く、
  cobra の透明性 help の質を捨てる。CLI 層に限り最小依存を許す方が実務的。
- **kong / urfave-cli**: 軽量だが cobra のエコシステム成熟（nixpkgs パッケージ・補完）に劣る。
- **engine を公開 import 可能な Go モジュールにする（`pkg/`）**: semver 互換の継続負債が生じる。安定面は `manifest.json`
  契約で足り、再利用要求も現状ない。
- **store path を `--print-out-paths` で stdout パース**: result 汚染を避けられるが GC 窓（dangling symlink リスク）を塞げない。
  out-link を profileDir 内に置けば汚染も起きず GC 安全も同時に得られる。
- **stdout パース + `nix-store --add-root --indirect`**: 窓は塞げるが取得と root 化が 2 コマンドに分かれ、out-link 一手より冗長。
- **gomod2nix**: flake input が増え、`gomod2nix.toml` 再生成の手間も残る。細粒度キャッシュの利得は cobra 程度の依存では小さい。
- **`vendor/` を commit し vendorHash = null**: hash 更新は消えるが vendor の差分がリポジトリに載る。
- **`O_EXCL` lock ファイル**: クラッシュで stale lock が残り、PID 生存チェック等の回収処理を自前で背負う。
- **`os.CopyFS`**: perm 非保存・symlink 非対応で「mode 保存」要件を満たさない。
- **`go.mod` の `toolchain` を正にする**: nixpkgs の go と不一致時にサンドボックスで DL を試みてビルド失敗する。二重管理になる。
