---
id: "CASE-a9ff732c-af30-4f13-862d-92e3074d390d"
type: test_case
name: "niface_payload_test.go — engine 結果からペイロードへの写像"
covers:
  - "TC-cf8189c4-d680-4d7e-bcd3-810543762c50"
  - "TC-71fdbbfb-cf81-4aba-8254-59d7cb45d5ab"
  - "TC-733ac4ed-dac3-4bb8-9bfd-fd1cbdc300c9"
  - "TC-4e0a14d6-342c-448b-964b-b8e87520e89b"
---
# CASE-a9ff732c: niface_payload_test.go — engine 結果からペイロードへの写像

## 対象

`cmd/nput/niface_payload_test.go`

## 検証内容

- **インベントリと change の写像** — item が新 manifest の entry と stale 除去された旧 entry の
  全数になること、差分のある遷移だけが change になること、状態の変わらない再 link が change を
  作らないこと、配置方式の変更（symlink → copy）が 1 item + 1 change へ束ねられること
- **dryrun での非抑制** — 実行結果と違い計画では再 link を抑制しないこと
- **到達状態のパーティション** — 完了 / 失敗（分類コードつき）/ 未到達の skip が区別されること
- **エラー層の置き場所** — entry スコープの失敗を subject の errors へ重複させないこと、
  主体起因の失敗はそこに 1 件だけ載ること、conflict がその一種として扱われること
- **warning の振り分け** — インベントリに存在する対象の警告は item へ、孤児・保守的に残した
  stale の警告は subject へ載ること。警告種別からコードへの対応が全種別で成立し、未知の種別が
  フォールバックすること。警告と error の同居でも適合を保つこと
- **世代スロット** — 変更前後の世代を持つこと（初回は変更前を省略、失敗時は動かない）、
  世代の概念を持たない実行ではスロット自体が不在であること、profile 未作成の dryrun で前後
  ともに省略されること
- **`info` の非出力** — 変更系がエンベロープにも結果にも `info` キーを出さないこと
- **実 engine を通した経路** — 一時ディレクトリ上で実際にコミットまで走らせ、同じ写像が
  成立すること

適合チェッカを併用しつつ、主眼は schema では縛れない**内容**の検証にある。
