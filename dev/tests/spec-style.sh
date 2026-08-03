#!/usr/bin/env bash
# requirement item の specification / specification_ja の様式検査（→ Issue #229）。
#
# 実行:
#   nix develop ./dev -c dev/tests/spec-style.sh   # devShell から直接（作業ツリーを見る）
#   nix flake check ./dev                           # checks.spec-style 経由（git 追跡分を見る）
#
# 前者は作業ツリーの docs/ をそのまま読むので編集中の item も検査できる。後者は
# サンドボックスへ持ち込んだ store 上の source を読むため、git add 済みの内容しか
# 見えない。編集中の確認には前者を使う。
#
# 検証対象。番号は下の節見出しに対応する:
#   1. specification（英語）に MUST / MUST NOT が無い（SHALL 系のみを使う）
#   2. specification_ja に写像表の規範助動詞が少なくとも 1 つある
#
# 規約の本文と写像表は docs/agents/sara-graph.md。sara 自身は specification に
# RFC2119 キーワードが「1 つでもあるか」しか見ず、どの綴りを使うかも
# specification_ja の文体も検査しないため、その差分をここで埋める。
#
# 判定は item 単位（ファイル単位）で行う。文単位で「規範文が平叙で終わっていないか」を
# 機械判定しようとすると、背景説明・担当分界の宣言・nput 以外の主体の記述といった
# 正しく平叙であるべき文を落とせず偽陽性が実用に耐えない。文単位の混在は
# レビューで見る（規約は sara-graph.md 側に散文として書いてある）。

# -e は使わない。dev/tests/sara-id.sh と同じ「1 回の実行で全失敗を報告する」集計方式で、
# -e があると最初の違反でスクリプトごと打ち切られ、残りの item の違反が見えなくなる。
set -uo pipefail

fail=0
violations=0

# 走査先はリポジトリルート基準で解決する（sara-id と同じ理由。dev/ から叩いても
# docs/ を見つけられるように）。nix のサンドボックスから走らせる場合は
# SPEC_STYLE_DOCS_DIR で明示的に渡す。
if [ -n "${SPEC_STYLE_DOCS_DIR:-}" ]; then
  req_dir="$SPEC_STYLE_DOCS_DIR/requirements"
else
  repo_root="$(git rev-parse --show-toplevel 2>/dev/null || printf '.')"
  req_dir="$repo_root/docs/requirements"
fi

if [ ! -d "$req_dir" ]; then
  printf 'FAIL - requirement ディレクトリが無い: %s\n' "$req_dir"
  exit 1
fi

# frontmatter の block scalar から 1 フィールドの本文を取り出す。
# 次のトップレベルキー（`^[a-z_]+:`）か frontmatter 終端（`---`）で打ち切る。
extract_field() { # $1=file $2=field
  awk -v field="$2" '
    !inblock && $0 == field ": |" { inblock = 1; next }
    inblock {
      if ($0 ~ /^[a-z_]+:/ || $0 == "---") { exit }
      print
    }
  ' "$1"
}

report() { # $1=file $2=message
  printf 'FAIL - %s: %s\n' "$(basename "$1")" "$2"
  fail=1
  violations=$((violations + 1))
}

checked=0

for f in "$req_dir"/*.md; do
  [ -e "$f" ] || continue
  checked=$((checked + 1))

  spec="$(extract_field "$f" specification)"
  spec_ja="$(extract_field "$f" specification_ja)"

  # --- 1. specification は SHALL 系のみ ------------------------------------
  #
  # MUST / MUST NOT は RFC2119 上 SHALL / SHALL NOT と同義だが、1 つの強度に
  # 綴りが 2 つあると item 群が「両者は違う」と読めてしまう。綴りを寄せる。
  # 語境界を見るのは MUSTARD のような語中一致を拾わないため。
  #
  # 検査対象は単語 MUST なので折り返しで分断されても各行で拾えるが、
  # specification_ja 側と同じ前処理を通して両検査の見ているものを揃える。
  spec_joined="$(printf '%s' "$spec" | sed 's/^[[:space:]]*//' | tr '\n' ' ')"

  if printf '%s' "$spec_joined" | grep -qE '(^|[^A-Za-z])MUST([^A-Za-z]|$)'; then
    report "$f" 'specification に MUST / MUST NOT がある（SHALL / SHALL NOT を使う）'
  fi

  # --- 2. specification_ja は規範助動詞を持つ ------------------------------
  #
  # 写像表の 4 強度の語尾。活用（「〜してはならず、」で後続節へ繋ぐ等）を許すため、
  # 「しなければ」「しては」の部分は見ずに語尾だけを見る。
  #
  # 検査の前に改行とインデントを畳んで 1 行にする。block scalar は 90 文字前後で
  # 折り返してあるので、畳まないと「〜しなけれ / ば ならない」のように助動詞が
  # 行をまたいで分断され、規約に従っている item を違反として弾いてしまう。
  #
  # specification_ja は required: true（docs/model.yaml）なので空欄は sara check が
  # 弾くが、ここでも空を違反として報告する。sara を通さずこのテスト単体で走らせた
  # ときに黙って素通りしないようにするため。
  spec_ja_joined="$(printf '%s' "$spec_ja" | sed 's/^[[:space:]]*//' | tr -d '\n')"

  if [ -z "$(printf '%s' "$spec_ja_joined" | tr -d '[:space:]')" ]; then
    report "$f" 'specification_ja が空'
  elif ! printf '%s' "$spec_ja_joined" |
    grep -qE 'なければならない|てはならない|てもならない|ならず|べきである|べきでない|べきではない|てもよい'; then
    report "$f" 'specification_ja に規範助動詞が無い（平叙形のみ）'
  fi
done

if [ "$checked" -eq 0 ]; then
  printf 'FAIL - requirement item が 1 件も見つからない: %s\n' "$req_dir"
  exit 1
fi

if [ "$fail" -eq 0 ]; then
  printf 'ok   - requirement %d 件の specification / specification_ja が様式規約に準拠する\n' "$checked"
else
  printf '\n%d 件の違反（requirement %d 件を検査）。規約は docs/agents/sara-graph.md\n' \
    "$violations" "$checked"
fi

exit "$fail"
