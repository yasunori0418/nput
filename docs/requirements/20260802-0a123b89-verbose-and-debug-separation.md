---
id: "REQ-0a123b89-0399-4f76-b988-56a5f7e0becf"
type: requirement
name: "冗長度は -v、デバッグは --debug に分離し --json と直交させる"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The placement report SHALL be written to stderr only when `-v` / `--verbose` is given
  (an opt-in on top of the default silence), including the skip notice and the
  `apply --all` completion summary. Disclosure of the nix commands run internally SHALL be
  separated into `--debug`, so that verbosity (`-v`) and debugging (`--debug`) are
  orthogonal. `--json` (stdout, machine-facing) SHALL be orthogonal to `-v` (stderr,
  human-facing) and usable together with it. `--quiet` SHALL NOT be provided, having been
  abolished along with the move to silence by default.
specification_ja: |
  配置レポートは `-v` / `--verbose` を付けたときだけ stderr に出さなければならない
  （既定沈黙の opt-in・skip 通知・`apply --all` 完了サマリも含む）。内部実行する nix
  コマンドの開示は `--debug` に分離しなければならず、冗長度（`-v`）とデバッグ
  （`--debug`）を直交させなければならない。`--json`（stdout・機械向け）は `-v`
  （stderr・人間向け）と直交し併用できなければならない。
  `--quiet` は既定沈黙化に伴い廃止し、提供してはならない。
---
# REQ-0a123b89: 冗長度は -v、デバッグは --debug に分離し --json と直交させる

## 仕様

**`-v` / `--verbose` を付けたときだけ配置レポートを stderr に出す**（既定沈黙の opt-in・
skip 通知・`apply --all` 完了サマリも含む）。**内部実行する nix コマンドの開示は
`--debug`** に分離する（冗長度＝`-v` と、デバッグ＝`--debug` を直交させる）。
`--json`（stdout・機械向け）は `-v`（stderr・人間向け）と直交・併用可。`--quiet` は
既定沈黙化に伴い廃止した。

既定沈黙そのものは REQ-8ef34101、`--help` 等での nix コマンド開示（透明性）は
REQ-4ffda99a の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」の `-v` / `--verbose` の
箇条書き、およびグローバルフラグ表の `-v` / `--debug`。

決定の実体は ADR-0031「`-v` で配置レポート opt-in・`--debug` で nix コマンド開示・
`--quiet` 廃止」。`--json` との直交は ADR-0043。
