---
id: "REQ-250d936c-1df0-491d-a7af-708f38b61f33"
type: requirement
name: "MVP は manifest v1 のみを発行・受理しマイグレーション機構を持たない"
specification: |
  The MVP SHALL emit and accept `schemaVersion = 1` only, and a migration mechanism for
  reading older schema versions MUST NOT be built up front. A backward-compatibility
  policy — namely that the engine must be able to read a manifest of an older version for
  stale removal and rollback, since stale removal right after an upgrade reads the
  previous generation's manifest — SHALL be revisited once a field addition makes v2
  necessary after the first release.
specification_ja: |
  MVP は `schemaVersion = 1` のみを発行・受理し、古い版の manifest を読む
  マイグレーション機構を最初から作ってはならない。後方互換ポリシー（engine が
  stale 除去 / rollback のために古い版の manifest を読めること。アップグレード直後の
  stale 除去は前世代 manifest を読む）は、最初のリリース後にフィールド追加で v2 が
  必要になった時点で改めて検討しなければならない。
---
# REQ-250d936c: MVP は manifest v1 のみを発行・受理しマイグレーション機構を持たない

## 仕様

**MVP は v1 のみを発行・受理し、マイグレーション（schema 後方互換）は現時点では
考慮しない**（古い版の受理機構を最初から作らない）。

**最初のリリース後、フィールド追加で v2 が必要になった時点で**、後方互換ポリシーを
改めて検討する。検討対象は「engine が古い版の manifest を stale 除去 / rollback のために
読めること」で、アップグレード直後の stale 除去が前世代 manifest を読むという事実が
その必要性の根拠になる。

engine が自身の対応版より新しい `schemaVersion` を拒否することは REQ-79ce0a09 が持つ。
本 item は「古い版を読む機構を作らない」側を規定する。

## 出典

`docs/spec.md`「manifest.json スキーマ（v1・Nix↔Go 契約）」の導入段落。
