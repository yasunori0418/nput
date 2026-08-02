---
id: "REQ-c9ab91c1-f778-4f87-a2ea-c66d6b3c2575"
type: requirement
name: "祖先 symlink は自己記録 stale のみ配置前除去し、それ以外はエラーで停止する"
specification: |
  Before placing a symlink, the engine SHALL walk every ancestor component of the target
  with `lstat` and SHALL treat an ancestor symlink asymmetrically. An ancestor that is
  self-recorded stale — recorded by the entry's own previous-generation manifest, whose
  on-disk destination matches that record, and which is absent from the next generation —
  SHALL be removed before placement (PreRemove) so that the entries below it are newly
  placed, thereby migrating them into the nest; this removal SHALL be silent by default
  and SHALL be made visible under `-v`. An ancestor that is foreign — not recorded, not
  matching the recorded destination, or having no previous generation — and a
  self-contradictory ancestor that also remains in the next generation SHALL cause the
  command to stop with an error, because nesting entries below such an ancestor would
  pollute the store or create dangling links.
specification_ja: |
  symlink の配置に先立ち、engine は target の各祖先 component を `lstat` で walk し、
  祖先の symlink を非対称に扱わなければならない。自己記録 stale の祖先（当該 entry 自身の
  前世代 manifest が記録し、on-disk が記録 dest と一致し、次世代には無い）は配置前に除去
  （PreRemove）して配下の子を新規配置し、ネストへ移行させる。この除去は既定で silent とし、
  `-v` で可視化する。foreign な祖先（記録なし / 記録 dest と不一致 / 前世代なし）と、次世代
  にも祖先が残る自己矛盾はエラーで停止しなければならない（配下にネストすると store 汚染 /
  dangling を招くため）。
---
# REQ-c9ab91c1: 祖先 symlink は自己記録 stale のみ配置前除去し、それ以外はエラーで停止する

## 仕様

```
target の各祖先 component を lstat で walk。symlink の祖先があれば非対称に扱う:
  - 自己記録 stale（自身の前世代 manifest が記録・on-disk が記録 dest と一致・次世代に無い）
    → その祖先を配置前に除去（PreRemove）し、配下子を新規配置してネスト移行する（silent・`-v` で可視）
  - foreign（記録なし / 記録 dest と不一致 / 前世代なし）または次世代にも祖先が残る自己矛盾
    → エラーで停止（配下にネストできない。store 汚染 / dangling を防ぐ）
```

> **上は原文の写しで、規範は frontmatter が正**。原文が参照する次の規範は本 item の
> 担当ではない。
>
> - 既定 silent と `-v` による可視化の出力規律そのもの（→ ADR-0031）→ REQ-8ef34101 /
>   REQ-0a123b89
> - `--backup` 有効時にこのエラー停止がどう変わるか → 祖先 symlink conflict は
>   `--backup` の対象外（REQ-5dd5a4e9・構造問題であり退避では解消しないため）

実 dir target の migration は REQ-7cee95dd、method 変更 symlink→copy の migration は
REQ-2b48620a の担当。

## 出典

`docs/spec.md`「配置動作仕様」→「symlink モード」の手順 0。

決定の実体は ADR-0046「自己記録の祖先 symlink 配下ネストを許可する — 前世代 manifest
判定の祖先拡張 + 配置前除去（PreRemove）」で、foreign 祖先のエラー停止は ADR-0015 §4 が
定めている。
