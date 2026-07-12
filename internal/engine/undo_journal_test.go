package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/planner"
)

// Tests for apply's end-to-end undo journal (→ ADR-0044, issue #168): a mid-run failure in any
// FS-mutating stage (PreRemove / place / copy materialization / stale removal) must roll back
// every FS change this run made before returning, leaving pre-apply state intact.

// blockWrite creates dir as a mode-0o555 (read+exec, no write) directory, so it exists and is
// lstat-able (the planner's pre-flight classification sees an ordinary, empty directory and
// schedules a plain PlaceNew/CopyAction for anything beneath it — no conflict), but any later
// write inside it at *execution* time (os.Symlink / os.OpenFile for a copy / os.Mkdir for a new
// child) fails with EACCES. This is what lets these tests fail a placement mid-*execution* batch
// rather than during planning: blocking write access (rather than substituting a regular file for
// the directory, which would make even Lstat of a path beneath it fail with ENOTDIR at *plan*
// time) keeps Lstat/ReadDir succeeding, so planner.Compute completes and the failure surfaces only
// once place/materializeCopies actually tries to write (→ ADR-0044). Skips the test outright when
// running as root, since root bypasses directory write-permission checks entirely and this
// technique would not observe any failure at all.
func blockWrite(t *testing.T, dir string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("running as root: directory write-permission checks are bypassed, cannot force a mid-execution failure this way")
	}
	if err := os.Mkdir(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
}

// TestApplyPlaceMidBatchFailureRollsBackEarlierPlacements verifies that when the second of three
// new-symlink placements fails (its target's parent directory denies write access), the first
// placement — already written to the real FS — is undone before Apply returns an error.
func TestApplyPlaceMidBatchFailureRollsBackEarlierPlacements(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src1 := realTempDir(t)
	src3 := realTempDir(t)

	blockWrite(t, filepath.Join(root, "ro"))

	lf := writeLinkFarm(t, projectManifest(
		storeEntry(src1, ".", "a"),
		storeEntry(realTempDir(t), ".", "ro/leaf"),
		storeEntry(src3, ".", "c"),
	))
	_, err := Apply(Options{LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil)})
	if err == nil {
		t.Fatal("expected an error from the blocked mid-batch placement, got nil")
	}

	// "a" was placed before the failure; it must be rolled back.
	if _, lerr := os.Lstat(filepath.Join(root, "a")); !os.IsNotExist(lerr) {
		t.Errorf("entry \"a\" must be rolled back after the mid-batch failure, lstat err = %v", lerr)
	}
	// "c" comes after the failing entry in the batch; it must never have been placed.
	if _, lerr := os.Lstat(filepath.Join(root, "c")); !os.IsNotExist(lerr) {
		t.Errorf("entry \"c\" must never have been placed, lstat err = %v", lerr)
	}
	// The read-only blocker directory itself must survive untouched, and "leaf" must never appear.
	info, serr := os.Lstat(filepath.Join(root, "ro"))
	if serr != nil || !info.IsDir() {
		t.Errorf("blocker at \"ro\" must survive as a directory: info=%v, err=%v", info, serr)
	}
	if _, lerr := os.Lstat(filepath.Join(root, "ro", "leaf")); !os.IsNotExist(lerr) {
		t.Errorf("\"ro/leaf\" must never have been created, lstat err = %v", lerr)
	}

	// No profile should have been committed (the generation commit is never reached).
	if _, serr := os.Stat(filepath.Join(profileDirFor(t, state, root, "c"), "profile")); !os.IsNotExist(serr) {
		t.Errorf("no generation must be committed after a rolled-back apply")
	}
}

// TestApplyPlaceMidBatchFailureRollsBackEarlierRelink verifies that a re-link (PlaceReplace)
// already performed earlier in the same batch is restored to its pre-apply destination when a
// later placement in the same batch fails.
func TestApplyPlaceMidBatchFailureRollsBackEarlierRelink(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	oldDest := realTempDir(t)

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(oldDest, ".", "a")))
	if _, err := Apply(Options{LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil)}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	newDest := realTempDir(t)
	blockWrite(t, filepath.Join(root, "ro"))
	lf2 := writeLinkFarm(t, projectManifest(
		storeEntry(newDest, ".", "a"),              // re-link: "a" already exists from gen 1
		storeEntry(realTempDir(t), ".", "ro/leaf"), // fails: "ro" denies write access
	))
	_, err := Apply(Options{LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil)})
	if err == nil {
		t.Fatal("expected an error from the blocked mid-batch placement, got nil")
	}

	got, rerr := os.Readlink(filepath.Join(root, "a"))
	if rerr != nil || got != oldDest {
		t.Errorf("re-linked entry \"a\" must be restored to its pre-apply dest: readlink=%q, err=%v, want %q", got, rerr, oldDest)
	}
}

// TestApplyCopyMidBatchFailureRollsBackEarlierCopy verifies a copy placement already materialized
// earlier in the same Apply call is removed when a later stage (removeStale) fails. materializeCopies
// runs after place but before removeStale (→ engine.go Apply step 8), so a copy entry always
// finishes placing before removeStale even starts; to observe copy's own journal entry get undone
// by a later-stage failure, this test forces removeStale to fail on a stale symlink whose parent
// directory has since had its write permission revoked (the unlink itself needs write access to
// the containing directory, matching blockWrite's technique applied to an existing parent instead
// of a fresh one).
func TestApplyCopyMidBatchFailureRollsBackEarlierCopy(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	copySrc := makeSrc(t, "tool.conf")
	staleDest := realTempDir(t)

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(staleDest, ".", "ro/stale")))
	if _, err := Apply(Options{LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil)}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root: directory write-permission checks are bypassed, cannot force removeStale to fail this way")
	}
	if err := os.Chmod(filepath.Join(root, "ro"), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "ro"), 0o755) })

	lf2 := writeLinkFarm(t, projectManifest(copyEntry(copySrc, "tool.conf", "copied.conf"))) // "ro/stale" dropped → removeStale target
	_, err := Apply(Options{LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil)})
	if err == nil {
		t.Fatal("expected an error from the permission-denied stale removal, got nil")
	}

	if _, lerr := os.Lstat(filepath.Join(root, "copied.conf")); !os.IsNotExist(lerr) {
		t.Errorf("copy placement must be rolled back after removeStale's failure, lstat err = %v", lerr)
	}
	got, rerr := os.Readlink(filepath.Join(root, "ro", "stale"))
	if rerr != nil || got != staleDest {
		t.Errorf("stale symlink removeStale failed to remove must still be there untouched: readlink=%q, err=%v", got, rerr)
	}
}

// TestApplyPreRemoveMidBatchFailureRollsBackUnlink verifies that when PreRemove's Unlink+Rmdir (a
// real-dir-target migration → ADR-0047) and the resulting new dir-symlink placement both succeed,
// but a later placement in the same Apply call fails, the whole migration is undone — the new dir
// symlink removed, the real directory recreated, and the original per-file leaf symlink restored
// inside it — landing back on the exact pre-apply state, not left half-migrated.
func TestApplyPreRemoveMidBatchFailureRollsBackUnlink(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	srcOld := makeSrc(t, "foo/main.sh")

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(srcOld, "foo/main.sh", ".claude/hooks/foo/main.sh")))
	if _, err := Apply(Options{LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil)}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// A second, unrelated entry in the same batch will fail to place, after PreRemove has already
	// migrated the occupying real directory (unlinked "foo/main.sh", rmdir-ed "foo" then
	// ".claude/hooks" itself) and place has already placed the new ".claude/hooks" dir symlink
	// (Place actions run after PreRemove but before this second entry, since planner.Compute emits
	// Place actions in manifest entry order).
	srcNew := realTempDir(t)
	blockWrite(t, filepath.Join(root, "ro"))
	lf2 := writeLinkFarm(t, projectManifest(
		storeEntry(srcNew, ".", ".claude/hooks"),
		storeEntry(realTempDir(t), ".", "ro/leaf"),
	))
	_, err := Apply(Options{LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil)})
	if err == nil {
		t.Fatal("expected an error from the blocked mid-batch placement, got nil")
	}

	// The whole migration must be undone: ".claude/hooks" is a real directory again (not the new
	// dir symlink place put there), and the original per-file leaf symlink is back inside it.
	info, serr := os.Lstat(filepath.Join(root, ".claude", "hooks"))
	if serr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf(".claude/hooks must be restored as a real directory (the pre-apply state), got mode %v, err %v", info, serr)
	}
	wantDest := filepath.Join(srcOld, "foo", "main.sh")
	got, rerr := os.Readlink(filepath.Join(root, ".claude", "hooks", "foo", "main.sh"))
	if rerr != nil || got != wantDest {
		t.Errorf("restored leaf readlink = %q, err %v; want %q", got, rerr, wantDest)
	}
}

// TestApplyRecopyMidBatchFailureRestoresAsideFile verifies --recopy's rename-aside is rolled back
// (rename back, not left as a stray .nput-recopy-aside file) when a later placement in the same
// batch fails, and that the fresh recopy content is discarded in favor of the pre-apply content.
func TestApplyRecopyMidBatchFailureRestoresAsideFile(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	copySrc := makeSrc(t, "tool.conf")

	lf1 := writeLinkFarm(t, projectManifest(copyEntry(copySrc, "tool.conf", "tool.conf")))
	if _, err := Apply(Options{LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil)}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	// Simulate a local edit so we can tell "restored" apart from "recopied".
	if err := os.WriteFile(filepath.Join(root, "tool.conf"), []byte("locally-edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Update the copy src's content so a successful recopy would be observably different from the
	// restored (locally-edited) content.
	if err := os.WriteFile(filepath.Join(copySrc, "tool.conf"), []byte("updated"), 0o644); err != nil {
		t.Fatal(err)
	}

	blockWrite(t, filepath.Join(root, "ro"))
	lf2 := writeLinkFarm(t, projectManifest(
		copyEntry(copySrc, "tool.conf", "tool.conf"),
		storeEntry(realTempDir(t), ".", "ro/leaf"),
	))
	_, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil), Recopy: true,
	})
	if err == nil {
		t.Fatal("expected an error from the blocked mid-batch placement, got nil")
	}

	data, rerr := os.ReadFile(filepath.Join(root, "tool.conf"))
	if rerr != nil || string(data) != "locally-edited" {
		t.Errorf("recopy target must be restored to its pre-apply (locally-edited) content: data=%q, err=%v", data, rerr)
	}
	if _, lerr := os.Lstat(filepath.Join(root, "tool.conf.nput-recopy-aside")); !os.IsNotExist(lerr) {
		t.Errorf("aside file must not be left behind after a rolled-back recopy, lstat err = %v", lerr)
	}
}

// TestPreRemoveJournalRmdirThenUnlinkOrder directly exercises preRemove with a batch containing
// both a RemoveUnlink and a RemoveRmdir (the shape classifyDirMigration produces: children before
// parents), verifying the journal records them such that unwind restores the parent directory
// before recreating the child symlink inside it — LIFO of a bottom-up forward batch naturally
// yields a top-down (parent-first) undo (→ ADR-0044 §5, ADR-0047).
func TestPreRemoveJournalRmdirThenUnlinkOrder(t *testing.T) {
	root := realTempDir(t)
	leafDest := realTempDir(t)
	dir := filepath.Join(root, "parent")
	leaf := filepath.Join(dir, "leaf")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(leafDest, leaf); err != nil {
		t.Fatal(err)
	}

	var warns []string
	a := &applier{opts: Options{Warnf: collectWarnings(&warns)}, result: &Result{}}
	a.root = root
	actions := []planner.RemoveAction{
		{Kind: planner.RemoveUnlink, Entry: storeEntry(leafDest, ".", "parent/leaf"), TargetAbs: leaf},
		{Kind: planner.RemoveRmdir, TargetAbs: dir},
	}
	if err := a.preRemove(actions); err != nil {
		t.Fatalf("preRemove: %v", err)
	}
	if _, lerr := os.Lstat(dir); !os.IsNotExist(lerr) {
		t.Fatalf("setup: dir must be gone after preRemove, lstat err = %v", lerr)
	}

	a.unwind(os.ErrInvalid)

	info, serr := os.Lstat(dir)
	if serr != nil || !info.IsDir() {
		t.Fatalf("parent dir must be recreated: info=%v, err=%v", info, serr)
	}
	got, rerr := os.Readlink(leaf)
	if rerr != nil || got != leafDest {
		t.Fatalf("leaf symlink must be recreated inside the recreated parent: readlink=%q, err=%v, want %q", got, rerr, leafDest)
	}
}

// TestApplyCommitFailureDoesNotUnwind verifies ADR-0044 §2's asymmetry: a commit (`nix-env --set`)
// failure is NOT rolled back, unlike every FS-mutating stage before it. Every FS write for this run
// already succeeded by the time commit runs, so there is nothing wrong to undo — the run simply
// fails to advance the generation, and idempotent re-apply converges (→ ADR-0006, ADR-0017).
func TestApplyCommitFailureDoesNotUnwind(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := realTempDir(t)

	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", "a")))
	failingCommit := func(string, string) error { return os.ErrPermission }

	var warns []string
	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: failingCommit,
		Warnf: collectFormatted(&warns),
	})
	if err == nil {
		t.Fatal("expected the commit failure to propagate, got nil")
	}

	// The placement must survive untouched — commit failure is not an unwind trigger.
	got, rerr := os.Readlink(filepath.Join(root, "a"))
	if rerr != nil || got != src {
		t.Errorf("placement must survive a commit failure untouched: readlink=%q, err=%v, want %q", got, rerr, src)
	}
	for _, w := range warns {
		if strings.Contains(w, "rolled back this run's filesystem changes") {
			t.Errorf("warns = %v, must not report a rollback for a commit-only failure", warns)
		}
	}
}

// TestApplyCommitFailureLeavesRecopyAsideFile verifies the same asymmetry for --recopy: a commit
// failure after a successful rename-aside overwrite must leave the aside file in place (neither
// cleaned up by discardJournal, which only runs after a successful commit, nor renamed back by
// unwind, which commit failure does not trigger) — the fresh copy stays live and recoverable from
// its pre-apply content is still sitting in the aside file for manual inspection if needed.
func TestApplyCommitFailureLeavesRecopyAsideFile(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	copySrc := makeSrc(t, "tool.conf")

	lf1 := writeLinkFarm(t, projectManifest(copyEntry(copySrc, "tool.conf", "tool.conf")))
	if _, err := Apply(Options{LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil)}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// A distinct link-farm (home mode has no generation-skip concept in the first place, but a
	// second, content-identical project-mode link-farm would still hit generationUnchanged and skip
	// straight to repairDrift, bypassing commit entirely) — home mode side-steps that branch so
	// this Apply call actually reaches step 9 (commit) and can be made to fail there.
	lf2 := writeLinkFarm(t, homeManifest(copyEntry(copySrc, "tool.conf", "tool.conf")))
	failingCommit := func(string, string) error { return os.ErrPermission }
	_, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: failingCommit, Recopy: true,
	})
	if err == nil {
		t.Fatal("expected the commit failure to propagate, got nil")
	}

	data, rerr := os.ReadFile(filepath.Join(root, "tool.conf"))
	if rerr != nil || string(data) != "content" {
		t.Errorf("the fresh recopy must survive a commit failure: data=%q, err=%v", data, rerr)
	}
	if _, lerr := os.Lstat(filepath.Join(root, "tool.conf.nput-recopy-aside")); lerr != nil {
		t.Errorf("the aside file must survive (neither cleaned up nor renamed back) after a commit failure, lstat err = %v", lerr)
	}
}
