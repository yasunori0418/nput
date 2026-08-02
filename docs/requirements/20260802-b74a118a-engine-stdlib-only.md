---
id: "REQ-b74a118a-1272-44eb-944c-7725163211c6"
type: requirement
name: "engine は第三者依存ゼロの stdlib-only とし内部層に閉じる"
specification: |
  The engine SHALL be stdlib-only with zero third-party dependencies, and SHALL be an
  internal layer rather than a public module, so that what it depends on is confined to the
  Go standard library and the runtime commands it invokes. File locking, recursive copying,
  the manifest contract and error wrapping SHALL all be realised with the standard library.
  Which external commands it may invoke at run time is stated by REQ-6c4e174a and SHALL NOT
  be restated here.
specification_ja: |
  engine は第三者依存ゼロの stdlib-only とし、公開モジュールではなく内部層としなければ
  ならない（依存を Go 標準ライブラリと実行時に叩くコマンドだけに閉じるため）。ファイル
  ロック・再帰コピー・manifest 契約・エラーラップはいずれも標準ライブラリで実現する。
  実行時に叩いてよい外部コマンドは REQ-6c4e174a の規範であり、ここでは再掲しない。
---
# REQ-b74a118a: engine は第三者依存ゼロの stdlib-only とし内部層に閉じる

## 仕様

| コンポーネント | 依存 |
|---|---|
| `internal/`（配置エンジン = 内部層分離。公開モジュールではない）| **stdlib-only 厳守**（第三者依存ゼロ）。`syscall.Flock`・`filepath.WalkDir`+`io.Copy`+`os.Chmod`・`encoding/json`・`fmt.Errorf`+`%w`。`manifest.json` を入力に取り runtime に `nix`（profile）/ `git`（toplevel）をサブプロセスで要求 |

> **上は原文の写しで、規範は frontmatter が正**。実行時に叩く外部コマンドを `nix` と
> `git` に限ることは REQ-6c4e174a、engine が `manifest.json` を入力に取る 2 層構成は
> REQ-f4d7d4ab、その `manifest.json` が唯一の安定契約であることは REQ-79ce0a09 の担当。
> 表が挙げる具体的な標準ライブラリ API 名は stdlib-only を満たす実装手段の例示であり、
> 規範文では「標準ライブラリで実現する」に留めた。

## 出典

`docs/spec.md`「依存関係」節の表の `internal/` 行。

決定の実体は ADR-0011「engine / CLI の技術スタックを確定する」で、engine を stdlib-only の
内部層とすることを定めている。runtime に `nix` / `git` をサブプロセスで要求する形は
ADR-0006「エンジンを固定の Go バイナリにし、lib はデータ生成に徹する」が定めるが、同 ADR は
stdlib-only 自体を決めていないため、側面の根拠として `justifies` は張らない（外部コマンドの
限定そのものの帰属は REQ-6c4e174a が担当する）。
