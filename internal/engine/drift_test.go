package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/manifest"
)

// tmpdir integration tests for generation skip + lstat drift repair (real FS · no nix · fakeCommit path).
// fakeCommit links the profile link directly to the link-farm, so applying the same link-farm twice
// makes generationUnchanged hold (project mode only), committing no new generation (--set omitted) and
// running only the lstat repair.

// applyOnce is a shared helper that runs a single apply in project mode (roothash key).
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
	// Use the same link-farm twice (same derivation = generation-skip candidate).
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	var commits [][2]string
	applyOnce(t, lf, "c", root, state, &commits, nil)
	if len(commits) != 1 {
		t.Fatalf("first apply commit calls = %d, want 1", len(commits))
	}

	// Second apply: same link-farm, no FS drift → generation skip, full no-op (commit count unchanged).
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
	// symlink stays as recorded.
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

	// foreign tool removes the target.
	tgt := filepath.Join(root, ".config", "foo")
	if err := os.Remove(tgt); err != nil {
		t.Fatal(err)
	}

	// Second apply: generation skip, but re-link the deleted entry (not a full no-op).
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

	// foreign tool rewrites the target to point elsewhere.
	tgt := filepath.Join(root, ".config", "foo")
	foreign := realTempDir(t)
	if err := os.Remove(tgt); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, tgt); err != nil {
		t.Fatal(err)
	}

	// Second apply: generation skip, but re-link the foreign rewrite with a warning (last-wins).
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

	// foreign tool removes the copy target → subject to place-once revert.
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

	// User edits the copy (content diff) → generation-skip repair leaves it untouched (src follow is --recopy only).
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
	// home mode does not skip generations (a new generation every time). In home mode the commit count grows even with the same link-farm.
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	// home mode manifest (RootKindHome). RootOverride redirects root to a tmpdir (→ ADR-0017 --root overrides all modes).
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
	// --recopy overwrites copies unconditionally even during a generation skip (recopy is an opt-in outside generations).
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

	// Second apply: same derivation so generation skip, but --recopy overwrites the copy. No new generation.
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
	// If the link-farm derivation changes (different src), no generation skip and a new generation is committed.
	root := realTempDir(t)
	state := realTempDir(t)

	src1 := makeSrc(t, "x")
	lf1 := writeLinkFarm(t, projectManifest(storeEntry(src1, ".", ".config/foo")))
	var commits [][2]string
	applyOnce(t, lf1, "c", root, state, &commits, nil)

	// Different link-farm (different src = different derivation) → no generation skip.
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
