---
id: "REQ-6419e4b0-629c-435f-b1e1-0bc996b5e61d"
type: requirement
name: "非 NixOS で動く主張を実 nix の一気通貫 E2E で検証する"
specification: |
  The claim that nput runs on a non-NixOS system given only nix SHALL be verified against a
  real nix by an E2E harness that drives the actual path end to end, including `nix build`,
  `nix eval` and `nix-env --set` from the flake entrypoint, rather than by stubbing nix out.
  Each scenario SHALL run under an isolated temporary `$HOME` and `$XDG_STATE_HOME`, so that
  it does not touch the environment of the machine that runs it, and its source SHALL be
  supplied either as a relative path inside a fixture flake, copied into the store at
  evaluation time, or as a live out-of-store directory. The harness SHALL cover, at minimum,
  placement and idempotency in project mode; placement, generation commit and rollback in
  home mode; stale removal; copy place-once and out-of-store symlinks; and evaluating and
  activating the home-manager module on a non-NixOS system. Each specification it verifies
  is stated by the item responsible for it and SHALL NOT be restated here.
specification_ja: |
  「非 NixOS でも nix さえあれば動く」という主張は、nix をスタブせず、flake entrypoint
  からの `nix build` / `nix eval` / `nix-env --set` を含む実経路を一気通貫で回す E2E
  ハーネスにより、実 nix で検証しなければならない。各シナリオは隔離した一時 `$HOME` /
  `$XDG_STATE_HOME` の下で動かし（実行マシンの環境に触れないようにするため）、配置元は
  fixture flake 内の相対パス（評価時に store へコピーされる）か out-of-store の live
  ディレクトリで用意する。ハーネスは最低限、project mode の配置と冪等性、home mode の
  配置・世代コミット・ロールバック、stale 除去、copy の place-once と out-of-store
  symlink、非 NixOS での home-manager モジュールの評価と activate を対象としなければ
  ならない。各シナリオが検証する仕様そのものは各担当 item の規範であり、ここでは
  再掲しない。
---
# REQ-6419e4b0: 非 NixOS で動く主張を実 nix の一気通貫 E2E で検証する

## 仕様

「非 NixOS でも nix さえあれば動く」主張を実 nix で検証する E2E ハーネス（`tests/e2e/`・
bash・詳細は `tests/e2e/README.md`）が、flake entrypoint からの `nix build` / `nix eval` /
`nix-env --set` を含む実経路を一気通貫で回す。各シナリオは隔離した一時 `$HOME` /
`$XDG_STATE_HOME` 下で動き、偽 src は fixture flake 内の相対パス（eval 時に store へ
コピー）か out-of-store の live ディレクトリで用意する。

| シナリオ | 検証する仕様 |
|---|---|
| project mode | 一時 git repo で `nput apply <name>` → git toplevel 配下に store symlink 配置・再 apply の冪等性 |
| home mode | 仮 `$HOME` で apply → `$HOME` 配下配置 + profile 世代コミット、entry 入替で世代を進め `nput rollback` で前世代の配置へ復帰 |
| stale 除去 | entry を config から削除 → 再 apply で旧 symlink が消える |
| copy place-once / out-of-store | copy が通常ファイル（書込可・mode に owner-write 付与）・place-once 冪等（ローカル編集を破棄しない）・out-of-store marker の live symlink |
| HM module | home-manager standalone configuration を非 NixOS で評価・activate し、activation が `nput apply --manifest` で engine を起動して配置する |

> **上は原文の写しで、規範は frontmatter が正**。表の右列が指す仕様そのものは、それぞれ
> REQ-9cb26ffd（project mode の root 解決）・REQ-1be4d678 / REQ-0e341430（世代とロール
> バック）・REQ-16aef46b（stale 除去）・REQ-d2277c7a / REQ-84e3c717 / REQ-a8a923ad（copy と
> out-of-store）・REQ-8085f194（HM activation 契約）の担当。`tests/e2e/` というパスと bash
> という実装手段は現況であり、規範文では実装手段を固定していない。

## 出典

`docs/spec.md`「E2E 検証範囲（非 NixOS）」節の導入文と表。

決定の実体は ADR-0012「CI とテスト実行」で、非 NixOS の実 nix 上で E2E を回す検証範囲を
定めている。
