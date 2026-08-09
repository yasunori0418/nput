---
id: "TC-36ea3609-d52e-42d4-975c-40fb89b23919"
type: test_condition
name: "rollback が FS を前世代へ再収束させてから最後にポインタを移す"
mitigates:
  - "RISK-fbf029f6-866c-4a08-a4eb-f09e3c7e907e"
---
# TC-36ea3609: rollback が FS を前世代へ再収束させてから最後にポインタを移す

## テスト条件

`Rollback` を tmpdir 上の実 FS で駆動し、現世代 N から N-1 への FS 再収束を検証する。
world は「gen1 と gen2 の link farm を書き、`profile-N-link` 世代リンクを張り、profile を
gen2 へ向け、FS を gen2 の配置状態にする」形で組む。

**素の再収束** — gen1 = {a, b}・gen2（current）= {a, c} のとき、c が stale 除去され、
b が再配置され、a は据え置かれる。ポインタ移動は FS 収束の後に起きる
（先に動かすと baseline が N-2 へずれる → REQ-0e341430）。

**祖先 symlink からの復帰** — 世代 N が配下の per-file entries を祖先 symlink 1 本へ
移行していた場合、N-1 へ戻るには plan の PreRemove がその祖先 symlink を先に除去し、
配下の子 entries を実ディレクトリへ置き直す必要がある。PreRemove を落とすと、
`ensureParentDir` の `MkdirAll` が symlink 上で no-op になり、子の `os.Symlink` が
symlink 越しに store を指して EEXIST / EROFS で落ちる（→ ADR-0046、issue #173）。
収束後に当該パスが実ディレクトリへ戻っていることを確認する。

**root 解決** — `rootKind` から絶対 root を得る側（home は `$HOME`、`--root` 上書きは
rootKind によらず優先）も同じ層の条件として併せて検証する。

## 対応する CASE

CASE-364ebb9d（`internal/engine/generations_test.go`）。
