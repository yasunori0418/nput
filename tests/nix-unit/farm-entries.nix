# nix-unit: symlink farm のアンカー対象抽出（`__internal.farmEntries`）と GC アンカー名
# （`__internal.anchorName`）・アンカー配置シェルの生成（`__internal.anchorLines`）を
# アサートする（→ ADR-0016, ADR-0019, #58, #71, #75, #289）。
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

  # ---- anchorLines の単体（内容の正しさをここで固定する）------------------------------
  # 生成式そのものを最小の手組み入力に適用し、静的な期待値で押さえる。manifest を経由しない
  # ので、期待値は共有式ではなくリテラルで書ける（同語反復にならない）。
  #
  # 1 行の形は `ln -s <escapeShellArg src> "$out/<anchorName target>"`。src は escapeShellArg
  # を通るので、clean なストアパスは素通りし、空白・記号を含むパスは quote される。
  testAnchorLinesSingleEntry = {
    expr = nput.__internal.anchorLines lib [
      {
        src = "/nix/store/00000000000000000000000000000000-fake-src";
        target = ".config/sym";
      }
    ];
    expected = ''ln -s /nix/store/00000000000000000000000000000000-fake-src "$out/029f105e76667554409c2422b0f61f1c"'';
  };

  # 複数エントリは改行連結（末尾に改行は付かない）。target ごとに anchor 名が変わる。
  testAnchorLinesJoinsWithNewline = {
    expr = nput.__internal.anchorLines lib [
      {
        src = "/nix/store/00000000000000000000000000000000-fake-src";
        target = ".config/sym";
      }
      {
        src = "/nix/store/00000000000000000000000000000000-fake-src";
        target = ".config/sym2";
      }
    ];
    expected = ''
      ln -s /nix/store/00000000000000000000000000000000-fake-src "$out/029f105e76667554409c2422b0f61f1c"
      ln -s /nix/store/00000000000000000000000000000000-fake-src "$out/1fa2d3541e7cab32b4961dfbdb6f1095"'';
  };

  # src 側は escapeShellArg を通る。空白・記号を含むパスが shell へ素通りしないこと。
  testAnchorLinesEscapesSrc = {
    expr = nput.__internal.anchorLines lib [
      {
        src = "/nix/store/x y & z";
        target = ".config/sym";
      }
    ];
    expected = ''ln -s '/nix/store/x y & z' "$out/029f105e76667554409c2422b0f61f1c"'';
  };

  # アンカー対象が皆無なら空文字列（derivation の builder に空行だけが残る）。
  testAnchorLinesEmptyWhenNoEntries = {
    expr = nput.__internal.anchorLines lib [ ];
    expected = "";
  };

  # ---- farm への配線（生成式は共有し、「何が入力に流れるか」だけを見る）------------------
  # mkManifest は pkgs を要るのでここでは組めない。代わりに manifest.nix と同じ経路
  # （normalizeManifest → farmEntries → anchorLines）を辿り、アンカー行の入力が混在 manifest
  # の全 entry ではなく farm 対象だけであることを見る。期待値は同じ生成式を「独立に選んだ対象
  # entry 列」へ適用して組むので、生成式を変えれば両辺が揃って動き（回帰検知は上の単体テストの
  # 担当）、ここでは選抜の誤りだけが落ちる。
  testAnchorLinesWiredToFarmEntries = {
    expr = nput.__internal.anchorLines lib farm;
    expected = nput.__internal.anchorLines lib (
      lib.filter (
        e:
        lib.elem e.target [
          ".config/sym"
          ".config/sym2"
        ]
      ) mixed.entries
    );
  };
}
