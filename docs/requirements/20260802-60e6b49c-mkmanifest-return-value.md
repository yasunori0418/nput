---
id: "REQ-60e6b49c-9ba1-4552-a0ec-d340421ec281"
type: requirement
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
name: "mkManifest は manifest.json と symlink farm を含む store オブジェクトを返す"
specification: |
  The return value of `lib.mkManifest` SHALL be the store object that the profile points
  at. It SHALL contain `manifest.json` (carrying `schemaVersion`, the engine's input
  contract) and an explicit symlink farm pointing at the placement sources (the GC
  anchor). `manifest.json` SHALL record, for every entry, the resolved placement source
  (store path or out-of-store absolute path), `subpath`, `target`, `method`, and the kind
  of root (project / home / system / fixed path).
specification_ja: |
  `lib.mkManifest` の返り値は profile が指す store オブジェクトでなければならない。
  これは `manifest.json`（`schemaVersion` 付き・エンジンの入力契約）と、配置元への
  明示 symlink farm（GC アンカー）を含む。`manifest.json` は各 entry の解決済み配置元
  （store パス / out-of-store 絶対パス）・`subpath`・`target`・`method`・root の kind
  （project / home / system / 固定パス）を記録する。
---
# REQ-60e6b49c: mkManifest は manifest.json と symlink farm を含む store オブジェクトを返す

## 仕様

返り値は profile が指す store オブジェクト。以下を含む。

- **`manifest.json`**: `schemaVersion` 付き・エンジンの入力契約。各 entry の解決済み
  配置元（store パス / out-of-store 絶対パス）・`subpath`・`target`・`method`・root の
  kind（project / home / system / 固定パス）を記録する（全フィールドは
  `docs/spec.md`「manifest.json スキーマ（v1）」）。
- **配置元への明示 symlink farm**: GC アンカー。profile が生きている限り配置元の
  store path が GC されない。

## 出典

`docs/spec.md`「lib API」→「`lib.mkManifest`」返り値。
