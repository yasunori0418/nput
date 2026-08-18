---
id: "DSG-16373ec2-3496-4b12-b3b1-ef74e0435b58"
type: design
name: "nput 自身の flake outputs は packages / templates / 各モジュール / flakeModules / lib の 5 系統で構成する"
satisfies:
  - "REQ-14f0aec9-abae-4621-82f3-40536a1ad904"
  - "REQ-6be1cbf1-6c6e-498b-8acb-7f4b80037169"
  - "REQ-c50df875-2cb0-4e72-8a21-858359a11cae"
  - "REQ-637599dc-a1ec-4af5-9e97-e882c7df56d0"
---
# DSG-16373ec2: nput 自身の flake outputs は packages / templates / 各モジュール / flakeModules / lib の 5 系統で構成する

## 設計

```nix
outputs = { ... }: {
  # nput CLI（一次 UX・配置エンジンを import）
  packages.<system>.nput = ...;   # buildGoModule（cmd/nput + internal）

  # 環境セットアップ（nput init = nix flake init -t のラッパー）
  templates.standalone = { path = ./templates/standalone; description = "..."; };
  templates.project    = { path = ./templates/project;    description = "..."; };

  # モジュール統合
  homeManagerModules.default = ./modules/home-manager.nix;
  nixosModules.default  = ./modules/nixos.nix;
  darwinModules.default = ./modules/nix-darwin.nix;

  # flake-parts module（perSystem.nput を flake.nput.<system> へ transpose）
  flakeModules.default = ./modules/flake-parts.nix;

  # 関数呼び出し（モジュールシステム不使用）
  lib = import ./lib;
};
```

5 系統は、nput が提供するものの**入手経路の違い**で分けている。要求は「何を提供するか」を
定めるが、それを flake のどの attr 名で公開するかは設計側の判断になる。

- **`packages.<system>.nput`**: REQ-14f0aec9 が求める PATH 常駐の CLI の配布口。
  `buildGoModule` で `cmd/nput` + `internal` をまとめてビルドする形にすることで、
  REQ-637599dc が求める vendorHash による依存の固定が output の作り方として表れる
  （Go を nixpkgs の go に pin することと `toolchain` ディレクティブ不使用は
  同 REQ の規範で、output の形では表れない）。project mode で devShell へ同梱する
  ときもこの attr を参照する
- **`templates.<name>`**: REQ-6be1cbf1 の `nput init` が `nix flake init -t` の
  透明なラッパーである以上、テンプレート実体は nix 標準の `templates` output に
  置く必要がある。`standalone` / `project` の 2 本を置くことも REQ-6be1cbf1 が定める
  （各テンプレートの中身は REQ-196ddabf の担当で、本 item は attr 名とパスの公開のみ）
- **`homeManagerModules` / `nixosModules` / `darwinModules`**: 各モジュールシステムの
  慣例 attr 名に合わせる。中身は `modules/` の各ファイルを指すだけ
  （NixOS / nix-darwin をスタブで公開する判断は DSG-0e186e89）
- **`flakeModules.default`**: REQ-c50df875 が求める flake-parts 経路の提供口。
  consumer が import すると `perSystem.nput.<name>` を書けるようになる
- **`lib`**: モジュールシステムを介さない関数呼び出し経路。`import ./lib` を
  そのまま公開し、公開 API 面は DSG-e4d5db6b の `lib/default.nix` が決める

**ユーザー側の entrypoint が公開する `nput.<system>.<name>` は、この 5 系統とは別**で、
nput 自身の outputs ではなく consumer 側の output である（規範は REQ-496b1a07）。

## 出典

`docs/design.md`「flake.nix outputs 設計」冒頭のコードブロック。
