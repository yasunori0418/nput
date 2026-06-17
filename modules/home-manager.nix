# home-manager モジュール: home.activation から nput エンジンを起動する薄い配線
# （→ ADR-0002, ADR-0003, ADR-0007, docs/spec.md「モジュール別動作仕様」）。
#
# 配置ロジックは持たず、home.file / systemd.tmpfiles へは翻訳しない。配置・stale 除去は
# 全層共通の nput エンジンが所有する（→ ADR-0003）。本モジュールは
#   1. root = homeRoot を pin（利用者は root を再指定しない・→ ADR-0007）、
#   2. nput.entries から link-farm（manifest.json + GC アンカー）を mkManifest でビルド、
#   3. home.activation でその link-farm を `nput apply --manifest` に渡してエンジンを起動
# するだけに徹する。
#
# 世代は nput 自前 profile（内部機構・前世代マニフェスト + stale 追跡）に乗る。MVP は固定名
# `default` の単一 profile（<state>/nix/profiles/nput/default）で role 分離は不可。ユーザー
# 向け rollback は host（home-manager --rollback）に一本化し、`nput rollback` は公開しない
# （host rollback は旧 config を再 activate して nput を再 kick することで FS を自動収束させる）
# （→ ADR-0002, ADR-0024, ADR-0025）。
#
# nputPackage（pin 版 nput CLI）は flake.nix の homeManagerModules.default 配線が
# _module.args として注入する（→ flake.nix）。nputLib は lib/ を直接 import する
# （nixpkgs.lib のみ依存・home-manager に依存しない・→ ADR-0006）。
{
  config,
  lib,
  pkgs,
  nputPackage,
  ...
}:
let
  cfg = config.nput;
  nputLib = import ../lib;

  # nput.entries から link-farm derivation（manifest.json + symlink farm）を生成する。
  # root は homeRoot を pin（実体 $HOME の解決は engine 実行時・marker は eval 時に展開しない）。
  manifest = nputLib.mkManifest {
    inherit pkgs;
    root = nputLib.homeRoot;
    entries = cfg.entries;
  };
in
{
  imports = [ ./common.nix ];

  config = lib.mkIf cfg.enable {
    # home.activation から engine を起動する。ビルド済み link-farm を --manifest で渡すため
    # activation 内で nix eval/build は行わない（entrypoint 経路ではない）。writeBoundary 後に
    # 走らせ、home.file の symlink 配置と順序衝突しないようにする。`run` は home-manager の
    # activation ヘルパで --dry-run（DRY_RUN_CMD）を尊重する。
    home.activation.nput = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
      run ${lib.getExe nputPackage} apply --manifest ${manifest}
    '';
  };
}
