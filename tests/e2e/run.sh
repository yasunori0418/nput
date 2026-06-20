#!/usr/bin/env bash
# 非 NixOS E2E ハーネスのオーケストレータ（→ docs/design.md「テスト戦略」・ADR-0012）。
#
# 実 nix を使って `nput` を end-to-end に駆動し、「非 NixOS でも nix さえあれば動く」主張を
# 検証する。CI からは `nix develop '.?dir=dev#ci' -c tests/e2e/run.sh` で起動する。
# scenarios/*.sh を辞書順に各々独立プロセスで実行し、1 つでも失敗すれば非ゼロ終了する。
set -uo pipefail

E2E_DIR="$(cd "$(dirname "$0")" && pwd)"
SCENARIO_DIR="$E2E_DIR/scenarios"

# nput / nix の所在を起動時に一度確認する（早期に分かりやすく落とす）。
NPUT="${NPUT:-nput}"
if ! command -v "$NPUT" >/dev/null 2>&1; then
	echo "run.sh: nput が PATH にありません（CI は ci devShell 経由で起動してください）" >&2
	exit 127
fi
if ! command -v nix >/dev/null 2>&1; then
	echo "run.sh: nix が PATH にありません" >&2
	exit 127
fi

echo "nput: $(command -v "$NPUT")"
echo "nix:  $(nix --version)"

fails=0
total=0
for scenario in "$SCENARIO_DIR"/*.sh; do
	[ -e "$scenario" ] || continue
	total=$((total + 1))
	name="$(basename "$scenario" .sh)"
	printf '\n========== scenario: %s ==========\n' "$name"
	if bash "$scenario"; then
		printf '========== %s: PASS ==========\n' "$name"
	else
		printf '========== %s: FAIL ==========\n' "$name" >&2
		fails=$((fails + 1))
	fi
done

printf '\n=== E2E 完了: %d シナリオ中 %d 件失敗 ===\n' "$total" "$fails"
[ "$fails" -eq 0 ]
