---
id: "REQ-d1b5b3f5-10a0-400d-9f03-ba00c63d1c34"
type: requirement
name: "mkManifest 自身が evalModules で入力を検査する単一ゲートになる"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  `lib.mkManifest` SHALL itself run `lib.evalModules` internally to validate and normalize
  `entries` and `root`. Declaring the types only on module options would make validation
  effective on the module path (`nput.entries`) alone, so `mkManifest` SHALL own the
  validation in order to cover the CLI / entrypoint path (a direct `mkManifest` call) as
  well. The entry type definition (the submodule in `lib/types.nix`) SHALL be shared with
  `attrsOf (submodule …)` in `modules/common.nix`, and the resulting double validation on
  the module path is acceptable because it is pure and idempotent.
specification_ja: |
  `lib.mkManifest` は内部で `lib.evalModules` を回して `entries` / `root` を検査・
  正規化しなければならない。型をモジュールオプションに書くだけでは検査がモジュール経路
  （`nput.entries`）でしか効かないため、CLI / entrypoint 経路（`mkManifest` 直呼び）でも
  検査を効かせる目的で `mkManifest` 自身が検査を持つ。entry の型定義（`lib/types.nix` の
  submodule）は `modules/common.nix` の `attrsOf (submodule …)` と共有し、モジュール経路
  での二重検査（純粋・冪等）を許容する。
---
# REQ-d1b5b3f5: mkManifest 自身が evalModules で入力を検査する単一ゲートになる

## 仕様

`mkManifest` は内部で `lib.evalModules` を回して `entries` / `root` を検査・正規化する。

- 型をオプションに書くだけだと検査が効くのはモジュール経路（`nput.entries`）のみだが、
  コアである CLI / entrypoint 経路（`mkManifest` 直呼び）でも検査を効かせるため
  `mkManifest` 自身が `evalModules` を回す。
- `lib.types` / `mkOption` / `evalModules` は `nixpkgs.lib` のコアなので「lib は
  nixpkgs.lib のみ依存」を満たす。
- entry の型定義（`lib/types.nix` の submodule）は `modules/common.nix` の
  `attrsOf (submodule …)` と共有する。モジュール経路では host の `evalModules`
  （`attrsOf`）と `mkManifest` の `evalModules` で二重に検査されるが、純粋・冪等で
  害はなく、`mkManifest` を「entries が必ず通る単一の検査ゲート」に保つ。

## 出典

`docs/spec.md`「lib API」→「入力検査（`evalModules` + `normalizeManifest`）」。
