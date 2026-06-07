# ADR-0003: 配置ロジックは全層 nput エンジンが所有し、モジュールは配線に徹する

- ステータス: 採用
- 日付: 2026-06-07
- 関連: ADR-0001, ADR-0002, ADR-0004, `docs/design.md`（レイヤー構成 / 各統合層の変換先）, `docs/spec.md`（モジュール別動作仕様）

## 背景

初期設計では、統合層が各プラットフォームのネイティブ機構へ「翻訳」する想定だった。

- home-manager: `home.file."<target>".source` / `config.lib.file.mkOutOfStoreSymlink`
- NixOS: `systemd.tmpfiles.rules`（`L` 型）
- nix-darwin: `system.activationScripts` + `ln`

しかしこの方式では配置の振る舞いが層ごとに二重化し、プラットフォームのネイティブ機構の挙動に依存する。
nput の本質は「テスト可能な純粋関数群で、ユーザーが配置を明示的に管理する」ことであり、
振る舞いの源は単一のコアであるべき、という方針と矛盾する。

## 決定

- **nput コア = 配置エンジン**が全層で配置を実行する。マニフェストを root に対して突き合わせ、
  symlink / place-once copy / out-of-store を **nput 自身の `ln` / `rsync`** で実現する。
  `home.file` / `systemd.tmpfiles` / `environment.etc` には委譲しない。
- **モジュール（HM / NixOS / nix-darwin）は薄い配線に徹する。** root（`$HOME` または
  `config.users.users.<user>.home`）と activation タイミング（`home.activation` /
  `system.activationScripts`）だけを供給し、nput エンジンを switch 時に起動する。
- standalone は profile モード（ADR-0002）でエンジンを起動する。モジュールは plain モード
  （独自 profile なし）でエンジンを起動し、ロールバックはホスト世代に委ねる。
- これに伴い ADR-0001 で「各層が realize」とした out-of-store の実現も、HM ネイティブの
  `config.lib.file.mkOutOfStoreSymlink` へは委譲せず、nput エンジン自身の `ln -s` で行う。
- **削除の安全不変条件**: エンジンが所有する stale 除去は保守的に行う。前世代の store マニフェスト（ADR-0002）が
  「nput が配置した」と記録し、かつ現状もその記録通りを指す symlink のみ削除し、ユーザーの実ファイルや
  nput 非管理の link には触れない。配置・cleanup 機構は home-manager の `linkGeneration`/`cleanup` を参考に再実装する。

## NixOS のシステムパスは tmpfiles を使う（部分ハイブリッド）

- **scoped 例外**: 将来拡張の NixOS 層で root=`/` の**システムパスへの symlink** を置く場合は、
  nput エンジンの命令的 `ln` ではなく `systemd.tmpfiles.rules`（`L`/`L+` 型）を使う。
  NixOS の宣言性・標準の追跡に乗せ、命令的 activation でシステムパスを直接変更する緊張を避けるため。
- 適用範囲は **NixOS の system パス symlink に限る**。standalone / home-manager および $HOME レベルの配置、
  copy（place-once）、out-of-store は引き続き nput エンジンが所有する（本 ADR の原則は維持）。
- **留意（将来の NixOS 作業で解決すべき open 事項）**:
  - `systemd.tmpfiles` の `L` 型は規則が消えても作成済み symlink を自動削除しない（NixOS 既知のギャップ）。
    NixOS でも stale 除去の正確性には別途手当てが要る。
  - copy をシステムパスで行う場合（`environment.etc` 相当か、エンジンか）、`/etc` の処理順・権限・所有者は未決。
  - NixOS のシステム世代との整合（rollback 時の再現）の詳細も未決。

## 根拠

- 配置の振る舞いが単一のコアに集約され、テスト可能性が上がる（純粋関数 + 1 本のエンジン）。
- distro ビジョン（NixOS モジュール生態系から独立した配置機構）と整合する。
- プラットフォーム横断で同一スキーマ・同一挙動を保証できる。

## 影響

- ネイティブ統合の恩恵（プラットフォーム標準の追跡・GC、`home.file` の宣言性）は標準/HM/$HOME 配置では捨てる。
  **stale 除去は全層で nput が所有する**が、世代由来の store マニフェスト（ADR-0002）で home-manager 同等の
  正確性を担保する。
- NixOS のシステムパス symlink のみ `systemd.tmpfiles` を使う（上記の部分ハイブリッド）。
- design.md の「各統合層の変換先」テーブルを「nput エンジン起動」に書き換える（NixOS system パスは tmpfiles と注記）。
  `home.file` は「明示的に採らない代替」として注記する。
- spec.md の「モジュール別動作仕様」を全層エンジン起動に書き換える（NixOS system パスは tmpfiles 例外）。

## 棄却した代替案

- **完全ハイブリッド（standalone はエンジン、全モジュールはネイティブ翻訳）**: ネイティブ統合の恩恵は得られるが、
  振る舞いが層ごとに二重化し、nput の「単一コア・ユーザー管理」方針と逆行する。
  NixOS の system パス symlink のみ tmpfiles を使う部分例外に留める。
