---
id: "REQ-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4"
type: requirement
name: "mkManifest は配置データを生成する純粋関数である"
specification: |
  `lib.mkManifest` SHALL be a pure function producing placement data (a derivation
  containing `manifest.json` and a symlink farm), with the signature
  `mkManifest :: { pkgs, entries, root } -> derivation`. It MUST NOT have side effects;
  profile swap and filesystem placement SHALL be the engine's runtime responsibility.
  The entrypoint SHALL expose it at `nput.<name>`, which the nput CLI builds with
  `nix build` and hands to the engine.
specification_ja: |
  `lib.mkManifest` は配置データ（`manifest.json` + symlink farm を含む derivation）を
  生成する純粋関数でなければならない。シグネチャは
  `mkManifest :: { pkgs, entries, root } -> derivation`。副作用（profile swap・FS 配置）を
  持ってはならず、それらは engine の実行時責務とする。entrypoint が `nput.<name>` に
  公開し、nput CLI が `nix build` してエンジンへ渡す。
---
# REQ-2b0c2bb8: mkManifest は配置データを生成する純粋関数である

## 仕様

`lib.mkManifest` は配置データ（`manifest.json` + symlink farm を含む derivation）を
生成する**純粋関数**。

```
mkManifest :: { pkgs, entries, root } -> derivation
```

- entrypoint が `nput.<name>` に公開し、nput CLI が `nix build` してエンジンに渡す。
- Nix 評価テスト（nix-unit / namaka）の単体対象でもある。
- 副作用は持たない。profile swap・FS 配置は engine の実行時責務。

配置単位名（profile を一意特定する `name`）は **entrypoint の `nput.<name>` 属性キー**が
供給する。CLI が選択した `<name>` をエンジンへ渡す。

## 出典

`docs/spec.md`「lib API」→「`lib.mkManifest`」。
