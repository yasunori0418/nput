---
id: "ADR-0050"
type: adr
name: "sara check を strict 化する — required status check 化と型別ゲートは見送り"
status: 採用
issues:
  - "#255"
  - "#203"
origin: "epic #203 Batch 2 完了後の残論点判断（2026-08-09）で確定"
justifies:
  - "INF-659b139d-0cf8-4c65-b30d-93c5ee2dfc71"
revises:
  - "ADR-0048"
references:
  - "ADR-0030"
---
# ADR-0050: sara check を strict 化する — required status check 化と型別ゲートは見送り

- ステータス: 採用
- 日付: 2026-08-09
- 関連: ADR-0048, ADR-0030, GitHub Issue #255, #203
- 改訂対象: **ADR-0048 §5 の `strict_mode = false` の決定だけ**。§5 のそれ以外の決定
  （required status check に登録しない・DoD に追加しない・CI job のローカル filter）は不変で、
  §1〜§4 も全て不変
- 起点: epic #203 の移行完了（Batch 2・2026-08-08）を受けた残論点判断（Issue #255）

## 背景

ADR-0048 §5 は `sara check` の CI 導入にあたり `strict_mode = false`（orphan は警告止まり）で
開始し、strict 化・必須チェック化は移行完了後に判断すると保留した。理由は移行中に
「テスト未着手の要求」「まだ親を持たない item」が正常に存在し、orphan をエラーにすると
移行そのものが進まないことだった。

この前提は epic #203 の移行完了で消滅した。Batch 2（2026-08-08）の全 PR マージにより
`sara check` は **warning 0 件**で通っており、「接続漏れ = orphan warning」の不変条件が
全型で成立している。一方でこの不変条件には機械的な担保が無く、新規 item が接続漏れのまま
追加されても警告止まりでマージが通る（Issue #255 が指摘した「宣言的にしか成立していない
不変条件」）。

## 決定

Issue #255 の 3 論点を次のとおり判断する。

1. **`sara.toml` の `strict_mode` を `true` にする**。orphan を含む全 warning が error になり、
   `sara check` は接続漏れで exit 1 を返す。warning 0 が実測で成立した今、strict 化は
   既存状態を壊さず、以後の接続漏れだけを機械的に落とす
2. **required status check には登録しない（ADR-0048 §5 の決定を維持）**。ruleset
   `main branch protection` と DoD（→ ADR-0030 の required check 集合・項目数上限）は
   変更しない。CI の `sara` job は接続漏れで赤くなるが、マージはブロックしない
3. **型別の担保（ADR- orphan だけを対象とする CI grep ゲート等）は設けない**。strict 化により
   orphan 全般が error になるため、型を限定したゲートは冗長。frontmatter 単体を検証する
   テストの新設も行わない

## 理由

- `--strict`（および `strict_mode`）は全 warning 一括の 0/1 スイッチで型単位の粒度指定を
  持たない（sara v0.9.4 のソース確認済み・ADR-0048 §5 起票時の調査）。粒度が選べないことは
  warning が残存していた時期には strict 化を退ける理由だったが、warning 0 が成立した現在は
  「一括で全部落とす」がそのまま望む挙動になる
- 非 required の維持は「検出は機械・判断は人」の分担を保つため。接続漏れの修正が自明でない
  ケース（item 新設を伴う等）で、CI が赤いままレビューで扱う余地を残す

## 帰結

- `sara check` の合格条件は「warning 0 件」に一本化される。`docs/adr/README.md` の
  セルフチェック節が持っていた「他の型の orphan は移行中のため残ることがあり判定対象外」の
  但し書きは前提を失うため、本 ADR と同じ変更の中で README を追随させる
- CI の `sara` job は非 required のまま赤/緑の信号としてはたらく（ADR-0048 §5 の filter 構成は
  不変）

## 再検討条件

テスト系型（risk / test_condition / test_case / defect）の工程着手時に再判断する。
`defect` は upstream relation を張る手段が無く**構造的に orphan になる**（`docs/model.yaml`
冒頭の既知例外）ため、defect item が最初に起こされた時点で strict 化と衝突する。その時点で
sara 側の型別設定の有無を再調査し、例外機構が無ければ `strict_mode` の解除を含めて
本 ADR を改訂する。
