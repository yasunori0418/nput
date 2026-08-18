# sara CLI の使い方

sara（と `sara-new` / `sara-gap`）は **PATH に通っている前提**で `sara ...` と直接叩く
（direnv 等で開発シェルが読み込まれた環境なら通っている）。通っていなければ環境構築を
プロジェクトの規約に従って済ませてから使う。設定はカレント（リポジトリルート）の
`sara.toml` を読むため、**リポジトリルートで実行する**。

## sara-new — item の起票（採番 + frontmatter の生成）

frontmatter は手書き・テンプレ写経をせず `sara-new` で起票する。採番・ファイル名規約の
適用・`sara init` の呼び出しを一手で済ませる:

```bash
sara-new <型> <slug> <配置ディレクトリ> [-- <sara init のオプション>...]

sara-new risk lock-ordering docs/risks -- \
  --name "<日本語の 1 行要約>" \
  --threatens "<REQ / DSG のフル ID>" \
  --likelihood medium --impact high --level high
```

- 返るのは機械可読の 2 行（`id:` = 採番された正式 ID、`file:` = 起票したファイルのパス）。
- `--` 以降は `sara init` へそのまま渡る。型ごとのオプション（relation・固有フィールド）は
  `sara init <型> --help` で確認する。requirement / quality / test_plan の
  `--specification` / `--specification-ja`、test_case の `--target` など、モデルの必須
  フィールドはここで渡す。
- **ADR は対象外**（連番を維持するため）。`sara init adr <パス>` を直接使う。
- 生成後に本文（`# <フル ID>: <name>` の見出し + 散文）を書き足し、`sara check` で green
  を確認する。
- 既存 item のフィールド・relation の変更は `sara edit <フル ID> --<フィールド> ...`
  でもよい（対話モードは TTY 前提なので、非対話ではフラグを明示する）。

## check — グラフ検証

```bash
sara check                 # broken reference / duplicate ID / cycle / orphan
sara check --format json   # items 配列（宣言辺つき）を機械可読で出す
```

- item を書き込んだら必ず回す。green（exit 0）が完了条件。
- strict 有効（`sara.toml` の `strict_mode = true`）なら orphan を含む全 warning が
  error になる。orphan は「upstream relation の張り忘れ」を意味することが多い。
- `--format json` の `items[].relationships[]` は**宣言辺（primary relation）のみ**で
  逆辺を持たない。逆引き（「この REQ を threatens している risk は?」）が要るときは
  `sara query -d` か `sara-gap` を使う。

## query — グラフを辿る

```bash
sara query <フル ID> -u    # 上流へ（例: requirement → use_case → solution）
sara query <フル ID> -d    # 下流へ（例: requirement → design、REQ ← risk ← TC ← CASE）
sara query <フル ID> -u --depth 1
sara query <フル ID> -d --format json
```

**`sara query` はフル ID しか受け付けない**が、散文もファイル名も relation リストも同じ
フル ID を書くので、目に付いた ID をそのままコピーして渡せる（省略形からの復元は要らない）。
同じ文字列で grep すれば、宣言側（frontmatter の `id:`）と参照側（relation リスト・散文）の
両方に当たる:

```bash
rg -l REQ-2b0c2bb8-964f-4e36-a121-c6ea0d4be1c4 docs/   # 宣言・参照元の列挙
```

## report — 俯瞰

```bash
sara report coverage   # 型ごとのカバー率
sara report matrix     # トレーサビリティマトリクス（--format json で機械可読）
```

工程固定の一覧文書を作らない代わりに、全体像はこの 2 つで取る。

## diff — 参照間の比較

```bash
sara diff <git-ref>    # 指定 ref とのグラフ差分（item / relation の増減）
```

PR レビューで「このブランチでグラフに何が増えたか」を見るときに使う。

## schema — モデルの確認

```bash
sara schema            # 有効な model schema を YAML で出力
```

型・relation・フィールド定義の正本は `docs/model.yaml`。編集前に schema で現状を確認する。

## sara-gap — 未カバー 3 段の列挙（別コマンド）

```bash
sara-gap           # text: セクション見出し + <ref>\t<name>\t<file>
sara-gap --json    # {unthreatened, unmitigated, uncovered}
```

sara 本体ではなく devShell 同梱の独立コマンド。valid なグラフに対してだけ動く
（`sara check` が落ちる間は exit 2 で一覧を出さない）。工程スキルは着手点の特定に使う。
