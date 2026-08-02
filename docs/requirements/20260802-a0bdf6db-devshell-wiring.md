---
id: "REQ-a0bdf6db-6c0c-476c-916a-61ee4e4510d9"
type: requirement
name: "devShell は shellHook から engine を起動する配線で、シェル入室のたびに project mode で配置する"
specification: |
  The devShell SHALL be wiring that kicks the engine from the `shellHook` of
  `devShells.<name>`, of the same form as the home-manager module: it SHALL hold no
  placement logic of its own and SHALL supply only the root — the git toplevel of project
  mode, `projectRoot` being pinned — and the timing of activation, which is entry into
  the shell. It SHALL be kicked the moment a shell is entered by `nix develop` or by
  direnv (`use flake`), and SHALL place within the project on the git toplevel as root.
  Because a `shellHook` runs at high frequency, the generation-skip short circuit of
  project mode, which makes an unchanged application a no-op, SHALL be a premise of this
  wiring.
specification_ja: |
  devShell は `devShells.<name>` の `shellHook` から engine を起動する配線でなければならない。
  HM モジュールと同型で、配置ロジックを持たず、root（project mode の git toplevel・
  `projectRoot` を pin）と activation タイミング（シェル入室）だけを供給する。
  `nix develop` / direnv（`use flake`）でシェルに入った瞬間にキックされ、git toplevel を
  root にプロジェクト内へ配置する。`shellHook` は高頻度で走るため、変更なしなら no-op に
  なる project mode の世代スキップ短絡をこの配線の前提とする。
---
# REQ-a0bdf6db: devShell は shellHook から engine を起動する配線で、シェル入室のたびに project mode で配置する

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
> 世代を積まず lstat ドリフト修復だけ行う）は REQ-46fccb80、project mode の root 解決は
> REQ-9cb26ffd、モジュールと devShell が一律配線であることは REQ-c1b3ca5f、`--no-wait` の
> try-lock 挙動は REQ-1c1526b1 の担当。上のコードブロックの `nput apply skills --no-wait` は
> 使用例であり、devShell が `--all --project-root` を使うべきであることは REQ-d95b814f、
> devShell への CLI 同梱が project mode の canonical であることは REQ-14f0aec9 が規定する。

## 出典

`docs/spec.md`「モジュール別動作仕様」→「devShell（配線）」節。

決定の実体は ADR-0005「project mode（プロジェクト相対配置）と ephemeral 配置原則」
（devShell からの project mode キックと世代スキップ）と、devShell を配線層として位置づける
ADR-0007。
