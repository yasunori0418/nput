package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/planner"
)

// --- staleremove unit-test helpers (branch prefix: staleErr_) ----------------

// staleErr_applier builds a minimal applier carrying a warning sink, for driving
// removeStale directly without going through the full Apply pipeline.
func staleErr_applier(warns *[]string) *applier {
	return &applier{
		opts:   Options{Warnf: collectWarnings(warns)},
		result: &Result{},
	}
}

// staleErr_action builds a RemoveAction whose recorded dest is `dest` (LinkDest of
// the entry) and whose on-disk target is targetAbs. reverifyStale unlinks only when
// targetAbs is a symlink pointing at `dest`.
func staleErr_action(dest, target, targetAbs string) planner.RemoveAction {
	return planner.RemoveAction{Entry: storeEntry(dest, ".", target), TargetAbs: targetAbs}
}

// --- tests -------------------------------------------------------------------

// TestStaleRemoveDriftKeepsAndWarns covers reverifyStale's post-plan drift re-check:
// a target that drifted away from the conservative invariant between planning and
// unlink (readlink mismatch / non-symlink / missing) is kept with a warning, never
// removed and never erroring (→ ADR-0002, staleremove.go:33-43).
func TestStaleRemoveDriftKeepsAndWarns(t *testing.T) {
	recordedDest := realTempDir(t) // the dest the previous-generation record points at

	cases := []struct {
		name       string
		setup      func(t *testing.T, targetAbs string)
		wantExists bool
	}{
		{
			// recorded symlink drifted to point at a foreign dest → readlink mismatch.
			name: "foreign symlink (readlink mismatch)",
			setup: func(t *testing.T, targetAbs string) {
				if err := os.Symlink(realTempDir(t), targetAbs); err != nil {
					t.Fatal(err)
				}
			},
			wantExists: true,
		},
		{
			// target replaced by a real file → no longer a symlink.
			name: "non-symlink regular file",
			setup: func(t *testing.T, targetAbs string) {
				if err := os.WriteFile(targetAbs, []byte("user"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantExists: true,
		},
		{
			// target replaced by a directory → no longer a symlink.
			name: "non-symlink directory",
			setup: func(t *testing.T, targetAbs string) {
				if err := os.Mkdir(targetAbs, 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantExists: true,
		},
		{
			// target vanished (e.g. concurrent removal) → Lstat fails.
			name:       "missing target",
			setup:      func(t *testing.T, targetAbs string) {},
			wantExists: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := realTempDir(t)
			targetAbs := filepath.Join(dir, "foo")
			tc.setup(t, targetAbs)

			var warns []string
			a := staleErr_applier(&warns)
			act := staleErr_action(recordedDest, "foo", targetAbs)

			if err := a.removeStale([]planner.RemoveAction{act}); err != nil {
				t.Fatalf("removeStale must not error on post-plan drift: %v", err)
			}
			if len(a.result.Removed) != 0 {
				t.Errorf("Removed = %v, want none (drifted target kept)", a.result.Removed)
			}
			if len(warns) != 1 {
				t.Errorf("warnings = %d, want 1 (single drift-keep warning)", len(warns))
			}
			_, err := os.Lstat(targetAbs)
			if tc.wantExists && err != nil {
				t.Errorf("drifted target should be left untouched: %v", err)
			}
			if !tc.wantExists && !os.IsNotExist(err) {
				t.Errorf("missing target should stay absent: lstat err = %v", err)
			}
		})
	}
}

// TestStaleRemoveContinuesAfterDrift verifies a drifted (kept+warned) action does not
// abort the loop: a subsequent action whose invariant still holds is still removed.
func TestStaleRemoveContinuesAfterDrift(t *testing.T) {
	dir := realTempDir(t)
	recordedDest := realTempDir(t)

	// action 1: drifted to a foreign dest → kept + warned.
	driftedAbs := filepath.Join(dir, "drift")
	if err := os.Symlink(realTempDir(t), driftedAbs); err != nil {
		t.Fatal(err)
	}
	// action 2: still a symlink pointing at the recorded dest → invariant holds, removed.
	validAbs := filepath.Join(dir, "valid")
	if err := os.Symlink(recordedDest, validAbs); err != nil {
		t.Fatal(err)
	}

	var warns []string
	a := staleErr_applier(&warns)
	actions := []planner.RemoveAction{
		staleErr_action(recordedDest, "drift", driftedAbs),
		staleErr_action(recordedDest, "valid", validAbs),
	}

	if err := a.removeStale(actions); err != nil {
		t.Fatalf("removeStale: %v", err)
	}
	if _, err := os.Lstat(driftedAbs); err != nil {
		t.Errorf("drifted symlink should be kept: %v", err)
	}
	if _, err := os.Lstat(validAbs); !os.IsNotExist(err) {
		t.Errorf("valid stale symlink should be removed: lstat err = %v", err)
	}
	if len(a.result.Removed) != 1 || a.result.Removed[0] != "valid" {
		t.Errorf("Removed = %v, want [valid] (continued past the kept drift)", a.result.Removed)
	}
	if len(warns) != 1 {
		t.Errorf("warnings = %d, want 1 (only the drifted action warns)", len(warns))
	}
}

// TestPreRemoveDriftErrors covers preRemove's post-plan drift re-check across the same drift
// equivalence classes as TestStaleRemoveDriftKeepsAndWarns (readlink mismatch / non-symlink file /
// non-symlink dir / missing target). Unlike removeStale, a drifted ancestor is not safe to skip
// (children were planned as unconditional new placements assuming the ancestor is gone), so
// preRemove aborts loudly for every class instead of keeping the drifted link (→ staleremove.go,
// ADR-0046). Rollback shares this executor with apply (→ generations.go), so the same error-stop
// semantics apply on the rollback path too.
func TestPreRemoveDriftErrors(t *testing.T) {
	recordedDest := realTempDir(t) // the dest the previous-generation record points at

	cases := []struct {
		name  string
		setup func(t *testing.T, targetAbs string) (wantExists bool, wantReadlink string)
	}{
		{
			name: "foreign symlink (readlink mismatch)",
			setup: func(t *testing.T, targetAbs string) (bool, string) {
				foreign := realTempDir(t)
				if err := os.Symlink(foreign, targetAbs); err != nil {
					t.Fatal(err)
				}
				return true, foreign
			},
		},
		{
			name: "non-symlink regular file",
			setup: func(t *testing.T, targetAbs string) (bool, string) {
				if err := os.WriteFile(targetAbs, []byte("user"), 0o644); err != nil {
					t.Fatal(err)
				}
				return true, ""
			},
		},
		{
			name: "non-symlink directory",
			setup: func(t *testing.T, targetAbs string) (bool, string) {
				if err := os.Mkdir(targetAbs, 0o755); err != nil {
					t.Fatal(err)
				}
				return true, ""
			},
		},
		{
			name: "missing target",
			setup: func(t *testing.T, targetAbs string) (bool, string) {
				return false, ""
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := realTempDir(t)
			targetAbs := filepath.Join(dir, "skills")
			wantExists, wantReadlink := tc.setup(t, targetAbs)

			var warns []string
			a := staleErr_applier(&warns)
			act := staleErr_action(recordedDest, ".claude/skills", targetAbs)

			err := a.preRemove([]planner.RemoveAction{act})
			if err == nil || !strings.Contains(err.Error(), "changed after planning") {
				t.Fatalf("expected ancestor-drift error, got %v", err)
			}
			// Error stop, not skip: no warning, no removal record, target left untouched.
			if len(warns) != 0 {
				t.Errorf("warnings = %v, want none (preRemove errors instead of warning)", warns)
			}
			if len(a.result.Removed) != 0 {
				t.Errorf("Removed = %v, want none", a.result.Removed)
			}
			_, lerr := os.Lstat(targetAbs)
			if wantExists && lerr != nil {
				t.Errorf("drifted target should be left untouched: %v", lerr)
			}
			if !wantExists && !os.IsNotExist(lerr) {
				t.Errorf("missing target should stay absent: lstat err = %v", lerr)
			}
			if wantReadlink != "" {
				got, rerr := os.Readlink(targetAbs)
				if rerr != nil || got != wantReadlink {
					t.Errorf("drifted ancestor after error = %q (err %v), want untouched %q", got, rerr, wantReadlink)
				}
			}
		})
	}
}

// TestStaleRemoveUnlinkError covers the os.Remove failure path: the invariant still
// holds (reverify passes), but the unlink fails, so removeStale returns a wrapped error
// and records no removal. Permission-denied is induced by an unwritable parent dir;
// skipped as root since root bypasses the permission bit (false negative).
func TestStaleRemoveUnlinkError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-denied unlink cannot be induced as root")
	}
	parent := realTempDir(t)
	recordedDest := realTempDir(t)
	targetAbs := filepath.Join(parent, "foo")
	if err := os.Symlink(recordedDest, targetAbs); err != nil { // invariant holds → reverify passes
		t.Fatal(err)
	}
	// Removing a directory entry requires write on its parent; drop it to force EACCES.
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) }) // restore so TempDir cleanup can recurse

	var warns []string
	a := staleErr_applier(&warns)
	act := staleErr_action(recordedDest, "foo", targetAbs)

	err := a.removeStale([]planner.RemoveAction{act})
	if err == nil {
		t.Fatal("expected an unlink error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot remove stale symlink") {
		t.Errorf("error = %v, want it to mention 'cannot remove stale symlink'", err)
	}
	// reverify passed, so this is a hard error rather than a kept+warn; nothing recorded.
	if len(a.result.Removed) != 0 {
		t.Errorf("Removed = %v, want none (unlink failed)", a.result.Removed)
	}
	if len(warns) != 0 {
		t.Errorf("warnings = %v, want none (drift warning only on reverify failure)", warns)
	}
}

// --- empty-ancestor pruning (Issue #174, epic #172 D4) ---------------------

// TestRemoveStalePrunesMultiLevelEmptyAncestors verifies that removing the last entry under
// a multi-level directory chain prunes the whole chain up to (but not including) root, the
// same as HM's `rmdir -p --ignore-fail-on-non-empty`.
func TestRemoveStalePrunesMultiLevelEmptyAncestors(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(src, "x", "a/b/c/file")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// New generation drops the entry entirely → a/b/c/file is stale-removed and the now-empty
	// a/b/c, a/b, a chain should be pruned back to root.
	lf2 := writeLinkFarm(t, projectManifest())
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}

	if len(res.Removed) != 1 || res.Removed[0] != "a/b/c/file" {
		t.Errorf("Removed = %v, want [a/b/c/file]", res.Removed)
	}
	wantPruned := map[string]bool{
		filepath.Join(root, "a", "b", "c"): true,
		filepath.Join(root, "a", "b"):      true,
		filepath.Join(root, "a"):           true,
	}
	if len(res.Pruned) != len(wantPruned) {
		t.Errorf("Pruned = %v, want %d entries covering a/b/c, a/b, a", res.Pruned, len(wantPruned))
	}
	for _, p := range res.Pruned {
		if !wantPruned[p] {
			t.Errorf("unexpected pruned dir: %s", p)
		}
	}
	for dir := range wantPruned {
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			t.Errorf("expected %s to be pruned away, lstat err = %v", dir, err)
		}
	}
	// root itself must survive.
	if _, err := os.Lstat(root); err != nil {
		t.Errorf("root must not be pruned: %v", err)
	}
}

// TestRemoveStaleLeavesNonEmptyAncestorInPlace verifies the conservative half of D4: a
// directory that still holds an unrelated entry's placement is left in place (ENOTEMPTY
// treated as success, not an error), and pruning stops there without touching its own parent.
func TestRemoveStaleLeavesNonEmptyAncestorInPlace(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	lf1 := writeLinkFarm(t, projectManifest(
		storeEntry(src, "x", "a/b/c/file1"),
		storeEntry(src, "x", "a/other/file2"),
	))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// Drop only a/b/c/file1; a/other/file2 (and thus a/) must survive.
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(src, "x", "a/other/file2")))
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}

	wantPruned := map[string]bool{
		filepath.Join(root, "a", "b", "c"): true,
		filepath.Join(root, "a", "b"):      true,
	}
	if len(res.Pruned) != len(wantPruned) {
		t.Errorf("Pruned = %v, want 2 entries covering a/b/c, a/b (a/ kept non-empty)", res.Pruned)
	}
	for _, p := range res.Pruned {
		if !wantPruned[p] {
			t.Errorf("unexpected pruned dir: %s", p)
		}
	}
	// a/ survives because a/other/file2 still lives under it.
	if _, err := os.Lstat(filepath.Join(root, "a")); err != nil {
		t.Errorf("a/ should survive (still holds a/other): %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "a", "other", "file2")); err != nil {
		t.Errorf("a/other/file2 should be untouched: %v", err)
	}
}

// TestRemoveStaleStopsAtRootBoundary verifies that when the removed target is a direct
// child of root, pruning has no ancestor to walk and root itself is never touched.
func TestRemoveStaleStopsAtRootBoundary(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(src, "x", "file")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	lf2 := writeLinkFarm(t, projectManifest())
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}

	if len(res.Pruned) != 0 {
		t.Errorf("Pruned = %v, want none (removed target's parent is root)", res.Pruned)
	}
	if _, err := os.Lstat(root); err != nil {
		t.Errorf("root must survive: %v", err)
	}
}

// TestRemoveStaleStopsAtSymlinkAncestor verifies that pruning stops (without touching it)
// when the walk reaches a directory component that is itself a symlink, rather than
// following/removing it. This guards the walk against ever unlinking something other than
// a plain empty directory it created.
func TestRemoveStaleStopsAtSymlinkAncestor(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(src, "x", "a/b/file")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// Replace a/b (about to become empty) so the *parent* the walk reaches after removing it
	// is a symlink: swap a itself for a symlink pointing elsewhere, simulating a foreign
	// ancestor introduced out-of-band. The walk must stop at "a" without touching it.
	if err := os.RemoveAll(filepath.Join(root, "a", "b")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	elsewhere := realTempDir(t)
	if err := os.Symlink(elsewhere, filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	// Re-create a/b/file directly (bypassing the engine) so removeStale's target lstat/readlink
	// reverify still finds the recorded symlink, and pruning walks up into the now-symlinked "a".
	if err := os.MkdirAll(filepath.Join(elsewhere, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(src, "x")
	if err := os.Symlink(dest, filepath.Join(elsewhere, "b", "file")); err != nil {
		t.Fatal(err)
	}

	lf2 := writeLinkFarm(t, projectManifest())
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}

	// The walk must not touch anything: "a" (the symlink) survives untouched, and critically
	// elsewhere/b (the real directory "a" resolves to) must also survive. Lstat("root/a/b")
	// resolves through the symlink "a" to elsewhere/b, so a walk that naively rmdir'd the
	// resolved path would delete a real directory outside root's tree without ever "touching
	// a" by name — the bug this test guards against (→ Issue #174).
	if len(res.Pruned) != 0 {
		t.Errorf("Pruned = %v, want none (walk must stop before crossing the symlink)", res.Pruned)
	}
	info, err := os.Lstat(filepath.Join(root, "a"))
	if err != nil {
		t.Fatalf("symlink ancestor a must survive: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("a must remain a symlink, got mode %v", info.Mode())
	}
	if _, err := os.Lstat(filepath.Join(elsewhere, "b")); err != nil {
		t.Errorf("elsewhere/b (what the symlink resolves to) must survive untouched: %v", err)
	}
}

// TestRemoveStalePruneFailureIsWarnedNotFatal verifies that a prune failure (the rmdir call
// itself erroring for a reason other than non-empty — here EACCES from a read-only
// grandparent) does not roll back or fail the already-successful target removal: the target
// stays removed, the failure is folded into a warning, and removeStale returns nil (→ Issue
// #174). Permission-denied is induced by an unwritable grandparent dir; skipped as root since
// root bypasses the permission bit (false negative).
func TestRemoveStalePruneFailureIsWarnedNotFatal(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-denied rmdir cannot be induced as root")
	}
	grandparent := realTempDir(t)
	parent := filepath.Join(grandparent, "a")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	recordedDest := realTempDir(t)
	targetAbs := filepath.Join(parent, "foo")
	if err := os.Symlink(recordedDest, targetAbs); err != nil {
		t.Fatal(err)
	}
	// Removing "parent" (once emptied by the target unlink) requires write on grandparent;
	// drop it so the prune's rmdir fails with EACCES while the target unlink itself (which
	// only needs write on "parent") still succeeds.
	if err := os.Chmod(grandparent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(grandparent, 0o755) })

	var warns []string
	a := staleErr_applier(&warns)
	a.root = grandparent
	act := staleErr_action(recordedDest, "a/foo", targetAbs)

	if err := a.removeStale([]planner.RemoveAction{act}); err != nil {
		t.Fatalf("removeStale must not error on a prune failure: %v", err)
	}
	if len(a.result.Removed) != 1 || a.result.Removed[0] != "a/foo" {
		t.Errorf("Removed = %v, want [a/foo] (target removal succeeds despite the prune failure)", a.result.Removed)
	}
	if len(a.result.Pruned) != 0 {
		t.Errorf("Pruned = %v, want none (prune failed)", a.result.Pruned)
	}
	if len(warns) != 1 {
		t.Errorf("warnings = %d, want 1 (prune failure folded into a warning)", len(warns))
	}
	if _, err := os.Lstat(targetAbs); !os.IsNotExist(err) {
		t.Errorf("target should still be removed: lstat err = %v", err)
	}
	if _, err := os.Lstat(parent); err != nil {
		t.Errorf("parent should survive (rmdir failed): %v", err)
	}
}

// TestRemoveStaleLeavesForeignFileNonEmptyAncestorInPlace covers the same conservative
// residency as TestRemoveStaleLeavesNonEmptyAncestorInPlace, but for a directory made
// non-empty by a file nput never placed (a user's own file), not by another entry's
// placement — the boundary case for "never touch what the removal did not empty" (→ Issue
// #174).
func TestRemoveStaleLeavesForeignFileNonEmptyAncestorInPlace(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(src, "x", "a/file")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// A file nput never recorded, dropped into "a/" out-of-band (e.g. by the user).
	foreign := filepath.Join(root, "a", "user-notes.txt")
	if err := os.WriteFile(foreign, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	lf2 := writeLinkFarm(t, projectManifest())
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}

	if len(res.Pruned) != 0 {
		t.Errorf("Pruned = %v, want none (a/ still holds the foreign file)", res.Pruned)
	}
	if _, err := os.Lstat(filepath.Join(root, "a")); err != nil {
		t.Errorf("a/ should survive (still holds the foreign file): %v", err)
	}
	if _, err := os.Lstat(foreign); err != nil {
		t.Errorf("the foreign file itself must be untouched: %v", err)
	}
}

// TestApplyAncestorSelfRecordedMigrationPrunesOuterAncestor verifies PreRemove's ancestor
// unlink also runs the pruning walk (Issue #174): PreRemove runs before Place, so when
// unlinking the ancestor symlink leaves "outer/" momentarily empty, the walk prunes it right
// there — res.Pruned reports it — and the following Place step (ensureParentDir's MkdirAll)
// recreates it fresh to hold the migrated child. The net effect is a real (non-symlink)
// "outer/" directory at the end, exactly as before, so the transient prune+recreate is
// invisible on disk; only res.Pruned reveals it happened.
func TestApplyAncestorSelfRecordedMigrationPrunesOuterAncestor(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	srcOld := realTempDir(t)
	if err := os.WriteFile(filepath.Join(srcOld, "foo"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(srcOld, ".", "outer/skills")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	srcNew := realTempDir(t)
	if err := os.WriteFile(filepath.Join(srcNew, "foo"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(srcNew, "foo", "outer/skills/foo")))
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("second Apply (migration): %v", err)
	}

	if len(res.Removed) != 1 || res.Removed[0] != "outer/skills" {
		t.Errorf("Removed = %v, want [outer/skills] (the pre-removed ancestor)", res.Removed)
	}
	if len(res.Pruned) != 1 || res.Pruned[0] != filepath.Join(root, "outer") {
		t.Errorf("Pruned = %v, want [%s] (outer/ momentarily emptied by the ancestor unlink, then recreated by Place)", res.Pruned, filepath.Join(root, "outer"))
	}

	// outer/skills is now a real directory holding the migrated child; outer/ survives on
	// disk as a real directory (recreated by Place after the transient prune above).
	info, err := os.Lstat(filepath.Join(root, "outer"))
	if err != nil {
		t.Fatalf("outer/ must survive: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("outer/ must be a real directory, not a symlink")
	}
	got, err := os.Readlink(filepath.Join(root, "outer", "skills", "foo"))
	if err != nil || got != filepath.Join(srcNew, "foo") {
		t.Fatalf("outer/skills/foo readlink = %q, err %v; want %q", got, err, filepath.Join(srcNew, "foo"))
	}
}

// TestPreRemovePruneCalledDirectly verifies pruneEmptyAncestors is actually invoked from
// preRemove's unlink loop (Issue #174), independent of the apply-level refill guarantee
// above. It drives preRemove directly against a two-level ancestor chain where nothing
// refills the freed space, so a successful prune is observable in isolation.
func TestPreRemovePruneCalledDirectly(t *testing.T) {
	root := realTempDir(t)
	dest := realTempDir(t)

	// outer/inner is the ancestor symlink to be pre-removed; outer/ holds nothing else, so
	// once inner is unlinked, outer/ itself becomes empty and should be pruned too.
	if err := os.MkdirAll(filepath.Join(root, "outer"), 0o755); err != nil {
		t.Fatal(err)
	}
	ancestorAbs := filepath.Join(root, "outer", "inner")
	if err := os.Symlink(dest, ancestorAbs); err != nil {
		t.Fatal(err)
	}

	a := &applier{opts: Options{Warnf: func(string, ...any) {}}, result: &Result{}, root: root}
	act := planner.RemoveAction{Entry: storeEntry(dest, ".", "outer/inner"), TargetAbs: ancestorAbs}

	if err := a.preRemove([]planner.RemoveAction{act}); err != nil {
		t.Fatalf("preRemove: %v", err)
	}

	if len(a.result.Pruned) != 1 || a.result.Pruned[0] != filepath.Join(root, "outer") {
		t.Errorf("Pruned = %v, want [%s] (outer/ emptied by the ancestor unlink)", a.result.Pruned, filepath.Join(root, "outer"))
	}
	if _, err := os.Lstat(filepath.Join(root, "outer")); !os.IsNotExist(err) {
		t.Errorf("outer/ should have been pruned away, lstat err = %v", err)
	}
}

// TestResetPrunesEmptyAncestors verifies the pruning walk also runs on the reset teardown
// path (Reset reuses removeStale for the symlink half, and this covers the copy-target
// deletion half too), matching apply's behavior (Issue #174).
func TestResetPrunesEmptyAncestors(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	symSrc := makeSrc(t, "sub/file")
	copySrc := makeSrc(t, "data/x")

	applyForReset(t, root, state, homeManifest(
		storeEntry(symSrc, "sub", "a/b/.link"),
		copyEntry(copySrc, "data", "c/d/.copied"),
	))

	res, err := Reset(resetOpts(root, state, nil, false, nil))
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}

	wantPruned := map[string]bool{
		filepath.Join(root, "a", "b"): true,
		filepath.Join(root, "a"):      true,
		filepath.Join(root, "c", "d"): true,
		filepath.Join(root, "c"):      true,
	}
	if len(res.Pruned) != len(wantPruned) {
		t.Errorf("Pruned = %v, want %d entries covering a/b, a, c/d, c", res.Pruned, len(wantPruned))
	}
	for _, p := range res.Pruned {
		if !wantPruned[p] {
			t.Errorf("unexpected pruned dir: %s", p)
		}
	}
	for dir := range wantPruned {
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			t.Errorf("expected %s to be pruned away, lstat err = %v", dir, err)
		}
	}
	if _, err := os.Lstat(root); err != nil {
		t.Errorf("root must survive: %v", err)
	}
}

// TestResetPrunesEmptyAncestorsCopyOnly isolates the copy-target deletion half of Reset's
// pruning (reset.go's copy-removal loop calls pruneEmptyAncestors independently of
// removeStale): with no symlink entries in the mix, a regression that dropped the copy
// loop's prune call would only be caught here, not by the combined symlink+copy case above.
func TestResetPrunesEmptyAncestorsCopyOnly(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	copySrc := makeSrc(t, "data/x")

	applyForReset(t, root, state, homeManifest(
		copyEntry(copySrc, "data", "c/d/.copied"),
	))

	res, err := Reset(resetOpts(root, state, nil, false, nil))
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}

	wantPruned := map[string]bool{
		filepath.Join(root, "c", "d"): true,
		filepath.Join(root, "c"):      true,
	}
	if len(res.Pruned) != len(wantPruned) {
		t.Errorf("Pruned = %v, want %d entries covering c/d, c", res.Pruned, len(wantPruned))
	}
	for _, p := range res.Pruned {
		if !wantPruned[p] {
			t.Errorf("unexpected pruned dir: %s", p)
		}
	}
	for dir := range wantPruned {
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			t.Errorf("expected %s to be pruned away, lstat err = %v", dir, err)
		}
	}
}
