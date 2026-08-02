---
id: "DSG-e4d5db6b-5bac-4fd6-b2fb-ed07bdec30f5"
type: design
name: "lib は公開 API・型定義・manifest 生成・マーカー構築子の 4 ファイルへ分割する"
satisfies:
  - "REQ-d85f0cef-0f1e-4897-a841-41b61a8dae51"
  - "REQ-97c1e088-a17e-46d9-a9a1-83d1757d0f7d"
  - "REQ-60e6b49c-9ba1-4552-a0ec-d340421ec281"
  - "REQ-eb363122-385a-499c-a074-c95efb949d07"
  - "REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66"
  - "REQ-901993e9-771c-480a-ba0d-ca4be042e206"
  - "REQ-b232ec98-af3b-41f3-a050-29d417322002"
---
# DSG-e4d5db6b: lib は公開 API・型定義・manifest 生成・マーカー構築子の 4 ファイルへ分割する

## 設計

```
lib/
├── default.nix        # 公開 API のまとめ（mkManifest / mkOutOfStoreSymlink / projectRoot / homeRoot / systemRoot）
├── types.nix          # entries の型定義（各モジュールで共有）
├── manifest.nix       # mkManifest（manifest.json + symlink farm derivation を生成する純粋関数）
└── out-of-store.nix   # mkOutOfStoreSymlink / projectRoot / homeRoot / systemRoot（マーカー構築子）
```

分割の軸は「公開 API 面 / 型 / 生成ロジック / マーカー」の 4 つで、次の理由による。

- **`default.nix` を公開 API のまとめに限る**ことで、REQ-97c1e088（`mkManifest` の引数は
  pkgs / entries / root の 3 つ）・REQ-60e6b49c（返り値）・REQ-eb363122
  （`mkOutOfStoreSymlink`）・REQ-37b56673（root マーカー 3 種）が定める API 面が 1 ファイルを
  読めば分かる。`__internal` はここから露出するが安定 API ではなく、REQ-901993e9 が定める
  テスト seam に限る
- **`types.nix` を独立させる**のは、entries の型定義を `lib` と `modules/` の双方が
  共有するため。共有できる形にしておかないと、モジュール側が型を再定義して二重管理になる
  （`modules/common.nix` からの共有は DSG-aeb5e219）
- **`manifest.nix` と `out-of-store.nix` を分ける**のは、前者が REQ-b232ec98 の
  `normalizeManifest`（検査・デフォルト適用・marker 変換）と derivation 組み立てを持つのに対し、
  後者はマーカーという不活性なデータ構造を返すだけで、依存の向きが一方向（manifest → marker）に
  なるため

いずれのファイルも nixpkgs.lib のみに依存し、REQ-d85f0cef の依存制約をファイル単位で保つ。

## 出典

`docs/design.md`「プロジェクト構成」の `lib/` 内訳。
