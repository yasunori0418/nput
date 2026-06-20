#!/usr/bin/env bash
# init + templates: `nput init <t>` でテンプレを展開し、展開後 flake が `nix flake check` を通ることを
# 検証する（→ docs/design.md「テスト戦略」・計画 #16 Q2）。
#
# テンプレ flake.nix の nput.url は無 pin github main を指すため、検証時のみ
# `--override-input nput path:$REPO_ROOT` で局所リポジトリへ差し替える（鶏卵問題の回避）。
# NPUT_TEMPLATE_REF で init の展開元も局所リポジトリへ向ける。配置動作（apply）は他シナリオに委任し、
# ここでは「展開できる + flake.nix が構造的に妥当（flake check 通過）」に限定する。
set -euo pipefail
source "$(dirname "$0")/../lib.sh"
e2e_isolate

# unknown flake output 'nput' warning は exit 0（許容）。それ以外の評価エラーは FAIL。
for t in standalone project; do
	e2e_step "nput init ${t}（NPUT_TEMPLATE_REF=path:\$REPO_ROOT で展開）"
	d="$E2E_WORK/init-${t}"
	mkdir -p "$d"
	(
		cd "$d"
		NPUT_TEMPLATE_REF="path:$REPO_ROOT" nput init "$t"
	)

	assert_exists "$d/flake.nix"
	if [ "$t" = "project" ]; then
		# project テンプレは .gitignore 雛形を同梱する。
		assert_exists "$d/.gitignore"
	fi

	e2e_step "展開した ${t} テンプレが nix flake check を通る（nput を局所リポジトリへ override）"
	if nix flake check "$d" --override-input nput "path:$REPO_ROOT"; then
		e2e_pass "nix flake check 通過: ${t}"
	else
		e2e_fail "nix flake check が失敗: ${t}"
	fi
done

e2e_finish
