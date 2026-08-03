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
#
# 終端判定が行頭アンカーなのは「block scalar の中身は必ずインデントされている」
# という前提に立っている（YAML がそう要求する）。本文中に `alike:` のような
# コロンを含む行があってもインデント済みなので終端と誤認しない。逆にこの前提が
# 崩れると、そこから後ろが見えなくなり違反を見逃す方向へ倒れる。
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
  #
  # 語境界（前後が英字でない）を要求するので、拾うのは助動詞として立つ MUST だけで、
  # 英字が続く MUSTARD / MUSTs は当たらない。これは意図した挙動で、REQ-2381d93a の
  # specification が niface 由来の名詞「the out-of-schema lint MUSTs」（= MUST 群と
  # いうドメイン用語）を含む。助動詞ではないので弾いてはいけない。
  #
  # 折り返しを畳んでから見る。連結の仕方は日本語側（下）と意図的に違えてある:
  # 英語は単語の切れ目が空白なので改行を空白へ置き換えないと語が繋がってしまう。
  # 日本語は折り返し位置に空白が入らないので、逆に改行を落として詰める必要がある。
  spec_joined="$(printf '%s' "$spec" | sed 's/^[[:space:]]*//' | tr '\n' ' ')"

  # 抽出が空になるのは specification が block scalar（`specification: |`）で
  # 書かれていないとき。その場合 grep は当然何にも当たらないので、ガードが無いと
  # MUST を含む item を「違反なし」として通してしまう（検査が黙って外れる方が、
  # 弾かれるより危ない）。specification は required: true なので未記入も同じく違反。
  if [ -z "$(printf '%s' "$spec_joined" | tr -d '[:space:]')" ]; then
    report "$f" 'specification が空、または block scalar（`specification: |`）ではない'
  elif printf '%s' "$spec_joined" | grep -qE '(^|[^A-Za-z])MUST([^A-Za-z]|$)'; then
    report "$f" 'specification に MUST / MUST NOT がある（SHALL / SHALL NOT を使う）'
  fi

  # --- 2. specification_ja は規範助動詞を持つ ------------------------------
  #
  # 語尾のリストは docs/agents/sara-graph.md の写像表が出典で、その 4 強度
  # （〜しなければならない / 〜してはならない / 〜すべきである・すべきでない /
  # 〜してもよい）と、実データに現れる活用形（「〜してはならず、」で後続節へ繋ぐ・
  # 「〜止めてもならない」）だけを載せる。「しなければ」「しては」の部分は活用で
  # 変わるので見ず、語尾だけを見る。写像表を増やすときはここも合わせて増やす。
  #
  # 「〜ものとする」は意図的に載せない。写像表の 4 強度のどれでもなく、どの強度を
  # 課しているのかが読み手に決まらないため。英語の SHALL をこの語尾で受けている
  # item が実際にあったが（REQ-8409db86 / REQ-c1b3ca5f / REQ-67095391）、
  # リストへ足して追認せず写像表どおりの語尾へ直した。同じものが現れたときも
  # ここへ足すのではなく item を直す。
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
    report "$f" 'specification_ja が空、または block scalar（`specification_ja: |`）ではない'
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

# --- 3. 判定器そのものの自己テスト -------------------------------------------
#
# ここまでは実 docs/ を読むだけなので、判定ロジックが壊れて「何も検出しなくなった」
# 退行を検知できない（実 item が全件準拠していれば緑のままになる）。合成 item を
# 置いた一時 docs/ を SPEC_STYLE_DOCS_DIR で食わせ、検出すべきものを検出し、
# 通すべきものを通すことを固定する。
#
# SPEC_STYLE_SELFTEST=1 の再帰呼び出しではこの節を実行しない（無限再帰の停止条件）。

if [ -n "${SPEC_STYLE_SELFTEST:-}" ]; then
  exit "$fail"
fi

selftest_dir="$(mktemp -d)"
trap 'rm -rf "$selftest_dir"' EXIT
mkdir -p "$selftest_dir/requirements"

# 自身を fixture に対して走らせ、stdout を出力・終了コードを status に入れる。
self_out=""
self_status=0
run_self() {
  self_out="$(SPEC_STYLE_SELFTEST=1 SPEC_STYLE_DOCS_DIR="$selftest_dir" bash "$0" 2>&1)" &&
    self_status=0 || self_status=$?
}

write_item() { # $1=filename $2=specification本文 $3=specification_ja本文
  cat >"$selftest_dir/requirements/$1" <<EOF
---
id: "REQ-00000000-0000-4000-8000-000000000000"
type: requirement
name: "selftest"
specification: |
$2
specification_ja: |
$3
---
# selftest
EOF
}

self_fail=0
expect() { # $1=説明 $2=条件が真なら pass
  if [ "$2" -eq 1 ]; then
    printf 'ok   - [selftest] %s\n' "$1"
  else
    printf 'FAIL - [selftest] %s\n' "$1"
    self_fail=1
  fi
}

# (a) 規約準拠の item は通る。
rm -f "$selftest_dir"/requirements/*.md
write_item ok.md '  The engine SHALL do it.' '  engine はそれをしなければならない。'
run_self
expect '準拠 item は exit 0 で通る' "$([ "$self_status" -eq 0 ] && echo 1 || echo 0)"

# (b) specification の MUST を検出する。
rm -f "$selftest_dir"/requirements/*.md
write_item must.md '  The engine MUST NOT do it.' '  engine はそれをしてはならない。'
run_self
expect 'specification の MUST を検出する' \
  "$(printf '%s' "$self_out" | grep -q 'MUST / MUST NOT がある' && echo 1 || echo 0)"

# (c) MUSTs（英字が続く語）は助動詞ではないので通す。
rm -f "$selftest_dir"/requirements/*.md
write_item musts.md '  The engine SHALL honour the lint MUSTs.' '  engine は lint MUSTs に従わなければならない。'
run_self
expect 'MUSTs は助動詞ではないので通す' "$([ "$self_status" -eq 0 ] && echo 1 || echo 0)"

# (d) specification_ja が平叙のみなら検出する。
rm -f "$selftest_dir"/requirements/*.md
write_item plain.md '  The engine SHALL do it.' '  engine はそれをする。'
run_self
expect 'specification_ja の平叙のみを検出する' \
  "$(printf '%s' "$self_out" | grep -q '規範助動詞が無い' && echo 1 || echo 0)"

# (e) 行またぎで折り返された助動詞を分断せず通す（連結処理の退行検知）。
#     実データでも REQ-a33a11e3 / REQ-b74a118a がこの経路に依存している。
rm -f "$selftest_dir"/requirements/*.md
write_item wrapped.md '  The engine SHALL do it.' '  engine はそれをしなけれ
  ばならない。'
run_self
expect '折り返しで分断された助動詞を通す' "$([ "$self_status" -eq 0 ] && echo 1 || echo 0)"

# (f) block scalar でない specification を素通りさせない（空ガード）。
rm -f "$selftest_dir"/requirements/*.md
cat >"$selftest_dir/requirements/inline.md" <<'EOF'
---
id: "REQ-00000000-0000-4000-8000-000000000000"
type: requirement
name: "selftest"
specification: "The engine MUST NOT do it."
specification_ja: |
  engine はそれをしてはならない。
---
# selftest
EOF
run_self
# メッセージは系統ごとに前半で見分ける。単に 'block scalar' で引くと
# specification_ja 側のガードが出した行にも当たり、specification 側の検査が
# 全く効いていなくても合格してしまう。
expect 'block scalar でない specification を違反として報告する' \
  "$(printf '%s' "$self_out" | grep -q 'specification が空、または' && echo 1 || echo 0)"

# (f2) specification_ja 側も同じく block scalar でなければ報告する。
#      2 系統のガードがそれぞれ独立に効いていることを、上の (f) と対で固定する。
rm -f "$selftest_dir"/requirements/*.md
cat >"$selftest_dir/requirements/inline-ja.md" <<'EOF'
---
id: "REQ-00000000-0000-4000-8000-000000000000"
type: requirement
name: "selftest"
specification: |
  The engine SHALL do it.
specification_ja: "engine はそれをしなければならない。"
---
# selftest
EOF
run_self
expect 'block scalar でない specification_ja を違反として報告する' \
  "$(printf '%s' "$self_out" | grep -q 'specification_ja が空、または' && echo 1 || echo 0)"

# (g) item が 1 件も無ければ失敗する（走査先を取り違えたまま緑になるのを防ぐ）。
#     exit code だけでは違反検出時の 1 と区別が付かないので、専用のメッセージで見る。
rm -f "$selftest_dir"/requirements/*.md
run_self
expect 'item が 1 件も無ければ専用メッセージで失敗する' \
  "$([ "$self_status" -eq 1 ] &&
    printf '%s' "$self_out" | grep -q 'requirement item が 1 件も見つからない' &&
    echo 1 || echo 0)"

if [ "$self_fail" -ne 0 ]; then
  fail=1
fi

exit "$fail"
