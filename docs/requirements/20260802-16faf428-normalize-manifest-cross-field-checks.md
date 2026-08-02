---
id: "REQ-16faf428-77f3-492f-b858-222c5274cbf7"
type: requirement
name: "未実装の組み合わせをクロスフィールドチェックで評価時に拒否する"
specification: |
  `normalizeManifest` SHALL reject, via `lib.throwIf`, the combinations that a
  single-field type check cannot express: `method = "copy"` together with an out-of-store
  marker, and `root = systemRoot` (not yet implemented).
---
# REQ-16faf428: 未実装の組み合わせをクロスフィールドチェックで評価時に拒否する

## 仕様

`normalizeManifest` はクロスフィールドの `lib.throwIf` チェックを持つ。

- `method = "copy"` かつ out-of-store marker の組み合わせを拒否する。
- `root = systemRoot` を未実装として拒否する。

いずれも単一フィールドの optionType では表現できない条件のため、型検査とは別に
`throwIf` で評価時に停止させる。

## 出典

`docs/spec.md`「lib API」→「入力検査（`evalModules` + `normalizeManifest`）」。
