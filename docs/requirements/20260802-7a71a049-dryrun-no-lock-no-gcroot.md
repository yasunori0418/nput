---
id: "REQ-7a71a049-5876-4cfc-a65e-44e9a0349856"
type: requirement
name: "--dryrun は root を解決するが flock も pending gcroot も取らない"
specification: |
  `--dryrun` SHALL evaluate the root kind and resolve the root, in order to display the
  plan, but because it is read-only and does not place anything even if it builds the
  link-farm, it SHALL take neither the flock nor the pending gcroot (the out-link).
specification_ja: |
  `--dryrun` はプラン表示のため root kind を eval し root を解決しなければならないが、
  link-farm を build しても配置しない読み取り専用であるため、flock も pending gcroot
  （out-link）も取ってはならない。
---
# REQ-7a71a049: --dryrun は root を解決するが flock も pending gcroot も取らない

## 仕様

`--dryrun` は root kind を eval し root を解決するが（プラン表示のため）、link-farm を
build しても配置しない読み取り専用なので、flock も pending gcroot（out-link）も取らない。

`apply --dryrun` が読み取り専用であること自体は REQ-02a33511、通常 apply が
out-link で gcroot を張ることは REQ-60c6b7ea の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「実行フロー」の `--dryrun` の箇条書き。

決定の実体は ADR-0011（out-link による pending gcroot の位置づけ）と ADR-0023
（実行フロー順序における flock の位置）。
