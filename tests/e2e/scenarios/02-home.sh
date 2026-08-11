#!/usr/bin/env bash
# home mode: 仮 $HOME（+ XDG_STATE_HOME）で apply → $HOME 配下配置 + profile 世代コミットを確認し、
# 世代をまたいで（entry 入替）`nput rollback` で前世代の配置へ復帰し、最後に `nput reset` が
# FS のみを撤去して profile / 世代を動かさないことをアサート。世代を戻す実行と撤去する実行は
# `--json` のエンベロープも実コマンド経路で検証する（→ issue #285）。
set -euo pipefail
source "$(dirname "$0")/../lib.sh"
e2e_isolate

PROJ="$E2E_WORK/cfg"
mkdir -p "$PROJ/srcrepo/a" "$PROJ/srcrepo/b"
echo "AAA" >"$PROJ/srcrepo/a/file"
echo "BBB" >"$PROJ/srcrepo/b/file"

# target / subpath を引数に取り fixture flake を書き出す（世代ごとに entry を入れ替える）。
write_flake() {
	local target="$1" sub="$2"
	cat >"$PROJ/flake.nix" <<EOF
{
$(e2e_flake_inputs)
  outputs = { self, nixpkgs, nput }: {
    nput = nixpkgs.lib.genAttrs $E2E_SYSTEMS (system: {
      home = nput.lib.mkManifest {
        pkgs = nixpkgs.legacyPackages.\${system};
        root = nput.lib.homeRoot;
        entries."$target" = { src = ./srcrepo; subpath = "$sub"; };
      };
    });
  };
}
EOF
}

cd "$PROJ"
write_flake ".cfg/a" "a"
git init -q
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm gen1

e2e_step "世代 1: apply（entry a）→ \$HOME 配下に配置"
nput apply home
assert_symlink "$HOME/.cfg/a"
assert_file_eq "$HOME/.cfg/a/file" "AAA"

e2e_step "成功時はデフォルト沈黙 / -v で配置レポート（→ ADR-0031）"
# 同一世代への再 apply（no-op）。nput 自身の配置レポート行は既定で出さず、-v で出す。
# （nix 自体の dirty-tree warning 等は通るため、nput の "完了" マーカー有無で判定する。）
silent_err="$(nput apply home 2>&1 >/dev/null || true)"
if printf '%s' "$silent_err" | grep -q 'nput: apply home done'; then
	e2e_fail "成功時に配置レポートが出てはいけない（既定は沈黙）: '$silent_err'"
else
	e2e_pass "成功時は配置レポート無し（既定沈黙）"
fi
verbose_err="$(nput apply home -v 2>&1 >/dev/null || true)"
if printf '%s' "$verbose_err" | grep -q 'nput: apply home done'; then
	e2e_pass "-v で配置レポートが出る"
else
	e2e_fail "-v で配置レポートが出るべき: '$verbose_err'"
fi

e2e_step "profile 世代がコミットされたか（home mode の profile レイアウト）"
PROFILE="$XDG_STATE_HOME/nix/profiles/nput/home/profile"
assert_symlink "$PROFILE"
GENS="$(nput list-generations home)"
echo "$GENS"
if [ "$(printf '%s\n' "$GENS" | grep -c .)" -ge 1 ]; then
	e2e_pass "list-generations が世代を返す"
else
	e2e_fail "世代が 1 つも無い"
fi

e2e_step "apply --dryrun --json: 既存 profile では generation.before = after（→ issue #132）"
ENV_DRYRUN="$E2E_WORK/dryrun-gen.json"
run_json 0 "$ENV_DRYRUN" apply home --dryrun
assert_json "$ENV_DRYRUN" "generation が before = after の観測を運ぶ" \
	'.results[0].generation | (.before != null) and (.before == .after)'

e2e_step "世代 2: entry を b に入替えて apply（a は stale 除去）"
write_flake ".cfg/b" "b"
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm gen2
nput apply home
assert_symlink "$HOME/.cfg/b"
assert_file_eq "$HOME/.cfg/b/file" "BBB"
assert_absent "$HOME/.cfg/a"

e2e_step "2 世代以上あること"
if [ "$(nput list-generations home | grep -c .)" -ge 2 ]; then
	e2e_pass "世代が 2 つ以上ある"
else
	e2e_fail "世代が 2 つ未満"
fi

e2e_step "nput rollback --json で前世代（entry a）へ復帰（RunE → emit の実経路・→ issue #285）"
ENV_ROLLBACK="$E2E_WORK/rollback.json"
run_json 0 "$ENV_ROLLBACK" rollback home
assert_symlink "$HOME/.cfg/a"
assert_file_eq "$HOME/.cfg/a/file" "AAA"
assert_absent "$HOME/.cfg/b"
assert_json "$ENV_ROLLBACK" "generation が 2 → 1 の遷移を運ぶ" \
	'.results[0].generation | .before == 2 and .after == 1'
assert_json "$ENV_ROLLBACK" "items は復帰先 .cfg/a と撤去元 .cfg/b を全在庫として運ぶ" \
	'[.results[0].result.items[] | .info.target] | sort == [".cfg/a", ".cfg/b"]'
assert_json "$ENV_ROLLBACK" "items は全て success（failed / skipped を含まない）" \
	'.results[0].result.items | all(.status == "success")'
assert_json "$ENV_ROLLBACK" "changes は .cfg/a の add と .cfg/b の remove の 2 件だけ" \
	'.results[0].result as $r
	 | ($r.items | map({key: .id, value: .info.target}) | from_entries) as $t
	 | [$r.changes[] | {kind, target: $t[.itemId]}]
	 | sort_by(.target) == [{kind: "add", target: ".cfg/a"}, {kind: "remove", target: ".cfg/b"}]'
assert_json "$ENV_ROLLBACK" "status=success・dryRun=false・command=rollback" \
	'.status == "success" and .dryRun == false and .command == "rollback"'

e2e_step "list-generations --json: result.info.generations（items=[]・→ issue #132）"
ENV_GENS="$E2E_WORK/list-generations.json"
run_json 0 "$ENV_GENS" list-generations home
assert_json "$ENV_GENS" "info.generations に 2 世代が {number, date, current} で載る" \
	'.results[0].result.info.generations | length == 2 and all(has("number") and has("date") and has("current"))'
assert_json "$ENV_GENS" "current は rollback 先の 1 世代だけ" \
	'[.results[0].result.info.generations[] | select(.current)] | map(.number) == [1]'
assert_json "$ENV_GENS" "items=[]・generation スロット無し・dryRun=false" \
	'.results[0].result.items == [] and (.results[0] | has("generation") | not) and .dryRun == false'

e2e_step "nput reset --json --yes: FS のみを撤去し profile / 世代は動かない（→ issue #285）"
ENV_RESET="$E2E_WORK/reset.json"
run_json 0 "$ENV_RESET" reset home --yes
assert_absent "$HOME/.cfg/a"
assert_json "$ENV_RESET" "items は撤去対象の entry（.cfg/a）で status は success" \
	'.results[0].result.items | map({target: .info.target, status}) == [{target: ".cfg/a", status: "success"}]'
assert_json "$ENV_RESET" "changes は .cfg/a の remove（symlink なので可逆）1 件だけ" \
	'.results[0].result as $r
	 | ($r.items | map({key: .id, value: .info.target}) | from_entries) as $t
	 | [$r.changes[] | {kind, reversible, target: $t[.itemId]}]
	   == [{kind: "remove", reversible: true, target: ".cfg/a"}]'
assert_json "$ENV_RESET" "generation スロットを持たない（FS のみの撤去）" \
	'.results[0] | has("generation") | not'
assert_json "$ENV_RESET" "status=success・dryRun=false・command=reset" \
	'.status == "success" and .dryRun == false and .command == "reset"'
assert_symlink "$PROFILE"
if [ "$(nput list-generations home | grep -c .)" -ge 2 ]; then
	e2e_pass "reset 後も世代は 2 つ以上残る（profile は不変）"
else
	e2e_fail "reset が世代を減らしてはいけない"
fi

e2e_finish
