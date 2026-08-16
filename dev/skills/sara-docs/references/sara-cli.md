# sara CLI の使い方

sara は devShell 経由で使う（例: `nix develop ./dev --command sara ...`。呼び出し形は
プロジェクトの CLAUDE.md に従う）。設定はカレント（リポジトリルート）の `sara.toml` を
読むため、**リポジトリルートで実行する**。

## init — item の作成（frontmatter の生成）

frontmatter は手書き・テンプレ写経をせず `sara init` で生成する。ID は先に `sara-id` で
採番し、`--id` で渡す（init 自身の自動採番は連番前提なので使わない）:

```bash
sara-id risk lock-ordering            # id / filename / ref の 3 行が返る
sara init risk docs/risks/<filename> \
  --id "<sara-id が返した正式 ID>" \
  --name "<日本語の 1 行要約>" \
  --threatens "<REQ / DSG のフル ID>" \
  --likelihood medium --impact high --level high
```

- 型ごとのオプション（relation・固有フィールド）は `sara init <型> --help` で確認する。
  requirement / quality / test_plan の `--specification` / `--specification-ja`、
  test_case の `--target` など、モデルの必須フィールドはここで渡す。
- 生成後に本文（`# <PREFIX>-<前方8>: <name>` の見出し + 散文）を書き足し、`sara check`
  で green を確認する。
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

**`sara query` はフル ID しか受け付けない**（省略形 `REQ-2b0c2bb8` は "Item not found"）。
散文中の省略形からフル ID を得る手順:

```bash
rg -l 2b0c2bb8 docs/                                              # 宣言・参照元の列挙
rg -o 'REQ-2b0c2bb8[0-9a-f-]*' docs/requirements/*2b0c2bb8*.md | head -1   # フル ID の復元
```

省略形は正式 ID の前方一致なので、8 文字で grep すれば宣言側（frontmatter の `id:`）と
参照側（relation リスト・散文）の両方に当たる。

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
