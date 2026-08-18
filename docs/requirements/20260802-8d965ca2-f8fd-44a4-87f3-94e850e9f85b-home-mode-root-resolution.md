---
id: "REQ-8d965ca2-f8fd-44a4-87f3-94e850e9f85b"
type: requirement
name: "home mode の root は層ごとに定まった供給元から解決する"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-d39c1994-f9a5-4860-80ba-f6e584adaf14"
specification: |
  In home mode (`root = homeRoot`) the root SHALL be resolved from the supplier determined
  by the layer. Standalone SHALL resolve it from `$HOME`, the shell environment variable
  at run time. The home-manager module SHALL resolve it from `$HOME` as resolved
  internally by home-manager, the module pinning `homeRoot`. Both SHALL work irrespective
  of the operating system. The NixOS module SHALL resolve it from
  `config.users.users.${cfg.user}.home`, which is `/home/<user>` where
  `isNormalUser = true`. The nix-darwin module SHALL likewise resolve it from
  `config.users.users.${cfg.user}.home`, which has no default value there and SHALL
  therefore require an explicit setting.
specification_ja: |
  home mode（`root = homeRoot`）の root は、層ごとに定まった供給元から解決しなければならない。
  standalone は実行時のシェル環境変数 `$HOME`、home-manager モジュールは HM が内部解決した
  `$HOME`（モジュールが `homeRoot` を pin する）から解決しなければならず、いずれも OS を
  問わず動作しなければならない。NixOS モジュールは `config.users.users.${cfg.user}.home` から
  解決しなければならない（`isNormalUser = true` なら `/home/<user>`）。nix-darwin モジュールも
  同じく `config.users.users.${cfg.user}.home` から解決しなければならないが、そこには
  デフォルト値が無いため
  明示設定を必須としなければならない。
---
# REQ-8d965ca2-f8fd-44a4-87f3-94e850e9f85b: home mode の root は層ごとに定まった供給元から解決する

## 仕様

| 層 | root の解決方法 | 備考 |
|---|---|---|
| standalone | `$HOME`（実行時のシェル環境変数）| OS 問わず動作 |
| home-manager | `$HOME`（HM が内部解決・モジュールが `homeRoot` を pin）| OS 問わず動作 |
| NixOS（将来）| `config.users.users.${cfg.user}.home` | `isNormalUser = true` で `/home/<user>` |
| nix-darwin（将来）| `config.users.users.${cfg.user}.home` | デフォルト値なし。明示設定が必須 |

> **上は原文の写しで、規範は frontmatter が正**。`homeRoot` が marker であり kind を運ぶこと
> （実体パス解決は engine の実行時責務）は REQ-3f541d39-da41-4ef8-858b-707f54cf6a29 / REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66、`--root` による全モード
> 共通の上書きは REQ-61c05e09-0bde-4f74-9a96-03185f9df606、`cfg.user` を持つ `nput.user` オプション自体は REQ-c2654ca5-62c2-4e4b-ad67-ffc5468f429b、
> home mode の profileDir キーは REQ-d5a2e289-40bc-45a9-9d44-21b8dc561b81 の担当。**モジュールが `homeRoot` を pin する
> こと自体は REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10 の担当**で、本 item の規範に現れる「モジュールが `homeRoot` を
> pin する」は、HM の `$HOME` が standalone の `$HOME` と供給経路として異なる（HM が内部
> 解決する）ことを説明する従属節であり、pin そのものを規範化するものではない。表の
> NixOS / nix-darwin 行が持つ「将来」の限定を規範に採らない理由（実装時期は満たすべきことでは
> ない）は REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9 の注記に集約している。

## 出典

`docs/spec.md`「root の解決」→「home mode（`root = homeRoot`）」節の表。

決定の実体は root を 3 マーカーへ昇格し `$HOME` を `homeRoot` としたうえで各層が root を
供給すると定めた ADR-0007「汎用 nput CLI を一次 UX に昇格し、entrypoint 発見＋root 明示
モデルへ移行する」と、各層を配線に徹させ root と activation タイミングだけを供給させる
ADR-0003。
