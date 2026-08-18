---
id: "REQ-a0bdf6db-6c0c-476c-916a-61ee4e4510d9"
type: requirement
name: "devShell は shellHook から engine を起動する配線で、シェル入室のたびに project mode で配置する"
derives_from:
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The devShell SHALL kick the engine from the `shellHook` of `devShells.<name>`, of the
  same form as the home-manager module. The timing of activation it supplies SHALL be
  entry into the shell: it SHALL be kicked the moment a shell is entered by `nix develop`
  or by direnv (`use flake`), and SHALL place within the project on the git toplevel as
  root. Because a `shellHook` runs at high frequency, the generation-skip short circuit of
  project mode, which makes an unchanged application a no-op, SHALL be a premise of this
  wiring. That it holds no placement logic of its own and supplies only the root and the
  timing of activation is stated by REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9, and that it pins `projectRoot` by
  REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10; neither is restated here.
specification_ja: |
  devShell は `devShells.<name>` の `shellHook` から engine を起動しなければならない
  （HM モジュールと同型）。供給する activation タイミングはシェル入室でなければならず、
  `nix develop` / direnv（`use flake`）でシェルに入った瞬間にキックされなければならず、
  git toplevel を root にプロジェクト内へ配置しなければならない。`shellHook` は高頻度で
  走るため、変更なしなら no-op になる project mode の世代スキップ短絡をこの配線の前提と
  しなければならない。配置ロジックを持たず root と
  activation タイミングだけを供給することは REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9、`projectRoot` を pin することは
  REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10 の担当で、いずれも本 item では規定しない。
---
# REQ-a0bdf6db-6c0c-476c-916a-61ee4e4510d9: devShell は shellHook から engine を起動する配線で、シェル入室のたびに project mode で配置する

## 仕様

`devShells.<name>` の `shellHook` から engine を起動する配線。HM モジュールと同型で、配置
ロジックは持たず root（project mode の git toplevel）と activation タイミング（シェル入室）を
供給するだけ。root は `projectRoot` を pin。

```nix
devShells.default = pkgs.mkShell {
  shellHook = ''
    nput apply skills --no-wait
  '';
};
```

- `nix develop` / direnv（`use flake`）でシェルに入った瞬間にキックされ、git toplevel を root に
  プロジェクト内へ配置する
- `shellHook` は高頻度で走るため、project mode の**世代スキップ短絡**（変更なしなら no-op）が
  前提

> **上は原文の写しで、規範は frontmatter が正**。世代スキップ短絡そのもの（derivation 同一なら
> 世代を積まず lstat ドリフト修復だけ行う）は REQ-46fccb80-4bae-4d37-bc19-dded88e9a9c0、project mode の root 解決は
> REQ-9cb26ffd-071e-4c68-a6fc-faac6373b75e、モジュールと devShell が一律配線であることは REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9、`--no-wait` の
> try-lock 挙動は REQ-1c1526b1-59e3-4264-bb7c-65a10a4aa461 の担当。上のコードブロックの `nput apply skills --no-wait` は
> 使用例であり、devShell が `--all --project-root` を使うべきであることは REQ-d95b814f-aa7a-470e-9320-c14f9c14da7b、
> devShell への CLI 同梱が project mode の canonical であることは REQ-14f0aec9-abae-4621-82f3-40536a1ad904 が規定する。

## 出典

`docs/spec.md`「モジュール別動作仕様」→「devShell（配線）」節。

決定の実体は ADR-0005「project mode（プロジェクト相対配置）と ephemeral 配置原則」
（devShell からの project mode キックと世代スキップ）と、devShell を配線層として位置づける
ADR-0007。
