---
id: "REQ-2f9205ee-cec5-4072-ac3e-890caae79904"
type: requirement
name: "mkManifest の返り値は passthru で root kind を露出する"
specification: |
  The derivation returned by `lib.mkManifest` SHALL carry `passthru.rootKind` (`"project"`
  / `"home"` / `"system"` / `"fixed"`), and for `fixed` it SHALL also carry the absolute
  path string in `passthru.root`. `rootKind` SHALL be determined at evaluation time
  (resolving the concrete path remains the engine's runtime work), so that the CLI can
  determine profileDir *before* building via
  `nix eval <ep>#nput.<system>.<name>.rootKind` and proceed in flock-then-build order.
---
# REQ-2f9205ee: mkManifest の返り値は passthru で root kind を露出する

## 仕様

返り値 derivation は `passthru.rootKind`（`"project"` / `"home"` / `"system"` /
`"fixed"`、`fixed` のときは `passthru.root` に絶対パス文字列）を持つ。

CLI が**ビルド前に** `nix eval <ep>#nput.<system>.<name>.rootKind` で profileDir を
確定し、flock → build の順に進める実行フロー（`docs/spec.md`「実行フロー」）を
成立させるため。

`rootKind` は eval 時に確定する（実体パス解決は engine 実行時）。

## 出典

`docs/spec.md`「lib API」→「`lib.mkManifest`」返り値の注記。
