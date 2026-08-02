---
id: "DSG-98d7fa5d-c590-441a-b682-0ab7afa14233"
type: design
name: "engine の起動を entrypoint 駆動とビルド済み manifest の 2 クラスに束ね、統合層の差を link-farm の取得方法だけに閉じる"
satisfies:
  - "REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9"
  - "REQ-dec58330-6dad-47f7-8f56-2402764a89c7"
  - "REQ-8085f194-c903-4ecb-abd8-c719fe7b3292"
  - "REQ-a0bdf6db-6c0c-476c-916a-61ee4e4510d9"
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

層ごとの割り当てと、engine の起動を担う配線は次の通り。

| 層 | クラス | 起動方法 |
|---|---|---|
| standalone（CLI）| entrypoint 駆動 | `nput apply <name>` を明示実行 |
| devShell | entrypoint 駆動 | `shellHook` から `nput apply <name>` |
| home-manager | ビルド済み manifest | `home.activation` から `apply --manifest` |
| NixOS（将来）| ビルド済み manifest | `system.activationScripts` から `apply --manifest` |
| nix-darwin（将来）| ビルド済み manifest | `system.activationScripts` から `apply --manifest` |

層ごとの root の供給元は REQ-8d965ca2 / REQ-9cb26ffd / REQ-c2654ca5、ユーザー向け
rollback を公開する層は REQ-05abce3e / REQ-844ee375 が定める。どちらも起動クラスとは
独立に決まるため、本 item の担当ではない。

**2 クラスに束ねる**ことが実現手段になっているのは次の点。

- **REQ-c1b3ca5f が求める「各層は engine をキックするだけの配線」**を、層の数（5）ではなく
  クラスの数（2）で担保できる。層を増やしても新しい engine 経路が増えず、
  既存 2 クラスのどちらかへ割り当てるだけで済む
- **クラスの分かれ目が「entrypoint を持つか」に一致する**。モジュールの `nput.configs` は
  モジュール config 内にあり flake output に現れないため、entrypoint 発見では到達できない。
  そこで REQ-dec58330 の `apply --manifest` と REQ-8085f194 の activation kick が
  必要になる。逆に entrypoint 駆動側（REQ-a0bdf6db の devShell 含む）は
  activation 内で `nix build` / `eval` を行わない制約が無いので発見経路を使える
- **層ごとに違う要素がクラスの分岐に持ち込まれない**。root の供給元は engine から見れば
  `manifest.json` の root kind に還元され、rollback の公開有無は profile の使い方の差に
  留まる。どちらも link-farm の取得方法とは無関係なので、2 クラスの外側で決まる

## 出典

`docs/design.md`「モジュール統合設計」→「各統合層の動作」の統合層テーブルと
engine kick 2 クラスの説明、および「実行タイミング」節。

なお `docs/design.md` の統合層テーブルは home-manager 行に「MVP は profile `<name>` =
`default` 固定の 1 profile・役割分離は不可」と書いているが、本 item はこれを採らない。
ADR-0035 が `nput.configs.<name>` を導入し、REQ-c6891aeb が HM 経由でも役割分離した
独立 profile を取れることを規範化しているため。
