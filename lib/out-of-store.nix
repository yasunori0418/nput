# マーカー構築子（→ ADR-0001, ADR-0004, ADR-0005, ADR-0007, ADR-0010）。
#
# マーカーは「実行時解決の種別を運ぶ入れ物」であり、パス文字列を返す糖衣ではない。
# `src`（derivation）も marker もどちらも attrset で構造判別できないため、marker には
# `_nputMarker` 判別タグを持たせ、`lib/types.nix` の custom optionType の `check` が判別する。
# `_nputMarker` は Nix 評価内で完結させ `manifest.json` には漏らさない（Go 契約は clean enum・→ ADR-0010）。
#
# 依存なし（nixpkgs.lib すら要らない純 attrset 構築子）。
{
  # ローカルパスへの out-of-store symlink を表すマーカー（→ ADR-0001）。
  # 引数は Nix 評価時に確定する絶対パスの文字列。実際の link 生成は engine の実行時責務。
  mkOutOfStoreSymlink = path: {
    _nputMarker = "outOfStore";
    inherit path;
  };

  # root マーカー（→ ADR-0004 改訂, ADR-0005, ADR-0007）。kind を運ぶだけで、実体パス解決は engine の実行時責務。
  projectRoot = {
    _nputMarker = "root";
    kind = "project";
  };
  homeRoot = {
    _nputMarker = "root";
    kind = "home";
  };
  systemRoot = {
    _nputMarker = "root";
    kind = "system";
  };
}
