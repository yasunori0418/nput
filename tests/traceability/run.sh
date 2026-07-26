#!/usr/bin/env bash
# テストプロセス成果物のトレーサビリティ集計（→ docs/test/<対象>/test-monitoring.md）。
#
# 仕様（docs/dev/<対象>/spec.md）の REQ / NFR / AC、テスト計画のプロダクトリスク R#、
# テスト分析のテスト条件 TC#、テストケース CASE# の ID 列を突き合わせ、下流成果物へ
# 引き継がれていない ID を列挙する。Nix 評価も Go ビルドも要さず、bash + awk のみで動く
# （docs のみの変更でも走らせられるようにするため。→ ADR-0027 の nix 実走最適化と無関係に保つ）。
#
# 計測・表示のみで閾値ゲートは持たない（go-coverage と同じ方針。→ ADR-0030 の required
# status check には登録しない）。カバー率が下がっても CI は赤くならず、数値の評価は
# test-report 工程が行う。
#
# 使い方: tests/traceability/run.sh <対象名>
# 出力: 標準出力へ markdown（CI では GITHUB_STEP_SUMMARY へリダイレクトする）。
# 終了コード: 集計できたら 0。対象の仕様が存在しない場合も「計測対象なし」として 0。
#            引数不正（対象名なし・パス区切りを含む対象名）のみ 2。
set -uo pipefail

target="${1:-}"
if [ -z "$target" ]; then
	echo "run.sh: 対象名を指定してください（例: run.sh quality-observability）" >&2
	exit 2
fi
# 対象名は docs 配下のディレクトリ名に展開するため、パス区切りと親参照を弾く。
case "$target" in
*/* | *..*)
	echo "run.sh: 対象名にパス区切りは使えません: $target" >&2
	exit 2
	;;
esac

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SPEC="$REPO_ROOT/docs/dev/$target/spec.md"
PLAN="$REPO_ROOT/docs/test/$target/test-plan.md"
ANALYSIS="$REPO_ROOT/docs/test/$target/test-analysis.md"
CASES="$REPO_ROOT/docs/test/$target/test-case.md"

# 仕様が無い対象は「計測対象なし」として正常終了する。機能単位の作業ディレクトリ
# （docs/dev/<対象>/）は完成後に削除される運用のため（→ docs/dev/definition-of-done.md）、
# ここで異常終了すると撤去のたびに CI が赤くなり、ゲートを持たない設計と矛盾する。
if [ ! -f "$SPEC" ]; then
	echo "## トレーサビリティ集計: $target"
	echo
	echo "計測対象なし（\`docs/dev/$target/spec.md\` が存在しません）。対象の作業ディレクトリが"
	echo "撤去済みであれば、\`.github/workflows/test.yml\` の \`traceability\` job も併せて"
	echo "削除してください。"
	exit 0
fi

# ---- ID 抽出 -----------------------------------------------------------------
# 表の第 1 セルに置かれた ID を拾う（`| AC-01 | ... |`）。出現順を保ち、重複は落とす。
ids_in_first_cell() {
	# $1: ファイル, $2: ID の正規表現（awk 用）
	[ -f "$1" ] || return 0
	awk -v pat="$2" -F'|' '
		/^\|/ {
			cell = $2
			gsub(/^[ \t]+|[ \t]+$/, "", cell)
			gsub(/\*/, "", cell)
			if (cell ~ ("^" pat "$") && !seen[cell]++) print cell
		}
	' "$1"
}

# REQ は `### REQ-01: ...` の見出し、NFR は `- **NFR-01**: ...` の箇条書きで宣言される。
req_ids() { grep -oE '^### REQ-[0-9]+' "$SPEC" | sed 's/^### //' | awk '!seen[$0]++'; }
nfr_ids() { grep -oE '\*\*NFR-[0-9]+\*\*' "$SPEC" | tr -d '*' | awk '!seen[$0]++'; }

# ---- 参照の有無を判定 --------------------------------------------------------
# $1: ID, $2... : 下流ファイル。いずれかに ID が現れれば 0。
# 前後を非 ID 文字で挟み、AC-01 が AC-019 に誤ヒットしないようにする。下流成果物の
# 対応 ID 列は範囲表記（`AC-07〜AC-10`）を使わず個別列挙する規約（→ test-monitoring.md）。
referenced_in() {
	local id="$1"
	shift
	local f
	for f in "$@"; do
		[ -f "$f" ] || continue
		if grep -qE "(^|[^A-Za-z0-9-])${id}([^0-9]|$)" "$f"; then
			return 0
		fi
	done
	return 1
}

# $1: 集計項目名, $2: 上流 ID の並び（改行区切り）, $3... : 下流ファイル
report_chain() {
	local title="$1" upstream="$2"
	shift 2
	local total=0 covered=0 missing=""
	local id
	while IFS= read -r id; do
		[ -n "$id" ] || continue
		total=$((total + 1))
		if referenced_in "$id" "$@"; then
			covered=$((covered + 1))
		else
			missing="$missing $id"
		fi
	done <<<"$upstream"

	local rate="—"
	if [ "$total" -gt 0 ]; then
		rate="$((covered * 100 / total))%"
	fi
	local missing_cell='なし'
	if [ -n "$missing" ]; then
		missing_cell="\`$(echo "$missing" | tr -s ' ' ',' | sed 's/^,//')\`"
	fi
	printf '| %s | %d | %d | %s | %s |\n' \
		"$title" "$total" "$((total - covered))" "$rate" "$missing_cell"
}

# ---- 出力 --------------------------------------------------------------------
echo "## トレーサビリティ集計: $target"
echo
if [ ! -f "$ANALYSIS" ]; then
	echo "- \`test-analysis.md\` が未作成です。これを下流とする集計（REQ / NFR / AC / R# →"
	echo "  テスト条件）は参照先が 0 件のため、カバー率 0% と全 ID の未カバー列挙になります。"
	echo "  一方これを上流とする集計（テスト条件 TC# → ケース）は母数が 0 になり、"
	echo "  カバー率は \`—\` と表示されます（集計失敗ではありません）。"
	echo
fi
if [ ! -f "$CASES" ]; then
	echo "- \`test-case.md\` が未作成です。これを下流とする集計（テスト条件 TC# → ケース /"
	echo "  AC → ケース）は参照先が 0 件のため、カバー率 0% と全 ID の未カバー列挙になります。"
	echo
fi

echo "| 集計項目 | 母数 | 未カバー | カバー率 | 未カバー ID |"
echo "|---|---|---|---|---|"

report_chain 'REQ → テスト条件' "$(req_ids)" "$ANALYSIS"
report_chain 'NFR → テスト条件' "$(nfr_ids)" "$ANALYSIS"
report_chain 'AC → テスト条件' "$(ids_in_first_cell "$SPEC" 'AC-[0-9]+')" "$ANALYSIS"
report_chain 'リスク R# → テスト条件' "$(ids_in_first_cell "$PLAN" 'R[0-9]+')" "$ANALYSIS"
report_chain 'テスト条件 TC# → ケース' "$(ids_in_first_cell "$ANALYSIS" 'TC-[0-9]+')" "$CASES"
report_chain 'AC → ケース' "$(ids_in_first_cell "$SPEC" 'AC-[0-9]+')" "$CASES"

echo
echo "生成: \`tests/traceability/run.sh $target\`（計測のみ。評価は test-report 工程）"
