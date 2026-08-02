---
id: "REQ-053cfed2-265a-4997-a27b-97b0bed10d8a"
type: requirement
name: "target に通常ファイル・ディレクトリが在れば上書きせずエラーで停止する"
specification: |
  When a regular file or a directory exists at the target of a symlink entry, the engine
  SHALL stop with an error and SHALL NOT overwrite it. A real directory SHALL be the sole
  exception, being removed before placement and then placed when it satisfies the
  conditions for real-directory migration.
specification_ja: |
  symlink entry の target に通常ファイルまたはディレクトリが存在する場合、engine は
  エラーで停止しなければならず、上書きしてはならない。実 dir のみが例外であり、実 dir
  migration の条件を満たす場合に限り配置前に除去して配置する。
---
# REQ-053cfed2: target に通常ファイル・ディレクトリが在れば上書きせずエラーで停止する

## 仕様

target に通常ファイルまたはディレクトリが存在する場合はエラーで停止（**上書きしない**）。
ただし実 dir は条件を満たせば例外的に PreRemove で除去して配置する。

> **上は原文の写しで、規範は frontmatter が正**。実 dir が例外になる条件そのもの
> （全 leaf 判定と部分除去の禁止）は REQ-7cee95dd の担当。`--backup` 有効時にこの
> エラー停止が退避 + 配置へ変わることは REQ-5dd5a4e9 / REQ-9b0046e0 の担当。

## 出典

`docs/spec.md`「配置動作仕様」→「symlink モード」の箇条書き第 2 項。

決定の実体は ADR-0015「実装前レビューで surfaced した残セマンティクス」の祖先 symlink /
実体衝突の扱いで、実 dir の例外化は ADR-0047 が追加した。
