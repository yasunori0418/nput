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

TC-4e7cfae7（文書構造の不変条件）・TC-f9e927d0（fixed root の絶対パス併記）・TC-d9175bb5
（既定適用と決定的順序）・TC-81be084d（marker の判別と src 種別への変換）・TC-de6514e2
（文書全体のスナップショット回帰）で緩和する。

REQ-dd10d820 は project root の否定側を TC-4e7cfae7、fixed root の肯定側を TC-f9e927d0 が
見るので、評価層の分は両側が揃っている（→ Issue #288）。

**本区分の TC 群では緩和しきれない残余**が 4 つある。

第一に、`fixed` の `root` が実際に絶対パスであることを強制する検査がそもそも実装に無い。
`rootType` は marker でない文字列をすべて受理し（`lib.isString` のみ）、`normalizeManifest` は
それを無検査で `rootKind = "fixed"` へ倒すため、相対パス文字列も fixed として通る。
TC-f9e927d0 は常に絶対パスを渡すので、この穴は評価テストでも塞がらない。拒否する規範が
REQ 側に無い（REQ-dd10d820 / REQ-37b56673 は「絶対パス」と述べるが gate を要求していない）
ので、仕様の未決事項として残余に置く。

第二に、`homeRoot` を HM 統合の配線として確かめる側は評価層の担当外で、`checks.hm-module`
（`integration` 区分）が持つ。ただし REQ-dd10d820 の home 側そのもの——`rootKind = "home"` で
`root` フィールドを持たないこと——は TC-f9e927d0 が exact 一致で担保済みで、
`checks.hm-module` の grep（`"rootKind":"home"` の部分一致）は余分なフィールドの混入を
検出しない。`integration` 区分が見るのは「HM モジュールが root に home を pin する配線」であり、
文書の形そのものではない。

第三に、`rootKind = "system"` は REQ-dd10d820 が定める 4 値の 1 つで、ADR-0036（採用）が
ADR-0013 §5 の throw を撤回して manifest v1 の正規値にすると決めているが、実装は依然
`normalizeManifest` の assertion で `system` を弾く。評価層で `{ rootKind = "system"; }` が
発行される側を見るテストは存在せず、TC-e7ff0e6d（拒否ゲート）が見ているのは throw する
現状の側である。ADR-0036 が指示した実装更新が済むまでの残余で、更新時にはゲート側のテストと合わせて
組み替える必要がある。

第四に、REQ-250d936c / REQ-79ce0a09 は発行側と engine の受理側の双方を規範に含むが、本 risk が
射程に持つのは評価層が発行する側だけで、engine が新しい `schemaVersion` を拒否する側は
`cli-json` / `integration` 区分の担当になる。

## 出典

`tests/nix-unit/structure.nix` / `defaults.nix` / `resolve-marker.nix` と
`tests/namaka/manifest-project/expr.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
manifest 契約の設計判断は ADR-0010 / ADR-0014 が持つ。
