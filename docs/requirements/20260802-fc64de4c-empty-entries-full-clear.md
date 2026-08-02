---
id: "REQ-fc64de4c-c82b-419c-8706-07d8d97daa37"
type: requirement
name: "空の entries は正当な全クリアとして扱い、エラーにも警告にもしない"
specification: |
  `entries = {}` SHALL be treated as a legitimate full clear, and SHALL NOT be an error or
  a warning. Every nput symlink of the previous generation SHALL be removed by conservative
  stale removal, and the new generation SHALL be empty. What conservative stale removal
  removes is stated by REQ-16aef46b and SHALL NOT be restated here.
specification_ja: |
  `entries = {}` は正当な全クリアとして扱わなければならず、エラーにも警告にもして
  はならない。前世代の全 nput symlink を保守的 stale 除去で除去し、新世代は空とする。
  保守的 stale 除去が何を除去するかは REQ-16aef46b の規範であり、ここでは再掲しない。
---
# REQ-fc64de4c: 空の entries は正当な全クリアとして扱い、エラーにも警告にもしない

## 仕様

| 条件 | 動作 |
|---|---|
| `entries = {}`（空 manifest）| 正当な全クリア。前世代の全 nput symlink を保守的 stale 除去し新世代は空（警告なし）|

> **上は原文の写しで、規範は frontmatter が正**。保守的 stale 除去の不変条件そのもの
> （何を消し何を消さないか）は REQ-16aef46b、copy が stale 除去で消えず orphan 警告に
> なることも同 item の担当。

## 出典

`docs/spec.md`「エラー仕様」節の表の `entries = {}` 行。

決定の実体は ADR-0019「パス安全性・copy の farm/gitignore 扱い・空 entries の挙動を
確定する」で、空 entries をエラーにせず全クリアとして受理することを定めている。
