---
id: "REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9"
type: requirement
name: "全モジュールと devShell は engine をキックするだけの配線とし、ネイティブ機構へ翻訳しない"
specification: |
  Every module (home-manager / NixOS / nix-darwin) and the devShell SHALL alike be wiring
  that does no more than kick the nput engine, each layer supplying only the root and the
  timing of activation. They MUST NOT translate entries into a native mechanism of the
  platform, such as `systemd.tmpfiles` or `home.file`. Placement and stale removal SHALL
  be performed, in every layer, by the same engine and the same store manifest, so that
  the behaviour is not duplicated per layer. The mechanisms of the operating system SHALL
  remain outside the concern of nput.
specification_ja: |
  全モジュール（home-manager / NixOS / nix-darwin）と devShell は一律、nput エンジンを
  キックするだけの配線でなければならず、各層は root と activation タイミングだけを供給する。
  `systemd.tmpfiles` / `home.file` などプラットフォームのネイティブ機構へ翻訳しては
  ならない。配置・stale 除去は全層で同一の engine + store マニフェストが行うものとし、
  振る舞いを層ごとに二重化してはならない。OS の機構は nput の関心外とする。
---
# REQ-c1b3ca5f: 全モジュールと devShell は engine をキックするだけの配線とし、ネイティブ機構へ翻訳しない

## 仕様

全モジュール（HM / NixOS / nix-darwin）と devShell は **一律「nput エンジンをキックするだけ」の
配線**であり、各層は root と activation タイミングを供給するだけ。`systemd.tmpfiles` /
`home.file` へは翻訳しない。配置・stale 除去は全層で同一の engine + store マニフェスト。

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
> 原文が「基本的な利用は project mode と standalone CLI を中心に考え、モジュール対応は他の
> モジュールシステムの switch と一括で動いてほしいユースケースを拾うためだけに存在する」と
> 述べる positioning は、CLI を一次 UX とする REQ-14f0aec9 の担当。

## 出典

`docs/spec.md`「モジュール別動作仕様」の前文と、「NixOS モジュール（将来拡張）」
「nix-darwin モジュール（将来拡張）」節。将来拡張 2 節のうち root の供給元は REQ-8d965ca2、
rollback の一本化は REQ-844ee375 が担当し、残る「activation フックから engine を起動する
配線に徹する」を本 item に畳んだ。

決定の実体は ADR-0003「配置ロジックは全層 nput エンジンが所有し、モジュールは配線に徹する」。
