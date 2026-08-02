---
id: "REQ-cb77ea05-bab8-4ccf-b09e-d23d8f71cdc7"
type: requirement
name: "entry の識別子は属性キー = target とし一意性は Nix が担保する"
specification: |
  `entries` SHALL be an attrset whose attribute key is the target path, which SHALL serve
  as the entry's identifier. Uniqueness of identifiers SHALL be guaranteed natively by
  the fact that Nix attrsets cannot hold duplicate keys, and a `lib.throwIf` detecting a
  duplicate `name` MUST NOT exist. When the `target` field is omitted, the attribute key
  SHALL be used as its default.
specification_ja: |
  `entries` は attrset とし、属性キーが target パス（= entry の識別子）でなければ
  ならない。識別子の一意性は Nix attrset のキー重複不可によって native に担保し、
  重複 name を検出する `lib.throwIf` を持ってはならない。`target` フィールドを省略した
  場合は属性キーを既定値とする。
---
# REQ-cb77ea05: entry の識別子は属性キー = target とし一意性は Nix が担保する

## 仕様

`entries` は配置定義の attrset で、**属性キー = target パス**が識別子。

- 識別子の一意性は Nix attrset のキー重複不可で **native に担保**する。重複 name の
  `throwIf` は持たない。
- `target` フィールドは省略でき、省略時は属性キーが既定値になる。

```nix
entries = {
  # 属性キー = target。target フィールドは省略（キーから既定）
  ".local/share/nvim/site/pack/foo/start/foo" = { src = inputs.vim-foo; };
  ".zsh/plugins/autosuggestions" = { src = inputs.zsh-sugg; };
};
```

## 出典

`docs/spec.md`「lib API」→「`lib.mkManifest`」引数表・「入力検査」節・使用例。
