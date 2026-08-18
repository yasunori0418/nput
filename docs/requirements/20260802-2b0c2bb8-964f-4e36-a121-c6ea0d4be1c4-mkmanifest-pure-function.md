---
id: "REQ-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4"
type: requirement
name: "mkManifest は配置データを生成する純粋関数である"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  `lib.mkManifest` SHALL be a pure function with the signature
  `mkManifest :: { pkgs, entries, root } -> derivation`. It SHALL NOT have side effects;
  profile swap and filesystem placement SHALL be the engine's runtime responsibility.
  The entrypoint SHALL expose it at `nput.<name>`, which the nput CLI builds with
  `nix build` and hands to the engine. `mkManifest` SHALL also be a unit-test target for
  the Nix evaluation tests (nix-unit / namaka). What the returned derivation contains is
  stated by REQ-60e6b49c and is NOT restated here.
specification_ja: |
  `lib.mkManifest` はシグネチャ `mkManifest :: { pkgs, entries, root } -> derivation` の
  純粋関数でなければならない。副作用（profile swap・FS 配置）を持ってはならず、それらは
  engine の実行時責務でなければならない。entrypoint は `nput.<name>` に公開しなければ
  ならず、nput CLI がこれを `nix build` してエンジンへ渡す。また Nix 評価テスト
  （nix-unit / namaka）の単体対象と
  しなければならない。返り値 derivation が何を含むかは REQ-60e6b49c が規定しており、
  本 item では重ねて規定しない。
---
# REQ-2b0c2bb8: mkManifest は配置データを生成する純粋関数である

## 仕様

`lib.mkManifest` は配置データを生成する**純粋関数**。返り値 derivation が何を含むかは
REQ-60e6b49c が正で、本 item では規定しない。

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
