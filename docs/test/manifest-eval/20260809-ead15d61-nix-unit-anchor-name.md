---
id: "CASE-ead15d61-8ca7-41fb-9121-a5d247ef727a"
type: test_case
name: "nix-unit: anchor-name.nix — GC アンカー名の形式と決定性"
covers:
  - "TC-a6a14739-89e5-4eba-acee-3ed5be5a0b7e"
---
# CASE-ead15d61: nix-unit anchor-name.nix

## 対象

`tests/nix-unit/anchor-name.nix`（TP-403c55c7 のテスト seam `nput.__internal.anchorName` を
直接叩く）

## 検証内容

`anchorName = sha256(target)` の先頭 32 hex について、形式・決定性・特殊文字耐性をアサートする。

- 出力長が常に 32
- 出力が小文字 hex のみ（`[0-9a-f]{32}` に一致）
- 通常の target が外部固定の既知 hash リテラルに一致（決定性 + 値の正しさ。関数の再実装では
  なく ground-truth との突き合わせ）
- 同じ target を 2 回適用しても同値
- 異なる target が異なる hash になる（衝突しないことの示唆）
- cyrillic / 日本語 / 空白 / 記号（`& * " |`）を含む target でも、既知 hash に安定一致し、
  長さ 32・全 hex を保つ（FS 名として直に使えない文字が sha256 経由で潰れること）

## 出典

`tests/nix-unit/anchor-name.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
アンカー名を target の sha256 とする設計判断は ADR-0016 が持つ。
