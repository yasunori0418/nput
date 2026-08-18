---
id: "REQ-1dcc9a33-b0f2-43e0-8310-fc4b19e68fe7"
type: requirement
name: "marker は判別タグで識別し manifest.json には漏らさない"
derives_from:
  - "UC-01b896b4-04b9-40d0-bf9e-966eaf64c3d4"
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  Markers (`mkOutOfStoreSymlink` and the root markers) SHALL carry a `_nputMarker`
  discriminator tag so that a custom optionType `check` can distinguish them from a
  derivation. `_nputMarker` SHALL stay entirely inside Nix evaluation and SHALL NOT leak
  into `manifest.json`; what crosses into the Go contract SHALL be the clean enums
  `srcKind` and `rootKind`.
specification_ja: |
  marker（`mkOutOfStoreSymlink` / root マーカー）は `_nputMarker` 判別タグを持ち、
  custom optionType の `check` で derivation と判別できなければならない。`_nputMarker` は
  Nix 評価内で完結させ、`manifest.json` へ漏らしてはならない。Go 契約へ渡すのは
  `srcKind` / `rootKind` の clean enum でなければならない。
---
# REQ-1dcc9a33-b0f2-43e0-8310-fc4b19e68fe7: marker は判別タグで識別し manifest.json には漏らさない

## 仕様

`src` の `set`（derivation）と marker（`mkOutOfStoreSymlink`）はどちらも attrset で
構造判別できないため、marker には `_nputMarker` 判別タグを持たせ custom optionType の
`check` で判別する。

`_nputMarker` は Nix 評価内で完結させ `manifest.json` には漏らさない。Go 契約は
`srcKind` / `rootKind` の clean enum。

## 出典

`docs/spec.md`「lib API」→「入力検査（`evalModules` + `normalizeManifest`）」節の
「marker のタグ方式」注記。
