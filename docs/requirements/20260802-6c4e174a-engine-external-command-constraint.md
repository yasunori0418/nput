---
id: "REQ-6c4e174a-4d16-477a-96ff-17cb4eb5b564"
type: requirement
name: "engine が叩く外部コマンドは nix と git のみに限る"
derives_from:
  - "UC-f2436d68-91ff-4c48-b1df-47acefe4f464"
  - "UC-19a90989-0ae3-438f-8a75-4e1e2637f81c"
specification: |
  The engine SHALL invoke only `nix` (for profile operations) and `git` (for obtaining the
  toplevel) as external commands. Place / replace / remove and conservative stale removal
  SHALL be performed with native filesystem operations. Profile swap SHALL be done via
  `nix-env --set`.
specification_ja: |
  engine が外部コマンドとして叩いてよいのは `nix`（profile 操作）と `git`（toplevel の
  取得）のみでなければならない。place / replace / remove と保守的 stale 除去は
  ネイティブ FS 操作で行わなければならない。profile の切り替えは `nix-env --set` で行う。
---
# REQ-6c4e174a: engine が叩く外部コマンドは nix と git のみに限る

## 仕様

engine は `manifest.json` を入力に取り、外部コマンドは以下の 2 つだけを叩く。

- `nix`（profile 操作）
- `git`（toplevel の取得）

配置そのもの（place / replace / remove）と保守的 stale 除去は **ネイティブ FS 操作**で
行う。profile の切り替えは `nix-env --set`。

外部コマンドの依存を 2 つに閉じることで、engine の動作が実行環境のコマンド群に左右される
範囲を限定する。2 層構成そのものは REQ-f4d7d4ab が規定する。

## 出典

`docs/spec.md`「アーキテクチャ概要」の構成図・engine 層の 2 行
（「`manifest.json` を入力に取り nix(profile)/git(toplevel)のみ叩く」「ネイティブ FS
操作で place/replace/remove、保守的 stale 除去、nix-env --set」）。

決定の実体は ADR-0006「決定」節で、`rsync` / `ln` の runtime 依存を落とし FS 操作を
ネイティブ Go（`os.Symlink` / ネイティブ再帰コピー）で行うこと、profile 操作を
`nix-env --profile <dir> --set <link-farm>` にすることを定めている。
