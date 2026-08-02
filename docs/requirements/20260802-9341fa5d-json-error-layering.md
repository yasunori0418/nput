---
id: "REQ-9341fa5d-836e-4023-af53-cc7d273438d1"
type: requirement
name: "エンベロープのエラーは主体の有無で層を分けコードを分類する"
specification: |
  For a single-config command the subject is fixed at the start of RunE, at the point of
  the name argument, so every failure of that execution (entrypoint discovery, eval /
  build, lock, engine runtime) SHALL be carried in `results[0].errors[]`. The top-level
  `errors[]` SHALL carry only the failures of an execution that has no subject, that is,
  failures before any subject has been fixed. An item-caused failure SHALL be embedded in
  `item.error` and SHALL NOT be duplicated into the `errors[]` of the enclosing
  `SubjectResult`. Both SHALL always coexist with the
  human-facing text on stderr (wrapped by the op plus the target path), and the exit code
  table SHALL be unchanged. Error codes SHALL classify what can be determined by a
  sentinel or a type: `E_LOCK` for flock, `E_NPUT_BUILD` for a failure of an internal nix
  eval / build invocation, `E_NOTFOUND`, `E_PERMISSION`, `E_IO` for other filesystem /
  external I/O failures, and `E_NPUT_COLLISION` for the conflict of `apply --dryrun`
  (exit 2). Anything else SHALL fall back to the tool-generic `E_NPUT_FAILED`.
specification_ja: |
  単一 config コマンドは RunE 冒頭・name 引数の時点で subject が確定するため、その実行の
  失敗（entrypoint 発見・eval / build・lock・engine 実行時）は全て `results[0].errors[]` に
  載せなければならない。トップ `errors[]` に載せてよいのは subject を持たない実行の失敗
  （subject が 1 つも確定していない段階の失敗）のみとする。item 起因の失敗は `item.error` に
  埋め、包含する `SubjectResult` の `errors[]` へ重複させてはならない。どちらも stderr の人間向け
  テキスト（op + 対象パスの wrap 規約）を常時併存させ、終了コード表は不変とする。
  エラーコードは sentinel / 型で判定できるものを分類する — `E_LOCK`（flock）/
  `E_NPUT_BUILD`（内部 nix eval / build 呼び出しの失敗）/ `E_NOTFOUND` / `E_PERMISSION` /
  `E_IO`（それ以外の FS / 外部 I/O 失敗）/ `E_NPUT_COLLISION`（`apply --dryrun` の
  conflict = exit 2）。それ以外はツール総称 fallback `E_NPUT_FAILED` とする。
---
# REQ-9341fa5d: エンベロープのエラーは主体の有無で層を分けコードを分類する

## 仕様

**エラーの層と stderr 併存**: 単一 config コマンドは RunE 冒頭・name 引数の時点で subject が
確定するため、その実行の失敗（entrypoint 発見・eval / build・lock・engine 実行時）は
**全て `results[0].errors[]`** に載る。トップ `errors[]` に載るのは **subject を持たない
実行の失敗のみ**（ADR-0043 §6 の「主体列挙・解決の前段」に相当）。どちらも stderr の
人間向けテキスト（op + 対象パスの wrap 規約）を常時併存させ、終了コード表 0 / 1 / 2 は
不変。item 起因の失敗は `item.error` に埋め、包含する `SubjectResult` の `errors[]` へ
重複させない（niface §2）。

**エラーコード**: sentinel / 型で判定できるものを分類する — `E_LOCK`（flock）/
`E_NPUT_BUILD`（内部 nix eval / build 呼び出しの失敗）/ `E_NOTFOUND` / `E_PERMISSION` /
`E_IO`（上記に該当しない FS / 外部 I/O 失敗の形〔PathError 等〕）/ `E_NPUT_COLLISION`
（`apply --dryrun` の conflict = exit 2）。それ以外はツール総称 fallback
`E_NPUT_FAILED`（世代 commit 失敗等）。

> **上は原文の写しで、規範は frontmatter が正**。原文が併記する実装 issue の進捗
> （どのコードが #131 で追加されたか、`--all` が #164 以降どうなるか）は要求ではなく
> 履歴の注記。

**本 item がエラー層の振り分け規則を規定する**（単一 config / `--all` を問わない）。
`--all` 固有の差分（集約エラーをトップへ重ねないこと）は REQ-059eb4d5、変更系ペイロードで
どの失敗が item 起因になるか（entry 失敗・conflict の写像）は REQ-2ea19863 の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」→「niface 準拠の `--json`
出力」のサブ項目「エラーの層と stderr 併存」「エラーコード」。

決定の実体は ADR-0043 §6。
