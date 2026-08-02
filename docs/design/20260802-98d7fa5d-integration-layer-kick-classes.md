---
id: "DSG-98d7fa5d-c590-441a-b682-0ab7afa14233"
type: design
name: "engine の起動を entrypoint 駆動とビルド済み manifest の 2 クラスに束ね、統合層の差を link-farm の取得方法だけに閉じる"
satisfies:
  - "REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9"
  - "REQ-dec58330-6dad-47f7-8f56-2402764a89c7"
  - "REQ-8085f194-c903-4ecb-abd8-c719fe7b3292"
  - "REQ-a0bdf6db-6c0c-476c-916a-61ee4e4510d9"
  - "REQ-844ee375-919f-4341-81e1-a5f89fd32840"
  - "REQ-8d965ca2-f8fd-44a4-87f3-94e850e9f85b"
  - "REQ-c2654ca5-62c2-4e4b-ad67-ffc5468f429b"
---
# DSG-98d7fa5d: engine の起動を entrypoint 駆動とビルド済み manifest の 2 クラスに束ね、統合層の差を link-farm の取得方法だけに閉じる

## 設計

統合層は 5 つ（standalone CLI / home-manager / devShell / NixOS / nix-darwin）あるが、
**engine の起動方法は 2 クラスに束ねられる**。配置から世代コミットまでは両クラスで
同一の engine 経路を通り、異なるのは link-farm をどう手に入れるかだけになる。

| クラス | 該当層 | link-farm の取得 |
|---|---|---|
| **entrypoint 駆動** | standalone CLI / devShell | ユーザーが `nput.<name>` を entrypoint に公開し、CLI が発見 → `nix eval`（rootKind 先取り）→ `nix build` |
| **ビルド済み manifest** | home-manager（将来の NixOS / nix-darwin）| モジュール評価時に `mkManifest` でビルドした link-farm を、activation から `nput apply --manifest <link-farm>` へ渡す |

層ごとの内訳は次の通り。

| 層 | 起動方法 | root の解決 | ユーザー向け rollback |
|---|---|---|---|
| standalone（CLI）| `nput apply <name>` を明示実行 | マーカー（`homeRoot` / `projectRoot` 等）| `nput rollback <name>`（home mode 限定）|
| home-manager | `home.activation` から `apply --manifest` | `$HOME`（`homeRoot` を pin）| host（`home-manager --rollback`）|
| devShell | `shellHook` から `nput apply <name>` | project mode: git toplevel（`--root` 可）| なし（ephemeral 配置）|
| NixOS（将来）| `system.activationScripts` から `apply --manifest` | `config.users.users.<user>.home` | host（`nixos-rebuild` 世代）|
| nix-darwin（将来）| `system.activationScripts` から `apply --manifest` | `config.users.users.<user>.home` | host 世代 |

**2 クラスに束ねる**ことが実現手段になっているのは次の点。

- **REQ-c1b3ca5f が求める「各層は engine をキックするだけの配線」**を、層の数（5）ではなく
  クラスの数（2）で担保できる。層を増やしても新しい engine 経路が増えず、
  既存 2 クラスのどちらかへ割り当てるだけで済む
- **クラスの分かれ目が「entrypoint を持つか」に一致する**。モジュールの `nput.configs` は
  モジュール config 内にあり flake output に現れないため、entrypoint 発見では到達できない。
  そこで REQ-dec58330 の `apply --manifest` と REQ-8085f194 の activation kick が
  必要になる。逆に entrypoint 駆動側（REQ-a0bdf6db の devShell 含む）は
  activation 内で `nix build` / `eval` を行わない制約が無いので発見経路を使える
- **rollback の一本化（REQ-844ee375）がクラスと直交する**。nput profile は全層で持つが、
  ユーザー向け rollback を公開するのは standalone（home mode）だけで、モジュール層では
  host に一本化する。この差は起動クラスではなく層の性質で決まる

root の供給元が層ごとに違う点（REQ-8d965ca2 / REQ-c2654ca5）も、engine から見れば
`manifest.json` の root kind に還元されるため、engine 経路の分岐にはならない。

## 出典

`docs/design.md`「モジュール統合設計」→「各統合層の動作」の統合層テーブル（L295-301）と
engine kick 2 クラスの説明（L303-306）、および「実行タイミング」節（L321-329）。

なお `docs/design.md` の統合層テーブルは home-manager 行に「MVP は profile `<name>` =
`default` 固定の 1 profile・役割分離は不可」と書いているが、本 item はこれを採らない。
ADR-0035 が `nput.configs.<name>` を導入し、REQ-c6891aeb が HM 経由でも役割分離した
独立 profile を取れることを規範化しているため。
