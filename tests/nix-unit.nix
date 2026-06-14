# nix-unit: normalizeManifest のデフォルト適用・manifest 構造の不変条件・検査ゲートを
# アサートする（→ ADR-0006, ADR-0010, ADR-0013, ADR-0019, ADR-0024）。
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
  # ---- manifest 構造の不変条件 ----------------------------------------------
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

  # ---- 検査ゲート（eval 時に throw する不変条件）----------------------------
  # systemRoot は未実装（→ ADR-0013）。
  testSystemRootUnimplemented = {
    expr =
      (norm nput.systemRoot {
        "x" = {
          src = fakeSrc;
        };
      }).entries;
    expectedError.type = "ThrownError";
    expectedError.msg = "system mode";
  };

  # method = "copy" かつ out-of-store marker は意図矛盾（→ ADR-0013）。
  testCopyOutOfStoreRejected = {
    expr =
      (norm nput.projectRoot {
        ".config/x" = {
          src = nput.mkOutOfStoreSymlink "/home/me/dots";
          method = "copy";
        };
      }).entries;
    expectedError.type = "ThrownError";
    expectedError.msg = "out-of-store";
  };

  # target が絶対パス（→ ADR-0019）。
  testAbsoluteTargetRejected = {
    expr =
      (norm nput.projectRoot {
        "/etc/x" = {
          src = fakeSrc;
        };
      }).entries;
    expectedError.type = "ThrownError";
    expectedError.msg = "target";
  };

  # target が `..` で root の外（→ ADR-0019）。
  testEscapingTargetRejected = {
    expr =
      (norm nput.projectRoot {
        "../../etc/x" = {
          src = fakeSrc;
        };
      }).entries;
    expectedError.type = "ThrownError";
    expectedError.msg = "target";
  };

  # subpath が `..` で src の外（→ ADR-0019）。
  testEscapingSubpathRejected = {
    expr =
      (norm nput.projectRoot {
        ".config/x" = {
          src = fakeSrc;
          subpath = "../escape";
        };
      }).entries;
    expectedError.type = "ThrownError";
    expectedError.msg = "subpath";
  };

  # 別キーで target を同値に明示上書きした衝突（→ ADR-0024）。
  testDuplicateTargetRejected = {
    expr =
      (norm nput.projectRoot {
        "a" = {
          src = fakeSrc;
          target = ".config/same";
        };
        "b" = {
          src = fakeSrc;
          target = ".config/same";
        };
      }).entries;
    expectedError.type = "ThrownError";
    expectedError.msg = "同一 target";
  };

  # 未知キー（タイポ / 旧名）は submodule strict で弾く（→ ADR-0008, ADR-0010）。
  testUnknownKeyRejected = {
    expr =
      (norm nput.projectRoot {
        ".config/x" = {
          src = fakeSrc;
          source = "skills/nix"; # 旧名（正しくは subpath）
        };
      }).entries;
    expectedError.type = "ThrownError";
    expectedError.msg = "source";
  };

  # 素の文字列 src は拒否（out-of-store は marker で opt-in・→ ADR-0001）。
  testStringSrcRejected = {
    expr =
      (norm nput.projectRoot {
        ".config/x" = {
          src = "/home/me/dots";
        };
      }).entries;
    expectedError.type = "ThrownError";
    expectedError.msg = "src";
  };
}
