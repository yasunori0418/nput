---
id: "CASE-8b3feeb2-7b7b-482c-bba2-6db0b923647a"
type: test_case
name: "niface_all_test.go — --all の subject 件数不変性と部分失敗の非汚染"
covers:
  - "TC-1a46da17-1812-431b-966f-ec8dffd315ae"
  - "TC-733ac4ed-dac3-4bb8-9bfd-fd1cbdc300c9"
  - "TC-ddee6cc4-bc10-4107-bc7e-288a5fb62f1f"
  - "TC-cf8189c4-d680-4d7e-bcd3-810543762c50"
---
# CASE-8b3feeb2: niface_all_test.go — --all の subject 件数不変性と部分失敗の非汚染

## 対象

`cmd/nput/niface_all_test.go`

## 検証内容

- **件数によらない文書形状** — subject 0 件で空配列 + 成功、1 件、複数件が「長さの違う同じ
  文書」であること。1 件の `--all` が名前付き単一実行とバイト単位で同一であること（成功系・
  失敗系の双方で固定）
- **部分失敗の非汚染** — 1 つの config の失敗が兄弟 config のステータスとインベントリを
  汚さないこと、subject を列挙し終えた後はトップレベルに errors が載らないこと
- **2 層の同居** — 部分結果を返す失敗では item 側に誤りが載り subject の errors は空である
  こと、結果すら無い失敗では subject の errors に 1 件載ること、両者が混在したときの終了
  コードが一般エラー優先で決まること
- **item id の結果スコープ性** — config 名を identity に含めないため、同じ対象を持つ 2 つの
  config が同じ id を別々の結果に持つこと。変更が指す item id は自分の結果内を指すこと
- **skip の扱い** — 排他制御で見送った主体が成功として扱われ、集約も成功に留まること
- **読み取り系 `--all` の打ち切り** — 途中で失敗したときそこまでの subject を残し、失敗した
  config に error を載せ、以降を subject 化しないこと
- **config をまたいだ重複除去をしない** — 複数の config が同じ対象を持つとき、機械可読出力は
  それぞれの config の結果に残す（行指向のテキスト出力が重複除去と並び替えを行うのとは
  非対称で、config ごとに分離するのが機械可読側の契約）

集約の関数を直接駆動し、テスト側で配線を書き直さない。世代一覧は外部コマンドの stub を
`PATH` の先頭へ置き、profile のレイアウトを一時ディレクトリ上に実際に作って再現する。
