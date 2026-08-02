---
id: "REQ-97c1e088-a17e-46d9-a9a1-83d1757d0f7d"
type: requirement
name: "mkManifest の引数は pkgs / entries / root の 3 つとする"
specification: |
  `lib.mkManifest` SHALL take three arguments: `pkgs` (an nixpkgs attrset, REQUIRED, used
  for derivation building such as `runCommandLocal` and to obtain `pkgs.lib`), `entries`
  (an attrset of placement definitions whose attribute keys are target paths), and `root`
  (`string | marker`, REQUIRED, the base of the placement destination).
specification_ja: |
  `lib.mkManifest` は引数として `pkgs`（attrset・必須。derivation ビルドと `pkgs.lib` の
  取得に使う）、`entries`（配置定義の attrset。属性キーが target パス）、`root`
  （`string | marker`・必須。配置先の基準）を取らなければならない。
---
# REQ-97c1e088: mkManifest の引数は pkgs / entries / root の 3 つとする

## 仕様

| 引数 | 型 | デフォルト | 説明 |
|---|---|---|---|
| `pkgs` | attrset（nixpkgs） | **なし（必須）** | derivation ビルド（`runCommandLocal` 等）と `pkgs.lib` の取得に使う |
| `entries` | attrset of entry | — | 配置定義の attrset。**属性キー = target パス**が識別子 |
| `root` | string \| marker | **なし（必須）** | 配置先の基準。暗黙デフォルトを持たない |

`pkgs` を引数で受け取るのは lib 層が unparameterized（`lib` / `pkgs` を自身で保持しない）
であるため。各関数が呼び出し時に必要なものを明示引数として要求する。

## 出典

`docs/spec.md`「lib API」→「`lib.mkManifest`」引数表。
