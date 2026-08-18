---
id: "REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10"
type: requirement
name: "全モジュールは共通オプションの同一集合を公開し、entries は configs 経由・root はモジュールが pin する"
derives_from:
  - "UC-d39c1994-f9a5-4860-80ba-f6e584adaf14"
specification: |
  Every module (home-manager / NixOS / nix-darwin) SHALL expose the same set of common
  options: `nput.enable :: bool` defaulting to false, `nput.configs :: attrsOf (submodule
  { entries :: attrsOf entry; })` defaulting to `{}` and with `entries` likewise
  defaulting to `{}`, `nput.backup.enable :: bool` defaulting to false, and
  `nput.backup.suffix :: str` defaulting to `"nput-backup"`. The bare `nput.entries` SHALL
  be kept as a sugar renaming onto `configs.default.entries`, non-breaking and deprecated.
  These SHALL be defined jointly for every module, so that the set does not diverge per
  module. What the `<name>` dimension of `configs` means for the granularity of profiles
  is stated by REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a and is not restated here. A module SHALL pin the root by its
  own nature — home-manager to `homeRoot`, a devShell to `projectRoot` — and SHALL NOT
  expose a `root` option, so that a user of a module never restates it. This is a
  departure from `mkManifest` and the CLI entrypoint, where `root` is required to be
  stated explicitly (REQ-4ec3accc-8bb6-461f-9024-dcf0027849e4).
specification_ja: |
  全モジュール（home-manager / NixOS / nix-darwin）は共通オプションとして同一の集合を
  公開しなければならない: `nput.enable :: bool`（デフォルト false）・
  `nput.configs :: attrsOf (submodule { entries :: attrsOf entry; })`（デフォルト `{}`・
  `entries` のデフォルトも `{}`）・`nput.backup.enable :: bool`（デフォルト false）・
  `nput.backup.suffix :: str`（デフォルト `"nput-backup"`）。素の `nput.entries` は
  `configs.default.entries` への rename 糖衣として非破壊に残さなければならず、deprecated と
  しなければならない。
  これらは全モジュール共通に定義し、モジュールごとに集合が分岐しないようにしなければ
  ならない。`configs` の `<name>` 次元が profile の粒度にとって何を意味するかは
  REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a の担当で、本 item では規定しない。モジュールは自分の性質で root を pin
  しなければならず（home-manager は `homeRoot`・devShell は `projectRoot`）、`root`
  オプションを公開してはならない。モジュール利用者が root を再指定しないようにするため
  である。これは `root` を明示必須とする `mkManifest` / CLI entrypoint の層
  （REQ-4ec3accc-8bb6-461f-9024-dcf0027849e4）との差分である。
---
# REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10: 全モジュールは共通オプションの同一集合を公開し、entries は configs 経由・root はモジュールが pin する

## 仕様

```
nput.enable  :: bool          # デフォルト: false
nput.entries :: attrsOf entry # デフォルト: {}（属性キー = target・→ ADR-0014）
                              # ← ADR-0035 により nput.configs.<name>.entries の
                              #   deprecated 糖衣へ降格済み（規範は frontmatter・下の注記を参照）
nput.backup.enable :: bool    # デフォルト: false（→ ADR-0045）
nput.backup.suffix :: str     # デフォルト: "nput-backup"（→ ADR-0045）
```

モジュールは自分の性質で root を pin する（HM → `homeRoot` / devShell → `projectRoot`）ため、
モジュール利用者は `root` を再指定しない。

> **上は原文の写しで、規範は frontmatter が正**。`entries` の属性キーが target であることと
> entry submodule のフィールド定義は REQ-cb77ea05-bab8-4ccf-b09e-d23d8f71cdc7 / REQ-a33a11e3-830d-4142-88ed-4c1fc35e7f74、`root` が
> `mkManifest` / CLI 層で明示必須であることは REQ-4ec3accc-8bb6-461f-9024-dcf0027849e4、`--backup` の退避契約そのものは
> REQ-5dd5a4e9-6162-4fa5-b295-66844f5a4f3b の担当。`nput.backup` が manifest ではなく engine 起動の配線に効くことは
> REQ-e1e1114b-ba07-4d57-8e04-6e30e39a5da3 の担当。
>
> **写しの `nput.entries` を canonical としない理由**: **ADR-0035 §1〜§2 は
> `nput.configs.<name>.entries` を canonical とし、素の `nput.entries` を
> `configs.default.entries` への deprecated 糖衣へ降格することを決定済み**で、`docs/spec.md`
> の当該表はこれに未追従。ADR-0035 §1 は `configs` の定義先を `modules/common.nix`
> （全モジュール共通）と定めているため、これは HM 固有ではなく共通オプション集合の一部で
> あり、本 item の規範に含めた（REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66 / REQ-16faf428-77f3-492f-b858-222c5274cbf7 で ADR-0036 由来の未追従を
> 扱ったのと同じ扱い）。`configs` の `<name>` 次元が profile の粒度にとって何を意味するか
> （役割分離が可能になること・`<name>` 次元の導入が非破壊であること）は REQ-c6891aeb-13c0-4ae7-9ad1-5c343735266a、
> profile dir をどうキーするかは REQ-d5a2e289-40bc-45a9-9d44-21b8dc561b81 の担当。`docs/spec.md` の追従は本 item の
> 担当範囲外。
>
> **`nput.configs` のデフォルトを `{}` とする根拠**: ADR-0035 §1 は型
> （`attrsOf (submodule { entries = entriesType; })`）のみを定めデフォルトに触れていないが、
> `attrsOf` の既定が空 attrset であることは Nix module system の帰結であり、原文の表が他 3
> オプションのデフォルトを明示しているのに `configs` だけ欠けると非対称になるため規範に含めた。
> 原文が `nput.entries` のデフォルトとする `{}` は、糖衣を通じて `configs.default.entries` の
> デフォルトへ落ちる。

## 出典

`docs/spec.md`「モジュールオプション仕様」→「共通オプション（全モジュール）」節の
オプション表と、その直後の root pin の段落。

決定の実体は root 明示必須とモジュール側 pin を定めた ADR-0007「汎用 nput CLI を一次 UX に
昇格し、entrypoint 発見＋root 明示モデルへ移行する」と、配置ロジックを engine が所有し
モジュールは配線に徹すると定めた ADR-0003。`nput.backup` の追加は ADR-0045、`nput.configs`
の新設と `nput.entries` の糖衣化は ADR-0035。
