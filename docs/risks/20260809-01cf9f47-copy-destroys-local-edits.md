---
id: "RISK-01cf9f47-95bd-419a-863d-0d7c1f8188ba"
type: risk
name: "copy が place-once を破り、ユーザーのローカル編集を失わせる"
threatens:
  - "REQ-d2277c7a-7992-49af-a9dc-4cc73843a6f9"
  - "REQ-7cc32a2b-eee4-4a29-8dc1-a1dc23e7a065"
  - "REQ-b4e4b65d-6e35-40c3-a00e-20c14043df6f"
likelihood: medium
impact: high
level: high
---
# RISK-01cf9f47: copy が place-once を破り、ユーザーのローカル編集を失わせる

copy は「target 不在のときにだけマテリアライズし、以後は触らない」という place-once の契約で
成り立っている。ユーザーは配置後のファイルを自分で編集する前提で copy を選ぶため、再 apply が
黙って上書きすればその編集は復元手段なく失われる（copy は世代管理の対象外なのでロールバックで
戻せない）。

明示的な上書き手段である `apply --recopy` にも同種の脅威がある。上書きを「削除してから作成」で
行うと、置換の途中で失敗したときに元も新も無い状態が残る。同一親内への rename 退避で行うことが
その回避策であり、この経路が崩れると「更新のつもりが消失」になる。

## 想定する失敗

- 再 apply が既存 target を上書きし、ローカル編集を破棄する
- `--recopy` が config 内の一部 copy target しか更新せず、更新したつもりの取りこぼしが出る
- `--recopy` の上書きが削除ベースで行われ、途中失敗で target を失う
- 構造不一致（subpath がディレクトリ / target が通常ファイル、またはその逆）が停止せず、
  不整合なまま書き込む

## 張り先の判断

3 本とも requirement へ張る。place-once も recopy の上書き手段もユーザーが直接触れる契約で
あり、コピーの実装手段を差し替えても懸念は残る。

ただし「想定する失敗」3 点目（`--recopy` の削除ベース上書き）を受けるテスト条件は `copy` 対象に
置いていない。rename 退避の経路を検証しているのは undo journal を通す `atomicity` 対象の
テスト資産であり、この失敗モードの mitigate はそちらが引き受ける。
