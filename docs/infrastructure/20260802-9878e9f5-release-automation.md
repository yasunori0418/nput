---
id: "INF-9878e9f5-1ec0-4ef2-879b-859ea538cc43"
type: infrastructure
name: "リリース自動化（VERSION ファイル起点の bump PR・自動タグ・自動リリースノート）"
depends_on:
  - "INF-8b97573f-d4d6-4abf-85e2-d859afbd96c6"
---
# INF-9878e9f5: リリース自動化（VERSION ファイル起点の bump PR・自動タグ・自動リリースノート）

## 構成

バージョンの一次情報をリポジトリ直下の `VERSION`（プレーンテキスト 1 行・semver）に置き、
**その変更が main へマージされることがリリースを駆動する**。手動のタグ打ちは要らない。

`VERSION` は `flake.nix` が `builtins.readFile` で読んで `packages.nput` の `version` に反映し、
Go バイナリへは nix build の `ldflags`（`-X`）で埋め込む。単一の一次情報から flake とバイナリの
両方が導出され、二重管理にならない。

2 つの workflow で構成する。

| workflow | トリガ | 内容 |
|---|---|---|
| `bump-version.yml` | `workflow_dispatch` | バージョン文字列を入力に取り、非 main ブランチ上で `VERSION` を書き換えてコミット・push する |
| `release.yml` | `push: main` + `VERSION` の `paths` | `VERSION` を読み、`softprops/action-gh-release`（バージョン pin）でタグ `vX.Y.Z` 作成 + リリースノート自動生成 + GitHub Release 作成 |

マージゲート（INF-8b97573f）が効くのは **bump PR の側**で、`release.yml` 自体は main への push 後に
走るため required check の対象外。「bump PR のマージがリリースを駆動する」形が main 直コミット
禁止と整合する。
手元で `VERSION` を編集して PR を作る手動経路も同格に有効で、`bump-version.yml` は省力化で
あって唯一の経路ではない。

リリースノートは GitHub 標準の自動生成に任せる（Conventional Commits の履歴がそのまま読める）。
CHANGELOG.md はコミットしない。

## 成果物

リリースにバイナリは添付しない。nput は実行時に `nix` / `git` を叩く nix 前提ツールで、配布の
正規経路は flake ref（タグで pin できるようになる）。nix の無い環境ではバイナリが動かないため
添付の価値が薄い。将来需要が出た際に `release.yml` へステップを足す seam だけ残す。

## 出典

ADR-0042（リリースを bump PR 起点で自動化する）。
