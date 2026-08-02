---
id: "REQ-62eda895-efd4-4eaf-a58b-600e8637da75"
type: requirement
derives_from:
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
name: "symlink farm の GC アンカー名は target のハッシュとする"
specification: |
  The GC anchor name in the symlink farm SHALL be a hash of `target` (a shortened hex of
  SHA-256; fixed length and filesystem safe). Sanitizing `target` (for
  example by stripping `/`) MUST NOT be used, because two distinct targets could collapse
  into the same name and violate the key uniqueness constraint of `linkFarm`. The anchor
  name is not required to be human readable, since the farm exists solely as a GC anchor
  and the values the engine uses for placement are the resolved `src` strings in
  `manifest.json`.
specification_ja: |
  symlink farm の GC アンカー名は `target` のハッシュ（sha256 の短縮 hex・固定長・
  FS 安全）でなければならない。`target` をサニタイズ（`/` 除去等）する方式を採っては
  ならない。別 target が同名に潰れ `linkFarm` のキー一意制約に反するためである。farm は
  GC アンカー専用でアンカー名が可読である必要はなく、engine が配置に使う値は
  `manifest.json` の解決済み `src` 文字列である。
---
# REQ-62eda895: symlink farm の GC アンカー名は target のハッシュとする

## 仕様

symlink farm の GC アンカー名は **`target` のハッシュ（sha256 の短縮 hex・固定長・
FS 安全）**を用いる。

- `target` をサニタイズ（`/` 除去等）すると別 target が同名に潰れ linkFarm のキー
  一意制約に反するため採らない。
- farm は GC アンカー専用でアンカー名は可読である必要がなく、衝突不可能なハッシュで
  十分。engine が配置に使う値は `manifest.json` の解決済み `src` 文字列。

farm が GC アンカー専用であることとアンカー対象の範囲は REQ-b12fc3c0 が持つ。

## 出典

`docs/spec.md`「manifest.json スキーマ（v1・Nix↔Go 契約）」→「`entries[]`」末尾の
blockquote 注記。
