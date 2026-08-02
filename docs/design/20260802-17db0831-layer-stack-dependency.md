---
id: "DSG-17db0831-d7da-446d-ba3e-404df64c582d"
type: design
name: "層を CLI / engine / lib / common.nix / 統合層の 5 段に積み、依存を上から下への一方向に限る"
satisfies:
  - "REQ-f4d7d4ab-fbdb-48c6-b29f-08dd88e72645"
  - "REQ-d85f0cef-0f1e-4897-a841-41b61a8dae51"
  - "REQ-c1b3ca5f-d2f7-443c-bc4b-b18413ca97b9"
---
# DSG-17db0831: 層を CLI / engine / lib / common.nix / 統合層の 5 段に積み、依存を上から下への一方向に限る

## 設計

```
nput CLI (packages.nput)      ← 一次 UX。entrypoint 発見 + nix build/eval + エンジン駆動
        │ import
配置エンジン (internal/ Go ライブラリ)  ← 配置・stale 除去・profile swap の単一の源（manifest.json in）
        ↑ manifest.json を渡して起動
lib/ (mkManifest 他)          ← nixpkgs.lib のみ依存（純データ生成。manifest.json + symlink farm）
        ↑ 起動配線
modules/common.nix            ← options 型定義（nixpkgs.lib のみ依存）
        ↑
┌───────┼──────────────────┐
HM   NixOS  darwin  devShell  standalone(CLI)
（root と activation hook を供給して nput エンジンを起動する薄い配線のみ）
```

**上位層が下位層にのみ依存し、逆方向の依存を持たない**ことを層構造の規約とする。

REQ-f4d7d4ab が規範化しているのは CLI と engine の 2 層境界（`manifest.json`）までで、
lib・`common.nix`・統合層を含めた積み方と依存の向きは設計側の判断になる。この積み方が
実現手段として効くのは次の点。

- **engine を lib の上へ置き、lib から engine への依存を持たせない**ことで、
  REQ-d85f0cef の「lib は配置ロジックを持たず nixpkgs.lib のみに依存する」が
  層の位置として表現される。lib が engine を呼べる位置にあると、配置ロジックが
  lib 側へ漏れ出す経路が開いてしまう
- **統合層を最下段に置き、上位（lib・engine）へ向かってのみ依存させる**ことで、
  REQ-c1b3ca5f の「モジュールは engine をキックするだけの配線」が守られる。
  統合層が engine から参照される位置にあると、engine が層ごとの事情を知ることになり、
  配置の振る舞いが層ごとに分岐しうる
- **配置の振る舞いは全層で engine が単一の源**であり、各層はネイティブ機構へ翻訳しない。
  層をいくつ重ねても、配置を実行する主体は 1 つに保たれる

## 出典

`docs/design.md`「レイヤー構成」（L71-87）。
