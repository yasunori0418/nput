# nput lib 公開 API のまとめ（→ docs/design.md「flake.nix outputs 設計」）。
#
# nixpkgs.lib のみ依存（home-manager / NixOS / nix-darwin は引かない・→ ADR-0006）。
# マーカー群は純 attrset 構築子で依存なし。mkManifest は引数 attrset の `pkgs` から
# linkFarm 相当（runCommandLocal）と `pkgs.lib` を得て derivation を組む。
let
  markers = import ./out-of-store.nix;
  manifest = import ./manifest.nix;
in
{
  # lib.mkManifest { pkgs, root, entries } -> derivation（manifest.json + symlink farm・→ ADR-0006, ADR-0023）
  inherit (manifest) mkManifest;

  # 検査・正規化の純データ関数。nix-unit / namaka の単体対象 + 将来 modules から再利用（→ ADR-0010）。
  # normalizeManifest { lib, root, entries } -> attrset
  inherit (manifest) normalizeManifest;

  # lib.mkOutOfStoreSymlink "/abs/path" -> marker（src に渡す・→ ADR-0001）
  # lib.projectRoot / homeRoot / systemRoot -> marker（root に渡す・→ ADR-0004, ADR-0005, ADR-0007）
  inherit (markers)
    mkOutOfStoreSymlink
    projectRoot
    homeRoot
    systemRoot
    ;
}
