---
id: "REQ-1dcc9a33-b0f2-43e0-8310-fc4b19e68fe7"
type: requirement
name: "marker は判別タグで識別し manifest.json には漏らさない"
specification: |
  Markers (`mkOutOfStoreSymlink` and the root markers) SHALL carry a `_nputMarker`
  discriminator tag so that a custom optionType `check` can distinguish them from a
  derivation. `_nputMarker` SHALL stay entirely inside Nix evaluation and MUST NOT leak
  into `manifest.json`; what crosses into the Go contract SHALL be the clean enums
  `srcKind` and `rootKind`.
---
# REQ-1dcc9a33: marker は判別タグで識別し manifest.json には漏らさない

## 仕様

`src` の `set`（derivation）と marker（`mkOutOfStoreSymlink`）はどちらも attrset で
構造判別できないため、marker には `_nputMarker` 判別タグを持たせ custom optionType の
`check` で判別する。

`_nputMarker` は Nix 評価内で完結させ `manifest.json` には漏らさない。Go 契約は
`srcKind` / `rootKind` の clean enum。

## 出典

`docs/spec.md`「lib API」→「入力検査」節の「marker のタグ方式」注記。
