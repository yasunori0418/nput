---
id: "TC-4f315cfc-446c-4c3f-8dc9-c36b10073e9d"
type: test_condition
name: "コピー中の syscall 失敗が伝播し、失敗した entry が結果へ記録されないこと"
mitigates:
  - "RISK-bc7526c2-2a2d-427f-9490-92d7d262fce3"
---
# TC-4f315cfc: コピー失敗の伝播

コピー経路の各 syscall（mkdir / open / open-file / read-write / readlink / lstat）の
失敗がそれぞれエラーとして呼び出し元へ伝わることを検証する。エラーメッセージが失敗した操作を
特定できることを含む（どの段で落ちたか分からないと診断できない）。

失敗した entry が結果レコードへ「コピー済み」として残らないことも条件に含む。非 ENOENT の
lstat 失敗が「target 不在」と同一視されないことは、recopy 経路で特に重要になる。

foreign な実ファイルの skip が warning として可視化されることは、engine 側の配置経路で
検証する（`engine-core` の CASE-31fdb776 が併せて覆う）。
