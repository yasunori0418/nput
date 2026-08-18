---
id: "TC-596d697f-4ba6-4ec1-b71e-8b5375806c08"
type: test_condition
name: "copy が target 不在時のみマテリアライズし、既存 target を触らないこと"
mitigates:
  - "RISK-01cf9f47-95bd-419a-863d-0d7c1f8188ba"
---
# TC-596d697f: place-once の冪等性

target が不在のときにだけコピーが行われ、既に存在するときは内容・属性ともに一切変更されない
ことを検証する。「再 apply しても壊れない」だけでは不足で、配置後にユーザーが編集した内容が
そのまま残ることまでを条件とする。

構造不一致（subpath がディレクトリのとき target が通常ファイル、またはその逆）が黙って
書き込まれず conflict として停止することも同じ条件の下に置く。
