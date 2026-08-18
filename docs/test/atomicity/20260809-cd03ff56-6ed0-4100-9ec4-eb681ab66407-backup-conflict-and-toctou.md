---
id: "TC-cd03ff56-6ed0-4100-9ec4-eb681ab66407"
type: test_condition
name: "退避先が既存なら停止し、rename 直前に退避先を再検証する"
mitigates:
  - "RISK-3719d298-8175-4501-95d5-75f3a29568fe"
---
# TC-cd03ff56-6ed0-4100-9ec4-eb681ab66407: 退避先が既存なら停止し、rename 直前に退避先を再検証する

## テスト条件

`--backup` が退避先 `<target>.<suffix>` を潰さないことを、plan 時点と execute 時点の
両方で検証する。

**plan 時点で既存** — 前回の退避の残りなど、退避先が既に存在する状態で `--backup` を
付けた apply は conflict で停止する。黙って上書きしない（→ REQ-5dd5a4e9-6162-4fa5-b295-66844f5a4f3b）。

**plan と execute の間に出現（TOCTOU）** — plan 計算時には退避先が存在せず、`backup()`
が実行される時点までに他プロセスがそこを作った場合、rename 直前の再検証がその並行生成物を
検出して停止する。plan 時の判定を信じて rename すれば、並行して作られた実データを黙って
潰す（→ ADR-0017 の実行直前再検証、ADR-0045）。

この条件は full `Apply` では窓を作れないため、`BackupAction` を組み立てて `applier.backup`
を直接駆動し、plan 済みの状態から execute までの間に退避先を作る、という形で検証する。

上位の規範は TC-ed4992c0-8513-4383-be0a-e45acbbc229f と同じく TP-e7c25263-6d2d-4a37-8275-26906889d912。

## 対応する CASE

CASE-ed4b32fd-4f2e-497c-baa4-cc91f8a34e4a（`internal/engine/backup_test.go`）。
