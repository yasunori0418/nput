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
- standalone もモジュールも、エンジンは nput 自身の profile を使って起動する（ADR-0002：全モード自前 profile）。
  standalone は profile をユーザー向け rollback に使い、モジュールは内部機構（前世代マニフェスト + stale 追跡）に留め、
  ユーザー向けロールバックは host に一本化する。
- これに伴い ADR-0001 で「各層が realize」とした out-of-store の実現も、HM ネイティブの
  `config.lib.file.mkOutOfStoreSymlink` へは委譲せず、nput エンジン自身の `ln -s` で行う。
- **削除の安全不変条件**: エンジンが所有する stale 除去は保守的に行う。前世代の store マニフェスト（ADR-0002）が
  「nput が配置した」と記録し、かつ現状もその記録通りを指す symlink のみ削除し、ユーザーの実ファイルや
  nput 非管理の link には触れない。配置・cleanup 機構は home-manager の `linkGeneration`/`cleanup` を参考に再実装する。

## nput は OS のファイル管理機構とは別の機構である

nput は「nix store の物を配置・管理する、一つのことをうまくやる」純粋関数（UNIX 思想）である。
`systemd.tmpfiles` / `environment.etc` などは **OS（NixOS）が自身の宣言的ファイル管理のために持つ別ツール**であり、
nput の関心事ではない。nput を NixOS 上で使う ＝ ただ nput を実行するだけで、OS の機構には一切翻訳・委譲しない
（stow や git を NixOS 上で動かすのに tmpfiles が関係ないのと同じ）。

- したがって `systemd.tmpfiles`（`L` 型）への翻訳は **採らない**。配置は全環境で nput エンジン（`ln` / `rsync`）+
  世代由来の store マニフェスト（ADR-0002）の一機構に統一する。
- nput は NixOS-宣言的であろうとしないため、「NixOS の宣言性を放棄する」という懸念はそもそも成立しない
  （放棄すべき宣言性を最初から負っていない）。
- 将来 NixOS モジュールを設けるとしても、それは activation hook から nput エンジンを起動する**ランチャー**であり、
  tmpfiles への翻訳ではない。

## 根拠

- 配置の振る舞いが単一のコアに集約され、テスト可能性が上がる（純粋関数 + 1 本のエンジン）。
- distro ビジョン（NixOS モジュール生態系から独立した配置機構）と整合する。
- プラットフォーム横断で同一スキーマ・同一挙動を保証できる。
- tmpfiles は「作成」しか宣言的に扱わず、規則消滅時の stale 除去をしない（NixOS 既知のギャップ）。
  どのみち削除は nput が担う必要があり、tmpfiles を併用すると機構が二重化するだけで利点が薄い。

## 影響

- ネイティブ統合の恩恵（プラットフォーム標準の追跡・GC、`home.file` の宣言性）は全層で捨てる。
  **stale 除去は全層で nput が所有する**が、世代由来の store マニフェスト（ADR-0002）で home-manager 同等の
  正確性を担保する。
- design.md の「各統合層の変換先」テーブルを「nput エンジン起動」に書き換える。
  `home.file` / `systemd.tmpfiles` は「明示的に採らない代替」として注記する。
- spec.md の「モジュール別動作仕様」を全層エンジン起動に書き換える。

## 棄却した代替案

- **ハイブリッド（standalone はエンジン、モジュールはネイティブ翻訳）**: ネイティブ統合の恩恵は得られるが、
  振る舞いが層ごとに二重化し、nput の「単一コア・ユーザー管理」方針と逆行する。
- **NixOS の system パス symlink のみ tmpfiles を使う部分ハイブリッド**: tmpfiles は作成のみで stale 除去をせず、
  結局 nput のマニフェスト削除が別途必要になり機構が二重化する。nput を「OS とは別の一機構」と割り切り、不採用。
