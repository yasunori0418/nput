---
id: "REQ-dedd2c28-bba3-4ecf-80c9-8c77347e8e1f"
type: requirement
name: "manifest.json のトップレベルは schemaVersion / root / entries の 3 フィールドとする"
specification: |
  The top level of `manifest.json` SHALL consist of exactly three fields: `schemaVersion`
  (int, the contract version), `root` (object, the kind of the placement base), and
  `entries` (array of object, the placement definitions).
specification_ja: |
  `manifest.json` のトップレベルは `schemaVersion`（int・契約バージョン）・
  `root`（object・配置先基準の kind）・`entries`（array of object・配置定義）の
  3 フィールドで構成しなければならない。
---
# REQ-dedd2c28: manifest.json のトップレベルは schemaVersion / root / entries の 3 フィールドとする

## 仕様

| フィールド | 型 | 説明 |
|---|---|---|
| `schemaVersion` | int | 契約バージョン。v1 は `1` |
| `root` | object | 配置先基準の kind |
| `entries` | array of object | 配置定義 |

上の表は原文の写しで、規範は frontmatter が正。`schemaVersion` の値と engine の拒否
挙動は REQ-79ce0a09、`root` オブジェクトの中身は REQ-dd10d820、`entries[]` の中身は
REQ-0b0cd1e3 が持つ。

```json
{
  "schemaVersion": 1,
  "root": { "rootKind": "project" },
  "entries": [
    {
      "srcKind": "store",
      "src": "/nix/store/abcd1234...-source",
      "subpath": "skills/nix",
      "target": ".claude/skills/nix",
      "method": "symlink"
    },
    {
      "srcKind": "outOfStore",
      "src": "/home/me/dotfiles",
      "subpath": "home/.config/nvim",
      "target": ".config/nvim",
      "method": "symlink"
    }
  ]
}
```

## 出典

`docs/spec.md`「manifest.json スキーマ（v1・Nix↔Go 契約）」→「トップレベル」・「例」。
