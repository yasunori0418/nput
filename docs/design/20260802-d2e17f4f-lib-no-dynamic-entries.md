---
id: "DSG-d2e17f4f-0d32-45c1-8125-17e589664c85"
type: design
name: "動的 entry 生成のヘルパを lib に置かず、readDir する idiom をドキュメントで示す"
satisfies:
  - "REQ-d85f0cef-0f1e-4897-a841-41b61a8dae51"
  - "REQ-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4"
---
# DSG-d2e17f4f: 動的 entry 生成のヘルパを lib に置かず、readDir する idiom をドキュメントで示す

## 設計

「リポジトリ内のファイルを列挙して entries を組み立てる」需要に対し、**lib は関数を
提供しない**。既に realise 済みの store パスまたは flake input を `builtins.readDir`
する idiom をドキュメント側（`docs/spec.md` 応用節）で示すに留める。

lib へヘルパを置かない理由は、置いた場合に **IFD（import from derivation）を
誘発する API になる**こと。列挙対象を「まだ realise していない derivation」で
受け取れる形にすると、評価中にビルドが走る経路が lib の公開 API から生えてしまう。

これは次の 2 つの要求の実現手段になっている。

- **REQ-d85f0cef（lib は nixpkgs.lib のみに依存する純データ生成器）**: 列挙ヘルパは
  実質的に「FS を読む」機能で、純データ生成の枠を越える。API 面に置かないことで
  lib の役割が entries を受け取って manifest を組むことに限定され続ける
- **REQ-2b0c2bb8（mkManifest は純粋関数）**: 動的生成を lib の内側へ入れると、
  `mkManifest` の入力が「評価時に確定したデータ」でなくなる余地が生まれる。
  列挙をユーザー側の評価コードに置けば、`mkManifest` が受け取る `entries` は
  常に確定済みの attrset になる

代償として、列挙の書き方はユーザーが自分で書くことになる。これを idiom として
ドキュメントで補うのが本設計の対になる措置。

## 出典

`docs/design.md`「flake.nix outputs 設計」の `lib` 公開 API 一覧末尾のコメント行（L119）。
