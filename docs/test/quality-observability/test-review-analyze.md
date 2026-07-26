# テストレビュー記録: quality-observability / analyze

- レビュー対象: `docs/test/quality-observability/test-analysis.md`
- 実施日: 2026-07-26（計 5 回。初回差し戻し〔must 2 / want 3 / nit 2〕→ 2 回目差し戻し
  〔must 1 / want 2 / nit 2〕→ 3 回目差し戻し〔must 2 / want 1 / nit 2〕→ 4 回目差し戻し
  〔must 0 / want 1 / nit 1〕→ 5 回目で全指摘解消）
- 判定: 通過（利用者判定）
- 機械検査: OK 7 / NG 0 / SKIP 0（review-check.sh analyze）

差し戻しが 4 回続いた主因は**リスク紐づけの誤り**で、3 回連続で差し戻し理由の中核だった
（TC-07・TC-08 への R1 の誤付与 → R3 への誤った付け替え → TC-06 の R3 維持と TC-05 の
R3 欠落）。最終版では全 14 条件の対応リスク列を R1〜R8 の定義文と 1 件ずつ悉皆で照合し、
不一致が無いことを AI 定性レビュー（test-reviewer サブエージェント）が独立に確認した。
分析側も 3 章へ「リスク紐づけの原則」（定義文に一致するものだけを書く / 外延を広げない /
一致しなければ `—`）を明文化し、判断基準を成果物へ固定している。

初回・2 回目の must 指摘は、受け入れ条件（AC）を分母にした自己検証では原理的に検出できない
欠落だった（REQ-01 の表構造と REQ-03 の `docs/quality/README.md` はいずれも対応する AC を
持たない）。最終版はこれらを TC-06・TC-14 として条件化し、2 章へ「AC の網羅だけを完了判定に
使うと当該要求が未実装のまま全 AC が充足しうる」旨の注記を残している。

最終版に対しては、リスク紐づけの悉皆照合に加えて、要求側からの悉皆突合
（REQ-01〜REQ-10 / NFR-01〜NFR-05 / AC-01〜AC-19 のすべてがいずれかのテスト条件へ対応）、
`test-monitoring.md` のリスクカバー率 100% の前提維持、粒度ガードへの非抵触、実物照合
（`docs/spec.md` の見出し構成・root/dev 両 `flake.nix` の `apps` 出力不在・
`docs/design.md` の `docs/quality/` 未言及）を確認済み。TC-11 を生成モードへ限定した判断も、
仕様に検証モードの Nix 非依存を要求する記述が無いこと、REQ-08 が `nix run` での起動を
規定していることから妥当と確認した。

## 次のステップ

- test-analyze へ戻り、mini サマリ（`mini-test-analysis.md`）を作成して完了宣言する。
- 後続工程は `/test-design quality-observability`（テスト条件からのケース導出）。
