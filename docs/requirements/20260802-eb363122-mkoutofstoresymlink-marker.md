---
id: "REQ-eb363122-385a-499c-a074-c95efb949d07"
type: requirement
name: "mkOutOfStoreSymlink は out-of-store symlink を表すマーカーを返す"
derives_from:
  - "UC-01b896b4-04b9-40d0-bf9e-966eaf64c3d4"
specification: |
  `lib.mkOutOfStoreSymlink` SHALL be a pure function
  (`mkOutOfStoreSymlink :: string -> marker`) returning a marker that denotes an
  out-of-store symlink to a local path. Its argument SHALL be an absolute path string
  fixed at Nix evaluation time; a shell `$HOME` SHALL NOT be usable. In the core lib it
  SHALL do nothing but wrap the path in a marker attrset, and creating the actual link
  SHALL be done by the engine. It SHALL NOT delegate to a platform-native mechanism such
  as home-manager's `config.lib.file.mkOutOfStoreSymlink`.
specification_ja: |
  `lib.mkOutOfStoreSymlink` は、ローカルパスへの out-of-store symlink を表すマーカーを
  返す純粋関数（`mkOutOfStoreSymlink :: string -> marker`）でなければならない。
  引数は Nix 評価時に確定する絶対パスの文字列とし、シェルの `$HOME` は使えない。
  core lib ではパスをマーカー attrset に包むだけとし、実際の link 生成は engine が行う。
  プラットフォームのネイティブ機構（home-manager の
  `config.lib.file.mkOutOfStoreSymlink` 等）へは委譲しない。
---
# REQ-eb363122: mkOutOfStoreSymlink は out-of-store symlink を表すマーカーを返す

## 仕様

ローカルパスへの out-of-store symlink を表すマーカーを返す。`entry.src` に渡すことで、
その entry を Nix ストア経由ではなくローカル FS への symlink にする。ファイル編集が
即座に反映されるライブ用途のための明示的退避路である。

```
mkOutOfStoreSymlink :: string -> marker
```

- core lib（nixpkgs のみ依存）では **パスをマーカー attrset に包むだけの純粋関数**。
- 実際の link 生成は engine が行う。プラットフォームのネイティブ機構（HM の
  `config.lib.file.mkOutOfStoreSymlink` 等）へは委譲しない。

```nix
src = nput.lib.mkOutOfStoreSymlink "/home/me/dotfiles";
```

**制約**: 引数は Nix 評価時に確定する絶対パスの文字列。シェルの `$HOME` は使えない。
ローカルパスをポータブルにしたい場合は flake 内で変数として定義する
（→ `docs/spec.md`「ローカルパス（out-of-store）の扱い」）。

## 出典

`docs/spec.md`「lib API」→「`lib.mkOutOfStoreSymlink`」。
