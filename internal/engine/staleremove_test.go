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
