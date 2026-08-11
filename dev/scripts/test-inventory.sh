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

usage() {
  cat >&2 <<'EOF'
usage: test-inventory.sh (--static | --full)

  --static  ファイル粒度で列挙する（fd / glob のみ。go test も nix eval も呼ばない）
  --full    テスト名粒度で列挙する（go test -json + nix eval を呼ぶため重い）

出力は TSV。詳細はスクリプト冒頭のコメントを参照。
EOF
}

case "${1-}" in
  -h | --help)
    usage
    exit 0
    ;;
  --static) mode=static ;;
  --full) mode=full ;;
  *)
    usage
    exit 2
    ;;
esac

if [ "$#" -ne 1 ]; then
  usage
  exit 2
fi

# 走査はリポジトリルート基準で行う（dev/ 等から叩いても同じ結果を出す）。
repo_root=$(git rev-parse --show-toplevel 2>/dev/null || printf '.')
cd "$repo_root" || exit 1

# flake check の静的リスト。`nix eval` を避けるため列挙を持つ（--static が毎 PR で
# 回る契約テストの入力であり、flake 評価を挟むと sara ジョブに nix ビルドが要る）。
#
# ここは flake.nix / dev/flake.nix の checks 定義と手で揃える契約。CI の path filter が
# flake.nix を含むため、check を増減して当リストを更新し忘れると契約テストが落ちる
# （除外リストにも CASE にも無い check が現れる / 消えた check の CASE が残る）。
# treefmt は treefmt-nix flakeModule が自動生成する。
FLAKE_CHECKS=(
  checks.go-vet
  checks.golangci-lint
  checks.hm-module
  checks.namaka
  checks.nix-unit
  checks.nput
  checks.treefmt
  dev:checks.sara-id
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
list_namaka_dirs() {
  find tests/namaka -mindepth 1 -maxdepth 1 -type d 2>/dev/null |
    sed 's|^\./||' |
    grep -v '/_' |
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

if [ "$mode" = static ]; then
  emit_static | LC_ALL=C sort
  exit 0
fi

# --- 列挙（テスト名粒度・--full のみ） ---------------------------------------

for cmd in go nix jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "test-inventory.sh: --full は $cmd を要する（nix develop ./dev から実行する）" >&2
    exit 1
  fi
done

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# Go: 実行ベースでテスト名を採る（サブテスト込み）。`go test -json` の run イベントを
# 集め、`Test` を取り出す。ビルド失敗・テスト失敗でも run イベントは出るため、
# 終了コードは見ない（列挙が目的で合否判定ではない）。
go test -json ./... 2>/dev/null |
  jq -r 'select(.Action == "run" and .Test != null) | .Test' |
  LC_ALL=C sort -u > "$work/go-names"

if [ ! -s "$work/go-names" ]; then
  echo "test-inventory.sh: go test -json からテスト名を採れなかった" >&2
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
# `{ lib, nput }` で import して `//` マージするだけなので、同じシグネチャで直接呼ぶ。
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
