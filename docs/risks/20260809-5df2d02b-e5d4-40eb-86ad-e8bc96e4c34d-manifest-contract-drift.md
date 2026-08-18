---
id: "RISK-5df2d02b-e5d4-40eb-86ad-e8bc96e4c34d"
type: risk
name: "manifest.json の形が engine の読む契約から静かにずれる"
likelihood: medium
impact: medium
level: medium
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
# RISK-5df2d02b-e5d4-40eb-86ad-e8bc96e4c34d: manifest.json の形が engine の読む契約から静かにずれる

## リスク

`manifest.json` は Nix 側と engine の唯一の安定契約（REQ-79ce0a09-f9bd-4e61-ba7f-45fb5643137b）で、`schemaVersion` は
1 に固定され v1 のみを発行・受理する（REQ-250d936c-1df0-491d-a7af-708f38b61f33）。`normalizeManifest` が出す文書の形
——トップレベル 3 フィールド（REQ-dedd2c28-bba3-4ecf-80c9-8c77347e8e1f）・`rootKind` と fixed 時のみの絶対パス併記
（REQ-dd10d820-e453-4099-a47a-ffb9a7de02fb）・entry の 5 フィールドと辞書順の配列化（REQ-0b0cd1e3-bfeb-45c1-978d-e2e11c568336）・marker の判別タグを
漏らさないこと（REQ-1dcc9a33-b0f2-43e0-8310-fc4b19e68fe7）——が、評価層の変更で気づかれずに変わりうる。

**顕在化したときに起きること**: engine 側は不正な文書を読んで失敗するか、より悪いことに
フィールドの欠落や順序変化を黙って受理し、意図しない配置を行う。`mkManifest` は純粋関数
（REQ-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4）で FS も nix daemon も介さないため、この破れは実行するまで観測されず、
E2E まで落ちてこないと検出できない（REQ-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4 の純粋性そのものが脅かされるわけでは
ないので `threatens` は張らない。検出が難しい理由としてここに書くに留める）。`method` の既定
（REQ-77689c68-953c-4cbb-ab31-1ac1e4f5f2fe）や defaults の適用（REQ-b232ec98-af3b-41f3-a050-29d417322002）が変われば、世代管理下に置くつもりの entry が
世代外へ回る。

## 評価

- likelihood: medium — 契約は 1 箇所（`lib/manifest.nix`）に閉じているが、フィールド追加・
  既定値変更は評価層の通常の改修で起こりうる
- impact: medium — engine の全経路が読む唯一の契約で、破れは配置結果そのものを狂わせる。
  ただし狂うのは nput が管理する配置であり、`lib/manifest.nix` を直したうえでの再 apply が
  正しい形へ収束させる。破れが実行するまで観測されない（上記のとおり `mkManifest` は純粋関数で
  E2E まで落ちないと気づけない）ことは沈黙性の問題で、回復可能性は下げない。
  **root の種別を取り違えて配置先そのものが別ツリーへ移る facet は本 item の射程ではない**。
  受理の段で止める側は RISK-7808768d-bd87-429a-af42-52e8559d940c が、engine の解決規則そのものは RISK-24e0805d-53cd-40dd-9e7a-b5c4bbf2a298 が持ち、
  どちらも「記録に残らない場所へ書くので回収できない」ことを理由に high を採っている。本 item が
  持つのは生成側が出す文書の形であり、下の残余第一（`fixed` の相対パスを受理する穴）も、
  拒否する規範が REQ 側に無いという仕様の未決事項として置いたもので、impact はその 2 item 側で
  計上する

## 対処

TC-4e7cfae7-72bc-4af6-a1f5-1ead7db564b1（文書構造の不変条件）・TC-f9e927d0-8e10-4b8e-9870-5b5486949af6（fixed root の絶対パス併記）・TC-d9175bb5-d7ec-41e0-8bee-71de928a71fb
（既定適用と決定的順序）・TC-81be084d-709f-481b-9b61-5d2d11c317a0（marker の判別と src 種別への変換）・TC-de6514e2-9105-45a6-a5b9-d474911a401b
（文書全体のスナップショット回帰）で緩和する。

REQ-dd10d820-e453-4099-a47a-ffb9a7de02fb は project root の否定側を TC-4e7cfae7-72bc-4af6-a1f5-1ead7db564b1、fixed root の肯定側を TC-f9e927d0-8e10-4b8e-9870-5b5486949af6 が
見るので、評価層の分は両側が揃っている（→ Issue #288）。

**本区分の TC 群では緩和しきれない残余**が 5 つある。

第一に、`fixed` の `root` が実際に絶対パスであることを強制する検査がそもそも実装に無い。
`rootType` は marker でない文字列をすべて受理し（`lib.isString` のみ）、`normalizeManifest` は
それを無検査で `rootKind = "fixed"` へ倒すため、相対パス文字列も fixed として通る。
TC-f9e927d0-8e10-4b8e-9870-5b5486949af6 は常に絶対パスを渡すので、この穴は評価テストでも塞がらない。拒否する規範が
REQ 側に無い（REQ-dd10d820-e453-4099-a47a-ffb9a7de02fb / REQ-37b56673-6e40-4a1b-a2a7-5d3c084e3e66 は「絶対パス」と述べるが gate を要求していない）
ので、仕様の未決事項として残余に置く。

第二に、`homeRoot` を HM 統合の配線として確かめる側は評価層の担当外で、`checks.hm-module`
（`integration` 区分）が持つ。ただし REQ-dd10d820-e453-4099-a47a-ffb9a7de02fb の home 側そのもの——`rootKind = "home"` で
`root` フィールドを持たないこと——は TC-f9e927d0-8e10-4b8e-9870-5b5486949af6 が exact 一致で担保済みで、
`checks.hm-module` の grep（`"rootKind":"home"` の部分一致）は余分なフィールドの混入を
検出しない。`integration` 区分が見るのは「HM モジュールが root に home を pin する配線」であり、
文書の形そのものではない。

第三に、`rootKind = "system"` は REQ-dd10d820-e453-4099-a47a-ffb9a7de02fb が定める 4 値の 1 つで、ADR-0036（採用）が
ADR-0013 §5 の throw を撤回して manifest v1 の正規値にすると決めているが、実装は依然
`normalizeManifest` の assertion で `system` を弾く。評価層で `{ rootKind = "system"; }` が
発行される側を見るテストは存在しない。throw する現状を写すアサート
（`testSystemRootUnimplemented`）は CASE-879a93da-d22f-4397-84da-3544f8249af1 に残っているが、TC-e7ff0e6d-32d7-4ed6-8c2f-449dba34b2f6 は撤回済み決定の
残骸としてこれを条件に含めておらず、RISK-09df40d3-2752-433e-9ab0-2816fbd14969 も同じ理由で規範から外している。つまり
`system` は規範側にも条件側にも所在を持たない。ADR-0036 が指示した実装更新が済むまでの残余で、
更新時にはゲート側のテストと合わせて組み替える必要がある。

第四に、`fixed` の絶対パスが `mkManifest` の `passthru.root`（REQ-2f9205ee-cec5-4072-ac3e-890caae79904）へ写る側は
derivation 層の担当で、本区分の TC 群は見ない（TC-f9e927d0-8e10-4b8e-9870-5b5486949af6 が射程外と宣言している）。この
分岐は fixed のときだけ通り、CLI が build 前に `nix eval` で読む経路（`cli-json` 区分）の
入力になるが、現状どの risk も残余として引き取っておらず、CASE も無い。評価層が出す文書の
形とは別物なので本 risk の射程には入れず、`cli-json` 区分側で拾うべき穴としてここに記録する
に留める。解消は、同区分に passthru を threatens する risk を立てて残余を移送した時点。
それまでは本 risk が仮の置き場になる。

第五に、REQ-250d936c-1df0-491d-a7af-708f38b61f33 / REQ-79ce0a09-f9bd-4e61-ba7f-45fb5643137b は発行側と engine の受理側の双方を規範に含むが、本 risk が
射程に持つのは評価層が発行する側だけで、engine が新しい `schemaVersion` を拒否する側は
`cli-json` / `integration` 区分の担当になる。

## 出典

`tests/nix-unit/structure.nix` / `defaults.nix` / `resolve-marker.nix` と
`tests/namaka/manifest-project/expr.nix` の現行実装からの逆算（→ Issue #273「L1〜L4」節）。
manifest 契約の設計判断は ADR-0010 / ADR-0014 が持つ。
