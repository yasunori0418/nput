---
id: "DSG-901351ea-f01b-4bb0-a470-b890d92c980b"
type: design
name: "NixOS VM テストは runNixOSTest でモジュール経路の実装時に追加し、E2E ハーネスとは別系統に置く"
satisfies:
  - "REQ-690f2730-2628-420d-8e72-ed1ce747ac1e"
  - "REQ-c2654ca5-62c2-4e4b-ad67-ffc5468f429b"
---
# DSG-901351ea: NixOS VM テストは runNixOSTest でモジュール経路の実装時に追加し、E2E ハーネスとは別系統に置く

## 設計

NixOS / nix-darwin モジュール経路の実 activate は、**`runNixOSTest`（nixpkgs の VM
テストフレームワーク）で検証する**。追加するのはモジュール経路を実装する段で、
`tests/e2e/` の bash ハーネス（DSG-2947b4a5）のスコープ外に置く。

**E2E ハーネスへ入れずに別系統とする**理由は、要求する実行環境が違うため。
REQ-690f2730 が「NixOS / nix-darwin モジュール経路の実 activate は E2E ハーネスの
対象外」と定めているのは、E2E が非 NixOS（ubuntu ランナー）で「nix さえあれば動く」を
検証する趣旨で組まれているのに対し、NixOS モジュールの activate には NixOS 自体が
要るからである。同じハーネスへ押し込むと、前提の異なるシナリオが 1 つの実行系に混ざる。

**`runNixOSTest` を選ぶ**のは、`system.activationScripts` からの engine 起動を
検証するには実際に system activation を走らせる必要があり、nixpkgs が提供する
VM テストがその唯一の現実的な手段だからである。REQ-c2654ca5 が定める `user`
オプション（配置先ユーザーの特定）も、実際に複数ユーザーが存在する system 上でないと
意味のある検証にならない。

**追加時期をモジュール経路の実装段まで遅らせる**のは、実装スコープの線引き
（DSG-4a84f282）が NixOS / nix-darwin を将来拡張に置いていることの帰結である。
テスト対象が存在しない段階で VM テストの土台だけ用意しても、空の枠が残るだけになる。

## 出典

`docs/design.md`「テスト戦略」の NixOS VM テストの段落（L466）。
