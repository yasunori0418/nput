---
id: "REQ-0b0cd1e3-bfeb-45c1-978d-e2e11c568336"
type: requirement
name: "manifest.json の entries は attrset を配列へ正規化し 5 フィールドを記録する"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The `entries` of `manifest.json` SHALL be the Nix attrset normalized into an array, so
  that the engine reads an array. Each element SHALL record five fields: `srcKind`
  (`"store"` or `"outOfStore"`), `src` (string; the resolved store path for `store`, the
  marker's absolute path for `outOfStore`), `subpath` (string, recorded after defaulting,
  so an omitted one is recorded as `"."`), `target` (string, the placement target
  relative to root and the entry's identity), and `method` (`"symlink"` or `"copy"`,
  recorded after defaulting). An entry `name` field SHALL NOT exist.
specification_ja: |
  `manifest.json` の `entries` は Nix の attrset を配列へ正規化したものでなければ
  ならない（Go は配列を読む）。各要素は次の 5 フィールドを記録しなければならない: `srcKind`
  （`"store"` / `"outOfStore"`）・`src`（string。`store` は解決済み store パス、
  `outOfStore` は marker の絶対パス）・`subpath`（string。デフォルト適用後を記録し、
  省略形も `"."` で記録する）・`target`（string。root 相対の配置先で entry の
  identity）・`method`（`"symlink"` / `"copy"`。デフォルト適用後）。entry の `name`
  フィールドを持ってはならない。
---
# REQ-0b0cd1e3: manifest.json の entries は attrset を配列へ正規化し 5 フィールドを記録する

## 仕様

attrset を**配列に正規化**して記録する（Go は配列を読む）。identity は `target`。

| フィールド | 型 | 説明 |
|---|---|---|
| `srcKind` | `"store"` \| `"outOfStore"` | 配置元の種別 |
| `src` | string | `store`: 解決済み store パス文字列 / `outOfStore`: marker の絶対パス |
| `subpath` | string | src 内の相対パス（デフォルト適用後。省略形も `"."` で記録）|
| `target` | string | root 相対の配置先。**entry の identity**（属性キー由来・stale 除去の diff キー）|
| `method` | `"symlink"` \| `"copy"` | 配置種別（デフォルト適用後。旧名 `mode`）|

上の表は原文の写しで、規範は frontmatter が正。

entry の `name` フィールドは持たない（識別子が属性キー = target であることは
REQ-cb77ea05）。

## 出典

`docs/spec.md`「manifest.json スキーマ（v1・Nix↔Go 契約）」→「`entries[]`」。
