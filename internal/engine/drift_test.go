package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/manifest"
)

// 世代スキップ + lstat ドリフト修復の tmpdir 統合テスト（実 FS・nix 不使用・fakeCommit 経路）。
// fakeCommit は profile リンクを link-farm へ直接張るため、同一 link-farm を 2 度 apply すると
// generationUnchanged が成立し（project mode 限定）、新世代を積まず（--set 省略）lstat 修復だけ走る。

// applyOnce は project mode（roothash キー）の 1 回分 apply を回す共通ヘルパ。
func applyOnce(t *testing.T, lf, name, root, state string, commits *[][2]string, warns *[]string) *Result {
	t.Helper()
	opts := Options{
		LinkFarm: lf, Name: name, RootOverride: root, StateDir: state, Commit: fakeCommit(commits),
	}
	if warns != nil {
		opts.Warnf = collectWarnings(warns)
	}
	res, err := Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return res
}

func TestApplyGenerationSkipNoDrift(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")
	// 同一 link-farm を 2 度使う（derivation 同一 = 世代スキップ対象）。
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	var commits [][2]string
	applyOnce(t, lf, "c", root, state, &commits, nil)
	if len(commits) != 1 {
		t.Fatalf("first apply commit calls = %d, want 1", len(commits))
	}

	// 2 回目: link-farm 同一・FS もドリフトなし → 世代スキップ・完全 no-op（commit 増えない）。
	res := applyOnce(t, lf, "c", root, state, &commits, nil)
	if !res.GenerationSkipped {
		t.Errorf("GenerationSkipped = false, want true (same derivation)")
	}
	if len(commits) != 1 {
		t.Errorf("commit calls = %d, want 1 (generation skip omits --set)", len(commits))
	}
	if len(res.Placed)+len(res.Replaced)+len(res.Removed) != 0 {
		t.Errorf("no-drift skip should touch nothing: Placed=%v Replaced=%v Removed=%v", res.Placed, res.Replaced, res.Removed)
	}
	// symlink は記録通りのまま。
	if dest, _ := os.Readlink(filepath.Join(root, ".config", "foo")); dest != src {
		t.Errorf("symlink dest = %q, want %q (untouched)", dest, src)
	}
}

func TestApplyGenerationSkipRepairsDeletedSymlink(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	var commits [][2]string
	applyOnce(t, lf, "c", root, state, &commits, nil)

	// foreign tool が target を消す。
	tgt := filepath.Join(root, ".config", "foo")
	if err := os.Remove(tgt); err != nil {
		t.Fatal(err)
	}

	// 2 回目: 世代スキップだが、消えた entry を再張りする（完全 no-op にしない）。
	res := applyOnce(t, lf, "c", root, state, &commits, nil)
	if !res.GenerationSkipped {
		t.Errorf("GenerationSkipped = false, want true")
	}
	if len(commits) != 1 {
		t.Errorf("commit calls = %d, want 1 (no new generation)", len(commits))
	}
	if len(res.Placed) != 1 || res.Placed[0] != ".config/foo" {
		t.Errorf("Placed = %v, want [.config/foo] (deleted symlink re-placed)", res.Placed)
	}
	if dest, _ := os.Readlink(tgt); dest != src {
		t.Errorf("re-placed dest = %q, want %q", dest, src)
	}
}

func TestApplyGenerationSkipRepairsForeignSymlinkWithWarn(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	var commits [][2]string
	applyOnce(t, lf, "c", root, state, &commits, nil)

	// foreign tool が target を別の先へ書き換える。
	tgt := filepath.Join(root, ".config", "foo")
	foreign := realTempDir(t)
	if err := os.Remove(tgt); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, tgt); err != nil {
		t.Fatal(err)
	}

	// 2 回目: 世代スキップだが foreign 書き換えを warning 付きで再張り（後勝ち）。
	var warns []string
	res := applyOnce(t, lf, "c", root, state, &commits, &warns)
	if !res.GenerationSkipped {
		t.Errorf("GenerationSkipped = false, want true")
	}
	if len(commits) != 1 {
		t.Errorf("commit calls = %d, want 1 (no new generation)", len(commits))
	}
	if len(res.Replaced) != 1 || res.Replaced[0] != ".config/foo" {
		t.Errorf("Replaced = %v, want [.config/foo] (foreign re-placed)", res.Replaced)
	}
	foundForeign := false
	for _, w := range warns {
		if strings.Contains(w, "foreign") {
			foundForeign = true
		}
	}
	if !foundForeign {
		t.Errorf("expected a foreign warning during drift repair, got %v", warns)
	}
	if dest, _ := os.Readlink(tgt); dest != src {
		t.Errorf("re-placed dest = %q, want %q (last-wins)", dest, src)
	}
}

func TestApplyGenerationSkipRepairsDeletedCopy(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeROSrc(t, "file.txt", "store-content")
	lf := writeLinkFarm(t, projectManifest(copyEntry(src, "file.txt", ".config/foo")))

	var commits [][2]string
	applyOnce(t, lf, "c", root, state, &commits, nil)
	tgt := filepath.Join(root, ".config", "foo")
	if _, err := os.Stat(tgt); err != nil {
		t.Fatalf("copy target should exist after first apply: %v", err)
	}

	// foreign tool が copy target を消す → place-once 復帰の対象。
	if err := os.Remove(tgt); err != nil {
		t.Fatal(err)
	}

	res := applyOnce(t, lf, "c", root, state, &commits, nil)
	if !res.GenerationSkipped {
		t.Errorf("GenerationSkipped = false, want true")
	}
	if len(commits) != 1 {
		t.Errorf("commit calls = %d, want 1 (no new generation)", len(commits))
	}
	if len(res.Copied) != 1 || res.Copied[0] != ".config/foo" {
		t.Errorf("Copied = %v, want [.config/foo] (deleted copy re-materialized)", res.Copied)
	}
	if data, _ := os.ReadFile(tgt); string(data) != "store-content" {
		t.Errorf("re-copied content = %q, want store-content", data)
	}
}

func TestApplyGenerationSkipKeepsEditedCopy(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeROSrc(t, "file.txt", "store-content")
	lf := writeLinkFarm(t, projectManifest(copyEntry(src, "file.txt", ".config/foo")))

	var commits [][2]string
	applyOnce(t, lf, "c", root, state, &commits, nil)

	// ユーザーが copy を編集する（内容差）→ 世代スキップの修復では触らない（src 追従は --recopy 限定）。
	tgt := filepath.Join(root, ".config", "foo")
	if err := os.WriteFile(tgt, []byte("user-edit"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := applyOnce(t, lf, "c", root, state, &commits, nil)
	if !res.GenerationSkipped {
		t.Errorf("GenerationSkipped = false, want true")
	}
	if len(res.Copied) != 0 {
		t.Errorf("Copied = %v, want none (edited copy untouched)", res.Copied)
	}
	if data, _ := os.ReadFile(tgt); string(data) != "user-edit" {
		t.Errorf("edited copy content = %q, want user-edit (place-once keeps edits)", data)
	}
}

func TestApplyGenerationSkipOnlyProjectMode(t *testing.T) {
	// home mode は世代スキップしない（毎回新世代）。home mode は同一 link-farm でも commit が増える。
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	// home mode manifest（RootKindHome）。RootOverride で root を tmpdir に逃がす（→ ADR-0017 --root 全モード上書き）。
	hm := manifest.Manifest{
		SchemaVersion: 1,
		Root:          manifest.Root{RootKind: manifest.RootKindHome},
		Entries:       []manifest.Entry{storeEntry(src, ".", ".config/foo")},
	}
	lf := writeLinkFarm(t, hm)

	var commits [][2]string
	apply := func() *Result {
		res, err := Apply(Options{
			LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
		})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		return res
	}

	apply()
	res := apply()
	if res.GenerationSkipped {
		t.Errorf("home mode must not skip generations (GenerationSkipped = true)")
	}
	if len(commits) != 2 {
		t.Errorf("home mode commit calls = %d, want 2 (new generation each apply)", len(commits))
	}
}

func TestApplyGenerationSkipRecopyOverwrites(t *testing.T) {
	// --recopy は世代スキップ時も copy を無条件上書きする（recopy は世代外の opt-in）。
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeROSrc(t, "file.txt", "store-content")
	lf := writeLinkFarm(t, projectManifest(copyEntry(src, "file.txt", ".config/foo")))

	var commits [][2]string
	applyOnce(t, lf, "c", root, state, &commits, nil)
	tgt := filepath.Join(root, ".config", "foo")
	if err := os.WriteFile(tgt, []byte("user-edit"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2 回目: derivation 同一で世代スキップだが --recopy で copy を上書き。世代は積まない。
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Recopy: true, Commit: fakeCommit(&commits),
	})
	if err != nil {
		t.Fatalf("recopy Apply: %v", err)
	}
	if !res.GenerationSkipped {
		t.Errorf("GenerationSkipped = false, want true")
	}
	if len(commits) != 1 {
		t.Errorf("commit calls = %d, want 1 (recopy stays out of generations)", len(commits))
	}
	if data, _ := os.ReadFile(tgt); string(data) != "store-content" {
		t.Errorf("recopy should restore src content, got %q", data)
	}
	if len(res.Recopied) != 1 || res.Recopied[0] != ".config/foo" {
		t.Errorf("Recopied = %v, want [.config/foo]", res.Recopied)
	}
}

func TestApplyNewDerivationCommitsNewGeneration(t *testing.T) {
	// link-farm derivation が変われば（別 src）世代スキップせず新世代を積む。
	root := realTempDir(t)
	state := realTempDir(t)

	src1 := makeSrc(t, "x")
	lf1 := writeLinkFarm(t, projectManifest(storeEntry(src1, ".", ".config/foo")))
	var commits [][2]string
	applyOnce(t, lf1, "c", root, state, &commits, nil)

	// 別 link-farm（別 src = 別 derivation）→ 世代スキップしない。
	src2 := makeSrc(t, "x")
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(src2, ".", ".config/foo")))
	res := applyOnce(t, lf2, "c", root, state, &commits, nil)
	if res.GenerationSkipped {
		t.Errorf("GenerationSkipped = true, want false (derivation changed)")
	}
	if len(commits) != 2 {
		t.Errorf("commit calls = %d, want 2 (new derivation = new generation)", len(commits))
	}
	if dest, _ := os.Readlink(filepath.Join(root, ".config", "foo")); dest != src2 {
		t.Errorf("dest = %q, want %q", dest, src2)
	}
}
