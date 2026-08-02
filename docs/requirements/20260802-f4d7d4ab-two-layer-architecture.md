---
id: "REQ-f4d7d4ab-fbdb-48c6-b29f-08dd88e72645"
type: requirement
name: "nput は CLI とエンジンの 2 層で構成する"
specification: |
  nput SHALL be composed of two layers: the nput CLI (`packages.nput`, the primary UX
  installed on PATH) and the engine (a Go library). The CLI SHALL discover an entrypoint,
  run `nix build` / `nix eval` internally to obtain the store path of the named manifest,
  and import the engine to drive placement, stale removal and profile swap. The engine
  SHALL take `manifest.json` as its input and SHALL drive placement through native
  filesystem operations. The boundary between the two layers SHALL be `manifest.json`
  alone, so that the engine does not depend directly on Nix evaluation results. Which
  files are discovered as entrypoints is stated by the CLI specification and is NOT
  restated here.
specification_ja: |
  nput は nput CLI（`packages.nput`・PATH 常駐の一次 UX）と engine（Go ライブラリ）の
  2 層で構成しなければならない。CLI は entrypoint を発見し、内部で `nix build` /
  `nix eval` を回して named manifest の store path を得て、engine を import して配置・
  stale 除去・profile swap を駆動する。engine は `manifest.json` を入力に取り、
  ネイティブ FS 操作で配置を行う。2 層の境界は `manifest.json` だけとし、engine が Nix の
  評価結果へ直接依存しないようにしなければならない。どのファイルを entrypoint として
  発見するかは CLI 仕様の担当で、本 item では規定しない。
---
# REQ-f4d7d4ab: nput は CLI とエンジンの 2 層で構成する

## 仕様

nput は **2 層**で構成する。

```
[nput CLI]  packages.nput（PATH 常駐・一次 UX）
  ・entrypoint(flake.nix/shell.nix/default.nix)を発見（CWD 既定 / -f 上書き）
  ・内部で nix build/eval を回し named manifest(manifest.json + symlink farm)の store path を得る
  ・engine(ライブラリ)を import して配置・stale 除去・profile swap を駆動
   ↓ manifest.json in
[engine]  Go ライブラリ
  ・manifest.json を入力に取り nix(profile)/git(toplevel)のみ叩く
  ・ネイティブ FS 操作で place/replace/remove、保守的 stale 除去、nix-env --set
```

層の境界は `manifest.json` が担う。CLI と engine の間で受け渡すのはこの JSON 契約のみで、
engine は Nix の評価結果へ直接依存しない。

## 出典

`docs/spec.md`「アーキテクチャ概要」。
