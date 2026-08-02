---
id: "REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9"
type: requirement
name: "全モジュールと devShell は engine をキックするだけの配線とし、ネイティブ機構へ翻訳しない"
specification: |
  Basic use SHALL be conceived around project mode and the standalone CLI, support for
  modules existing only to pick up the use case of wanting placement to run together with
  the switch of another module system. Every module (home-manager / NixOS / nix-darwin)
  and the devShell SHALL alike be wiring that does no more than kick the nput engine, each
  layer supplying only the root and the timing of activation. They MUST NOT translate
  entries into a native mechanism of the platform, such as `systemd.tmpfiles` or
  `home.file`. Placement and stale removal SHALL be performed, in every layer, by the same
  engine and the same store manifest, so that the behaviour is not duplicated per layer.
  The mechanisms of the operating system SHALL remain outside the concern of nput. The
  NixOS and nix-darwin modules SHALL kick the engine from `system.activationScripts.nput`;
  the hook the home-manager module uses is stated by REQ-8085f194.
specification_ja: |
  基本的な利用は project mode（プロジェクト内配置）と standalone CLI を中心に考えるものとし、
  モジュール対応は、他のモジュールシステムの switch と一括で動いてほしいユースケースを拾う
  ためだけに存在するものとする。全モジュール（home-manager / NixOS / nix-darwin）と devShell は
  一律、nput エンジンをキックするだけの配線でなければならず、各層は root と activation
  タイミングだけを供給する。`systemd.tmpfiles` / `home.file` などプラットフォームの
  ネイティブ機構へ翻訳してはならない。配置・stale 除去は全層で同一の engine + store
  マニフェストが行うものとし、振る舞いが層ごとに二重化しないようにする。OS の機構は nput の
  関心外とする。NixOS / nix-darwin モジュールは `system.activationScripts.nput` から engine を
  起動しなければならない（home-manager モジュールが使うフックは REQ-8085f194 の担当）。
---
# REQ-c1b3ca5f: 全モジュールと devShell は engine をキックするだけの配線とし、ネイティブ機構へ翻訳しない

## 仕様

基本的な利用は **project mode（プロジェクト内配置）と standalone CLI** を中心に考える。
モジュール対応は、他のモジュールシステムの switch と**一括で動いてほしいユースケース**を拾う
ためだけに存在する。全モジュール（HM / NixOS / nix-darwin）と devShell は
**一律「nput エンジンをキックするだけ」の配線**であり、各層は root と activation タイミングを
供給するだけ。`systemd.tmpfiles` / `home.file` へは翻訳しない。配置・stale 除去は全層で同一の
engine + store マニフェスト。

- NixOS モジュール（将来拡張）は `system.activationScripts.nput` から engine を起動する
  **配線**に徹する。`systemd.tmpfiles` へは翻訳しない。OS の機構（tmpfiles 等）は nput の
  関心外
- nix-darwin モジュール（将来拡張）も `system.activationScripts.nput` から engine を起動する

> **上は原文の写しで、規範は frontmatter が正**。各層が供給する root の具体（HM の `$HOME`・
> NixOS / nix-darwin の `config.users.users.${cfg.user}.home`）は REQ-8d965ca2、HM の
> activation フックと起動契約は REQ-8085f194、devShell の起動タイミングは REQ-a0bdf6db、
> module 経路の rollback を host 世代へ一本化することは REQ-844ee375、engine 自身が
> ネイティブ FS 操作で配置することは REQ-6c4e174a の担当。
>
> CLI が PATH 常駐の一次 UX であることと、その導入形（standalone のグローバル導入・
> project mode の devShell 同梱）は REQ-14f0aec9 の担当。本 item はモジュール対応の側から見た
> 位置づけ（一括 switch 用途を拾うためだけに存在する配線であること）を規定する。
>
> **原文の「将来拡張」の限定を規範に採らない理由**: NixOS / nix-darwin モジュールは
> ADR-0036 の時点でも「将来・別マイルストーン」のままで、ADR による反転は無い（system mode の
> engine / CLI 対応だけが先行実装された）。それでも限定を規範へ持ち込まないのは、
> **実装時期は「満たすべきこと」ではない**ため。本 item が規定するのは「これらのモジュールを
> 実装するときに満たすべき形」であり、いつ実装するかはマイルストーンの管理事項として
> `docs/spec.md` 側の見出しと出典に残す。同じ扱いは NixOS / nix-darwin の `user` オプション
> （REQ-c2654ca5）と root 供給元（REQ-8d965ca2）にも適用している。

## 出典

`docs/spec.md`「モジュール別動作仕様」の前文と、「NixOS モジュール（将来拡張）」
「nix-darwin モジュール（将来拡張）」節。将来拡張 2 節のうち root の供給元は REQ-8d965ca2、
rollback の一本化は REQ-844ee375 が担当し、残る「activation フックから engine を起動する
配線に徹する」を本 item に畳んだ。

決定の実体は ADR-0003「配置ロジックは全層 nput エンジンが所有し、モジュールは配線に徹する」で、
モジュール対応を一括 switch 用途に限る positioning は ADR-0007「汎用 nput CLI を一次 UX に
昇格し、entrypoint 発見＋root 明示モデルへ移行する」が定めている。
