# frontmatter テンプレート（11 型）

共通の骨格。`id` / `type` / `name` は全型必須で、relation キーはフル ID のリストで書く。
本文は `# <PREFIX>-<前方8>: <name>` の見出しで始め、根拠・補足を散文で書く。

```markdown
---
id: "<PREFIX>-<フル UUIDv4>"
type: <型名>
name: "<日本語の 1 行要約>"
<relation キー>:
  - "<張り先のフル ID>"
---
# <PREFIX>-<前方8>: <name>

<本文>
```

`id` と `name` は必ずダブルクオートで囲む（`:` や特殊文字で YAML が壊れるのを防ぐ）。
以下、型ごとの relation キーと固有フィールドだけを差分で示す。

## solution（根）

relation なし。`docs/solution/` に置く。

## use_case

```yaml
type: use_case
refines:
  - "SOL-…"
```

## requirement

```yaml
type: requirement
derives_from:
  - "UC-…"
depends_on:          # 任意（同型 peer）
  - "REQ-…"
specification: |
  The system SHALL <describe the requirement>.
specification_ja: |
  <対象> は <要求> を満たさなければならない。
```

`specification` は英語（RFC2119 の SHALL 系）・`specification_ja` は日本語の規範助動詞。
規約の詳細は SKILL.md の「specification 規約」。

## design

```yaml
type: design
satisfies:
  - "REQ-…"          # または TP-…（テストハーネスの実装形など）
depends_on:          # 任意
  - "DSG-…"
```

## quality

```yaml
type: quality
derives_from:
  - "SOL-…"
depends_on:          # 任意
  - "QA-…"
specification: |
  The project SHALL <describe the quality policy>.
specification_ja: |
  <プロジェクト> は <品質方針> を満たさなければならない。
```

## test_plan

```yaml
type: test_plan
derives_from:
  - "SOL-…"
depends_on:          # 任意
  - "TP-…"
specification: |
  The test process SHALL <describe the test planning decision>.
specification_ja: |
  テストプロセスは <テスト計画上の決定> を満たさなければならない。
```

## infrastructure

```yaml
type: infrastructure
satisfies:
  - "QA-…"           # dev 基盤は quality を、runtime 基盤は design を satisfy する
```

プロジェクトによっては固有フィールド（DoD 対応など）を持つ。`docs/model.yaml` を確認する。

## adr（分離型・連番）

```yaml
id: "ADR-NNNN"       # 連番。sara-id は使わない（docs/adr/ の最大値 + 1 を手で採る）
type: adr
status: 採用
justifies:           # 必須（1 本以上）
  - "REQ-…"          # requirement / design / infrastructure / quality / test_plan
revises:             # 任意（部分改訂した旧 ADR）
  - "ADR-NNNN"
references:          # 任意（改訂を伴わない関連 ADR）
  - "ADR-NNNN"
```

## risk

```yaml
type: risk
threatens:
  - "REQ-…"          # または DSG-…
likelihood: medium   # high / medium / low
impact: high         # high / medium / low
level: high          # likelihood × impact から導出（手で判断しない）
```

- threatens を requirement / design のどちらへ張るか: **設計を差し替えたら消える懸念だけが
  design 行き**。要求そのものが満たされない懸念は requirement へ（判断は edge ごと）。
- `level` の導出マトリクス（likelihood \ impact）:

  | | high | medium | low |
  |---|---|---|---|
  | **high** | high | high | medium |
  | **medium** | high | medium | low |
  | **low** | medium | low | low |

  プロジェクトが独自のマトリクスや契約テストを持つ場合はそちらが正本。

## test_condition

```yaml
type: test_condition
mitigates:
  - "RISK-…"
```

固有フィールドなし。優先度もフィールドにしない（mitigates 先 risk の `level` から導出する。
複数なら最高 level。例外は本文に根拠付きで書く）。

## test_case

```yaml
type: test_case
target: "<テスト資産の正準表記（例: リポジトリ相対パス）>"
covers:
  - "TC-…"
```

`target` の正準表記の規則（パス形式・check 名など）はプロジェクト規約に従う。
