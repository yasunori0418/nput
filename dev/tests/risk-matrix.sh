#!/usr/bin/env bash
# risk の level 導出マトリクス整合の契約テスト（→ Issue #303）。
#
# 実行:
#   nix develop '.?dir=dev#sara' -c dev/tests/risk-matrix.sh   # devShell / CI から直接
#   nix flake check ./dev                                       # checks.risk-matrix 経由
#
# 検証対象。番号は下の節見出しに対応する:
#   1.  下の LEVEL_MATRIX が docs/agents/sara-graph.md の level 表と一致する（契約テスト）
#   2.  docs/risks/*.md 全件で、frontmatter の level が likelihood × impact の
#       マトリクス導出と一致する
#
# §1 が要るのは、コーパスが 9 セルのうち一部しか踏まないため。実測（2026-08-13）で
# 32 件が踏むのは 5 セルだけで、残り 4 セルの転記を誤っても §2 は全件緑のまま通る。
# 表の正本は散文側にあるので、その 3 行を読んで突き合わせる（sara-id.sh §6b が
# docs/model.yaml と dev/flake.nix を突合するのと同じ構図）。
#
# フィールドの存在・enum 妥当性（high / medium / low のいずれか）は docs/model.yaml が
# 宣言し `sara check` が検証する担当なので、ここでは重複させない。ここが見るのは
# 3 フィールドの「関係」だけで、値が enum 外・欠損のときはマトリクスを引けないため
# その旨を FAIL として報告する（sara check の代替をするのではなく、判定不能を黙って
# 素通りさせないための扱い）。
#
# ## なぜ機械化するか
#
# level 導出は `docs/agents/sara-graph.md` の規約のうち唯一「散文の解釈を伴わない
# 純粋な写像」で、スクリプトが判定しきれる。目視レビューはこの検査の最も弱い形。

# -e は使わない（sara-id.sh・test-doc-map.sh と同じ理由）。このテストは「1 回の実行で
# 全失敗を報告する」集計方式で、-e があると最初の非ゼロ終了で以降のアサーションが
# 走らず、退行時の診断が先頭 1 件で切れる。
set -uo pipefail

fail=0
pass() { printf 'ok   - %s\n' "$1"; }
fault() {
  printf 'FAIL - %s\n' "$1"
  fail=1
}

# level 導出マトリクス。正本は docs/agents/sara-graph.md「How a risk is scored」節の
# `level` 表で、ここはその 9 セルを機械可読に写したもの。写しが正本とずれていないことは
# §1 が機械的に突き合わせる（9 セル固定・共用する消費者もいないため TSV 等へ外出ししない）。
#
#   likelihood \ impact | high   | medium | low
#   high                | high   | high   | medium
#   medium              | high   | medium | low
#   low                 | medium | low    | low
declare -A LEVEL_MATRIX=(
  [high:high]=high
  [high:medium]=high
  [high:low]=medium
  [medium:high]=high
  [medium:medium]=medium
  [medium:low]=low
  [low:high]=medium
  [low:medium]=low
  [low:low]=low
)

# 走査基点をリポジトリルートへ解決する。git 管理外ではカレント基準へフォールバックする
# （checks 派生のサンドボックスは作業ツリーを持たないためこの経路を通る）。
repo_root=$(git rev-parse --show-toplevel 2>/dev/null || printf '.')

# 読む正本 2 つの在り処は 2 経路ある（sara-id.sh §6b と同じ扱い）:
#   1. RISK_DOCS_DIR / SARA_GRAPH_MD（checks.risk-matrix のサンドボックス。作業ツリーが
#      無いので nix が store path を渡す）
#   2. git のリポジトリルート基準（`nix develop` からの直接実行・CI の sara job）
risks_dir="${RISK_DOCS_DIR:-}"
[[ -d "$risks_dir" ]] || risks_dir="$repo_root/docs/risks"

graph_md="${SARA_GRAPH_MD:-}"
[[ -f "$graph_md" ]] || graph_md="$repo_root/docs/agents/sara-graph.md"

if [[ ! -d "$risks_dir" ]]; then
  echo "risk-matrix.sh: risk item のディレクトリを解決できない（$risks_dir）" >&2
  exit 1
fi

# frontmatter（先頭の `---` から次の `---` まで）から 1 フィールドを取り出す。
# yq を足さず sed で済ませる。範囲を frontmatter に限るのは予防措置で、現コーパスの
# 本文に `^likelihood:` の形で当たる行は無い（根拠記述は「likelihood を medium と
# するのは …」の形）。本文へその形の行が現れた時点で判定が狂うのを未然に防ぐ。
#
# 値は引用符と前後の空白を剥がす。剥がさないと `impact: "high"` のような表記ゆれが
# 「マトリクスのセルが無い」として報告され、enum 違反と区別が付かなくなる。
field_of() {
  sed -n '1{/^---$/!q}; 1,/^---$/{ /^---$/d; s/^'"$2"':[[:space:]]*//p }' "$1" |
    head -1 |
    sed -e 's/[[:space:]]*$//' -e 's/^"\(.*\)"$/\1/' -e "s/^'\(.*\)'$/\1/"
}

# --- 1. LEVEL_MATRIX が sara-graph.md の level 表と一致する -------------------
#
# 正本の表は 3 行 4 列の Markdown テーブルで、行頭が likelihood、以降が impact
# high / medium / low の順に並ぶ:
#
#   | **`high`** | `high` | `high` | `medium` |
#
# 行頭が `| **` で始まる行だけを拾えば、上の見出し行（`| `likelihood` \ …`）と
# 区切り行（`|---|`）を巻き込まずに 3 行が取れる。バッククォート・アスタリスク・
# パイプ・空白を落として 4 語にすると、そのまま likelihood + 3 セルになる。
if [[ ! -f "$graph_md" ]]; then
  fault "level 表の正本を解決できない（$graph_md）"
else
  matrix_rows=0
  matrix_ok=1
  while read -r row_likelihood cell_high cell_medium cell_low; do
    matrix_rows=$((matrix_rows + 1))
    for impact_col in high medium low; do
      case "$impact_col" in
        high) doc_level="$cell_high" ;;
        medium) doc_level="$cell_medium" ;;
        low) doc_level="$cell_low" ;;
      esac
      script_level="${LEVEL_MATRIX[$row_likelihood:$impact_col]:-}"
      if [[ "$script_level" != "$doc_level" ]]; then
        fault "マトリクスが正本とずれている（likelihood=$row_likelihood impact=$impact_col: sara-graph.md=$doc_level スクリプト=$script_level）"
        matrix_ok=0
      fi
    done
  done < <(
    awk '
      /^### `level`/ { in_section = 1; next }
      in_section && /^## / { in_section = 0 }
      !in_section        { next }
      /^\| \*\*/ {
        gsub(/[`*|]/, " ")
        print $1, $2, $3, $4
      }
    ' "$graph_md"
  )

  # 抽出が壊れた状態で突合すると、ループが 0 周回って全て緑に見える。
  # 表の形（3 行）を先に固定する。
  if [[ "$matrix_rows" -ne 3 ]]; then
    fault "sara-graph.md の level 表から 3 行を抽出できない（実際: $matrix_rows 行）"
  elif [[ "$matrix_ok" -eq 1 ]]; then
    pass "LEVEL_MATRIX の 9 セルが sara-graph.md の level 表と一致する"
  fi
fi

# --- 2. 全 risk item の level が導出と一致する --------------------------------

# ファイル一覧は改行区切りで受ける（この corpus のファイル名は日付 + UUID8 + slug で
# 空白・改行を含まない → CLAUDE.md の ID 規約）。
mapfile -t risk_files < <(find "$risks_dir" -maxdepth 1 -type f -name '*.md' | LC_ALL=C sort)

# 対象 0 件を緑にしない。走査先の解決を誤ったとき（ディレクトリ移動・環境変数の
# 誤配線）、件数を見ないと「全件合格」と報告してしまう。
if [[ "${#risk_files[@]}" -eq 0 ]]; then
  fault "docs/risks に risk item が 1 件も無い（走査先: $risks_dir）"
  exit "$fail"
fi

checked=0
for f in "${risk_files[@]}"; do
  name="$(basename "$f")"
  likelihood="$(field_of "$f" likelihood)"
  impact="$(field_of "$f" impact)"
  level="$(field_of "$f" level)"

  if [[ -z "$likelihood" || -z "$impact" || -z "$level" ]]; then
    fault "$name: frontmatter から 3 フィールドを読めない（likelihood=$likelihood impact=$impact level=$level）"
    continue
  fi

  want="${LEVEL_MATRIX[$likelihood:$impact]:-}"
  if [[ -z "$want" ]]; then
    # 生値を引用符で囲んで出す。enum 外の値と、剥がしきれなかった表記ゆれ
    # （全角空白・内側の引用符など）を目視で区別できるようにする。
    fault "$name: likelihood='$likelihood' impact='$impact' に対応するマトリクスのセルが無い（enum 外の値か表記ゆれ）"
    continue
  fi

  if [[ "$level" != "$want" ]]; then
    fault "$name: level が導出と食い違う（likelihood=$likelihood impact=$impact → $want、実際: $level）"
    continue
  fi

  checked=$((checked + 1))
done

if [[ "$checked" -eq "${#risk_files[@]}" ]]; then
  pass "risk $checked 件の level が likelihood × impact の導出と一致する"
fi

exit "$fail"
