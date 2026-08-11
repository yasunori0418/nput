---
id: "RISK-cdcc6faf-9164-409b-b584-2921fa036d10"
type: risk
name: "run が返す世代観測と到達状態が実態とずれ、機械可読出力の消費側を誤らせる"
threatens:
  - "REQ-1be4d678-959c-44d7-a346-44bfd95af56e"
  - "REQ-05abce3e-9797-432b-b93f-37c55d09afde"
likelihood: medium
impact: high
level: high
---
# RISK-cdcc6faf: run が返す世代観測と到達状態が実態とずれ、機械可読出力の消費側を誤らせる

## リスク

`Apply` / `Rollback` / `Reset` は、実行結果として全 entry のインベントリ・世代観測
（before / after）・失敗時の到達状態（どの target で落ちたか・どこへ到達しなかったか・
巻き戻したか）・構造化された planner warning を返す。これが `--json` の envelope として
そのまま外へ出るため、値のずれは「FS の実態と機械可読出力の乖離」になる。

食い違い方は失敗経路に集中する。成功時は FS も出力も期待どおりで、失敗時にだけ
「配置したことになっている target が実は配置されていない」「巻き戻したのに `Unwound` が
false」「到達しなかった entry が一覧から消える」といった形で崩れる。CI が exit code だけ
でなく JSON を読んで判断する運用では、この乖離が誤った成功判定へ直結する。

世代観測の nil 表現も同じ系統にある。世代を観測できない状況（profile リンクが世代リンク
でない・そもそも世代を進めない run）で、観測できなかったことを nil で表さずに 0 や現在値を
返せば、消費側は「世代 0 へ戻った」「世代が進んだ」と読む。project mode が世代をユーザーへ
公開しない（`rollback` / `list-generations` を出さない → REQ-05abce3e）という区別も、
観測値が漏れ出せば崩れる。ただしその公開／非公開の強制は CLI 側（`rootKind` を見て
project mode の `rollback` / `list-generations` を拒否する分岐）にあり、engine 層の
観測値が nil を正しく運ぶところまでが TC-1d19aebc / TC-527b5034 の守備範囲になる。
非公開の強制そのものは、入口の評価で root の種別を解決した後にしか効かないため、実際に
評価を通す e2e で TC-f304d682 が持つ。

## 実現性

**likelihood: medium** — 値の組み立ては失敗経路ごとに分岐しており、段を増やすたびに
どの段までを「到達済み」と数えるかの判断が要る。

**impact: high** — FS を直接壊しはしないが、誤りは沈黙の不整合として現れる。「配置した
ことになっている target が実は配置されていない」「巻き戻したのに `Unwound` が false」は
どちらも失敗として現れず、CI が JSON を読んで判断する運用では誤った成功判定に直結する。

## 緩和

TC-1d19aebc（実行結果の観測フィールドと到達状態の分割）と TC-527b5034（e2e での
`--json` 観測）が緩和する。世代の公開／非公開の強制は TC-f304d682（project mode では
世代を公開するコマンドが拒否され理由が利用者へ届く）が緩和する。
