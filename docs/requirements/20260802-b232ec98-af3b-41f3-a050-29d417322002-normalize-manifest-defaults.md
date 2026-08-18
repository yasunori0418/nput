---
id: "REQ-b232ec98-af3b-41f3-a050-29d417322002"
type: requirement
name: "normalizeManifest が検査・デフォルト適用・marker 変換を行い mkManifest が derivation を組む"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The input validation implementation SHALL be split in two stages.
  `normalizeManifest { lib, root, entries } -> attrset` SHALL be a pure data function that
  performs `evalModules` validation, applies defaults (`subpath` → `"."`, `method` →
  `"symlink"`, `target` → the attribute key), and converts the internal marker tag
  (`_nputMarker`) into the clean enums (`srcKind` / `rootKind`).
  `normalizeManifest` SHALL be a unit-test target for the Nix evaluation tests (nix-unit /
  namaka). `mkManifest = args: derivation` SHALL write the output of `normalizeManifest`
  into `manifest.json` and assemble the symlink farm. The lib layer SHALL be
  unparameterized: each function SHALL require `lib` (`normalizeManifest`) or `pkgs`
  (`mkManifest`) as an explicit argument at call time.
specification_ja: |
  入力検査の実装は 2 段に分かれなければならない。`normalizeManifest { lib, root, entries }
  -> attrset` は `evalModules` 検査・デフォルト適用（`subpath` → `"."` / `method` →
  `"symlink"` / `target` → 属性キー）・内部 marker タグ（`_nputMarker`）から clean enum
  （`srcKind` / `rootKind`）への変換を行う純データ関数とし、Nix 評価テスト
  （nix-unit / namaka）の単体対象としなければならない。`mkManifest = args:
  derivation` は `normalizeManifest` の出力を `manifest.json` に書き symlink farm を
  組まなければならない。lib 層は unparameterized でなければならず、各関数が呼び出し時に
  `lib`（`normalizeManifest`）または `pkgs`（`mkManifest`）を明示引数として要求しなければ
  ならない。
---
# REQ-b232ec98: normalizeManifest が検査・デフォルト適用・marker 変換を行い mkManifest が derivation を組む

## 仕様

実装は 2 段に分かれる。lib 層は unparameterized（`lib` / `pkgs` を自身で保持しない）で、
各関数が呼び出し時に `lib`（`normalizeManifest`）または `pkgs`（`mkManifest`）を明示
引数として要求する。

- **`normalizeManifest { lib, root, entries } -> attrset`**: 純データ関数。
  nix-unit / namaka の単体対象。
  - `evalModules` 検査
  - デフォルト適用: `subpath` → `"."` / `method` → `"symlink"` / `target` → 属性キー
  - 内部 marker タグ（`_nputMarker`）→ clean enum（`srcKind` / `rootKind`）変換
- **`mkManifest = args: derivation`**: `normalizeManifest` の出力を `manifest.json` に
  書き symlink farm を組む。

クロスフィールドチェック・パス安全性検査・target 衝突検出も `normalizeManifest` が
担う（別 item）。

## 出典

`docs/spec.md`「lib API」→「入力検査（`evalModules` + `normalizeManifest`）」。
