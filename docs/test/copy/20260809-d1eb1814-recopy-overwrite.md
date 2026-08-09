---
id: "TC-d1eb1814-ac5e-4576-b092-7db4929fba43"
type: test_condition
name: "apply --recopy が全 copy target を無条件に更新し、上書きを退避経由で行うこと"
mitigates:
  - "RISK-01cf9f47-95bd-419a-863d-0d7c1f8188ba"
---
# TC-d1eb1814: recopy の上書き

`apply --recopy` が config 内の copy target を（place-once の判定を経ずに）src から
無条件に再コピーすることを検証する。foreign な実ファイルが占有している target も対象に含まれる
ため、通常 apply では skip される対象が recopy では更新されることを併せて確認する。

この条件が見るのは「無条件に上書きが行われる」ところまでで、上書き結果が src の内容へ戻ることと
更新対象が結果へ記録されることを確認する。上書きが削除ではなく同一親内への rename 退避で
行われること（途中失敗時に target を失わないための経路）の直接検証は、undo journal を通じて
その経路を見る `atomicity` 対象の担当であり、ここでは扱わない。
