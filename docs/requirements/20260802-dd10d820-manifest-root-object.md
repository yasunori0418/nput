---
id: "REQ-dd10d820-e453-4099-a47a-ffb9a7de02fb"
type: requirement
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
name: "manifest.json の root は rootKind を持ち fixed のときだけ絶対パスを併記する"
specification: |
  The `root` object of `manifest.json` SHALL carry `rootKind`, one of `"project"`,
  `"home"`, `"system"` or `"fixed"`. `project`, `home` and `system` are resolved by the
  engine at runtime (git toplevel / `$HOME` / `/`) and therefore SHALL NOT carry a path.
  Only `rootKind = "fixed"` SHALL additionally carry a `root` field holding the absolute
  path determined at evaluation time; for every other kind that field SHALL be omitted.
specification_ja: |
  `manifest.json` の `root` オブジェクトは `rootKind` を持ち、値は `"project"` /
  `"home"` / `"system"` / `"fixed"` のいずれかでなければならない。`project` / `home` /
  `system` は engine が実行時に解決する（git toplevel / `$HOME` / `/`）ためパスを
  持ってはならない。`rootKind = "fixed"` のときのみ、評価時に確定した絶対パスを
  `root` フィールドに併記し、それ以外の kind では当該フィールドを省略する。
---
# REQ-dd10d820: manifest.json の root は rootKind を持ち fixed のときだけ絶対パスを併記する

## 仕様

| フィールド | 型 | 説明 |
|---|---|---|
| `rootKind` | `"project"` \| `"home"` \| `"system"` \| `"fixed"` | root マーカーの種別。engine が実行時に解決 |
| `root` | string | `rootKind = "fixed"` のときのみ存在する絶対パス。それ以外は省略 |

上の表は原文の写しで、規範は frontmatter が正。

`project` / `home` / `system` は実行時解決（git toplevel / `$HOME` / `/`）のため
パスを持たない。`fixed` のみ評価時確定の絶対パスを `root` に持つ。

Nix 側の `root` 引数がどの値を取るか（3 マーカー + 絶対パス文字列の union）は
REQ-37b56673、marker が kind を運ぶ入れ物であることは REQ-3f541d39 が持つ。本 item は
それが `manifest.json` にどう写るかを規定する。

## 出典

`docs/spec.md`「manifest.json スキーマ（v1・Nix↔Go 契約）」→「`root`」。
