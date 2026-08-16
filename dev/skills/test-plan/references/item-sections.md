# test-plan 工程の item 本文の節構成

frontmatter の書式は sara-docs の references/frontmatter.md が正本。ここは本文
（Markdown 部分）の節構成を定める。どちらの item も本文は
`# <PREFIX>-<前方8>: <name>` の見出しで始める。

## test_plan item（`docs/test-plan/`）

```markdown
# TP-<前方8>: <name>

## 仕様

<specification の内容を人間向けに展開する散文・表。
 - スコープ内 / スコープ外（スコープ外は理由付き）
 - リスクレベル別の重点配分: どこを・なぜ厚く / 薄くするか。
   対応する risk への省略形参照（RISK-xxxxxxxx）で紐づける
 - 採用するテストレベル（コンポーネント / 統合 / システム / 受け入れ）・テストタイプ
 - 開始基準（entry criteria）と前提>

## 出典

<この計画の根拠。参照した REQ / DSG の省略形、設計文書、利用者との決定
（リスク許容度・しきい値）の要約。ADR やイシューがあればその参照>
```

- 完了基準（exit criteria）の規範そのものは frontmatter の `specification` /
  `specification_ja` に置く。本文はその展開と背景で、規範を二重に書き下さない
  （ずれの温床になる）。
- 成果物一覧・次のステップの節は持たない。下流の把握は `sara query <TP のフル ID> -d`。

## risk item（`docs/risks/`）

```markdown
# RISK-<前方8>: <name>

<リード文: 何がどう壊れると、誰にどんな悪影響が出るか。1〜2 段落>

## 想定する失敗

- <具体的な失敗モードの列挙。コードで実在性を確認済みのものだけ>

## 評価

- likelihood: <値> — <根拠: 変更頻度・複雑さ・既存の検証ゲートの届き方>
- impact: <値> — <根拠: 回復可能性・失われるもの・気づく契機>

## 張り先の判断

<threatens を requirement / design のどちらへ張ったかの理由。
 設計を差し替えたら消える懸念か（design 行き）、要求そのものが満たされない
 懸念か（requirement 行き）。複数 edge を張ったならその内訳>
```

- `level` は frontmatter のフィールドで、likelihood × impact から導出する（本文には
  導出根拠を書かない — マトリクスが規範なので書くことがない）。
- 失敗モードの一部を別の risk が引き受けるなら、その分担を「張り先の判断」に明記する
  （重い方向の脱落を防ぐ）。
