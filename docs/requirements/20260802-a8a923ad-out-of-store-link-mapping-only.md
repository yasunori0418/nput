---
id: "REQ-a8a923ad-07fb-4582-b90a-07a6e0c41baa"
type: requirement
name: "out-of-store symlink は marker の絶対パスを指し、版管理はリンク先マッピングのみとする"
specification: |
  An out-of-store entry SHALL be placed as a symlink whose destination is the absolute
  path held by the marker, on the local filesystem. Generations SHALL version only the
  mapping of which absolute path is pointed at, and SHALL NOT version the content of the
  destination, which is live by design and SHALL never be snapshotted.
specification_ja: |
  out-of-store entry は symlink として配置しなければならず、指す先は marker の絶対パス
  （ローカル FS）とする。世代が版管理するのは「どの絶対パスを指すか」というリンク先
  マッピングのみとし、指す先の内容は版管理してはならない（設計上ライブであり、永遠に
  スナップショットしない）。
---
# REQ-a8a923ad: out-of-store symlink は marker の絶対パスを指し、版管理はリンク先マッピングのみとする

## 仕様

- symlink として配置する。指す先は marker の絶対パス（ローカル FS）
- 世代では「どの絶対パスを指すか」のリンク先マッピングのみ版管理する。指す先の内容は設計上
  ライブで、永遠にスナップショットしない

> **上は原文の写しで、規範は frontmatter が正**。marker の生成と識別（`lib.mkOutOfStoreSymlink`・
> marker の tag 判別）は REQ-eb363122 / REQ-1dcc9a33、method と src の対応表は REQ-77689c68、
> symlink 配置の手順そのものは REQ-622787dc の担当。

## 出典

`docs/spec.md`「配置動作仕様」→「out-of-store symlink」節。

決定の実体は ADR-0002「世代管理を nix profile に乗せる」で確定した、out-of-store のリンク先
マッピングのみを版管理する方針。
