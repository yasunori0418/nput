# nix-unit: resolveEntry の src 文字列化・marker→enum 変換と types の marker 判別をアサートする
# （→ ADR-0001, ADR-0010, ADR-0013）。
#
# #71 が unit-test seam として露出した `nput.__internal.resolveEntry` と、`lib/types.nix` の
# `isRootMarker` / `isOutOfStoreMarker` を直接突く。resolveEntry は manifest 全体を介さず
# 単一 entry の src 種別判定・文字列化のみを検証できる最小の境界。
#
# store パスの hash 揺れを避けるため src には toString が安定する fake な flake-input 相当
# （`{ outPath = …; }`）を使う。これは srcType の store-backed 判定（`? outPath`）を通る正当な test double。
{ lib, nput }:
let
  inherit (nput.__internal) resolveEntry;
  types = import ../../lib/types.nix lib;
  inherit (types) isRootMarker isOutOfStoreMarker;

  fakeSrc = {
    outPath = "/nix/store/00000000000000000000000000000000-fake-src";
  };

  resolve = resolveEntry lib;

  # store-backed src（path / derivation / flake input 相当）→ srcKind="store" / src=toString。
  storeEntry = resolve {
    src = fakeSrc;
    subpath = "skills/nix";
    target = ".claude/skills/nix";
    method = "symlink";
  };

  # out-of-store marker → srcKind="outOfStore" / src=marker の path。
  outOfStoreEntry = resolve {
    src = nput.mkOutOfStoreSymlink "/home/me/dotfiles/nvim";
    subpath = "lua";
    target = ".config/nvim";
    method = "copy";
  };
in
{
  # ---- store src の文字列化（toString）------------------------------------------------
  testResolveMarkerStoreSrcKind = {
    expr = storeEntry.srcKind;
    expected = "store";
  };

  testResolveMarkerStoreSrcString = {
    expr = storeEntry.src;
    expected = "/nix/store/00000000000000000000000000000000-fake-src";
  };

  # store entry の exact 形状。余分なキー（_nputMarker 等）が残れば exact 一致で fail する。
  testResolveMarkerStoreEntryShape = {
    expr = storeEntry;
    expected = {
      srcKind = "store";
      src = "/nix/store/00000000000000000000000000000000-fake-src";
      subpath = "skills/nix";
      target = ".claude/skills/nix";
      method = "symlink";
    };
  };

  # ---- out-of-store marker → clean enum 変換（src=marker の path）------------------------
  testResolveMarkerOutOfStoreSrcKind = {
    expr = outOfStoreEntry.srcKind;
    expected = "outOfStore";
  };

  testResolveMarkerOutOfStoreSrcPath = {
    expr = outOfStoreEntry.src;
    expected = "/home/me/dotfiles/nvim";
  };

  testResolveMarkerOutOfStoreEntryShape = {
    expr = outOfStoreEntry;
    expected = {
      srcKind = "outOfStore";
      src = "/home/me/dotfiles/nvim";
      subpath = "lua";
      target = ".config/nvim";
      method = "copy";
    };
  };

  # ---- _nputMarker 判別タグが出力に漏れないこと（Go contract は clean enum・→ ADR-0010）----
  testResolveMarkerNoTagLeakStore = {
    expr = storeEntry ? _nputMarker;
    expected = false;
  };

  testResolveMarkerNoTagLeakOutOfStore = {
    expr = outOfStoreEntry ? _nputMarker;
    expected = false;
  };

  # ---- isOutOfStoreMarker の判別 -------------------------------------------------------
  testResolveMarkerIsOutOfStoreTrue = {
    expr = isOutOfStoreMarker (nput.mkOutOfStoreSymlink "/home/me/x");
    expected = true;
  };

  # root marker は outOfStore ではない（タグ値で判別）。
  testResolveMarkerIsOutOfStoreFalseOnRoot = {
    expr = isOutOfStoreMarker nput.projectRoot;
    expected = false;
  };

  # store-backed src（マーカーでない attrset）も outOfStore ではない。
  testResolveMarkerIsOutOfStoreFalseOnStore = {
    expr = isOutOfStoreMarker fakeSrc;
    expected = false;
  };

  # ---- isRootMarker の判別 -------------------------------------------------------------
  testResolveMarkerIsRootTrueProject = {
    expr = isRootMarker nput.projectRoot;
    expected = true;
  };

  testResolveMarkerIsRootTrueHome = {
    expr = isRootMarker nput.homeRoot;
    expected = true;
  };

  testResolveMarkerIsRootTrueSystem = {
    expr = isRootMarker nput.systemRoot;
    expected = true;
  };

  # out-of-store marker は root ではない。
  testResolveMarkerIsRootFalseOnOutOfStore = {
    expr = isRootMarker (nput.mkOutOfStoreSymlink "/home/me/x");
    expected = false;
  };

  # 文字列 root（絶対パス）は marker ではない。
  testResolveMarkerIsRootFalseOnString = {
    expr = isRootMarker "/abs/root";
    expected = false;
  };
}
