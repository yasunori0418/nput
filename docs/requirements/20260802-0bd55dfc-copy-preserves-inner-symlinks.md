---
id: "REQ-0bd55dfc-b28c-4ad2-ad50-bb1f76b2246c"
type: requirement
name: "copy は src ツリー内の symlink を deref せず symlink のまま複製する"
derives_from:
  - "UC-403fbe32-b146-401b-8b53-fe67c1e169c5"
specification: |
  A symlink inside the source tree SHALL be reproduced as a symlink and SHALL NOT be
  dereferenced, so as to avoid cycles and size inflation. A relative symlink SHALL be kept
  as it stands.
specification_ja: |
  src ツリー内の symlink は symlink のまま複製しなければならず、deref してはならない
  （循環・サイズ膨張を避けるため）。相対 symlink はそのまま保つ。
---
# REQ-0bd55dfc: copy は src ツリー内の symlink を deref せず symlink のまま複製する

## 仕様

**src ツリー内の symlink は symlink のまま複製する**（deref しない。循環・サイズ膨張回避）。
相対 symlink はそのまま保つ。

> **上は原文の写しで、規範は frontmatter が正**。原文はこれに続けて「ただし store 内への
> 絶対 symlink を複製すると store 依存（read-only / GC 後 dangling）が残る点に注意」と
> 注記するが、これは規範ではなく利用上の帰結の注意喚起なので規範文には含めない。

この複製規則は通常の place-once コピーと `apply --recopy` の再コピーの双方に適用される。
place-once そのものは REQ-d2277c7a、recopy そのものは REQ-7cc32a2b / REQ-b4e4b65d の担当。

## 出典

`docs/spec.md`「配置動作仕様」→「copy モード（place-once・ユーザー管理）」節の箇条書き第 5 項
（および「recopy」節の「symlink 複製は通常 copy と同じ」）。

決定の実体は ADR-0016「実装前レビュー第 2 巡で surfaced した残セマンティクス」の
「copy 内 symlink」。
