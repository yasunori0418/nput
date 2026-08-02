---
id: "REQ-d95b814f-aa7a-470e-9320-c14f9c14da7b"
type: requirement
name: "--all は root モードフィルタで対象 config を絞れる"
specification: |
  When `--project-root` / `--home-root` / `--system-root` is given together with
  `apply --all`, only the configs of the corresponding root mode SHALL be applied
  (an opt-in filter named after the root markers). A bare `--all` SHALL apply every
  config. Because a devShell `shellHook` that runs a bare `--all` in an entrypoint mixing
  home mode and project mode configs would also place the home mode configs under `$HOME`,
  a devShell SHOULD use `nput apply --all --project-root` or an explicitly named
  `nput apply <name>`. The filter SHALL be a modifier of `--all` only, since a named apply
  already pins a single config.
specification_ja: |
  `apply --all` に `--project-root` / `--home-root` / `--system-root` を付けたときは、
  `nput.*` のうち該当 root モードの config のみを適用しなければならない（root マーカー名に
  揃えたフィルタ・opt-in）。素の `--all` は全 config を適用する。home mode と project mode の
  config が混在する entrypoint で devShell の `shellHook` から素の `--all` を打つと home mode
  config も `$HOME` に配置される footgun があるため、devShell は
  `nput apply --all --project-root`（または名指し `nput apply <name>`）を使うべきである。
  フィルタは `--all` の修飾に限る（名指し apply では `<name>` が 1 config を pin するため
  無意味）。
---
# REQ-d95b814f: --all は root モードフィルタで対象 config を絞れる

## 仕様

`apply --all` に **`--project-root` / `--home-root` / `--system-root`** を付けると、
`nput.*` のうち該当 root モードの config **のみ**を適用する（root マーカー名に揃えた
フィルタ・opt-in）。素の `--all` は全 config を適用する。

home mode と project mode の config が混在する entrypoint で devShell の `shellHook` から
`--all` を打つと home mode config も `$HOME` に配置される footgun があるため、devShell は
**`nput apply --all --project-root`**（または名指し `nput apply <name>`）を使う。
フィルタは `--all` 修飾で、名指し apply では `<name>` が 1 config を pin するため無意味。

`--system-root` は system mode が未実装のため当面マッチしない将来 seam である
（`docs/spec.md`「グローバルフラグ」の注記）。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `apply --all --project-root` の
箇条書き、およびグローバルフラグ表の `--project-root` / `--home-root` / `--system-root`。

決定の実体は ADR-0017「`--all` の root モードフィルタ」。
