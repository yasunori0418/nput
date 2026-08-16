#!/usr/bin/env bash
# テスト資産の列挙を一元化する（→ Issue #304、epic #283）。
#
# 実行:
#   dev/scripts/test-inventory.sh --static   # ファイル粒度（契約テストが使う）
#   dev/scripts/test-inventory.sh --full     # テスト名粒度（対応表生成が使う）
#
# 出力（TSV・1 行 1 レコード・資産識別子でソート）:
#   --static: <資産識別子>\t<種別>
#   --full:   <資産識別子>\t<種別>\t<テスト名>
#             テスト名を持たない種別（e2e / namaka / flake-check）は 3 列目が空。
#
# 資産識別子の正準表記は CASE frontmatter の target と同一の 2 規則:
#   - 実在するファイル → リポジトリ相対パス（namaka だけは末尾スラッシュのディレクトリ）
#   - flake check      → checks.<name> 形式（dev flake は dev:checks.<name>）
#
# ## 2 モードの役割分担
#
# --static はファイル走査（fd / glob）だけで完結し、go test も nix eval も呼ばない。
# 契約テストは毎 PR で回るため、Go のビルドや flake 評価に依存させない（sara ジョブは
# sara devShell だけで完結する）。flake check の一覧は静的リストで持つ（下記）。
#
# --full は Go のテスト名を `go test -json` の実行ベースで採る（サブテスト込み。
# 静的 AST 解析は採らない → grilling 2026-08-11 で確定）。テスト名 → ファイルの
# 帰属は `^func Test` の grep で決め、サブテストは親のファイルへ寄せる。
# nix-unit は per-file の `builtins.attrNames` を引く。どちらも重いため、
# 対応表生成（main push / workflow_dispatch）でのみ使う。

set -uo pipefail

# shellcheck source=dev/scripts/lib-testdoc.sh
. "$(dirname "$0")/lib-testdoc.sh"

usage() {
  cat >&2 <<'EOF'
usage: test-inventory.sh (--static | --full | --module-generated-checks)

  --static  ファイル粒度で列挙する（fd / glob のみ。go test も nix eval も呼ばない）
  --full    テスト名粒度で列挙する（go test -json + nix eval を呼ぶため重い）
  --module-generated-checks
            flakeModule 由来（flake ファイルに定義行を持たない）check だけを列挙する。
            契約テストが flake ⟷ 静的リストの grep 突合から除くために使う

出力は TSV。詳細はスクリプト冒頭のコメントを参照。
EOF
}

# 引数個数は mode 判定より先に見る（`--help extra` も `--static extra` も同じく exit 2）。
if [ "$#" -ne 1 ]; then
  usage
  exit 2
fi

case "$1" in
  -h | --help)
    usage
    exit 0
    ;;
  --static) mode=static ;;
  --full) mode=full ;;
  --module-generated-checks) mode=module-generated-checks ;;
  *)
    usage
    exit 2
    ;;
esac

# 走査はリポジトリルート基準で行う（dev/ 等から叩いても同じ結果を出す）。
cd "$(testdoc_repo_root)" || exit 1

# flake check の静的リスト。`nix eval` を避けるため列挙を持つ（--static が毎 PR で
# 回る契約テストの入力であり、flake 評価を挟むと sara ジョブに nix ビルドが要る）。
#
# ここは flake.nix / dev/flake.nix の checks 定義と揃える契約で、その突合は
# dev/tests/test-doc-map.sh の §5 が機械的に行う（flake ファイルから `checks.<name> =`
# の定義名を grep して当リストと集合比較する）。**この突合が無いと片側更新漏れは
# 検知できない**: check を flake へ足して当リストへ足し忘れると、その check は列挙に
# 現れないので順方向・逆方向のどちらも触れず黙って緑になる（実際に踏んだ →
# review 2026-08-11）。逆向き（リストに残った消えた check）も、対応する CASE / 除外行が
# 一緒に残っていれば通ってしまう。
#
# MODULE_GENERATED_CHECKS は flakeModule が生成し flake ファイルに定義行を持たないため
# grep 突合の対象外にする。こちらは手で揃えるほか手段が無い。
FLAKE_CHECKS=(
  checks.go-vet
  checks.golangci-lint
  checks.hm-module
  checks.namaka
  checks.nix-unit
  checks.nput
  checks.treefmt
  dev:checks.risk-matrix
  dev:checks.sara-gap
  dev:checks.sara-id
  dev:checks.test-doc-map
)

# flakeModule 由来（flake ファイルに `checks.<name> =` の定義行が無い）check。
# nix-unit は nix-unit の flakeModule、treefmt は treefmt-nix の flakeModule が生む。
# 契約テストの §5 はこの 2 件を grep 突合から除く。
MODULE_GENERATED_CHECKS=(
  checks.nix-unit
  checks.treefmt
)

# --- 列挙（ファイル粒度） ----------------------------------------------------
#
# fd ではなく find + glob を使う。契約テストは sara devShell（fd 無し）から走るため。

list_go_files() {
  find cmd internal -name '*_test.go' -type f 2>/dev/null | sed 's|^\./||'
}

list_nix_unit_files() {
  find tests/nix-unit -maxdepth 1 -name '*.nix' -type f 2>/dev/null | sed 's|^\./||'
}

# namaka はスナップショット実体（tests/namaka/_snapshots/）と expr.nix が対になるため、
# CASE の粒度はディレクトリ（末尾スラッシュ）。`_` 始まりの内部ディレクトリは除く。
# 除外は find の -name で行う（basename だけを見る）。`grep -v '/_'` はパス中のどの位置の
# `/_` にも当たるため、走査基点が変わると意図より広く落とす。
list_namaka_dirs() {
  find tests/namaka -mindepth 1 -maxdepth 1 -type d -not -name '_*' 2>/dev/null |
    sed 's|^\./||' |
    sed 's|$|/|'
}

list_e2e_files() {
  find tests/e2e/scenarios -name '*.sh' -type f 2>/dev/null | sed 's|^\./||'
}

emit_static() {
  list_go_files | sed 's|$|\tgo|'
  list_nix_unit_files | sed 's|$|\tnix-unit|'
  list_namaka_dirs | sed 's|$|\tnamaka|'
  list_e2e_files | sed 's|$|\te2e|'
  printf '%s\tflake-check\n' "${FLAKE_CHECKS[@]}"
}

if [ "$mode" = module-generated-checks ]; then
  printf '%s\n' "${MODULE_GENERATED_CHECKS[@]}" | LC_ALL=C sort
  exit 0
fi

if [ "$mode" = static ]; then
  emit_static | LC_ALL=C sort
  exit 0
fi

# --- 列挙（テスト名粒度・--full のみ） ---------------------------------------

require_commands "test-inventory.sh --full（nix develop ./dev から実行する）" go nix jq || exit 1

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# Go: 実行ベースでテスト名を採る（サブテスト込み）。`go test -json` の run イベントを
# 集め、`Test` を取り出す。テスト失敗でも run イベントは出るため終了コードは見ない
# （列挙が目的で合否判定ではない）。
#
# ただしビルド失敗は別扱いにする。ビルドできなかったパッケージは run イベントを 1 つも
# 出さないので、他パッケージが成功していれば全体は非空のまま**一部だけ静かに欠ける**。
# 欠けた対応表を artifact として出すより、列挙が不完全であることを報告して落とす。
go test -json ./... 2>/dev/null > "$work/go-json"

jq -r 'select(.Action == "run" and .Test != null) | .Test' "$work/go-json" |
  LC_ALL=C sort -u > "$work/go-names"

if [ ! -s "$work/go-names" ]; then
  echo "test-inventory.sh: go test -json からテスト名を採れなかった" >&2
  exit 1
fi

# 列挙が不完全になる失敗の検出。テストを 1 件も走らせずに fail したパッケージを拾う。
# ビルド失敗のほか TestMain / init の異常終了も同じ形（Action=fail・Test=null・run
# イベント無し）で出るため、診断は両義に留める（どちらでも列挙は欠ける）。
jq -r 'select(.Action == "run" and .Package != null) | .Package' "$work/go-json" |
  LC_ALL=C sort -u > "$work/go-ran-packages"
jq -r 'select(.Action == "fail" and .Package != null and .Test == null) | .Package' "$work/go-json" |
  LC_ALL=C sort -u > "$work/go-failed-packages"

unbuilt=$(LC_ALL=C comm -13 "$work/go-ran-packages" "$work/go-failed-packages")
if [ -n "$unbuilt" ]; then
  echo "test-inventory.sh: テストを 1 件も走らせずに失敗したパッケージがある" >&2
  echo "  （ビルド失敗、または TestMain / init の異常終了）:" >&2
  printf '%s\n' "$unbuilt" | sed 's/^/  /' >&2
  echo "test-inventory.sh: 列挙が不完全なので中断する（go test <パッケージ> で原因を確認する）" >&2
  exit 1
fi

# テスト名 → ファイルの帰属表。トップレベル関数は `^func Test` の grep で決まる。
# 同名のトップレベル関数はパッケージを跨いでも Go の慣習上まず衝突しないが、
# 万一衝突したら両ファイルへ出す（列挙の欠落より重複のほうが害が小さい）。
: > "$work/func-map"
while IFS= read -r file; do
  grep -oE '^func (Test[A-Za-z0-9_]*)' "$file" |
    sed 's/^func //' |
    while IFS= read -r fn; do
      printf '%s\t%s\n' "$fn" "$file" >> "$work/func-map"
    done
done < <(list_go_files)

# サブテスト（Parent/Sub）は親のファイルへ寄せる。
while IFS= read -r name; do
  toplevel=${name%%/*}
  files=$(awk -F'\t' -v fn="$toplevel" '$1 == fn { print $2 }' "$work/func-map")
  if [ -z "$files" ]; then
    # 帰属先不明。列挙から落とすと対応表が黙って欠けるため、識別子を空にせず報告する。
    echo "test-inventory.sh: 警告: テスト $name の定義ファイルを特定できなかった" >&2
    continue
  fi
  while IFS= read -r file; do
    printf '%s\tgo\t%s\n' "$file" "$name"
  done <<< "$files"
done < "$work/go-names" > "$work/go-rows"

# nix-unit: per-file の attrNames。アグリゲータ（tests/nix-unit.nix）は各ファイルを
# `{ lib, nput }` で import し、マージ前にファイル横断の名前衝突を検査してから
# `//` マージする（→ Issue #287）。ここが要るのは per-file の名前一覧だけなので、
# 検査を経ずに同じシグネチャで leaf を直接呼ぶ。衝突の検出は評価時に
# `nix flake check` の checks.nix-unit が担い、ここでは二重に持たない
# （bash / nix の二重実装はドリフトする → Issue #308）。
# getFlake のため --impure が要る（--full は対応表生成専用なので許容する）。
: > "$work/nix-unit-rows"
while IFS= read -r file; do
  names=$(nix eval --impure --json --expr "
    let
      flake = builtins.getFlake (builtins.toString ./.);
      lib = flake.inputs.nixpkgs.lib;
      nput = import ./lib;
    in
    builtins.attrNames (import ./$file { inherit lib nput; })
  " 2>/dev/null | jq -r '.[]')
  if [ -z "$names" ]; then
    echo "test-inventory.sh: 警告: $file から nix-unit のテスト名を採れなかった" >&2
    printf '%s\tnix-unit\t\n' "$file" >> "$work/nix-unit-rows"
    continue
  fi
  while IFS= read -r name; do
    printf '%s\tnix-unit\t%s\n' "$file" "$name" >> "$work/nix-unit-rows"
  done <<< "$names"
done < <(list_nix_unit_files)

# e2e / namaka / flake check は static と同粒度（テスト名の内訳を持たない）。
{
  cat "$work/go-rows"
  cat "$work/nix-unit-rows"
  list_namaka_dirs | sed 's|$|\tnamaka\t|'
  list_e2e_files | sed 's|$|\te2e\t|'
  printf '%s\tflake-check\t\n' "${FLAKE_CHECKS[@]}"
} | LC_ALL=C sort
