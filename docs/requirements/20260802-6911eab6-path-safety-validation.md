---
id: "REQ-6911eab6-12b4-457c-9db4-d7430a9e9b3f"
type: requirement
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
name: "target / subpath のパス安全性を評価時に検査する"
specification: |
  `normalizeManifest` SHALL validate the path safety of `target` and `subpath`. An
  absolute path (starting with `/`) SHALL be an error, and after normalization equivalent
  to `filepath.Clean` any path that escapes root or src via `..` SHALL be rejected. The
  static absolute / `..` decision for `target` SHALL be possible at evaluation time,
  independently of the concrete value of root (which is resolved at runtime).
specification_ja: |
  `normalizeManifest` は `target` / `subpath` のパス安全性を検査しなければならない。
  絶対パス（`/` 始まり）はエラーとし、`filepath.Clean` 相当で正規化したうえで `..` に
  よって root / src の外へ出るものを拒否する。`target` の静的な絶対 / `..` 判定は root の
  実体値（実行時解決）に依らず eval 時に可能でなければならない。
---
# REQ-6911eab6: target / subpath のパス安全性を評価時に検査する

## 仕様

`normalizeManifest` は**パス安全性検査**を行う。

- `target` / `subpath` が絶対パス（`/` 始まり）ならエラー。
- `filepath.Clean` 相当で正規化し、`..` で root / src の外へ出るものを拒否する。

`target` の静的な絶対 / `..` 判定は root の実体値（実行時解決）に依らず eval 時に可能。
root が `homeRoot` / `projectRoot` のように実行時解決であっても、「root からの相対で
外へ出るか」は文字列だけで判定できるため、engine 実行時ではなく評価時に止める。

## 出典

`docs/spec.md`「lib API」→「入力検査（`evalModules` + `normalizeManifest`）」。
