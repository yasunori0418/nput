# normalizeManifest（純データ・検査ゲート）+ mkManifest（derivation 生成）（→ ADR-0006, ADR-0010, ADR-0016, ADR-0019, ADR-0023）。
#
# 2 段に分ける（→ ADR-0010）:
#   - normalizeManifest { lib, root, entries } -> attrset
#       evalModules 検査・デフォルト適用・パス安全性 / クロスフィールド throwIf・marker タグ → clean enum 変換。
#       derivation の外に出すことで nix-unit / namaka の単体対象にする。
#   - mkManifest { pkgs, root, entries } -> derivation
#       normalizeManifest の出力を manifest.json に書き、store src への symlink farm を組む。
#
# 本スライスの最小スコープ: root = projectRoot のみ / src = path・set の store-backed symlink entry のみ
# （→ Issue #5）。型・throwIf は将来スライス（home / copy / out-of-store）も見据えて完全形で定義する。
let
  # ---- パス安全性（→ ADR-0019）----------------------------------------------
  # target は root 相対・subpath は src 内相対。絶対パス（`/` 始まり）と
  # `..` で外へ出るパスを eval 時に拒否する。root の実体値（実行時解決）に依らず静的に判定可能。
  pathChecks =
    lib:
    let
      isAbsolute = lib.hasPrefix "/";
      # `..` を辿って深さが負になる（base の外へ出る）かを判定する。
      escapesBase =
        p:
        let
          comps = lib.filter (c: c != "" && c != ".") (lib.splitString "/" p);
          step =
            acc: c:
            if acc.bad then
              acc
            else if c == ".." then
              {
                bad = acc.depth == 0;
                depth = if acc.depth == 0 then 0 else acc.depth - 1;
              }
            else
              {
                bad = false;
                depth = acc.depth + 1;
              };
        in
        (lib.foldl' step {
          bad = false;
          depth = 0;
        } comps).bad;
    in
    {
      isUnsafe = p: isAbsolute p || escapesBase p;
    };

  normalizeManifest =
    {
      lib,
      root,
      entries ? { },
    }:
    let
      t = import ./types.nix lib;
      checks = pathChecks lib;

      # 両経路（CLI 直呼び / モジュール）で検査を効かせるため mkManifest 自身が evalModules を回す（→ ADR-0010）。
      evaluated = lib.evalModules {
        modules = [
          {
            options = {
              root = lib.mkOption { type = t.rootType; };
              entries = lib.mkOption {
                type = t.entriesType;
                default = { };
              };
            };
            config = { inherit root entries; };
          }
        ];
      };
      cfg = evaluated.config;

      # root の marker タグ → clean enum（→ ADR-0010）。
      rootInfo =
        if t.isRootMarker cfg.root then
          { rootKind = cfg.root.kind; }
        else
          {
            rootKind = "fixed";
            root = cfg.root;
          };

      # entry の marker タグ → clean enum + 解決済み src 文字列（→ ADR-0010）。
      # 属性キー = target なので attrNames の辞書順で決定的に配列化する（Go は配列を読む・→ ADR-0014）。
      resolveEntry =
        e:
        let
          srcInfo =
            if t.isOutOfStoreMarker e.src then
              {
                srcKind = "outOfStore";
                src = e.src.path;
              }
            else
              {
                srcKind = "store";
                src = toString e.src;
              };
        in
        {
          inherit (srcInfo) srcKind src;
          inherit (e) subpath target method;
        };

      normEntries = map (key: resolveEntry cfg.entries.${key}) (lib.attrNames cfg.entries);

      targets = map (e: e.target) normEntries;

      # ---- クロスフィールド / パス検査（→ ADR-0013, ADR-0019, ADR-0024）-------
      assertions = lib.concatLists [
        # systemRoot は未実装（→ ADR-0013）。
        (lib.optional (
          rootInfo.rootKind == "system"
        ) "nput: root = systemRoot (system mode) は未実装です（→ ADR-0013）")
        # method = "copy" かつ out-of-store marker は意図矛盾（→ ADR-0013）。
        (map (
          e: "nput: method = \"copy\" と out-of-store marker は同時指定できません（target: ${e.target}・→ ADR-0013）"
        ) (lib.filter (e: e.method == "copy" && e.srcKind == "outOfStore") normEntries))
        # 別キーで target を同値に明示上書きした衝突（→ ADR-0024）。
        (lib.optional (
          lib.length targets != lib.length (lib.unique targets)
        ) "nput: 複数 entry が同一 target に解決されました（→ ADR-0024）")
        # target / subpath の絶対パス・`..` エスケープ拒否（→ ADR-0019）。
        (map (e: "nput: target が不正です（絶対パスまたは `..` で root の外）: ${e.target}（→ ADR-0019）") (
          lib.filter (e: checks.isUnsafe e.target) normEntries
        ))
        (map (
          e: "nput: subpath が不正です（絶対パスまたは `..` で src の外）: ${e.subpath}（target: ${e.target}・→ ADR-0019）"
        ) (lib.filter (e: checks.isUnsafe e.subpath) normEntries))
      ];

      result = {
        schemaVersion = 1;
        root = rootInfo;
        entries = normEntries;
      };
    in
    # 全 assertion を throwIf で評価ゲートに通す。
    lib.foldl' (acc: msg: lib.throwIf true msg acc) result assertions;

  # symlink farm の GC アンカー名 = target の sha256 短縮 hex（固定長・FS 安全・衝突不可能・→ ADR-0016）。
  anchorName = lib: target: lib.substring 0 32 (builtins.hashString "sha256" target);

  mkManifest =
    {
      pkgs,
      root,
      entries ? { },
    }:
    let
      lib = pkgs.lib;
      norm = normalizeManifest { inherit lib root entries; };

      manifestJson = pkgs.writeText "manifest.json" (builtins.toJSON norm);

      # farm アンカーは「store-backed かつ method = symlink」の entry 限定（→ ADR-0016, ADR-0019）。
      # out-of-store / copy は farm アンカーを持たない（copy は世代外・place-once で store から独立）。
      farmEntries = lib.filter (e: e.srcKind == "store" && e.method == "symlink") norm.entries;

      anchorLines = lib.concatMapStringsSep "\n" (
        e: "ln -s ${lib.escapeShellArg e.src} \"$out/${anchorName lib e.target}\""
      ) farmEntries;
    in
    # derivation は manifest.json（engine の入力契約）+ store src への symlink farm（GC アンカー）を含む（→ ADR-0006）。
    pkgs.runCommandLocal "nput-manifest"
      {
        # CLI が build 前に `nix eval … .rootKind` で読む（→ ADR-0023）。
        passthru = {
          inherit (norm.root) rootKind;
        }
        // lib.optionalAttrs (norm.root ? root) { inherit (norm.root) root; };
      }
      ''
        mkdir -p "$out"
        cp ${manifestJson} "$out/manifest.json"
        ${anchorLines}
      '';
in
{
  inherit normalizeManifest mkManifest;
}
