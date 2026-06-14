# entry submodule + srcType / rootType / marker custom type（→ ADR-0010, ADR-0014）。
#
# `mkManifest` の `evalModules` と `modules/common.nix` の `attrsOf (submodule …)` で
# 共有できるよう、nixpkgs.lib を受け取り型定義一式を返す純関数として書く。
# `lib.types` / `mkOption` / `evalModules` は nixpkgs.lib のコアなので
# 「lib は nixpkgs.lib のみ依存」を満たす（home-manager / NixOS / nix-darwin は引かない）。
lib:
let
  inherit (lib) types mkOption mkDefault;
  inherit (lib.options) mergeEqualOption;
  inherit (builtins) isAttrs isString;

  # marker 判別（`out-of-store.nix` が付ける `_nputMarker` タグで見分ける）。
  isOutOfStoreMarker = x: isAttrs x && (x._nputMarker or null) == "outOfStore";
  isRootMarker = x: isAttrs x && (x._nputMarker or null) == "root";

  # store-backed src: path / derivation / flake input（`{ outPath = …; }`）を 1 ブランチに集約。
  # 素の文字列は拒否し out-of-store の暗黙分岐を型で禁ずる（→ ADR-0001）。
  # path と set は挙動が同一（ともに store link）なので型レベルで分けない（→ ADR-0010）。
  isStoreBacked =
    x:
    lib.isPath x || lib.isDerivation x || (isAttrs x && x ? outPath && (x._nputMarker or null) == null);

  # srcType = either storeBacked outOfStoreMarker（→ ADR-0010）。
  srcType = types.mkOptionType {
    name = "nputSrc";
    description = "store-backed source (path / derivation / flake input) or out-of-store marker";
    check = x: isStoreBacked x || isOutOfStoreMarker x;
    merge = mergeEqualOption;
  };

  # rootType = either str rootMarker（→ ADR-0010）。`mkManifest` 専用で modules は共有しない
  # （modules は root を pin する・→ ADR-0003）。
  rootType = types.mkOptionType {
    name = "nputRoot";
    description = "absolute path string or root marker (projectRoot / homeRoot / systemRoot)";
    check = x: isString x || isRootMarker x;
    merge = mergeEqualOption;
  };

  # entry submodule（→ ADR-0014）。属性キー = target が識別子。strict（未知キー拒否）。
  entryModule =
    { name, ... }:
    {
      options = {
        src = mkOption {
          type = srcType;
          description = "配置元。store-backed（path / set）または out-of-store marker。";
        };
        subpath = mkOption {
          type = types.str;
          default = ".";
          description = "src 内の相対パス。省略 = リポジトリ全体（→ ADR-0008）。";
        };
        target = mkOption {
          type = types.str;
          # 既定 = 属性キー（→ ADR-0014）。
          default = name;
          defaultText = "属性キー";
          description = "root 相対の配置先。省略時は属性キー。";
        };
        method = mkOption {
          type = types.enum [
            "symlink"
            "copy"
          ];
          default = "symlink";
          description = "配置方法（旧名 mode・→ ADR-0015）。";
        };
      };
    };

  entriesType = types.attrsOf (types.submodule entryModule);
in
{
  inherit
    isOutOfStoreMarker
    isRootMarker
    isStoreBacked
    srcType
    rootType
    entryModule
    entriesType
    ;
}
