---
id: "REQ-690f2730-2628-420d-8e72-ed1ce747ac1e"
type: requirement
name: "NixOS / nix-darwin モジュール経路の実 activate は E2E ハーネスの対象外とする"
specification: |
  Actually activating the NixOS and nix-darwin module paths SHALL be outside the scope of
  the E2E harness, since it requires a VM or a sandbox whereas the harness is a single
  non-NixOS job, and a VM test SHALL NOT be a prerequisite for that harness. Verification
  of those paths SHALL be added separately at the point at which the module paths are
  implemented in earnest.
specification_ja: |
  NixOS / nix-darwin モジュール経路の実 activate は E2E ハーネスのスコープ外としなければ
  ならない（VM / sandbox を要するのに対しハーネスは非 NixOS の単一ジョブであるため）。
  VM テストを同ハーネスの前提条件にしてはならない。これらの経路の検証は、モジュール経路を
  本実装する段で別途追加する。
---
# REQ-690f2730: NixOS / nix-darwin モジュール経路の実 activate は E2E ハーネスの対象外とする

## 仕様

**NixOS VM テスト（`runNixOSTest`）は将来拡張**。NixOS / nix-darwin モジュール経路の実
activate は VM / sandbox を要し、本ハーネス（非 NixOS の単一ジョブ）のスコープ外。
モジュール経路を本実装する段で別途追加する。

> **上は原文の写しで、規範は frontmatter が正**。E2E ハーネス本体の検証範囲は
> REQ-6419e4b0、NixOS / nix-darwin モジュールが engine キックの配線であることは
> REQ-c1b3ca5f、その `user` オプションは REQ-c2654ca5 の担当。原文が併記する
> `runNixOSTest` という手段と `docs/design.md`「テスト戦略」への参照は、将来追加する際の
> 実装手段の示唆であり規範に採らない。

## 出典

`docs/spec.md`「E2E 検証範囲（非 NixOS）」節の末尾の注記。

決定の実体は ADR-0006「エンジンを固定の Go バイナリにし、lib はデータ生成に徹する
（生成 bash を廃する）」の「テスト戦略」節で、「NixOS VM テスト（`runNixOSTest`）は
モジュール経路の実装段で追加」と定めている。ハーネス側が非 NixOS の単一ジョブである
ことは ADR-0012「CI・テスト実行基盤を cryoflow 構成踏襲で確定する」が定めるが、同 ADR は
VM テストの扱いを決めていないため、側面の根拠として `justifies` は張らない
（ハーネス本体の検証範囲は REQ-6419e4b0 が担当し、そちらが ADR-0012 から張られる）。
