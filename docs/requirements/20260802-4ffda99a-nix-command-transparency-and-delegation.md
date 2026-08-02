---
id: "REQ-4ffda99a-7062-4c00-915f-70b525cb215b"
type: requirement
derives_from:
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
name: "内部実行する nix コマンドを開示し世代の切替と GC は標準の nix コマンドへ委譲する"
specification: |
  The CLI SHALL disclose the nix commands it runs internally, for instance through
  `nput --help`, so that the user can run them selectively by hand. Switching to an
  arbitrary generation and garbage-collecting generations SHALL be done with the standard
  `nix-env` / `nix-collect-garbage` against the profile path, rather than with dedicated
  nput subcommands.
specification_ja: |
  CLI は `nput --help` 等で内部実行する nix コマンドを開示し、ユーザーが選択的に手で
  実行できるようにしなければならない。任意世代への切替・世代の GC は nput 専用の
  サブコマンドではなく、標準の `nix-env` / `nix-collect-garbage` を profile パスに対して
  使うものとする。
---
# REQ-4ffda99a: 内部実行する nix コマンドを開示し世代の切替と GC は標準の nix コマンドへ委譲する

## 仕様

- 透明性: `nput --help` 等で内部実行する nix コマンドを開示し、ユーザーが選択的に手で
  実行できる。
- 任意世代への切替・世代の GC は標準の `nix-env` / `nix-collect-garbage` を profile パスに
  対して使う。

`--debug` による nix コマンドの開示（冗長度と直交させる分離）は REQ-0a123b89 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の透明性・任意世代切替の箇条書き 2 項。

決定の実体は ADR-0007 §3「透明性」（`nput --help` 等で内部実行する nix コマンドを開示し、
ユーザーが選択的に手で実行できるようにする）。
