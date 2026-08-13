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
# 正本の表は Markdown テーブルで、見出しが impact の列順を、以降の各行が likelihood
# ごとのセルを持つ:
#
#   | `likelihood` \ `impact` | `high` | `medium` | `low` |
#   | **`high`** | `high` | `high` | `medium` |
#
# 見出し行が impact の列順を宣言し、行頭が `| **` の 3 行が likelihood ごとのセルを
# 持つ。列順は見出しから読む（決め打ちにすると、見出しとデータ行を揃えて列を入れ替えた
# 表が素通りする）。行ラベルも読み取って、照合したセルの集合が LEVEL_MATRIX の 9 キーと
# 過不足なく一致することまで確かめる（行の重複・欠落・列の増減はここで落ちる）。
#
# awk はタグ付きで出す: 見出しは `H <impact> …`、データ行は `R <likelihood> <cell> …`。
# バッククォート・アスタリスク・パイプを落とすと語に割れる。見出しの第 1 セルは
# `likelihood \ impact` なのでラベルではなく、`\` の後ろまで含めて捨てる。
if [[ ! -f "$graph_md" ]]; then
  fault "level 表の正本を解決できない（$graph_md）"
else
  impact_cols=()
  matrix_rows=0
  matrix_ok=1
  declare -A compared=()

  while read -r kind rest; do
    case "$kind" in
      H)
        # shellcheck disable=SC2206 # 語分割で列名の配列にする（値は英小文字のみ）
        impact_cols=($rest)
        ;;
      R)
        matrix_rows=$((matrix_rows + 1))
        # shellcheck disable=SC2206
        row=($rest)
        row_label="${row[0]}"
        cells=("${row[@]:1}")
        if [[ "${#cells[@]}" -ne "${#impact_cols[@]}" ]]; then
          fault "level 表の行 $row_label のセル数が見出しと違う（見出し ${#impact_cols[@]} 列・行 ${#cells[@]} 列）"
          matrix_ok=0
          continue
        fi
        for i in "${!impact_cols[@]}"; do
          key="$row_label:${impact_cols[$i]}"
          if [[ -n "${compared[$key]:-}" ]]; then
            fault "level 表でセル $key が 2 回現れる（行ラベルか列名が重複している）"
            matrix_ok=0
            continue
          fi
          compared["$key"]=1
          script_level="${LEVEL_MATRIX[$key]:-}"
          if [[ "$script_level" != "${cells[$i]}" ]]; then
            fault "マトリクスが正本とずれている（$key: sara-graph.md=${cells[$i]} スクリプト=${script_level:-（キー無し）}）"
            matrix_ok=0
          fi
        done
        ;;
    esac
  done < <(
    awk '
      /^### `level`/ { in_section = 1; next }
      in_section && /^## / { in_section = 0 }
      !in_section        { next }
      # 見出し行。第 1 セルの `likelihood \ impact` を捨てて impact の列名だけ出す。
      /^\| `likelihood`/ {
        sub(/^[^\\]*\\/, "")
        gsub(/[`|]/, " ")
        line = "H"
        for (i = 2; i <= NF; i++) { line = line " " $i }
        print line
        next
      }
      # データ行。第 1 セルが likelihood のラベル。
      /^\| \*\*/ {
        gsub(/[`*|]/, " ")
        line = "R"
        for (i = 1; i <= NF; i++) { line = line " " $i }
        print line
      }
    ' "$graph_md"
  )

  # 抽出が壊れた状態で突合すると、ループが 0 周回って全て緑に見える。
  # 「照合したセルの集合 == LEVEL_MATRIX のキー集合」を最後に確かめれば、
  # 行数・列数・ラベルの取り違えが 1 つの判定に畳まれる。
  if [[ "${#impact_cols[@]}" -eq 0 || "$matrix_rows" -eq 0 ]]; then
    fault "sara-graph.md の level 表を抽出できない（見出し ${#impact_cols[@]} 列・データ $matrix_rows 行）"
  elif [[ "${#compared[@]}" -ne "${#LEVEL_MATRIX[@]}" ]]; then
    fault "level 表と照合したセルが LEVEL_MATRIX の全キーを覆っていない（照合 ${#compared[@]} セル・LEVEL_MATRIX ${#LEVEL_MATRIX[@]} セル）"
  else
    for key in "${!LEVEL_MATRIX[@]}"; do
      if [[ -z "${compared[$key]:-}" ]]; then
        fault "LEVEL_MATRIX の $key に対応するセルが level 表に無い"
        matrix_ok=0
      fi
    done
    if [[ "$matrix_ok" -eq 1 ]]; then
      pass "LEVEL_MATRIX の ${#LEVEL_MATRIX[@]} セルが sara-graph.md の level 表と一致する"
    fi
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
