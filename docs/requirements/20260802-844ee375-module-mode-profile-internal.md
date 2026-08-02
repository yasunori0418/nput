---
id: "REQ-844ee375-919f-4341-81e1-a5f89fd32840"
type: requirement
derives_from:
  - "UC-d39c1994-f9a5-4860-80ba-f6e584adaf14"
name: "module 時は rollback を host へ一本化し、nput profile は前進のみで追従する"
specification: |
  Under a module, `nput rollback` SHALL NOT be exposed and user-facing rollback SHALL be
  unified on the host (`home-manager --rollback` and its equivalents). A rollback on the
  host SHALL be followed automatically by re-activating the old config and kicking nput
  again, which converges the filesystem; the nput profile SHALL therefore only ever move
  forward, stacking a new generation holding the old content. What the host is required to
  provide to nput SHALL be no more than the kick on switch, and no plumbing of the host's
  `oldGenPath` SHALL be required.
specification_ja: |
  module では `nput rollback` を公開してはならず、ユーザー向けロールバックは host
  （`home-manager --rollback` 等）に一本化しなければならない。host rollback は旧 config を
  再 activate して nput を再 kick することで FS を自動収束させる。したがって nput profile は
  前進のみ（旧内容の新世代を積む）とする。host から nput へ要求するのは switch 時の kick だけ
  とし、ホストの `oldGenPath` 配管を要求してはならない。
---
# REQ-844ee375: module 時は rollback を host へ一本化し、nput profile は前進のみで追従する

## 仕様

nput は自前 profile を**持つ**が、module ではそれを前世代マニフェスト + stale 追跡のための
内部機構に留める。ユーザー向けロールバックは host に一本化し、host rollback は旧 config を
再 activate して nput を再 kick することで FS を自動収束させる（nput profile は前進のみ＝
旧内容の新世代を積む）。host から nput へ要求するのは「switch 時の kick」だけで、ホストの
oldGenPath 配管は不要。

> **上は原文の写しで、規範は frontmatter が正**。全モードで自前 profile を持つこと・module では
> それを内部機構に留めること・前世代を nput 自身の profile から読みホストの oldGenPath に
> 依存しないことは REQ-1be4d678 の担当で、本 item はユーザー向け rollback の一本化先と、
> それが成立する追従の仕組み（前進のみ）を規定する。home mode 側の rollback 手順は
> REQ-0e341430、`rollback` が home mode 限定であることは REQ-05abce3e の担当。

## 出典

`docs/spec.md`「世代管理仕様」→「モジュール時」節と、「ロールバック」節の module の箇条書き。
「モジュール別動作仕様」→「home-manager モジュール」「NixOS モジュール（将来拡張）」
「nix-darwin モジュール（将来拡張）」各節が述べる同趣旨（nput は自前 profile を内部機構として
持ち、ユーザー向け rollback は host 世代へ一本化する）も本 item の担当で、独立 item を立てず
本 item に畳んだ。

決定の実体は ADR-0002「世代管理を nix profile に乗せる（全モード自前 profile / rollback は
standalone 中心）」。
