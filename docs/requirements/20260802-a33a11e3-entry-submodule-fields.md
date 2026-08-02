---
id: "REQ-a33a11e3-830d-4142-88ed-4c1fc35e7f74"
type: requirement
name: "entry submodule のフィールドは src / subpath / target / method の 4 つとする"
specification: |
  The value of each `entries` attribute SHALL be an entry submodule holding exactly four
  fields: `src` (`path | set | marker`, required), `subpath` (`string`, optional),
  `target` (`string`, optional), and `method` (`"symlink" | "copy"`, optional). `target`
  SHALL be a path relative to the `root` selected explicitly in `mkManifest`. The
  concrete default of each optional field, and which component validates and applies
  those defaults, are stated by REQ-b232ec98, REQ-cb77ea05 and REQ-d1b5b3f5 and are NOT
  restated here.
specification_ja: |
  `entries` の各属性値は entry submodule とし、フィールドは `src`
  （`path | set | marker`・必須）・`subpath`（`string`・省略可）・`target`
  （`string`・省略可）・`method`（`"symlink" | "copy"`・省略可）の 4 つでなければ
  ならない。`target` は `mkManifest` で明示選択した `root` からの相対パスとする。
  各省略可フィールドの具体的なデフォルト値と、それを
  検査・適用する主体は REQ-b232ec98 / REQ-cb77ea05 / REQ-d1b5b3f5 が規定しており、
  本 item では重ねて規定しない。
---
# REQ-a33a11e3: entry submodule のフィールドは src / subpath / target / method の 4 つとする

## 仕様

`entries` の各値は entry submodule で、フィールドは次の 4 つ。

```
entry :: {
  src    : path | set | marker # 必須（type/marker によって挙動が変わる）
  subpath: string              # 省略可、デフォルト: "."（省略 = リポジトリ全体）
  target : string              # 省略可、デフォルト: 属性キー
  method : "symlink"
         | "copy"              # 省略可、デフォルト: "symlink"
}
```

上のコードブロックは原文の写しで、規範は frontmatter が正。各省略可フィールドの
デフォルト値そのもの（コメント欄の記述）は規範に含めない（理由は下の注記）。

`target` は配置先パスで、`mkManifest` の `root` で明示選択した基準からの相対パスで
指定する。省略可なので、属性キーをそのまま target とするのが canonical だが、キーを
論理ラベルにして `target` を明示上書きすることもできる（home-manager の `home.file` と
同型）。

各フィールドは `lib/types.nix` の entry submodule（`lib.types`）として定義され、
`mkManifest` の `evalModules` が検査・デフォルト適用する。この検査主体の規範は
REQ-d1b5b3f5（`mkManifest` が単一の検査ゲート）と REQ-b232ec98（`normalizeManifest` が
検査・デフォルト適用を担う）が持つ。

各フィールドの値域・意味も個別 item が持つ（`src` は REQ-99ca5381、`subpath` は
REQ-27b75fe6、`target` の識別子性とデフォルトは REQ-cb77ea05、`method` の動作は
REQ-77689c68）。本 item はフィールドの構成そのもの（4 つで過不足なく、どれが必須で
どれが省略可か）を規定する。5 つ目のフィールドが評価時エラーになること（strict）は
REQ-3e446ad9 の規範。

> **デフォルト値を規範に含めない理由**: `subpath` → `"."` / `method` → `"symlink"` /
> `target` → 属性キーの 3 つは、いずれも既存 item が規範として持つ（REQ-b232ec98 が
> `normalizeManifest` のデフォルト適用として 3 つとも、REQ-cb77ea05 が `target` の
> 分を）。本 item で重ねて規定すると規範の所在が二箇所になるため、frontmatter からは
> 落として委譲した。

## 出典

`docs/spec.md`「entries スキーマ仕様」→「フィールド定義（entry submodule）」、
および `#### target` の「root からの相対パス」。
