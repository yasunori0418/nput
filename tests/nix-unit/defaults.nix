# nix-unit: normalizeManifest のデフォルト適用・明示上書き・target 辞書順をアサートする
# （→ ADR-0010, ADR-0014, ADR-0016）。
#
# store パスの hash 揺れを避けるため src には toString が安定する fake な flake-input 相当
# （`{ outPath = …; }`）を使う。これは srcType の store-backed 判定（`? outPath`）を通る正当な test double。
{ lib, nput }:
let
  fakeSrc = {
    outPath = "/nix/store/00000000000000000000000000000000-fake-src";
  };
  norm = root: entries: nput.normalizeManifest { inherit lib root entries; };
in
{
  # ---- デフォルト適用（subpath="." / target=属性キー / method="symlink"）-----
  testDefaultsApplied = {
    expr =
      builtins.head
        (norm nput.projectRoot {
          ".config/foo" = {
            src = fakeSrc;
          };
        }).entries;
    expected = {
      srcKind = "store";
      src = "/nix/store/00000000000000000000000000000000-fake-src";
      subpath = ".";
      target = ".config/foo";
      method = "symlink";
    };
  };

  # 明示上書きが反映される
  testExplicitOverrides = {
    expr =
      builtins.head
        (norm nput.projectRoot {
          "label" = {
            src = fakeSrc;
            target = ".config/bar";
            subpath = "sub/dir";
            method = "copy";
          };
        }).entries;
    expected = {
      srcKind = "store";
      src = "/nix/store/00000000000000000000000000000000-fake-src";
      subpath = "sub/dir";
      target = ".config/bar";
      method = "copy";
    };
  };

  # entries は target（属性キー）の辞書順で決定的に配列化される（→ ADR-0014, ADR-0016）。
  testEntriesSortedByTarget = {
    expr =
      map (e: e.target)
        (norm nput.projectRoot {
          "b" = {
            src = fakeSrc;
          };
          "a" = {
            src = fakeSrc;
          };
          "c" = {
            src = fakeSrc;
          };
        }).entries;
    expected = [
      "a"
      "b"
      "c"
    ];
  };
}
