---
id: "DSG-aeb5e219-4784-4950-845c-35f9bab9179c"
type: design
name: "modules は common.nix に共通オプションを集約し、統合層ごとに 1 ファイルへ分ける"
satisfies:
  - "REQ-fc1c7ce6-dc9d-4dd3-98f5-7877d9f99d10"
  - "REQ-c2654ca5-62c2-4e4b-ad67-ffc5468f429b"
  - "REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9"
  - "REQ-8085f194-c903-4ecb-abd8-c719fe7b3292"
  - "REQ-c50df875-2cb0-4e72-8a21-858359a11cae"
---
# DSG-aeb5e219: modules は common.nix に共通オプションを集約し、統合層ごとに 1 ファイルへ分ける

## 設計

```
modules/
├── common.nix         # options 定義のみ（全モジュールが import）
├── flake-parts.nix    # flake-parts module（perSystem.nput を flake.nput.<system> へ transpose）
├── home-manager.nix   # home.activation から nput エンジンを起動（root = homeRoot を pin）
├── nixos.nix          # （将来拡張）system.activationScripts から nput エンジンを起動
└── nix-darwin.nix     # （将来拡張）system.activationScripts から nput エンジンを起動
```

**`common.nix` を options 定義のみに絞り、全モジュールが import する**のが分割の要。
REQ-fc1c7ce6 は「全モジュールが共通オプションの同一集合を公開し、モジュールごとに集合が
分岐しない」ことを求めるが、これは各モジュールが自前で options を定義していると
守れない（片方にだけ足す・デフォルトがずれる、といった分岐が構造的に起きる）。
定義を 1 ファイルに置いて import させれば、集合の同一性がファイル参照の一意性として
担保される。

`common.nix` が持つのは REQ-fc1c7ce6 が定める `enable` / `configs`（`entries` を含む）/
`backup` と、素の `entries` の deprecated 糖衣まで。root オプションは REQ-fc1c7ce6 の通り
どのモジュールも公開せず、各モジュールが自分の性質で pin する（home-manager → `homeRoot`）。

**`user` オプションだけは `common.nix` に含めず、`nixos.nix` / `nix-darwin.nix` の各
モジュール内で追加定義する**。REQ-c2654ca5 が `user` を要求するのは NixOS / nix-darwin に
限られ、home-manager と standalone は `$HOME` を直接参照するため不要だからである。
共通側へ置くと、使わない層にも空のオプションが生えて「共通集合の同一性」が
かえって曖昧になる。

統合層を 1 層 1 ファイルにするのは、各ファイルが持つのが REQ-c1b3ca5f の言う
「エンジンをキックするだけの配線」であって、層ごとに異なるのは activation の
フック名と root の供給元だけだからである（HM の kick 契約は REQ-8085f194、
配線の内容は DSG-98d7fa5d）。`flake-parts.nix` は他と性質が違い、
REQ-c50df875 が求める「flake-parts 経路が直書きと同一の derivation を生む」ための
transpose を担う。

## 出典

`docs/design.md`「プロジェクト構成」の `modules/` 内訳（L56-61）と
「モジュール統合設計」→「共通オプション」節（L272-286）。

なお `docs/design.md` の当該節は `options.nput.entries` を直下に置く旧形で書かれているが、
本 item は ADR-0035 が導入した `nput.configs.<name>` を反映した REQ-fc1c7ce6 /
REQ-c6891aeb の現行規範に合わせている。
