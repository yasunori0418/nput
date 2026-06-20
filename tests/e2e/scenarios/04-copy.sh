#!/usr/bin/env bash
# copy place-once / out-of-store: copy 配置が通常ファイル（書込可）になること・place-once の冪等性
# （ローカル編集が再 apply で破棄されないこと）・out-of-store marker が live symlink になることをアサート。
set -euo pipefail
source "$(dirname "$0")/../lib.sh"
e2e_isolate

LIVE="$E2E_WORK/live"
mkdir -p "$LIVE"
echo "LIVE1" >"$LIVE/note.txt"

PROJ="$E2E_WORK/cfg"
mkdir -p "$PROJ/srcrepo/data"
echo "ORIG" >"$PROJ/srcrepo/data/conf"

cat >"$PROJ/flake.nix" <<EOF
{
$(e2e_flake_inputs)
  outputs = { self, nixpkgs, nput }: {
    nput = nixpkgs.lib.genAttrs $E2E_SYSTEMS (system: {
      home = nput.lib.mkManifest {
        pkgs = nixpkgs.legacyPackages.\${system};
        root = nput.lib.homeRoot;
        entries = {
          ".cfg/copied" = { src = ./srcrepo; subpath = "data"; method = "copy"; };
          ".cfg/live"   = { src = nput.lib.mkOutOfStoreSymlink "$LIVE"; };
        };
      };
    });
  };
}
EOF

cd "$PROJ"
git init -q
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm init

e2e_step "apply（copy + out-of-store）"
nput apply home

e2e_step "copy: store symlink ではなく通常ディレクトリ・書込可・内容コピー"
assert_real_dir "$HOME/.cfg/copied"
assert_file_eq "$HOME/.cfg/copied/conf" "ORIG"
assert_writable "$HOME/.cfg/copied/conf"

e2e_step "out-of-store: live ディレクトリへの symlink"
assert_symlink "$HOME/.cfg/live" "$LIVE"
assert_file_eq "$HOME/.cfg/live/note.txt" "LIVE1"

e2e_step "place-once 冪等: copy したファイルを編集 → 再 apply で編集が残る（上書きされない）"
echo "EDITED" >"$HOME/.cfg/copied/conf"
nput apply home
assert_file_eq "$HOME/.cfg/copied/conf" "EDITED"

e2e_step "out-of-store は live: src 側の変更が target からそのまま見える"
echo "LIVE2" >"$LIVE/note.txt"
assert_file_eq "$HOME/.cfg/live/note.txt" "LIVE2"

e2e_finish
