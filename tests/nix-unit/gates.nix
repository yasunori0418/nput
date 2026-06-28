# nix-unit: eval 時に throw する検査ゲート群（systemRoot 未実装・copy×outOfStore・絶対/`..` escape・
# 重複 target・未知キー・素文字列 src・共有 entriesType）をアサートする
# （→ ADR-0008, ADR-0010, ADR-0013, ADR-0014, ADR-0019, ADR-0024）。
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
    expectedError.msg = "same target";
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

  # modules/common.nix が共有する entriesType（attrsOf (submodule entryModule)）を
  # evalModules で直接検査する。common.nix は同じ lib/types.nix の entriesType を使うため、
  # 未知キー（タイポ・旧名）はモジュール経路でも strict submodule で eval エラーになる
  # （→ AC「common.nix の entry submodule が lib/types.nix と共有され未知キーが eval エラー」・ADR-0010, ADR-0014）。
  testSharedEntriesTypeUnknownKey = {
    expr =
      let
        t = import ../../lib/types.nix lib;
        evaluated = lib.evalModules {
          modules = [
            { options.entries = lib.mkOption { type = t.entriesType; }; }
            {
              entries.".config/x" = {
                src = fakeSrc;
                bogus = true; # 未知キー
              };
            }
          ];
        };
      in
      evaluated.config.entries;
    expectedError.type = "ThrownError";
    expectedError.msg = "bogus";
  };
}
