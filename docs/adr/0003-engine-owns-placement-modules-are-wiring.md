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

## 根拠

- 配置の振る舞いが単一のコアに集約され、テスト可能性が上がる（純粋関数 + 1 本のエンジン）。
- distro ビジョン（NixOS モジュール生態系から独立した配置機構）と整合する。
- プラットフォーム横断で同一スキーマ・同一挙動を保証できる。

## 影響

- ネイティブ統合の恩恵（プラットフォーム標準の追跡・GC、`systemd.tmpfiles` の宣言性）は捨てる。
  **stale 除去まで全層で nput が所有する**（ADR-0002 の state マニフェスト）。
- design.md の「各統合層の変換先」テーブルを「nput エンジン起動」に全面書き換える。
  `systemd.tmpfiles` / `home.file` は「明示的に採らない代替」として注記する。
- spec.md の「モジュール別動作仕様」を全層エンジン起動に書き換える。

## 棄却した代替案

- **ハイブリッド（standalone はエンジン、モジュールはネイティブ翻訳）**: ネイティブ統合の恩恵は得られるが、
  振る舞いが層ごとに二重化し、nput の「単一コア・ユーザー管理」方針と逆行する。
