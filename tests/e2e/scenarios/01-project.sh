#!/usr/bin/env bash
# project mode: 一時 git repo で `nput apply` → git toplevel 配下に store symlink 配置されることをアサート。
set -euo pipefail
source "$(dirname "$0")/../lib.sh"
e2e_isolate

PROJ="$E2E_WORK/proj"
mkdir -p "$PROJ/srcrepo/skills/nix"
echo "SKILLBODY" >"$PROJ/srcrepo/skills/nix/SKILL.md"

# idvec は --json 検証専用の第 2 config（apply しない）。target ".zshrc" は niface の
# id-vectors に載る entry key で、エンベロープの item.id を適合ベクタと直接突き合わせる。
cat >"$PROJ/flake.nix" <<EOF
{
$(e2e_flake_inputs)
  outputs = { self, nixpkgs, nput }: {
    nput = nixpkgs.lib.genAttrs $E2E_SYSTEMS (system: {
      docs = nput.lib.mkManifest {
        pkgs = nixpkgs.legacyPackages.\${system};
        root = nput.lib.projectRoot;
        entries.".nput-out/docs" = { src = ./srcrepo; subpath = "skills/nix"; };
      };
      idvec = nput.lib.mkManifest {
        pkgs = nixpkgs.legacyPackages.\${system};
        root = nput.lib.projectRoot;
        entries.".zshrc" = { src = ./srcrepo; subpath = "skills/nix"; };
      };
    });
  };
}
EOF

cd "$PROJ"
git init -q
git -c user.email=e2e@nput.test -c user.name=e2e add -A
git -c user.email=e2e@nput.test -c user.name=e2e commit -qm init

e2e_step "apply --dryrun --json: 初回 plan（add のみ・generation 番号なし・→ issue #132）"
ENV_DRYRUN="$E2E_WORK/dryrun.json"
run_json 0 "$ENV_DRYRUN" apply docs --dryrun
assert_json "$ENV_DRYRUN" "dryRun=true・status=success" \
	'.dryRun == true and .status == "success"'
assert_json "$ENV_DRYRUN" "subject=docs の SubjectResult 1 要素" \
	'(.results | length) == 1 and .results[0].subject.name == "docs"'
assert_json "$ENV_DRYRUN" "items はフルインベントリ（1 entry・success）" \
	'[.results[0].result.items[] | select(.status == "success")] | length == 1'
assert_json "$ENV_DRYRUN" "changes は add のみ・reversible=true" \
	'[.results[0].result.changes[] | select(.kind == "add" and .reversible)] | length == 1'
assert_json "$ENV_DRYRUN" "profile 未作成の plan は generation 番号を両省略" \
	'.results[0].generation | (has("before") or has("after")) | not'

e2e_step "item.id が niface id-vectors と一致（.zshrc・→ issue #132）"
ENV_IDVEC="$E2E_WORK/idvec.json"
run_json 0 "$ENV_IDVEC" apply idvec --dryrun
VEC_EXPECTED="$(jq -r '.vectors[] | select(.identity.kind == "entry" and .identity.key == {target: ".zshrc"}) | .expected' "${NIFACE_ID_VECTORS:?NIFACE_ID_VECTORS が未設定（ci devShell 経由で実行してください）}")"
GOT_ID="$(jq -r '.results[0].result.items[0].id' "$ENV_IDVEC")"
if [ -n "$VEC_EXPECTED" ] && [ "$GOT_ID" = "$VEC_EXPECTED" ]; then
	e2e_pass "item.id が適合ベクタと一致: $GOT_ID"
else
	e2e_fail "item.id 不一致: got=$GOT_ID want=$VEC_EXPECTED"
fi

e2e_step "conflict の --dryrun --json: exit 2 と JSON 出力の両立（→ issue #132 受け入れ基準）"
echo "foreign" >"$PROJ/.zshrc"
ENV_CONFLICT="$E2E_WORK/conflict.json"
run_json 2 "$ENV_CONFLICT" apply idvec --dryrun
assert_json "$ENV_CONFLICT" "status=error・dryRun=true" \
	'.status == "error" and .dryRun == true'
assert_json "$ENV_CONFLICT" "conflict entry は failed item + E_NPUT_COLLISION" \
	'[.results[0].result.items[] | select(.status == "failed" and .error.code == "E_NPUT_COLLISION")] | length == 1'
assert_json "$ENV_CONFLICT" "item 起因エラーは subject errors[] に二重化しない" \
	'.results[0] | has("errors") | not'
rm "$PROJ/.zshrc"

e2e_step "nput apply docs（project mode）"
nput apply docs

e2e_step "git toplevel 配下に配置されたか"
TARGET="$PROJ/.nput-out/docs"
assert_symlink "$TARGET"
assert_file_eq "$TARGET/SKILL.md" "SKILLBODY"
# store symlink であること（/nix/store を指す）。
case "$(readlink "$TARGET")" in
	/nix/store/*) e2e_pass "store symlink を指す" ;;
	*) e2e_fail "store symlink を指すべき: $(readlink "$TARGET")" ;;
esac

e2e_step "再 apply は冪等（配置が壊れない）"
nput apply docs
assert_symlink "$TARGET"
assert_file_eq "$TARGET/SKILL.md" "SKILLBODY"

e2e_step "gitignore --json: result.info.paths（anchor 形・items=[]・→ issue #132）"
ENV_GITIGNORE="$E2E_WORK/gitignore.json"
run_json 0 "$ENV_GITIGNORE" gitignore docs
assert_json "$ENV_GITIGNORE" "info.paths が anchor 形の全 target" \
	'.results[0].result.info.paths == ["/.nput-out/docs"]'
assert_json "$ENV_GITIGNORE" "items=[]・dryRun=false" \
	'.results[0].result.items == [] and .dryRun == false'

e2e_step "gitignore はフラグ無しで従来の行出力（既存消費の互換・→ issue #132 受け入れ基準）"
if [ "$(nput gitignore docs)" = "/.nput-out/docs" ]; then
	e2e_pass "行指向出力は不変"
else
	e2e_fail "行指向出力が変化: $(nput gitignore docs)"
fi

e2e_finish
