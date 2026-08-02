---
id: "REQ-a480c183-40ce-4201-93b5-65a7a59c1b9e"
type: requirement
derives_from:
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
name: "gitignore は配置 target を stdout へ列挙するだけでファイルを書き込まない"
specification: |
  `nput gitignore <name>` SHALL only enumerate the placed targets for use in `.gitignore`
  and write them to stdout; it SHALL NOT write the file, the responsibility for updating
  it remaining with the project administrator. The output SHALL be in the anchored form of
  a root-relative target prefixed with `/` (for example `/.claude/skills/nix`), one entry
  per line. Because in project mode root = git toplevel = the location of `.gitignore`,
  the leading `/` anchors correctly and SHALL NOT mistakenly ignore an identically named
  path at another level. Neither directories nor files SHALL carry a trailing `/`.
specification_ja: |
  `nput gitignore <name>` は配置 target を `.gitignore` 向けに列挙して stdout へ出力する
  だけでなければならず、ファイルを書き込んではならない（更新責務はプロジェクト管理者）。
  出力は root 相対 target に先頭 `/` を付けたアンカー形式（例: `/.claude/skills/nix`）で
  1 行 1 件とする。project mode の root = git toplevel = `.gitignore` の置き場所なので
  先頭 `/` が正しくアンカーし、別階層の同名パスを誤って無視してはならない。
  ディレクトリ / ファイルとも末尾 `/` は付けない。
---
# REQ-a480c183: gitignore は配置 target を stdout へ列挙するだけでファイルを書き込まない

## 仕様

```bash
nput gitignore <name>          # 配置 target を .gitignore 向けに stdout 出力（書き込みなし）
```

`gitignore <name>` は配置 target を `.gitignore` 向けに列挙して stdout に出力するだけで、
ファイルは書き込まない。更新責務はプロジェクト管理者。出力は **root 相対 target に
先頭 `/` を付けたアンカー形式**（例: `/.claude/skills/nix`）で 1 行 1 件。project mode の
root = git toplevel = `.gitignore` 置き場所なので先頭 `/` が正しくアンカーし、別階層の
同名パスを誤って無視しない。ディレクトリ / ファイルとも末尾 `/` は付けない。

project mode 限定であることは REQ-eaa8c0df、`method` を区別しない点は REQ-60787ed2、
`--all` の集約は REQ-1f128917 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `gitignore <name>` の箇条書き。

決定の実体は ADR-0005（更新責務はプロジェクト管理者）と ADR-0013（アンカー形式・
末尾 `/` を付けない）。
