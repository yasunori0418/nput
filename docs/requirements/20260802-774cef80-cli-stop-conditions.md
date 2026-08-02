---
id: "REQ-774cef80-2872-4ea1-937b-a0fbabc305a9"
type: requirement
name: "対象を確定できないときは CLI がエラーで停止し、暗黙のフォールバックを採らない"
specification: |
  When the CLI cannot determine what to operate on, it SHALL stop with an error and SHALL
  NOT fall back implicitly. It SHALL stop when no entrypoint can be discovered — no
  entrypoint file in the current working directory and no explicit `-f` — and when the
  discovered entrypoint does not expose the requested `nput.<name>`. Likewise `rollback`
  SHALL print an error message and stop when no previous generation exists. Guessing
  another config, treating the situation as a no-op, or continuing with an empty manifest
  MUST NOT be adopted in any of these cases, since each would silently produce a placement
  the user did not request.
specification_ja: |
  CLI は操作対象を確定できないときエラーで停止しなければならず、暗黙のフォールバックを
  採ってはならない。entrypoint が発見できないとき（CWD に entrypoint ファイルが無く
  `-f` の明示も無い）と、発見した entrypoint に指定の `nput.<name>` が存在しないときは
  停止する。同様に `rollback` は前世代が存在しないとき、エラーメッセージを出力して停止する。
  いずれの場合も、別 config の推測・no-op 扱い・空 manifest での続行を採ってはならない
  （ユーザーが要求していない配置を黙って生むため）。
---
# REQ-774cef80: 対象を確定できないときは CLI がエラーで停止し、暗黙のフォールバックを採らない

## 仕様

| 条件 | 動作 |
|---|---|
| `rollback` で前世代が存在しない | エラーメッセージを出力して停止 |
| `nput.<name>` が entrypoint に存在しない | CLI がエラーで停止 |
| entrypoint が発見できない（CWD に flake.nix/shell.nix/default.nix なし・`-f` 未指定）| CLI がエラーで停止 |

> **上は原文の写しで、規範は frontmatter が正**。entrypoint の探索順と `-f` による上書きは
> REQ-1cc080f6、`nput.<name>` のアドレッシングは REQ-496b1a07、`apply` で name を省略した
> ときに `default` へ解決すること（および `nput.default` 未定義でのエラー停止）は
> REQ-c2d44626 / REQ-205d744d、`rollback` が名指し必須で home mode 限定であることは
> REQ-89c7baf9 / REQ-05abce3e、`gitignore` を非 project config へ与えたときのエラー停止は
> REQ-eaa8c0df、`--manifest` と `-f` / `--all` の併用エラーは REQ-dec58330、
> experimental-features 未有効の案内エラーは REQ-f9920c87 の担当。停止時の終了コードは
> REQ-2c5a10d8 の担当。

## 出典

`docs/spec.md`「エラー仕様」節の表のうち、対象を確定できないことによる CLI 停止の 3 行。

決定の実体は ADR-0007「CLI を一次 UX とし entrypoint を発見する」で、CLI が entrypoint と
named manifest を解決する責務を負うことを定めている。
