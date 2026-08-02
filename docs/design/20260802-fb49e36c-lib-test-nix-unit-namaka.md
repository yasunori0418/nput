---
id: "DSG-fb49e36c-eb20-4efa-8cac-88ef0873db8e"
type: design
name: "lib は nix-unit の評価テストと namaka のスナップショットの 2 手段で検証する"
satisfies:
  - "REQ-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4"
  - "REQ-60e6b49c-9ba1-4552-a0ec-d340421ec281"
  - "REQ-b232ec98-af3b-41f3-a050-29d417322002"
  - "REQ-d1b5b3f5-10a0-400d-9f03-ba00c63d1c34"
  - "REQ-901993e9-771c-480a-ba0d-ca4be042e206"
---
# DSG-fb49e36c: lib は nix-unit の評価テストと namaka のスナップショットの 2 手段で検証する

## 設計

| 手段 | 重点 |
|---|---|
| **nix-unit**（評価テスト）| `mkManifest` の不変条件 — 個別の入力に対して期待する出力・拒否すべき入力がエラーになること |
| **namaka**（スナップショット）| `manifest.json` 全体の回帰 — 出力構造がまるごと変わっていないこと |

2 手段を併用するのは、lib が生成するものが**構造化データ 1 個**であることに由来する。

- **nix-unit が担うのは「点」の検証**。REQ-b232ec98 の `normalizeManifest`（検査・
  デフォルト適用・marker 変換）と REQ-d1b5b3f5 の「`mkManifest` 自身が evalModules で
  入力を検査する単一ゲート」は、正常系より**拒否されるべき入力がきちんとエラーになるか**が
  肝になる。評価テストなら「この入力は eval が失敗する」を直接書ける
- **namaka が担うのは「面」の検証**。REQ-60e6b49c が定める返り値（`manifest.json` +
  symlink farm を含む store オブジェクト）は、フィールドを 1 つ書き換えただけでも
  engine 側の契約が壊れうる。全体をスナップショットに固定しておけば、
  意図しない構造変化が差分として出る。個別アサーションを網羅する形では
  「書き忘れたフィールド」を捕まえられない

いずれも**実際のビルドを伴わない評価だけで完結する**のが選択理由でもある。
REQ-2b0c2bb8 が `mkManifest` を純粋関数と定めている以上、検証も評価層で閉じるのが自然で、
FS も nix daemon も要らない。テスト対象の private helper へは、REQ-901993e9 が
テスト seam として公開する `lib.__internal` から到達する。

engine 側の検証手段は DSG-836aa5cb、実経路の一気通貫は DSG-2947b4a5 が担う。

## 出典

`docs/design.md`「テスト戦略」のテーブル 1 行目（L451-453）。
