---
id: "REQ-9cb26ffd-071e-4c68-a6fc-faac6373b75e"
type: requirement
name: "project mode の root は git toplevel から解決し、config 相対も CWD 相対も採らない"
derives_from:
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  In project mode (`root = projectRoot`) the root SHALL be resolved by default from
  `git rev-parse --show-toplevel`, so that running from any subdirectory resolves to the
  same root; `--root <path>` SHALL state it explicitly for use outside git or to point at
  another root. Resolution relative to the config file SHALL NOT be adopted, since a flake
  source is copied into the store by Nix and becomes a store path, so that it does not
  hold. Resolution relative to the current working directory (`$PWD`) SHALL NOT be adopted
  either, since the placement destination would shift with the place of execution and
  break idempotency.
specification_ja: |
  project mode（`root = projectRoot`）の root は、既定で
  `git rev-parse --show-toplevel` から解決しなければならない（どのサブディレクトリから
  実行しても同じ root に解決される）。git 外で使う場合や別ルートを指す場合は
  `--root <path>` で明示しなければならない。config ファイル相対は採ってはならない
  （Nix で flake source が store にコピーされ store path 化するため成立しないため）。
  CWD（`$PWD`）相対も採っては
  ならない（実行場所で配置先がズレ冪等性を壊すため）。
---
# REQ-9cb26ffd-071e-4c68-a6fc-faac6373b75e: project mode の root は git toplevel から解決し、config 相対も CWD 相対も採らない

## 仕様

| 解決方法 | 備考 |
|---|---|
| `git rev-parse --show-toplevel`（既定）| どのサブディレクトリから実行しても同じ root に解決される |
| `--root <path>`（上書き）| git 外で使う場合や別ルートを指す場合に明示 |

- config ファイル相対は採らない（Nix で flake source が store にコピーされ store path 化する
  ため成立しない）
- CWD（`$PWD`）相対は採らない（実行場所で配置先がズレ冪等性を壊すため）

> **上は原文の写しで、規範は frontmatter が正**。`--root` が project mode に限らず全モードの
> 解決 root を一律上書きすることは REQ-61c05e09-0bde-4f74-9a96-03185f9df606、明示時の profileDir キーイングは REQ-d5a2e289-40bc-45a9-9d44-21b8dc561b81、
> `projectRoot` が marker であることは REQ-3f541d39-da41-4ef8-858b-707f54cf6a29 / REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66、engine が外部コマンドとして
> `git` を叩くことは REQ-6c4e174a-4d16-477a-96ff-17cb4eb5b564、project mode の配置物が ephemeral であることは REQ-e79178f5-5865-4444-a05d-3ab06f33cd6d、
> 世代の扱い（非公開・世代スキップ）は REQ-46fccb80-4bae-4d37-bc19-dded88e9a9c0 の担当。

## 出典

`docs/spec.md`「root の解決」→「project mode（`root = projectRoot`）」節の表と、
同節の箇条書き第 1〜2 項。

決定の実体は ADR-0005「project mode（プロジェクト相対配置）と ephemeral 配置原則」
（git toplevel 解決・config 相対と CWD 相対の棄却）。
