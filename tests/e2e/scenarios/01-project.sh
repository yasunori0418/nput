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

# ここから --all の複数 SubjectResult（→ issue #164）。この flake は projectRoot の
# config を 2 つ（docs / idvec）持つので、--all は N=2 の results[] を出す。
e2e_step "gitignore --all --json: config ごとの SubjectResult（cross-config dedup なし・→ issue #164）"
ENV_GI_ALL="$E2E_WORK/gitignore-all.json"
run_json 0 "$ENV_GI_ALL" gitignore --all
assert_json "$ENV_GI_ALL" "status=success・results は選択順（辞書順）の 2 config" \
	'.status == "success" and [.results[].subject.name] == ["docs", "idvec"]'
# 真偽値そのものを評価する（select の結果を length で数えると、値が誤っていても要素数は
# 変わらないため恒真アサートになる）。docs / idvec が「自 config のパスだけ」を持つことと、
# 共有パス /.zshrc が dedup されず idvec 側に残ることを両方固定する。
assert_json "$ENV_GI_ALL" "docs は自 config のパスだけを info.paths に持つ" \
	'first(.results[] | select(.subject.name == "docs")).result.info.paths == ["/.nput-out/docs"]'
assert_json "$ENV_GI_ALL" "idvec も自 config のパスだけを持つ（cross-config dedup なし）" \
	'first(.results[] | select(.subject.name == "idvec")).result.info.paths == ["/.zshrc"]'
assert_json "$ENV_GI_ALL" "items=[] は単一実行と同一・トップ errors[] なし" \
	'([.results[] | select(.result.items == [])] | length) == 2 and (has("errors") | not)'

e2e_step "gitignore --all はフラグ無しで従来の dedup+sort テキスト（ADR-0018 不変・→ issue #164）"
if [ "$(nput gitignore --all)" = "$(printf '/.nput-out/docs\n/.zshrc')" ]; then
	e2e_pass "テキスト集約 / JSON per-config の非対称を保つ"
else
	e2e_fail "テキスト行出力が変化: $(nput gitignore --all)"
fi

e2e_step "apply --all --dryrun --json: 全 config が clean なら status=success（→ issue #164）"
ENV_ALL_CLEAN="$E2E_WORK/apply-all-clean.json"
run_json 0 "$ENV_ALL_CLEAN" apply --all --dryrun
assert_json "$ENV_ALL_CLEAN" "status=success・dryRun=true・2 config" \
	'.status == "success" and .dryRun == true and (.results | length) == 2'
assert_json "$ENV_ALL_CLEAN" "全 subject が success" \
	'[.results[] | select(.status == "success")] | length == 2'

e2e_step "apply --all --dryrun --json: 一部 conflict は item 起因 + 集約 status=error（exit 2・→ issue #164 受け入れ基準）"
echo "foreign" >"$PROJ/.zshrc"
ENV_ALL_CONFLICT="$E2E_WORK/apply-all-conflict.json"
run_json 2 "$ENV_ALL_CONFLICT" apply --all --dryrun
assert_json "$ENV_ALL_CONFLICT" "集約 status=error（conflict は 1 件でも error）" \
	'.status == "error"'
assert_json "$ENV_ALL_CONFLICT" "conflict の無い docs は success のまま残る（部分失敗で成功分を失わない）" \
	'[.results[] | select(.subject.name == "docs")] | .[0].status == "success" and (.[0].result.items | length) > 0'
assert_json "$ENV_ALL_CONFLICT" "idvec は failed item + E_NPUT_COLLISION で subject status=error" \
	'[.results[] | select(.subject.name == "idvec")] | .[0].status == "error" and ([.[0].result.items[] | select(.status == "failed" and .error.code == "E_NPUT_COLLISION")] | length) == 1'
assert_json "$ENV_ALL_CONFLICT" "item 起因は subject errors[] にもトップ errors[] にも重ねない" \
	'([.results[] | select(has("errors"))] | length) == 0 and (has("errors") | not)'
rm "$PROJ/.zshrc"

e2e_step "apply --all --json（非 dryrun）: 実際に配置した 2 config の SubjectResult（→ issue #164）"
ENV_ALL_APPLY="$E2E_WORK/apply-all.json"
run_json 0 "$ENV_ALL_APPLY" apply --all
assert_json "$ENV_ALL_APPLY" "status=success・dryRun=false・2 config が選択順" \
	'.status == "success" and .dryRun == false and [.results[].subject.name] == ["docs", "idvec"]'
assert_json "$ENV_ALL_APPLY" "各 subject が自 config の items を持ち generation を観測している" \
	'(.results | length) == 2 and all(.results[]; (.result.items | length) == 1 and (.generation | has("profile")))'
assert_json "$ENV_ALL_APPLY" "トップ errors[] なし・どの subject にも errors[] なし" \
	'(has("errors") | not) and ([.results[] | select(has("errors"))] | length) == 0'
assert_symlink "$PROJ/.zshrc"
rm "$PROJ/.zshrc"

e2e_step "apply --all --json（非 dryrun・部分失敗）: exit 1 + 成功 config は残る（→ issue #164 受け入れ基準）"
# 第 3 config broken を足し、その target を実ファイルで塞いで conflict させる（非 dryrun の
# conflict は engine が停止して error を返す = exit 1 経路。dryrun の conflict=2 とは別物）。
# --all は残りを続行するので、集約 status:error / exit 1 と「成功 config の SubjectResult は
# 全て残る」を同時に見る。
cp "$PROJ/flake.nix" "$E2E_WORK/flake.nix.bak"
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
      broken = nput.lib.mkManifest {
        pkgs = nixpkgs.legacyPackages.\${system};
        root = nput.lib.projectRoot;
        entries.".nput-out/broken" = { src = ./srcrepo; subpath = "skills/nix"; };
      };
    });
  };
}
EOF
git -c user.email=e2e@nput.test -c user.name=e2e add -A
# broken の target を実ファイルで塞ぐ（foreign entity → conflict で engine が停止する）。
mkdir -p "$PROJ/.nput-out"
echo "foreign" >"$PROJ/.nput-out/broken"
ENV_ALL_PARTIAL="$E2E_WORK/apply-all-partial.json"
run_json 1 "$ENV_ALL_PARTIAL" apply --all
assert_json "$ENV_ALL_PARTIAL" "集約 status=error（1 config でも失敗すれば error）" \
	'.status == "error"'
assert_json "$ENV_ALL_PARTIAL" "選択した 3 config すべてが results に載る（続行して全部処理する）" \
	'[.results[].subject.name] == ["broken", "docs", "idvec"]'
assert_json "$ENV_ALL_PARTIAL" "失敗した broken だけが error・conflict は item 起因で errors[] に重ねない" \
	'first(.results[] | select(.subject.name == "broken")) | .status == "error" and (.errors | not) and ([.result.items[] | select(.status == "failed" and .error.code == "E_NPUT_COLLISION")] | length) == 1'
assert_json "$ENV_ALL_PARTIAL" "成功した docs / idvec は success のまま items を保つ" \
	'all(.results[] | select(.subject.name != "broken"); .status == "success" and (.errors | not) and (.result.items | length) == 1)'
assert_json "$ENV_ALL_PARTIAL" "集約エラーをトップ errors[] へ重ねない（各 subject が既に持つ）" \
	'has("errors") | not'
cp "$E2E_WORK/flake.nix.bak" "$PROJ/flake.nix"
git -c user.email=e2e@nput.test -c user.name=e2e add -A
rm -f "$PROJ/.zshrc" "$PROJ/.nput-out/broken"

e2e_step "apply --all --json: フィルタが 0 件でも results:[] + status=success（→ issue #164 受け入れ基準）"
ENV_ALL_EMPTY="$E2E_WORK/apply-all-empty.json"
run_json 0 "$ENV_ALL_EMPTY" apply --all --home-root
assert_json "$ENV_ALL_EMPTY" "results:[]・status=success（N=0 でも同一形状）" \
	'.results == [] and .status == "success"'

# 世代を公開するコマンドは home mode 限定（→ REQ-05abce3e・issue #284）。拒否は evalRoot で
# rootKind を解決した後に効くため、実 nix を通る経路でしか観測できない。エラーは exit 1 の
# 一般エラーで、stderr に home mode を要する旨を含む（利用者が config の root を疑える）。
BEFORE_TARGET="$(readlink "$TARGET")"
for cmd in rollback list-generations; do
	e2e_step "$cmd は project mode で拒否（exit 1 + home mode を要する旨と rootKind・→ issue #284）"
	reject_err="$E2E_WORK/reject-$cmd.err"
	reject_out="$E2E_WORK/reject-$cmd.out"
	reject_code=0
	nput "$cmd" docs >"$reject_out" 2>"$reject_err" || reject_code=$?
	if [ "$reject_code" -eq 1 ]; then
		e2e_pass "exit 1: nput $cmd docs"
	else
		e2e_fail "exit $reject_code (期待 1): nput $cmd docs"
	fi
	# home mode を要する旨は文言に閉じず語の出現で見る（system mode の追加でモード名が
	# 増える改訂が控える・→ ADR-0036）。一方 rootKind の値表現はその改訂の対象外なので、
	# 判断に使った種別は値ごと固定する（rollback のメッセージは末尾に "project / fixed do
	# not expose generations" という静的な文言を持ち、値に依らず project の語を含むため、
	# 語の出現だけを見ると恒真になる）。
	if grep -q 'home mode' "$reject_err"; then
		e2e_pass "stderr が home mode を要する旨を伝える"
	else
		e2e_fail "stderr に home mode の説明が無い: $(cat "$reject_err")"
	fi
	if grep -qF 'rootKind="project"' "$reject_err"; then
		e2e_pass "stderr が拒否に使った rootKind の値を示す"
	else
		e2e_fail "stderr に rootKind の値が無い: $(cat "$reject_err")"
	fi
done

# 一覧の出力先は stdout（→ ADR-0023）。拒否した以上そこへ 1 バイトも出さない。拒否分岐が
# printGenerations の後段へずれれば一覧が漏れるが、exit code と stderr だけでは検知できない。
# rollback は元々 stdout へ何も書かないので、この観点は list-generations だけが持つ。
e2e_step "拒否した list-generations は stdout へ世代を出さない"
if [ ! -s "$E2E_WORK/reject-list-generations.out" ]; then
	e2e_pass "stdout へ世代を出さない"
else
	e2e_fail "stdout へ出力があった: $(cat "$E2E_WORK/reject-list-generations.out")"
fi

# 拒否した実行のあとも docs の配置が保たれていることの sanity check。project mode は
# generation skip で世代を 1 つしか持たないため、これは「rollback が engine へ到達しない」
# ことまでは示さない（到達しても最古世代として止まる）。到達の検知は世代を複数持つ
# home mode の担当。
e2e_step "拒否のあとも docs の配置は変わらない"
assert_symlink "$TARGET" "$BEFORE_TARGET"

e2e_finish
