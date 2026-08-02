---
id: "REQ-35f485ff-4d37-4dac-a442-73c0babdbac0"
type: requirement
name: "engine は配置をネイティブ FS 操作で行い外部コマンドに委ねない"
specification: |
  The engine SHALL carry out placement through native filesystem operations and SHALL NOT
  delegate it to external commands such as `ln` or `rsync`. This SHALL apply to every
  placement method (symlink, out-of-store symlink and copy) and to the removal paths.
specification_ja: |
  engine は配置をネイティブ FS 操作で行わなければならず、`ln` / `rsync` などの外部
  コマンドへ委ねてはならない。これは全ての配置方法（symlink・out-of-store symlink・copy）
  と除去経路に適用する。
---
# REQ-35f485ff: engine は配置をネイティブ FS 操作で行い外部コマンドに委ねない

## 仕様

配置は engine が**ネイティブ FS 操作**で行う（`ln` / `rsync` は使わない）。

> **上は原文の写しで、規範は frontmatter が正**。engine が `nix` / `git` 以外の
> サブプロセスを使わないという配置エンジン層一般の制約は REQ-6c4e174a の担当で、
> 本 item は配置動作がネイティブ FS 操作であることだけを規定する。

## 出典

`docs/spec.md`「配置動作仕様」節の導入文。

決定の実体は ADR-0006「エンジンを固定の Go バイナリにし、lib はデータ生成に徹する」。
