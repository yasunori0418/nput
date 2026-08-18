---
id: "REQ-d2277c7a-7992-49af-a9dc-4cc73843a6f9"
type: requirement
name: "copy は target 不在のときだけマテリアライズする place-once で世代管理の対象外とする"
derives_from:
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
specification: |
  A copy entry SHALL be materialized only while the target is absent. When the subpath
  denotes a directory and the target is absent, `<root>/<target>` SHALL be created and the
  contents of `<src>/<subpath>/` copied into it recursively through native filesystem
  operations; when the subpath denotes a file and the target is absent, the parent
  directory SHALL be created and `<src>/<subpath>` copied to `<root>/<target>`. Where the
  target already exists, nothing SHALL be done, the object being left to the user's own
  management; this is place-once, meaning that after the first materialization a present
  target SHALL NOT be touched. Reflecting an update of the store SHALL therefore be done
  either by `apply --recopy` or by removing the object with `reset` and applying again. A
  structural mismatch — a regular file existing at the target while the subpath is a
  directory, or the target being a directory while the subpath is a file — SHALL stop with
  an error. Copies SHALL be outside generation management and SHALL NOT be rolled back.
specification_ja: |
  copy entry は target が不在のときにだけマテリアライズしなければならない。subpath が
  ディレクトリで target が不在のときは `<root>/<target>` を作成して `<src>/<subpath>/` の内容を
  ネイティブ FS 操作で再帰コピーしなければならず、subpath がファイルで target が不在のときは
  親ディレクトリを作成して `<src>/<subpath>` を `<root>/<target>` へコピーしなければならない。
  target が既に存在するときは何もせずユーザー管理に委ねなければならない（place-once。初回
  マテリアライズ後、target が在れば触ってはならない）。したがってストア更新の反映は
  `apply --recopy`、または `reset` で撤去後に再 apply で行わなければならない。
  構造不一致（subpath がディレクトリのとき target に通常ファイルが存在する / subpath がファイルの
  とき target がディレクトリである）はエラーで停止しなければならない。copy は世代管理の対象外と
  し、ロールバックしてはならない。
---
# REQ-d2277c7a: copy は target 不在のときだけマテリアライズする place-once で世代管理の対象外とする

## 仕様

```
subpath がディレクトリの場合:
  target が不在のとき: <root>/<target> を作成しネイティブ再帰コピー（<src>/<subpath>/ → <root>/<target>/）
  target が存在するとき: 何もしない（ユーザー管理に委ねる）

subpath がファイルの場合:
  target が不在のとき: 親ディレクトリを作成しネイティブコピー（<src>/<subpath> → <root>/<target>）
  target が存在するとき: 何もしない
```

- **place-once**: 初回マテリアライズ後、target が在れば触らない。ストア更新の反映は
  **`apply --recopy`**、または `reset <name> [target]` で撤去後に再 apply で行う
- `subpath` がディレクトリのとき `target` に通常ファイルが存在する場合、または `subpath` が
  ファイルのとき `target` がディレクトリの場合は構造不一致でエラーで停止する
- 世代管理の対象外。ロールバックされない

> **上は原文の写しで、規範は frontmatter が正**。原文が併記する次の規範は本 item の
> 担当ではない。
>
> - コピー時の mode 保存 + owner-write 付与 → REQ-84e3c717
> - src ツリー内 symlink の複製 → REQ-0bd55dfc
> - foreign 実ファイルの skip を warning で可視化すること → REQ-07c3b735
> - `--backup` 有効時に構造不一致が退避 + 配置へ変わること → REQ-5dd5a4e9 / REQ-9b0046e0
> - `apply --recopy` の契約 → REQ-7cc32a2b、`reset` の契約 → REQ-31f2882e
> - method の値が copy であることの意味と世代管理の有無 → REQ-77689c68

## 出典

`docs/spec.md`「配置動作仕様」→「copy モード（place-once・ユーザー管理）」節のコードブロックと、
箇条書き第 2・3・6 項。

決定の実体は ADR-0020「copy の明示上書き（`apply --recopy`）と配置物のリセット
（`nput reset`）を追加する」で、place-once そのものは ADR-0002 が定めている。
