---
id: "REQ-774cef80-2872-4ea1-937b-a0fbabc305a9"
type: requirement
name: "要求された操作が成立しないときは CLI がエラーで停止し、暗黙のフォールバックを採らない"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
  - "UC-0b6f60cb-3e98-4ee7-8929-4d94a29f0af6"
specification: |
  When the operation the user asked for does not hold, the CLI SHALL stop with an error and
  SHALL NOT fall back implicitly. It SHALL stop when no entrypoint can be discovered — no
  entrypoint file in the current working directory and no explicit `-f` — when the
  discovered entrypoint does not expose the requested `nput.<name>`, and, for `rollback`,
  when no previous generation exists, in which case it SHALL print an error message before
  stopping. Guessing another config, treating the situation as a no-op, or continuing with
  an empty manifest SHALL NOT be adopted in any of these cases, since each would silently
  produce a placement the user did not request.
specification_ja: |
  ユーザーが要求した操作が成立しないとき、CLI はエラーで停止しなければならず、暗黙の
  フォールバックを採ってはならない。entrypoint が発見できないとき（CWD に entrypoint
  ファイルが無く `-f` の明示も無い）、発見した entrypoint に指定の `nput.<name>` が存在
  しないとき、および `rollback` で前世代が存在しないときに停止する（`rollback` は停止前に
  エラーメッセージを出力する）。いずれの場合も、別 config の推測・no-op 扱い・空 manifest
  での続行を採ってはならない（ユーザーが要求していない配置を黙って生むため）。
---
# REQ-774cef80: 要求された操作が成立しないときは CLI がエラーで停止し、暗黙のフォールバックを採らない

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

`docs/spec.md`「エラー仕様」節の表のうち、要求された操作が成立しないことによる CLI 停止の
3 行（`rollback` で前世代なし・`nput.<name>` が entrypoint に不在・entrypoint が発見できない）。
同じ述語を満たす他の行（`nput.default` 未定義・`gitignore` の非 project config・`--manifest`
の併用・experimental-features 未有効）は、下記のとおり各担当 item が規範を持つため除く。

この 3 行が挙げる停止そのものに対応する決定を持つ ADR は無く、`docs/spec.md` が一次記述に
あたる。よって本 item に `justifies` は張られないが、これは張り漏れではない。前提となる
「CLI が entrypoint と named manifest を解決する」ことは ADR-0007「汎用 nput CLI を一次 UX に
昇格し、entrypoint 発見＋root 明示モデルへ移行する」が定めるが、同 ADR は解決できなかった
ときの扱いを決めていないため、側面の根拠として `justifies` は張らない（前提そのものの帰属は
REQ-1cc080f6 / REQ-496b1a07 が担当する）。
