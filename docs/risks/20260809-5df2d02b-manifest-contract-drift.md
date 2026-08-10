---
id: "RISK-5df2d02b-e5d4-40eb-86ad-e8bc96e4c34d"
type: risk
name: "manifest.json の形が engine の読む契約から静かにずれる"
likelihood: medium
impact: high
level: high
threatens:
  - "REQ-dedd2c28-bba3-4ecf-80c9-8c77347e8e1f"
  - "REQ-dd10d820-e453-4099-a47a-ffb9a7de02fb"
  - "REQ-0b0cd1e3-bfeb-45c1-978d-e2e11c568336"
  - "REQ-79ce0a09-f9bd-4e61-ba7f-45fb5643137b"
  - "REQ-250d936c-1df0-491d-a7af-708f38b61f33"
  - "REQ-b232ec98-af3b-41f3-a050-29d417322002"
  - "REQ-1dcc9a33-b0f2-43e0-8310-fc4b19e68fe7"
  - "REQ-77689c68-953c-4cbb-ab31-1ac1e4f5f2fe"
---
# RISK-5df2d02b: manifest.json の形が engine の読む契約から静かにずれる

## リスク

`manifest.json` は Nix 側と engine の唯一の安定契約（REQ-79ce0a09）で、`schemaVersion` は
1 に固定され v1 のみを発行・受理する（REQ-250d936c）。`normalizeManifest` が出す文書の形
——トップレベル 3 フィールド（REQ-dedd2c28）・`rootKind` と fixed 時のみの絶対パス併記
（REQ-dd10d820）・entry の 5 フィールドと辞書順の配列化（REQ-0b0cd1e3）・marker の判別タグを
漏らさないこと（REQ-1dcc9a33）——が、評価層の変更で気づかれずに変わりうる。

**顕在化したときに起きること**: engine 側は不正な文書を読んで失敗するか、より悪いことに
フィールドの欠落や順序変化を黙って受理し、意図しない配置を行う。`mkManifest` は純粋関数
（REQ-2b0c2bb8）で FS も nix daemon も介さないため、この破れは実行するまで観測されず、
E2E まで落ちてこないと検出できない（REQ-2b0c2bb8 の純粋性そのものが脅かされるわけでは
ないので `threatens` は張らない。検出が難しい理由としてここに書くに留める）。`method` の既定
（REQ-77689c68）や defaults の適用（REQ-b232ec98）が変われば、世代管理下に置くつもりの entry が
世代外へ回る。

## 評価

- likelihood: medium — 契約は 1 箇所（`lib/manifest.nix`）に閉じているが、フィールド追加・
  既定値変更は評価層の通常の改修で起こりうる
- impact: high — engine の全経路が読む唯一の契約で、破れは配置結果そのものを狂わせる

## 対処

TC-4e7cfae7（文書構造の不変条件）・TC-d9175bb5（既定適用と決定的順序）・TC-81be084d
（marker の判別と src 種別への変換）・TC-de6514e2（文書全体のスナップショット回帰）で緩和する。

**本区分の TC 群では緩和しきれない残余**が 2 つある。第一に、REQ-dd10d820 のうち fixed root で
絶対パスを併記する側は評価テストに無く、TC-4e7cfae7 が見るのは project root 分だけである
（`homeRoot` の `rootKind` は `checks.hm-module`、fixed root はどの区分も現状カバーしていない）。
第二に、REQ-250d936c / REQ-79ce0a09 は発行側と engine の受理側の双方を規範に含むが、本 risk が
射程に持つのは評価層が発行する側だけで、engine が新しい `schemaVersion` を拒否する側は
`cli-json` / `integration` 区分の担当になる。

## 出典

`tests/nix-unit/structure.nix` / `defaults.nix` / `resolve-marker.nix` と
`tests/namaka/manifest-project/expr.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
manifest 契約の設計判断は ADR-0010 / ADR-0014 が持つ。
