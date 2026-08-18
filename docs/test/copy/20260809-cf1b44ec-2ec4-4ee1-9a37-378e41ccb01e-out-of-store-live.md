---
id: "TC-cf1b44ec-2ec4-4ee1-9a37-378e41ccb01e"
type: test_condition
name: "out-of-store marker が live symlink として配置され、src 側の変更が透過すること"
mitigates:
  - "RISK-ffb5100d-c553-4dd8-937a-1263ad84a8c7"
---
# TC-cf1b44ec: out-of-store の live 性

`mkOutOfStoreSymlink` で作った marker が、コピーではなく marker の絶対パスを指す symlink
として配置されることを検証する。配置後に src 側のファイルを書き換えたとき、target 越しに
その変更がそのまま見えることまでを条件とする（コピーとして落ちていれば見えない）。

nix store を介さず live なディレクトリを指すことが目的であるため、実 nix を通した e2e で
検証する。engine 単体での out-of-store の分類・stale 除去は `engine-core` 対象の担当。
