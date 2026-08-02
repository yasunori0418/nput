---
id: "REQ-16faf428-77f3-492f-b858-222c5274cbf7"
type: requirement
name: "意図が矛盾する組み合わせをクロスフィールドチェックで評価時に拒否する"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
  - "UC-01b896b4-04b9-40d0-bf9e-966eaf64c3d4"
specification: |
  `normalizeManifest` SHALL reject, via `lib.throwIf`, a combination that a single-field
  type check cannot express: `method = "copy"` together with an out-of-store marker.
  Out-of-store means a live symlink while copy means a place-once snapshot, so the two
  intents contradict each other and SHALL fail early at evaluation time rather than being
  silently downgraded.
specification_ja: |
  `normalizeManifest` は単一フィールドの型検査では表現できない組み合わせを
  `lib.throwIf` で拒否しなければならない。対象は `method = "copy"` かつ out-of-store
  marker の組み合わせである。out-of-store は「ライブ symlink」、copy は「place-once
  スナップショット」で意図が矛盾するため、暗黙降格ではなく評価時の早期エラーとする。
---
# REQ-16faf428: 意図が矛盾する組み合わせをクロスフィールドチェックで評価時に拒否する

## 仕様

`normalizeManifest` はクロスフィールドの `lib.throwIf` チェックを持つ。

- `method = "copy"` かつ out-of-store marker の組み合わせを拒否する。
- `root = systemRoot` を未実装として拒否する。

上の箇条書きは原文の写しで、規範は frontmatter が正。`root = systemRoot` の拒否は
規範に含めない（理由は下の注記）。

単一フィールドの optionType では表現できない条件のため、型検査とは別に `throwIf` で
評価時に停止させる。out-of-store は「ライブ symlink」、copy は「place-once
スナップショット」で意図が矛盾するので、暗黙に降格させず早期エラーにする（→ ADR-0013 §4）。

## 出典

`docs/spec.md`「lib API」→「入力検査（`evalModules` + `normalizeManifest`）」。

> **`root = systemRoot` の拒否を本 item に含めない理由**: `docs/spec.md` の当該記述は
> 「`root = systemRoot` の未実装拒否」も同じ `throwIf` の対象として挙げるが、この決定
> （ADR-0013 §5）は **ADR-0036 が撤回済み**で、現行は `rootKind = "system"` を正規の値
> として通す（ADR-0013 冒頭の 2026-07-04 改訂注記）。`docs/spec.md` 側がこの改訂に
> 追従できていないため、分割にあたって撤回済みの決定を規範文へ持ち込まない判断をした。
> `docs/spec.md` の追従は本 item の担当範囲外（→ #209 の後続・別 issue）。
