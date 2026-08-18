---
id: "REQ-97c1e088-a17e-46d9-a9a1-83d1757d0f7d"
type: requirement
name: "mkManifest の引数は pkgs / entries / root の 3 つとする"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  `lib.mkManifest` SHALL accept exactly three arguments: `pkgs`, `entries` and `root`.
  `pkgs` SHALL be an nixpkgs attrset used for derivation building (such as
  `runCommandLocal`) and to obtain `pkgs.lib`, and it SHALL be REQUIRED with no default.
  `entries` SHALL be an attrset of placement definitions. `root` SHALL be the base of the
  placement destination. The obligations specific to `root` — that it is REQUIRED and that
  its type is `string | marker` — are stated by REQ-4ec3accc-8bb6-461f-9024-dcf0027849e4 and REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66 and are NOT
  restated here.
specification_ja: |
  `lib.mkManifest` は引数として `pkgs` / `entries` / `root` の 3 つを取らなければ
  ならない。`pkgs` は derivation ビルド（`runCommandLocal` 等）と `pkgs.lib` の取得に
  使う nixpkgs の attrset でなければならず、デフォルトを持たない必須引数でなければ
  ならない。`entries` は配置定義の attrset でなければならない。`root` は配置先の基準で
  なければならない。`root` 固有の規範（明示必須であること・
  型が `string | marker` であること）は REQ-4ec3accc-8bb6-461f-9024-dcf0027849e4 と REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66 が規定しており、
  本 item では重ねて規定しない。
---
# REQ-97c1e088-a17e-46d9-a9a1-83d1757d0f7d: mkManifest の引数は pkgs / entries / root の 3 つとする

## 仕様

| 引数 | 型 | デフォルト | 説明 |
|---|---|---|---|
| `pkgs` | attrset（nixpkgs） | **なし（必須）** | derivation ビルド（`runCommandLocal` 等）と `pkgs.lib` の取得に使う |
| `entries` | attrset of entry | — | 配置定義の attrset。**属性キー = target パス**が識別子 |
| `root` | string \| marker | **なし（必須）** | 配置先の基準。暗黙デフォルトを持たない |

`pkgs` を引数で受け取るのは lib 層が unparameterized（`lib` / `pkgs` を自身で保持しない）
であるため。各関数が呼び出し時に必要なものを明示引数として要求する。

`root` の必須性と型の内訳は REQ-4ec3accc-8bb6-461f-9024-dcf0027849e4（明示必須・暗黙デフォルトなし）と
REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66（`string | marker` の union と各マーカーの対応）が正とする。上の表は
原文の引数表をそのまま写したもので、規範の重複を避けるため本 item の規範文では
`root` 固有の規定を持たない。

## 出典

`docs/spec.md`「lib API」→「`lib.mkManifest`」引数表。
