---
id: "REQ-3f541d39-da41-4ef8-858b-707f54cf6a29"
type: requirement
name: "root マーカーは kind を運ぶ入れ物でパス解決は engine が行う"
specification: |
  `lib.projectRoot` / `lib.homeRoot` / `lib.systemRoot` SHALL be pure functions returning
  a marker attrset that carries the kind of runtime resolution. They MUST NOT be sugar
  that returns a path string. `mkManifest` SHALL record the kind in `manifest.json`, and
  resolving the concrete path SHALL be the engine's runtime responsibility.
specification_ja: |
  `lib.projectRoot` / `lib.homeRoot` / `lib.systemRoot` は、実行時解決の種別（kind）を
  運ぶマーカー attrset を返す純粋関数でなければならない。パス文字列を返す糖衣であっては
  ならない。`mkManifest` は kind を `manifest.json` に記録し、実体パスの解決は engine の
  実行時責務とする。
---
# REQ-3f541d39: root マーカーは kind を運ぶ入れ物でパス解決は engine が行う

## 仕様

`root` 引数に渡す **root マーカー**は、`mkOutOfStoreSymlink` と同じ「マーカーを渡して
挙動を opt-in する」パターンに従う。

```
projectRoot :: marker   # 実行時に git rev-parse --show-toplevel（--root で上書き可）
homeRoot    :: marker   # 実行時に $HOME
systemRoot  :: marker   # /（distro 構想・将来）
```

- core lib（nixpkgs のみ依存）では **kind を運ぶマーカー attrset を返す純粋関数**。
  実体パス解決は engine の実行時責務。
- マーカーは「**実行時解決の種別を運ぶ入れ物**」であってパス文字列を返す糖衣ではない。
  `homeRoot` / `projectRoot` は評価時にパスへ展開できない（`$HOME` / git toplevel は
  実行環境依存）。
- `mkManifest` は kind を `manifest.json` に記録し、エンジンが実行時に解決する。

## 出典

`docs/spec.md`「lib API」→「`lib.projectRoot` / `lib.homeRoot` / `lib.systemRoot`」、
および「`root` の値」節の注記。
