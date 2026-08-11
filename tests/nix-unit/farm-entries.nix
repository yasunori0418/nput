# nix-unit: symlink farm のアンカー対象抽出（`__internal.farmEntries`）と GC アンカー名
# （`__internal.anchorName`）・アンカー配置シェルの生成（`__internal.anchorLines`）、および
# それらを mkManifest がビルドスクリプトへ埋める配線をアサートする
# （→ ADR-0016, ADR-0019, #58, #71, #75, #289）。
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
  # 抽出テストと配線テストが同じ入力を見ていることを構文で保つため、entries は 1 箇所で宣言し
  # normalizeManifest 経由・mkManifest 経由の両方から参照する。
  mixedEntries = {
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

  # normalizeManifest は target 辞書順で配列化するため mixed.entries の順は
  # [".config/copy", ".config/out", ".config/sym", ".config/sym2"]。
  mixed = norm nput.projectRoot mixedEntries;

  farm = nput.__internal.farmEntries lib mixed.entries;

  # copy しか無い manifest（アンカー対象が皆無になる入力）。
  copyOnlyEntries = {
    ".config/copy" = {
      src = fakeSrc;
      method = "copy";
    };
  };

  # 配線検証用の fake pkgs。mkManifest が pkgs から使うのは lib / writeText / runCommandLocal の
  # 3 つだけなので、後 2 者を「引数をそのまま持ち帰る」double に差し替えると、derivation を
  # 組まずにビルドスクリプト本文を純評価で取り出せる（src の fake flake-input double と同じ
  # イディオム）。実ビルドによる検証は評価テストの枠を超えるため採らない（→ #289）。
  #
  # 持ち帰り先を `buildCommand` と名付けるのは実 nixpkgs の runCommandLocal に合わせるため
  # （実 derivation の `.builder` は bash 本体のパスであって本文ではない）。
  fakePkgs = {
    inherit lib;
    writeText = name: _text: "/nix/store/fake-${name}";
    runCommandLocal = _name: _attrs: script: { buildCommand = script; };
  };

  buildCommandOf =
    entries:
    (nput.mkManifest {
      pkgs = fakePkgs;
      root = nput.projectRoot;
      inherit entries;
    }).buildCommand;
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

  # アンカー対象が皆無なら空文字列（埋め込み先には空行だけが残る）。
  testAnchorLinesEmptyWhenNoEntries = {
    expr = nput.__internal.anchorLines lib [ ];
    expected = "";
  };

  # ---- farm derivation への配線（ビルドスクリプトに何が埋まるかを見る）--------------------
  # mkManifest が埋めるアンカー行が、生成式へ farm 対象**だけ**を通した結果であること。fake
  # pkgs 経由でビルドスクリプト本文を取り出し、manifest.json のコピーに続いてアンカー行が並ぶ
  # 全体を突き合わせる。期待値のアンカー行は共有式で組むので、生成式を変えれば両辺が揃って動き
  # （内容の正しさは上の単体テストが固定する）、ここで落ちるのは配線の誤り
  # （フィルタ漏れ・生成結果の埋め込み忘れ・順序の崩れ）だけである。
  testBuildCommandEmbedsAnchorLinesForFarmEntriesOnly = {
    expr = buildCommandOf mixedEntries;
    expected = ''
      mkdir -p "$out"
      cp /nix/store/fake-manifest.json "$out/manifest.json"
      ${nput.__internal.anchorLines lib farm}
    '';
  };

  # アンカー対象が皆無なら `ln -s` は 1 行も現れない。空の生成結果を埋めた跡（空行）が残るか
  # 否かは整形の都合なので、行の有無だけを見て全文一致には依存しない。
  testBuildCommandHasNoAnchorLinesWhenNoFarmEntries = {
    expr = lib.filter (l: lib.hasPrefix "ln -s " l) (
      lib.splitString "\n" (buildCommandOf copyOnlyEntries)
    );
    expected = [ ];
  };

  # ただしアンカー対象が皆無でも manifest.json のコピーまでは行う（アンカーが無いことと
  # ビルドスクリプトが空になることを取り違えない）。
  testBuildCommandStillCopiesManifestWhenNoFarmEntries = {
    expr = lib.filter (l: l != "") (lib.splitString "\n" (buildCommandOf copyOnlyEntries));
    expected = [
      ''mkdir -p "$out"''
      ''cp /nix/store/fake-manifest.json "$out/manifest.json"''
    ];
  };
}
