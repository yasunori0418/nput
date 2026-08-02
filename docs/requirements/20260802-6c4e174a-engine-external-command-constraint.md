---
id: "REQ-6c4e174a-4d16-477a-96ff-17cb4eb5b564"
type: requirement
name: "engine が叩く外部コマンドは nix と git のみに限る"
specification: |
  The engine SHALL invoke only `nix` (for profile operations) and `git` (for obtaining the
  toplevel) as external commands. Everything else — place / replace / remove, conservative
  stale removal — SHALL be performed with native filesystem operations rather than by
  shelling out. Profile swap SHALL be done via `nix-env --set`.
specification_ja: |
  engine が外部コマンドとして叩いてよいのは `nix`（profile 操作）と `git`（toplevel の
  取得）のみでなければならない。それ以外——place / replace / remove、保守的 stale 除去
  ——は外部コマンドの呼び出しではなくネイティブ FS 操作で行わなければならない。
  profile の切り替えは `nix-env --set` で行う。
---
# REQ-6c4e174a: engine が叩く外部コマンドは nix と git のみに限る

## 仕様

engine は `manifest.json` を入力に取り、外部コマンドは以下の 2 つだけを叩く。

- `nix`（profile 操作）
- `git`（toplevel の取得）

配置そのもの（place / replace / remove）と保守的 stale 除去は、生成した shell へ委ねず
**ネイティブ FS 操作**で行う。profile の切り替えは `nix-env --set`。

外部コマンドの依存を 2 つに閉じることで、engine の動作が実行環境のコマンド群に左右される
範囲を限定する。2 層構成そのものは REQ-f4d7d4ab が規定する。

## 出典

`docs/spec.md`「アーキテクチャ概要」の engine 層の記述。
