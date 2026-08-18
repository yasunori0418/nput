---
id: "REQ-1cc080f6-ae91-4c1f-973e-b7054cfc0198"
type: requirement
name: "entrypoint は CWD で flake.nix → shell.nix → default.nix の順に探し -f で上書きする"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The CLI SHALL discover an entrypoint file, that is, a nix file exposing a named manifest
  at `nput.<name>`. By default it SHALL search the current working directory in the
  priority order `flake.nix` → `shell.nix` → `default.nix`. The `-f` / `--file <path>`
  flag SHALL specify the entrypoint explicitly and SHALL override the automatic search.
specification_ja: |
  CLI は entrypoint ファイル（`nput.<name>` に named manifest を公開する nix ファイル）を
  発見しなければならない。既定では CWD を `flake.nix` → `shell.nix` → `default.nix` の
  優先順で探さなければならない。`-f` / `--file <path>` は entrypoint を明示指定し、自動探索を
  上書きしなければならない。
---
# REQ-1cc080f6-ae91-4c1f-973e-b7054cfc0198: entrypoint は CWD で flake.nix → shell.nix → default.nix の順に探し -f で上書きする

## 仕様

CLI は **entrypoint ファイル**（`nput.<name>` に named manifest を公開する nix ファイル）を
発見する。

| 方法 | 挙動 |
|---|---|
| 既定（自動探索）| CWD で `flake.nix` → `shell.nix` → `default.nix` の優先順で探す |
| `-f` / `--file <path>` | entrypoint を明示指定（自動探索を上書き）|

## 出典

`docs/spec.md`「CLI 仕様」→「entrypoint の発見」。

決定の実体は ADR-0007 §3「CLI の責務」1.（既定: CWD で `flake.nix` → `shell.nix` →
`default.nix` の優先順。`-f` / `--file <path>` で明示上書き）。
