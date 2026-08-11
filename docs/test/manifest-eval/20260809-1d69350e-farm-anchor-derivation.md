---
id: "TC-1d69350e-db3c-4d74-a24e-7a3fabb31b0a"
type: test_condition
name: "GC アンカー対象の抽出条件と farm の組み立て結果をアサートする"
mitigates:
  - "RISK-f67a0883-950a-4458-8b6d-1f95cb039cb1"
---
# TC-1d69350e: アンカー対象の抽出と farm の組み立てをアサートする

## テスト条件

symlink farm が GC アンカー専用であることを、対象の抽出と組み立て結果で見る。
アンカー名そのものの形式・決定性は TC-a6a14739 の担当で、ここでは扱わない。

**抽出条件**: store-backed かつ `method = "symlink"` の entry だけがアンカー対象になること。
copy（世代外・置き切り）と out-of-store（store 非依存）は対象外であること。対象が皆無なら
空になること。store×symlink / store×copy / out-of-store×symlink が混在する manifest を 1 つ
用意し、採用と除外を同時に見る。

**組み立て結果**: 抽出とアンカー名の合成が `ln -s <src> "$out/<anchor>"` の行として期待どおり
に並び、target ごとに異なるアンカー名を持つこと。derivation は組み立てず、生成式そのもの
（`__internal.anchorLines`・→ Issue #289）を叩く。

生成式の複製は持たない。回帰検知は次の 2 層で担う。

- **単体**: 生成式を最小の手組みエントリ列へ適用し、リテラルの期待値で 1 行の形・改行連結・
  `escapeShellArg` の作用・空入力を固定する。manifest を経由しないので期待値を式で組む必要が
  なく、生成式を変えればここが落ちる
- **配線**: 混在 manifest を抽出経路へ通し、生成の入力が farm 対象エントリだけであることを、
  同じ生成式を独立に選んだ対象 target 列へ適用した期待値と突き合わせる。生成式の変更には
  両辺が揃って追随するので、ここで落ちるのは抽出と生成の繋ぎ違いだけである

アンカー名は先に対象 target の 1 つについて外部固定の既知 hash リテラルとして単独で確定させ、
単体の期待値の足場にする。

## 覆う CASE

- CASE-3a1f403d（`tests/nix-unit/farm-entries.nix`）

## 出典

`tests/nix-unit/farm-entries.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
