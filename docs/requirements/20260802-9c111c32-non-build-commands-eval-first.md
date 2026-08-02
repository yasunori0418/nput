---
id: "REQ-9c111c32-8f6c-4eda-859c-dae02c0624fc"
type: requirement
name: "非 build コマンドも eval 先行を共通前段に持つ"
specification: |
  The non-building commands (`reset` / `rollback` / `list-generations`) SHALL also have
  eval-first as a common preliminary stage. They do not build, but determining
  `profileDir` (that is, the preliminary rootKind eval followed by root resolution) is a
  precondition for the per-`profileDir` flock and for reading the previous generation
  manifest. When overridden with `--root`, `profileDir` SHALL be looked up under the same
  roothash key as elsewhere, so that operating on a generation created with `--root`
  requires the same `--root`.
specification_ja: |
  非 build コマンド（`reset` / `rollback` / `list-generations`）も eval 先行を共通前段に
  持たなければならない。build はしないが、profileDir 単位の flock と前世代 manifest 読みの
  ため profileDir 確定（= rootKind 先取り eval → root 解決）が前提になる。`--root` 上書き
  時は同じ roothash キーで profileDir を引く（`--root` を付けた世代を操作するには同じ
  `--root` が要る）。
---
# REQ-9c111c32: 非 build コマンドも eval 先行を共通前段に持つ

## 仕様

**非 build コマンド（`reset` / `rollback` / `list-generations`）も eval 先行を共通前段に
持つ**。build はしないが、profileDir 単位の flock / 前世代 manifest 読みのため
profileDir 確定（= rootKind 先取り eval → root 解決）が前提になる。`--root` 上書き時は
「root の解決」と同じ roothash キーで profileDir を引く（`--root` を付けた世代を操作するには
同じ `--root` が要る）。

**本 item が非 build コマンド一般の eval 先行を規定する**。個々のコマンドが加える固有の
前段（`reset` が entries 読みのため entrypoint eval も行うこと）は各コマンドの item が
規定し、本 item では扱わない（→ REQ-a8edc58f）。

apply を含む実行フロー全体の順序規範は REQ-60c6b7ea、`--root` のキーイングは
REQ-61c05e09 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「実行フロー」の非 build コマンドの箇条書き。

決定の実体は ADR-0024「非 build コマンドの eval 先行」。
