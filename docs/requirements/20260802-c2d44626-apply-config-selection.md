---
id: "REQ-c2d44626-d8f4-446a-a80a-319a500129b4"
type: requirement
name: "apply の config 選択は name 省略で default・明示で単一・--all で全件"
specification: |
  When the name is omitted, `nput apply` SHALL apply `nput.default`, following the flake
  `default` convention, and SHALL fail with an error when `default` is not defined.
  Specifying `<name>` explicitly SHALL apply that config, and `--all` SHALL apply all of
  `nput.*`. That a profile is atomic per config is stated by the generation management
  specification and is not restated here.
specification_ja: |
  `nput apply` は name 省略時に `nput.default` を適用しなければならない（flake の
  `default` 慣例に倣う）。`default` が未定義ならエラーとする。`<name>` を明示すれば
  その config を、`--all` で `nput.*` 全てを適用する。profile が config 単位で atomic で
  あることは世代管理仕様の担当で、本 item では規定しない。
---
# REQ-c2d44626: apply の config 選択は name 省略で default・明示で単一・--all で全件

## 仕様

```bash
nput apply                     # name 省略時は nput.default を適用（flake の default 慣例。無ければエラー）
nput apply <name>              # nput.<name> をビルドし新世代を作って適用
```

`apply` の **name 省略時は `nput.default` を適用**する（flake の `default` 慣例に倣う。
`default` が未定義ならエラー）。`<name>` を明示すればその config を、`--all` で
`nput.*` 全てを適用する。profile は config 単位で atomic。

> **上は原文の写しで、規範は frontmatter が正**。`--all` の適用順・部分失敗時の挙動・
> root モードフィルタは本 item の規範ではなく、REQ-4cbd9a0d / REQ-d95b814f の担当。
> profile の atomic 性そのもの（→ ADR-0002）は REQ-1be4d678 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」のコードブロックと、直後の箇条書き
第 1 項。

決定の実体は ADR-0007 §4「config 選択」（name 省略時は `nput.default`・未定義ならエラー・
一括適用は `--all`）。
