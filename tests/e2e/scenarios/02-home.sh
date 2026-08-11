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
# 第 2 config の proj は projectRoot（apply するのは `--all` の除外検証に入る直前だけ）。
# home mode の home と同居させることで、`list-generations --all` の除外が「元から空」ではなく
# 「除外が効いた」ものであることを観測できるようにする（→ issue #312）。
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
      proj = nput.lib.mkManifest {
        pkgs = nixpkgs.legacyPackages.\${system};
        root = nput.lib.projectRoot;
        entries.".nput-out/proj" = { src = ./srcrepo; subpath = "$sub"; };
      };
    });
  };
}
EOF
}

# 世代の本数を数える。nput 自体の失敗は非ゼロで返し（末尾の `|| true` が関数の終了コードを
# 決めてしまうため、取得の失敗はその場で return する）、呼び出し側のコマンド置換代入を set -e で
# 落とす。0 件で非ゼロ終了する grep だけを守り、0 件は「本数が壊れた」観測としてアサートへ渡す。
gens_count() {
	local out
	out="$(nput list-generations home)" || return 1
	printf '%s\n' "$out" | grep -c . || true
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
# 既定契約（--json 無し）の rollback はここでは通さない。前世代へ戻す実行はこのシナリオでは
# 一度しか成立せず（世代 1 からは戻せない）、旧版も出力は何もアサートしていなかったため、
# 一度きりの実行はエンベロープ検証のある --json 側へ寄せる。
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
assert_json "$ENV_ROLLBACK" "changes は .cfg/a の add と .cfg/b の remove（どちらも可逆）2 件だけ" \
	'.results[0].result as $r
	 | ($r.items | map({key: .id, value: .info.target}) | from_entries) as $t
	 | [$r.changes[] | {kind, reversible, target: $t[.itemId]}]
	 | sort_by(.target) == [{kind: "add", reversible: true, target: ".cfg/a"},
	                        {kind: "remove", reversible: true, target: ".cfg/b"}]'
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

# ここから `list-generations --all`（→ REQ-05abce3e 第 3 文・issue #312）。除外の検証は home mode の
# config が実在する配置でしか意味を持たない（project mode だけの環境では「除外が効いた」と
# 「元から空」を区別できない）ため、home の profile が既に 2 世代を持つこの位置に置く。
e2e_step "project mode の proj を apply（--all の除外検証に実体を与える）"
# proj の profileDir は <state>/nix/profiles/nput/<roothash>/proj で、<roothash> の直下には
# profile リンクが無い。--all はこの構造差だけで除外するので、除外対象を disk 上に実在させる。
nput apply proj
assert_symlink "$PROJ/.nput-out/proj"
PROJ_HASHDIR="$(find "$XDG_STATE_HOME/nix/profiles/nput" -mindepth 2 -maxdepth 2 -type d -name proj -print -quit)"
if [ -n "$PROJ_HASHDIR" ] && [ -L "$PROJ_HASHDIR/profile" ]; then
	e2e_pass "proj の profile が roothash 階層に実在する: ${PROJ_HASHDIR#"$XDG_STATE_HOME/"}"
else
	e2e_fail "proj の profile が roothash 階層に見つからない（除外検証の前提が崩れる）"
fi

e2e_step "list-generations --all --json: home mode の config だけを列挙する（→ issue #312）"
ENV_GENS_ALL="$E2E_WORK/list-generations-all.json"
run_json 0 "$ENV_GENS_ALL" list-generations --all
# 陽性対照と除外を 1 つの等式で固定する。subject 名の集合そのものを見るので、home が落ちても
# proj が混ざっても落ちる（select + length では値が誤っていても件数が変わらず恒真になりうる）。
assert_json "$ENV_GENS_ALL" "results は home mode の home だけ（proj は roothash 階層なので除外）" \
	'[.results[].subject.name] == ["home"]'
assert_json "$ENV_GENS_ALL" "home の info.generations は名指し列挙と同じ 2 世代（current は世代 1）" \
	'first(.results[] | select(.subject.name == "home")).result.info as $i
	 | ($i.generations | length) == 2
	 and ([$i.generations[] | select(.current) | .number] == [1])'
assert_json "$ENV_GENS_ALL" "items=[]・generation スロット無し・status=success・dryRun=false" \
	'.results[0].result.items == [] and (.results[0] | has("generation") | not)
	 and .status == "success" and .dryRun == false'

e2e_step "list-generations --all は読み取り専用（profile も世代も配置も動かない）"
# 「読み取り専用」を profile リンク先・世代の本数・配置の 3 点で見る。--json 経路は既に通した
# ので、ここは既定（テキスト）経路で実行して両方の出力経路が状態を変えないことを固定する。
ALL_PROFILE_BEFORE="$(readlink "$PROFILE")"
ALL_GENS_BEFORE="$(gens_count)"
nput list-generations --all >/dev/null
assert_symlink "$PROFILE" "$ALL_PROFILE_BEFORE"
ALL_GENS_AFTER="$(gens_count)"
if [ "$ALL_GENS_AFTER" -eq "$ALL_GENS_BEFORE" ]; then
	e2e_pass "--all のあとも世代の本数が変わらない（$ALL_GENS_BEFORE 件）"
else
	e2e_fail "--all が世代の本数を動かしてはいけない: $ALL_GENS_BEFORE → $ALL_GENS_AFTER"
fi
assert_symlink "$HOME/.cfg/a"
assert_file_eq "$HOME/.cfg/a/file" "AAA"

e2e_step "nput reset --json --yes: FS のみを撤去し profile / 世代は動かない（→ issue #285）"
# 撤去対象は config（この時点の flake は .cfg/b を宣言している）ではなく、profile が指す世代の
# manifest 由来（記録された真実）。rollback で世代 1 に戻った後なので .cfg/a が在庫になる。
PROFILE_BEFORE="$(readlink "$PROFILE")"
GENS_BEFORE="$(gens_count)"
# 前後一致の基準そのものが壊れていないこと（直前の list-generations --json の length == 2 と同じ数を
# 非 JSON 経路でも観測できること）を先に固定する。
if [ "$GENS_BEFORE" -eq 2 ]; then
	e2e_pass "reset 前の世代は 2 件"
else
	e2e_fail "reset 前の世代が 2 件でない: $GENS_BEFORE"
fi
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
# profile の非遷移: リンク先（現行世代）と世代の本数がどちらも reset 前後で変わらない。
assert_symlink "$PROFILE" "$PROFILE_BEFORE"
GENS_AFTER="$(gens_count)"
if [ "$GENS_AFTER" -eq "$GENS_BEFORE" ]; then
	e2e_pass "reset 後も世代の本数が変わらない（$GENS_BEFORE 件）"
else
	e2e_fail "reset が世代の本数を動かしてはいけない: $GENS_BEFORE → $GENS_AFTER"
fi

e2e_finish
