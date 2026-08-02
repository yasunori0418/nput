---
id: "REQ-a33a11e3-830d-4142-88ed-4c1fc35e7f74"
type: requirement
name: "entry submodule のフィールドは src / subpath / target / method の 4 つとする"
specification: |
  The value of each `entries` attribute SHALL be an entry submodule holding exactly four
  fields: `src` (`path | set | marker`, required), `subpath` (`string`, optional,
  defaulting to `"."`), `target` (`string`, optional, defaulting to the attribute key),
  and `method` (`"symlink" | "copy"`, optional, defaulting to `"symlink"`). `target`
  SHALL be a path relative to the `root` selected explicitly in `mkManifest`. The fields
  SHALL be defined as an entry submodule in `lib/types.nix` and SHALL be validated and
  defaulted by the `evalModules` run inside `mkManifest`.
specification_ja: |
  `entries` の各属性値は entry submodule とし、フィールドは `src`
  （`path | set | marker`・必須）・`subpath`（`string`・省略可・デフォルト `"."`）・
  `target`（`string`・省略可・デフォルトは属性キー）・`method`
  （`"symlink" | "copy"`・省略可・デフォルト `"symlink"`）の 4 つでなければならない。
  `target` は `mkManifest` で明示選択した `root` からの相対パスとする。これらの
  フィールドは `lib/types.nix` の entry submodule として定義し、`mkManifest` 内の
  `evalModules` が検査・デフォルト適用する。
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

`target` は配置先パスで、`mkManifest` の `root` で明示選択した基準からの相対パスで
指定する。省略可なので、属性キーをそのまま target とするのが canonical だが、キーを
論理ラベルにして `target` を明示上書きすることもできる（home-manager の `home.file` と
同型）。

各フィールドは `lib/types.nix` の entry submodule（`lib.types`）として定義され、
`mkManifest` の `evalModules` が検査・デフォルト適用する。

各フィールドの値域・意味は個別 item が持つ（`src` は REQ-99ca5381、`subpath` は
REQ-27b75fe6、`target` の識別子性は REQ-cb77ea05、`method` の動作は REQ-77689c68）。
本 item はフィールドの構成そのもの（4 つで過不足なく、どれが必須でどれが省略可か）を
規定する。

## 出典

`docs/spec.md`「entries スキーマ仕様」→「フィールド定義（entry submodule）」、
および `#### target` の「root からの相対パス」。
