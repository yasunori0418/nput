---
id: "TC-1a46da17-1812-431b-966f-ec8dffd315ae"
type: test_condition
name: "エンベロープが全形状で niface 適合を保ち subject 件数で形が揺れない"
mitigates:
  - "RISK-4936a47d-796b-4f1f-8ae4-eb87f6c64e71"
---
# TC-1a46da17: エンベロープが全形状で niface 適合を保ち subject 件数で形が揺れない

## 条件

emit した文書を niface の適合チェッカ（embed 正本 schema + schema 外 lint）へ通し、
出現しうる全形状で適合することを確かめる。形状とは、subject の有無、成功と失敗、主体確立前の
失敗と主体起因の失敗、conflict、および `--all` の subject 0 件 / 1 件 / 複数件である。

件数による形の揺れをここで潰す。0 件でも `results` は空配列として残り、1 件の `--all` は
名前付き単一実行と同一の文書になる（成功系・失敗系の双方）。複数件は「長さの違う同じ文書」
であって、件数ごとに別形状にならない。

item id は identity から導出される決定論的な値であり、他実装と一致しなければ相互運用が成り立た
ない。niface が配布する id ベクタと突き合わせて導出則の一致を確かめ、あわせて nput 側の
entry 用 identity（kind と key の形）が同じ id を再現することを確かめる。

エンベロープを出す実行と出さない実行の線引きも条件に含む。RunE を持つ全サブコマンドが run を
開始して文書を公開する一方、help / completion のような成立条件を満たさない実行は文書を出さない。

> 適合検証だけでは文書の**内容**は守れない（schema は形しか縛らない）。内容の正しさは
> TC-cf8189c4 / TC-733ac4ed の担当。
