---
id: "TC-1d69350e-db3c-4d74-a24e-7a3fabb31b0a"
type: test_condition
name: "GC アンカー対象の抽出条件と farm の組み立て結果をアサートする"
mitigates:
  - "RISK-f67a0883-950a-4458-8b6d-1f95cb039cb1"
---
# TC-1d69350e: アンカー対象の抽出と farm の組み立てをアサートする

## テスト条件

symlink farm が GC アンカー専用であることを、対象の抽出と derivation の組み立てで見る。
アンカー名そのものの形式・決定性は TC-a6a14739 の担当で、ここでは扱わない。

**抽出条件**: store-backed かつ `method = "symlink"` の entry だけがアンカー対象になること。
copy（世代外・置き切り）と out-of-store（store 非依存）は対象外であること。対象が皆無なら
空になること。store×symlink / store×copy / out-of-store×symlink が混在する manifest を 1 つ
用意し、採用と除外を同時に見る。

**組み立て結果**: 抽出とアンカー名の合成が farm derivation の
`ln -s <src> "$out/<anchor>"` 行として期待どおりに並び、target ごとに異なるアンカー名を
持つこと。

## 覆う CASE

- CASE-3a1f403d（`tests/nix-unit/farm-entries.nix`）

## 出典

`tests/nix-unit/farm-entries.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
