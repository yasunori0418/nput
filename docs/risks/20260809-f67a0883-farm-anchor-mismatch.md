---
id: "RISK-f67a0883-950a-4458-8b6d-1f95cb039cb1"
type: risk
name: "GC アンカーの対象と名前を取り違えて世代が回収不能になる"
likelihood: medium
impact: high
level: high
threatens:
  - "REQ-62eda895-efd4-4eaf-a58b-600e8637da75"
  - "REQ-b12fc3c0-d7fe-4003-922c-f3ac0d969b66"
  - "REQ-60e6b49c-9ba1-4552-a0ec-d340421ec281"
---
# RISK-f67a0883: GC アンカーの対象と名前を取り違えて世代が回収不能になる

## リスク

symlink farm は GC アンカー専用で、アンカーは store-backed な symlink entry に限る
（REQ-b12fc3c0）。アンカー名は target のハッシュ（REQ-62eda895）で、farm derivation は
`mkManifest` の返り値に含まれる（REQ-60e6b49c）。

**顕在化したときに起きること**: 抽出条件が緩めば copy や out-of-store の entry までアンカーを
持ち、世代外・置き切りであるはずの配置が store の生存に紐づく。逆に厳しすぎれば store-backed
な symlink がアンカーを失い、世代が積まれた後に前世代の store パスが GC で回収されて
ロールバック先が壊れる。アンカー名が非決定的になれば、同じ target が世代ごとに別名のアンカーを
持ち、間引きの対象を特定できなくなる。いずれも配置直後には観測されず、GC が走った後に
初めて壊れた形で現れる。

**この破れが requirement に張る理由**: 検証手段を変えても、GC アンカーの取り違えという破れ
自体は残る。`docs/agents/sara-graph.md` の判別規約に従い requirement 側へ張る。

## 評価

- likelihood: medium — 抽出条件は method × srcKind の組み合わせで分岐し、条件の追加・変更で
  取りこぼしが起こりうる
- impact: high — アンカーの欠落は前世代の store パス回収 = ロールバック不能を招く

## 対処

TC-1d69350e（アンカー対象の抽出と farm の組み立て）・TC-a6a14739（アンカー名の形式と決定性）で
緩和する。

ただし本区分が見るのは抽出条件とアンカー名、および derivation のビルドスクリプトへの配線
までで、**REQ-60e6b49c の「返り値が symlink farm を含む store オブジェクトである」ことそのもの
は残余**として残る。TC-1d69350e は fake な pkgs 越しにスクリプト本文を取り出すだけで実ビルドは
行わないため、配置後の farm derivation の実体は評価テストの射程外にあり、`integration` 区分の
E2E が担当する（→ Issue #289）。

## 出典

`tests/nix-unit/farm-entries.nix` / `tests/nix-unit/anchor-name.nix` の現行実装からの逆算
（→ Issue #273「L1〜L4」節）。GC アンカーの設計判断そのものは ADR-0016 が持つ。
