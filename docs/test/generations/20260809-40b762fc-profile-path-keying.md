---
id: "TC-40b762fc-3b48-471c-b495-c5ffe1f584b5"
type: test_condition
name: "state 基底・profileDir のキーイング・世代リンク名が root と mode ごとに正しく決まる"
mitigates:
  - "RISK-ac5cd9b4-58f9-43d8-9fae-66f7b5940beb"
---
# TC-40b762fc: state 基底・profileDir のキーイング・世代リンク名が root と mode ごとに正しく決まる

## テスト条件

profile の on-disk レイアウトを決めるパス解決層を、環境変数と入力の組み合わせに対する
関数として検証する。

**`<state>` の解決** — `$XDG_STATE_HOME` があればそれを使い（`$HOME` の値によらず）、
無ければ `$HOME/.local/state` へ落ちる。`$HOME` も解決できなければエラーにする
（黙って相対パスや空文字を使わない）。

**`<roothash>` の性質** — 同じ絶対 root に対して決定的で、長さが固定であること。
決定的でなければ apply のたびに別の世代系列が生まれ、長さが揺れれば FS 上の名前として
使えない。

**キーイングの分岐** — mode と `--root` の有無で、`<name>` 直キーと `<roothash>` キーの
どちらを取るかが決まる。

| 入力 | キー |
|---|---|
| project mode | `<roothash>` |
| home mode（`--root` なし）| `<name>` 直キー |
| home mode + `--root` 上書き | `<roothash>` |
| fixed root mode | `<roothash>` |

**世代リンクの命名** — profile リンクに対する `profile-N-link` の形が `nix-env` の
規約どおりで、同一ディレクトリ内の兄弟として並ぶこと。

## 対応する CASE

CASE-ff4a842e（`internal/paths/paths_test.go`）。root 解決そのもの（`rootKind` と
`--root` から絶対 root を得る側）は CASE-364ebb9d が併せて覆う。
