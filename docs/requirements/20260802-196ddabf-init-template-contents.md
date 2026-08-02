---
id: "REQ-196ddabf-6569-4303-942e-050872972501"
type: requirement
name: "template は動く example を 1 config だけ置きバリエーションはコメントで示す"
derives_from:
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  Each template SHALL contain exactly one working example config, and SHALL present its
  variations (`subpath`, `method = "copy"`, `mkOutOfStoreSymlink`, multiple entries,
  dynamic generation) as comments, so that the starter stays small and the user's effort
  to delete what is unnecessary is minimized. The `standalone` template SHALL provide a
  `flake.nix` with one `homeRoot` config example plus variation comments. The `project`
  template SHALL provide a `flake.nix` with one `projectRoot` config example, a devShell
  (`packages = [ nput.packages.${system}.nput ]`, `shellHook` being a named apply), and a
  comment stating that placed objects are ephemeral and that the output of
  `nput gitignore <name>` should be appended to `.gitignore`; it SHALL also provide a
  `.gitignore` skeleton headed by the comment
  `# nput: regenerate with 'nput gitignore <name>'`. The `shellHook` of the `project`
  template SHALL default to the named `nput apply <name> --no-wait`, that being clearest
  for a one-config example and free of the mixed-mode footgun, and SHALL present
  `nput apply --all --project-root --no-wait` for multiple configs as a comment. An
  `.envrc` (direnv) SHALL NOT be bundled, so as not to add a file unnecessary to
  non-users; it SHALL be mentioned in a comment instead.
specification_ja: |
  各 template は動く example を 1 config だけ置き、バリエーション（`subpath` /
  `method = "copy"` / `mkOutOfStoreSymlink` / 複数 entry / 動的生成）はコメントで示さなければ
  ならない。starter を小さく保ち、ユーザーが不要分を削除する手間を最小化するため。
  `standalone` template は `homeRoot` の 1 config 例 + バリエーションコメントを持つ
  `flake.nix` を提供する。`project` template は `projectRoot` の 1 config 例 + devShell
  （`packages = [ nput.packages.${system}.nput ]`・`shellHook` は名指し apply）+
  「配置物は ephemeral・`.gitignore` へ `nput gitignore <name>` 出力を追記」コメントを持つ
  `flake.nix` と、`# nput: regenerate with 'nput gitignore <name>'` ヘッダコメント付きの
  `.gitignore` 雛形を提供する。`project` template の `shellHook` は名指しの
  `nput apply <name> --no-wait` を既定とする（example が 1 config なので最も明確で混在
  footgun が起きないため）。複数 config 向けの
  `nput apply --all --project-root --no-wait` はコメントで示す。`.envrc`（direnv）は
  同梱してはならず（非利用者に不要ファイルを増やすため）、コメント案内に留める。
---
# REQ-196ddabf: template は動く example を 1 config だけ置きバリエーションはコメントで示す

## 仕様

各 template は**動く example を 1 config だけ**置き、バリエーション（`subpath` /
`method = "copy"` / `mkOutOfStoreSymlink` / 複数 entry / 動的生成）は**コメントで示す**。
starter を小さく保ち、ユーザーが不要分を削除する手間を最小化する。

| template | ファイル | 内容 |
|---|---|---|
| `standalone` | `flake.nix` | `homeRoot` の 1 config 例（`nput.<system>.<name> = mkManifest { root = homeRoot; entries = {...}; }`）+ バリエーションコメント |
| `project` | `flake.nix` | `projectRoot` の 1 config 例 + devShell（`packages = [ nput.packages.${system}.nput ]`・shellHook = 名指し apply）+ 「配置物は ephemeral・`.gitignore` へ `nput gitignore <name>` 出力を追記」コメント |
| `project` | `.gitignore` | 先頭に `# nput: regenerate with 'nput gitignore <name>'` ヘッダコメント付きの雛形 |

- project template の `shellHook` は **`nput apply <name> --no-wait`（名指し）**を既定に
  する。example が 1 config なので最も明確で混在 footgun が起きない。「複数 config なら
  `nput apply --all --project-root --no-wait`」をコメントで示す。
- `.envrc`（direnv）は同梱しない（非利用者に不要ファイルを増やすため。コメント案内に
  留める）。

混在 footgun そのものは REQ-d95b814f の担当。

## 出典

`docs/spec.md`「CLI 仕様」→「`nput init`」→「テンプレートの内容（最小 + 手厚いコメント）」。

決定の実体は ADR-0018「templates の具体内容」。
