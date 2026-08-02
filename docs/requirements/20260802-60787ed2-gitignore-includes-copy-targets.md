---
id: "REQ-60787ed2-4176-4bdd-800f-1600c0315551"
type: requirement
name: "gitignore は method を区別せず copy target も含めて全 target を列挙する"
specification: |
  `nput gitignore` SHALL enumerate every target without distinguishing `method`,
  including copy targets. A copy target in project mode SHALL also be treated as
  ephemeral: it is re-materialized place-once in each clone, and edits to it are
  clone-local and disposable. Making a copy committed (vendoring) SHALL be outside the
  responsibility of nput, and the ephemeral principle of project mode SHALL NOT be broken.
specification_ja: |
  `nput gitignore` は `method` を区別せず、copy target も含めて全 target を列挙しなければ
  ならない。project mode の copy target も ephemeral 扱いとし、各 clone で place-once で
  再マテリアライズされ、その編集は clone local / 使い捨てとする。copy を committed
  （vendoring）にするのは nput の責務外であり、project mode の ephemeral 原則を
  崩してはならない。
---
# REQ-60787ed2: gitignore は method を区別せず copy target も含めて全 target を列挙する

## 仕様

`gitignore` は **`method` を区別せず全 target を列挙**する（copy target も含む）。
project mode の copy target も ephemeral 扱いで、各 clone で place-once で再マテリアライズ
され**編集は clone local / 使い捨て**（`git clean` で消える）。copy を committed
（vendoring）にするのは nput の責務外（手動コミット）で、project mode の ephemeral 原則は
崩さない。

## 出典

`docs/spec.md`「CLI 仕様」→「サブコマンド体系」の `gitignore` の `method` 非区別の
箇条書き。

決定の実体は ADR-0019「copy の farm/gitignore 扱い」。
