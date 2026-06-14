# listFilesInSrc { src, subpath ? "." } -> { filename = fileType; }（→ ADR-0008, ADR-0009, ADR-0023, ADR-0024）。
#
# src 内の subpath ディレクトリを `builtins.readDir` で走査し、readDir 互換の
# `{ filename = "regular" | "directory" | "symlink" | "unknown"; }` を返す。
# entries の動的生成（subdir 列挙 → 1 entry 展開）に使う（→ docs/spec.md 応用節）。
#
# src は path（store パス）限定。eval 時に readDir するため次を型で弾く：
#   - out-of-store marker: 実行時解決の入れ物で eval 時にパスへ展開できない（→ ADR-0023）。
#   - set（生 derivation）: 未 realise の derivation を readDir すると IFD（import-from-derivation）を
#     誘発し flake pure eval が破綻する（→ ADR-0024）。entries の src が set を許容するのは
#     engine 実行時解決だからで、eval 時 readDir の本関数とは非対称。
# flake = false の flake input（`{ outPath = …; }` の既 realise store path）は path 同様に通す。
#
# subpath は readDir するため**ディレクトリ限定**。非 dir は eval エラーで弾く（readDir の IO エラーは
# tryEval で捕捉できないため、`builtins.readFileType` で先に判定し clean な throw に変換する）。
#
# 依存なし（nixpkgs.lib すら引かない・builtins のみ）。型判別は lib/types.nix と独立に builtins で行う：
# srcType は set を許容するが本関数は IFD 回避のため弾くので、optionType は共有できない。
{
  src,
  subpath ? ".",
}:
let
  isAttrs = builtins.isAttrs src;
  isOutOfStoreMarker = isAttrs && (src._nputMarker or null) == "outOfStore";
  isDerivation = isAttrs && (src.type or null) == "derivation";
  # path（store パス）または outPath 付き非 derivation attrset（flake input）を許容する。
  isPathLike =
    builtins.typeOf src == "path" || (isAttrs && src ? outPath && !isDerivation && !isOutOfStoreMarker);
in
if isOutOfStoreMarker then
  throw "nput: listFilesInSrc の src は path 限定です（out-of-store marker は eval 時にパスへ展開できないため不可・→ ADR-0023）"
else if isDerivation then
  throw "nput: listFilesInSrc の src は path 限定です（set（derivation）は readDir が IFD を誘発するため不可・→ ADR-0024）"
else if !isPathLike then
  throw "nput: listFilesInSrc の src は path（store パス）限定です（→ ADR-0023, ADR-0024）"
else if builtins.readFileType "${src}/${subpath}" != "directory" then
  throw "nput: listFilesInSrc の subpath はディレクトリ限定です（非 dir: ${subpath}・→ ADR-0008）"
else
  builtins.readDir "${src}/${subpath}"
