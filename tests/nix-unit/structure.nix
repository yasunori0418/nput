# nix-unit: manifest 構造の不変条件（schemaVersion / root 系 / store・outOfStore エントリ構造）を
# アサートする（→ ADR-0006, ADR-0010, ADR-0014）。
#
# store パスの hash 揺れを避けるため src には toString が安定する fake な flake-input 相当
# （`{ outPath = …; }`）を使う。これは srcType の store-backed 判定（`? outPath`）を通る正当な test double。
{ lib, nput }:
let
  fakeSrc = {
    outPath = "/nix/store/00000000000000000000000000000000-fake-src";
  };
  norm = root: entries: nput.normalizeManifest { inherit lib root entries; };

  basic = norm nput.projectRoot {
    ".claude/skills/nix" = {
      src = fakeSrc;
      subpath = "skills/nix";
    };
  };
in
{
  testSchemaVersion = {
    expr = basic.schemaVersion;
    expected = 1;
  };

  testRootKindProject = {
    expr = basic.root.rootKind;
    expected = "project";
  };

  # project は実行時解決なので固定 root パスを持たない（→ ADR-0010）。
  testProjectHasNoFixedRoot = {
    expr = basic.root ? root;
    expected = false;
  };

  testStoreEntry = {
    expr = builtins.head basic.entries;
    expected = {
      srcKind = "store";
      src = "/nix/store/00000000000000000000000000000000-fake-src";
      subpath = "skills/nix";
      target = ".claude/skills/nix";
      method = "symlink";
    };
  };

  # out-of-store marker → clean enum 変換（→ ADR-0001, ADR-0010, ADR-0013）。
  # srcKind = "outOfStore" / src = marker の絶対パスが記録され、_nputMarker は漏れない
  # （expected は exact 一致なので余分なキーが残れば fail する）。
  testOutOfStoreEntry = {
    expr =
      builtins.head
        (norm nput.projectRoot {
          ".config/nvim" = {
            src = nput.mkOutOfStoreSymlink "/home/me/dotfiles/nvim";
            subpath = "lua";
          };
        }).entries;
    expected = {
      srcKind = "outOfStore";
      src = "/home/me/dotfiles/nvim";
      subpath = "lua";
      target = ".config/nvim";
      method = "symlink";
    };
  };

  # out-of-store entry に _nputMarker 判別タグが漏れていないことを明示アサートする。
  testOutOfStoreMarkerNotLeaked = {
    expr =
      (builtins.head
        (norm nput.projectRoot {
          ".config/nvim" = {
            src = nput.mkOutOfStoreSymlink "/home/me/dotfiles/nvim";
          };
        }).entries
      ) ? _nputMarker;
    expected = false;
  };
}
