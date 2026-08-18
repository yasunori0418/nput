---
id: "REQ-2bd0d35f-dc29-4771-9cfe-6998247afa0f"
type: requirement
name: "modules/common.nix は nixpkgs.lib のみに依存する"
derives_from:
  - "UC-d39c1994-f9a5-4860-80ba-f6e584adaf14"
specification: |
  `modules/common.nix` SHALL depend only on nixpkgs.lib. It SHALL NOT introduce a
  dependency on the module system of home-manager, NixOS or nix-darwin, so that the common
  option set it holds can be imported from every integration layer without one layer's
  module system leaking into another. Which options it defines is stated by REQ-fc1c7ce6,
  and the same constraint on `lib` by REQ-d85f0cef; neither is restated here. What the
  per-layer files (`modules/home-manager.nix` and its counterparts) may depend on is
  outside the scope of this item and is not stated here; that they are wiring which kicks
  the engine is stated by REQ-c1b3ca5f.
specification_ja: |
  `modules/common.nix` は nixpkgs.lib のみに依存しなければならない。home-manager /
  NixOS / nix-darwin の module system への依存を持ち込んではならない。同ファイルが持つ
  共通オプション集合を、ある統合層の module system を別の統合層へ漏らすことなく全統合層
  から import できるようにするためである。同ファイルがどのオプションを定義するかは
  REQ-fc1c7ce6、`lib` に対する同種の制約は REQ-d85f0cef の担当で、いずれも本 item では
  規定しない。統合層ごとのファイル（`modules/home-manager.nix` 等）が何に依存してよいかは
  本 item の射程外であり、ここでは規定しない。それらが engine を起動する配線であることは
  REQ-c1b3ca5f の担当である。
---
# REQ-2bd0d35f: modules/common.nix は nixpkgs.lib のみに依存する

## 仕様

| コンポーネント | 依存 |
|---|---|
| `modules/common.nix` | nixpkgs.lib のみ |

> **上は原文の写しで、規範は frontmatter が正**。同ファイルが定義する共通オプション集合
> （`enable` / `configs`（`entries` を含む）/ `backup.*` と `nput.entries` の糖衣）は
> REQ-fc1c7ce6、entry submodule の型定義を `lib/types.nix` と共有することは REQ-d1b5b3f5、
> `lib` が nixpkgs.lib のみに依存する純データ生成器であることは REQ-d85f0cef、engine が
> stdlib-only であることは REQ-b74a118a の担当。統合層ごとのファイルが engine を起動する
> 配線に徹することは REQ-c1b3ca5f が定めるが、**それらが何に依存してよいかの規範は現状
> どの item も持たない**（原文の `modules/home-manager.nix` 行以下も `modules/common.nix`
> 行と同じく縮退で落ちたまま未回収）。層の積み方と依存の向きを一方向に限る設計は
> DSG-17db0831 が持つ。
>
> **新規 item にした理由（REQ-d85f0cef との粒度合わせ）**: 依存制約は層ごとに 1 item
> 立てるのが既存の粒度で、`lib` は REQ-d85f0cef、engine は REQ-b74a118a が各層 1 item
> として持つ。`modules/common.nix` は DSG-17db0831 の 5 段のうち `lib` とも統合層とも
> 別の 1 段であり、同じ粒度に揃えるなら独立した item になる。既存 item への編入も
> 検討したが、REQ-fc1c7ce6 は「共通オプションとして何を公開するか」の item で軸が別
> （公開する集合 vs 依存の範囲）であり、REQ-d85f0cef へ相乗りさせると `lib` と
> `common.nix` という別の層に対する 2 つの依存制約が 1 item に同居して 1 item 1 主張が
> 崩れるため、いずれも採らなかった。DSG-17db0831 の層構成図は `common.nix` を「options 型定義
> （nixpkgs.lib のみ依存）」と記載するが、これは設計の図であって規範文（specification）
> ではないため、本 item が規範を持つ。

## 出典

`docs/spec.md`「依存関係」節の表の `modules/common.nix` 行「nixpkgs.lib のみ」。#209 の
分割では同表の `internal/` 行が REQ-b74a118a へ入った一方、`modules/common.nix` 行は
どの item の規範にも入らないまま縮退で落ちていた（epic #203 / issue #228 で回収）。
`lib` に対する同種の依存制約は、同表の `lib/` 行ではなく別出典（「lib API」節）の
REQ-d85f0cef が持つ。

同表の他行と異なり原文の当該行は ADR 参照を伴わず、`modules/common.nix` の依存制約
そのものを決定した ADR も無いため、`justifies` は張らない（側面としては、lib を
データ生成に徹させる ADR-0006、entry submodule の型定義を `lib/types.nix` と
`modules/common.nix` で共有すると定めた ADR-0010、モジュールを配線に徹させる ADR-0003 が
この制約と整合する）。
