---
id: "REQ-57137302-de29-4f71-a565-034cd5de080b"
type: requirement
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
name: "item id は identity の JCS を SHA-256 した小文字 hex とする"
specification: |
  The item id SHALL be derived as `id = lowercase-hex(sha256(JCS(identity)))`, where the
  identity is `kind="entry"` and `key={target}`. The key SHALL contain only the
  root-relative target and SHALL NOT include the config name.
specification_ja: |
  item id は `id = lowercase-hex(sha256(JCS(identity)))` で導出しなければならない。
  identity は `kind="entry"`, `key={target}` とする。key は root 相対 target のみを含み、
  config 名を含めてはならない。
---
# REQ-57137302: item id は identity の JCS を SHA-256 した小文字 hex とする

## 仕様

**item id 導出 seam**: `id = lowercase-hex(sha256(JCS(identity)))`・identity は
`kind="entry"`, `key={target}`（root 相対 target のみ・config 名は含めない）。

> **上は原文の写しで、規範は frontmatter が正**。原文が併記する実装 issue の進捗
> （#130 で導出関数を配線し id-vectors との整合をテストで固定した／変更系への実配線は
> #131／読み取り系は #132）は要求ではなく履歴の注記。id-vectors との突合そのものは
> REQ-2381d93a（適合検証）の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「出力ストリームと終了コード」→「niface 準拠の `--json`
出力」のサブ項目「item id 導出 seam」。

決定の実体は ADR-0043 §3。
