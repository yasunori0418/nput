---
id: "REQ-b12fc3c0-d7fe-4003-922c-f3ac0d969b66"
type: requirement
name: "symlink farm は GC アンカー専用でアンカーは store-backed な symlink entry に限る"
specification: |
  The symlink farm contained in the derivation returned by `mkManifest` SHALL exist
  solely as a GC anchor; the values the engine uses for placement SHALL be the resolved
  `src` strings in `manifest.json`. A farm anchor SHALL be held only by an entry that is
  both store-backed (`srcKind = "store"`) and `method = "symlink"`, so that the profile
  generation holds every store `src` as a GC root. An out-of-store entry
  (`srcKind = "outOfStore"`) points
  outside the store and SHALL NOT hold an anchor. A `method = "copy"` entry SHALL NOT
  hold an anchor either, even when its `src` is store-backed, because copy is place-once
  and independent of the store once materialized, so its store `src` may be freed by
  `nix-collect-garbage`; it SHALL still be recorded in `manifest.json` for orphan warnings
  and stale detection.
specification_ja: |
  `mkManifest` が返す derivation が含む symlink farm は GC アンカー専用でなければ
  ならず、engine が配置に使う値は `manifest.json` の解決済み `src` 文字列とする。
  farm アンカーを持つのは store 由来（`srcKind = "store"`）かつ `method = "symlink"` の
  entry に限り、profile 世代が GC root として全 store src を掴む。out-of-store entry
  （`srcKind = "outOfStore"`）は store 外を指すためアンカーを持ってはならない。
  `method = "copy"` の entry は store 由来であってもアンカーを持ってはならない。copy は
  place-once でマテリアライズ後は store から独立するため store src を掴む必要がなく、
  `nix-collect-garbage` で解放されてよいためである。ただし copy entry も orphan 警告・
  stale 判定のため `manifest.json` には記録する。
---
# REQ-b12fc3c0: symlink farm は GC アンカー専用でアンカーは store-backed な symlink entry に限る

## 仕様

derivation は `manifest.json` と symlink farm を含む。farm は **GC アンカー専用**で、
engine が配置に使う値は `manifest.json` の解決済み `src` 文字列。

- store-backed entry（`srcKind = "store"`）**かつ `method = "symlink"`** は farm に
  store パスへの symlink アンカーを持ち、profile 世代が GC root として全 store src を
  掴む。
- out-of-store entry（`srcKind = "outOfStore"`）は store 外を指すため farm アンカーを
  持たない。
- **`method = "copy"` entry は farm アンカーを持たない**（store src でも）。copy は
  place-once でマテリアライズ後は store から独立（世代外）なので store src を掴む必要が
  なく、`nix-collect-garbage` で解放されてよい。`manifest.json` には記録する
  （orphan 警告・stale 判定のため）。

アンカー名の決め方は REQ-62eda895 が持つ。derivation が manifest.json と farm を含む
こと自体は REQ-60e6b49c が持つ。

## 出典

`docs/spec.md`「manifest.json スキーマ（v1・Nix↔Go 契約）」→「symlink farm との対応」。
