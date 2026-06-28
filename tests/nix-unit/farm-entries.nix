# nix-unit: symlink farm のアンカー対象抽出（`__internal.farmEntries`）と GC アンカー名
# （`__internal.anchorName`）・manifest.nix が生成する anchorLines シェルをアサートする
# （→ ADR-0016, ADR-0019, #58, #71, #75）。
#
# farmEntries は store-backed かつ method = symlink のエントリのみをアンカー対象とする。copy /
# out-of-store はアンカーを持たない（copy は世代外・置き切り、out-of-store はストア非依存）。
#
# store パスの hash 揺れを避けるため src には toString が安定する fake な flake-input 相当
# （`{ outPath = …; }`）を使う。これは srcType の store-backed 判定（`? outPath`）を通る正当な test double。
{ lib, nput }:
let
  fakeSrc = {
    outPath = "/nix/store/00000000000000000000000000000000-fake-src";
  };
  norm = root: entries: nput.normalizeManifest { inherit lib root entries; };

  # store×symlink（採用）/ store×copy（除外）/ out-of-store×symlink（除外）が混在する manifest。
  # normalizeManifest は target 辞書順で配列化するため norm.entries の順は
  # [".config/copy", ".config/out", ".config/sym", ".config/sym2"]。
  mixed = norm nput.projectRoot {
    ".config/copy" = {
      src = fakeSrc;
      method = "copy";
    };
    ".config/out" = {
      src = nput.mkOutOfStoreSymlink "/home/me/dotfiles/x";
    };
    ".config/sym" = {
      src = fakeSrc;
    };
    ".config/sym2" = {
      src = fakeSrc;
    };
  };

  farm = nput.__internal.farmEntries lib mixed.entries;

  # manifest.nix:111-113 と同一の式で anchorLines を再構成し、実 helper（farmEntries /
  # anchorName）と escapeShellArg の合成結果がリテラル期待値と一致することを検証する。
  anchorLines = lib.concatMapStringsSep "\n" (
    e: "ln -s ${lib.escapeShellArg e.src} \"$out/${nput.__internal.anchorName lib e.target}\""
  ) farm;
in
{
  # farmEntries は store×symlink のみを採用し、copy / out-of-store を除外する（→ ADR-0016）。
  testFarmEntriesIncludesOnlyStoreSymlink = {
    expr = map (e: e.target) farm;
    expected = [
      ".config/sym"
      ".config/sym2"
    ];
  };

  # store×symlink が皆無なら farmEntries は空（copy / out-of-store だけではアンカーを持たない）。
  testFarmEntriesEmptyWhenNoStoreSymlink = {
    expr =
      nput.__internal.farmEntries lib
        (norm nput.projectRoot {
          ".config/copy" = {
            src = fakeSrc;
            method = "copy";
          };
          ".config/out" = {
            src = nput.mkOutOfStoreSymlink "/home/me/dotfiles/x";
          };
        }).entries;
    expected = [ ];
  };

  # GC アンカー名は target の sha256 短縮 hex（32 文字・固定長・FS-safe・衝突回避・→ ADR-0016）。
  testAnchorNameSha256ShortHex = {
    expr = nput.__internal.anchorName lib ".config/sym";
    expected = "029f105e76667554409c2422b0f61f1c";
  };

  # anchorLines は farm エントリごとに `ln -s <src> "$out/<anchorName>"` を改行連結する。
  # src（clean なストアパス）は escapeShellArg でそのまま、target ごとに anchor 名が変わる。
  testAnchorLinesGenerated = {
    expr = anchorLines;
    expected = ''
      ln -s /nix/store/00000000000000000000000000000000-fake-src "$out/029f105e76667554409c2422b0f61f1c"
      ln -s /nix/store/00000000000000000000000000000000-fake-src "$out/1fa2d3541e7cab32b4961dfbdb6f1095"'';
  };
}
