package engine

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- undo unit tests (→ ADR-0044, issue #168) --------------------------------

// TestUndoOneUnlinksNewSymlink verifies undoUnlinkNew removes a freshly placed symlink.
func TestUndoOneUnlinksNewSymlink(t *testing.T) {
	dir := realTempDir(t)
	target := filepath.Join(dir, "link")
	if err := os.Symlink(dir, target); err != nil {
		t.Fatal(err)
	}
	if err := undoOne(undoOp{kind: undoUnlinkNew, path: target}); err != nil {
		t.Fatalf("undoOne: %v", err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("target must be gone after undo, lstat err = %v", err)
	}
}

// TestUndoOneUnlinkNewToleratesAlreadyGone verifies undoUnlinkNew does not error when the
// forward operation's own place already vanished (e.g. removed by an earlier, unrelated FS
// change) — undo should not fail just because there is nothing left to remove.
func TestUndoOneUnlinkNewToleratesAlreadyGone(t *testing.T) {
	dir := realTempDir(t)
	target := filepath.Join(dir, "already-gone")
	if err := undoOne(undoOp{kind: undoUnlinkNew, path: target}); err != nil {
		t.Fatalf("undoOne on an absent path must not error, got: %v", err)
	}
}

// TestUndoOneRestoresRelink verifies undoRelinkOld removes the current symlink and recreates
// it pointing at prevDest — the inverse of both a re-link (place's PlaceReplace/PlaceForeign)
// and a plain removal (removeStale / PreRemove's RemoveUnlink).
func TestUndoOneRestoresRelink(t *testing.T) {
	dir := realTempDir(t)
	target := filepath.Join(dir, "link")
	oldDest := realTempDir(t)
	newDest := realTempDir(t)
	if err := os.Symlink(newDest, target); err != nil {
		t.Fatal(err)
	}
	if err := undoOne(undoOp{kind: undoRelinkOld, path: target, prevDest: oldDest}); err != nil {
		t.Fatalf("undoOne: %v", err)
	}
	got, err := os.Readlink(target)
	if err != nil || got != oldDest {
		t.Fatalf("readlink = %q, err %v; want %q", got, err, oldDest)
	}
}

// TestUndoOneRestoresRelinkWhenTargetAbsent verifies undoRelinkOld recreates the symlink even
// when the current path is already gone (removeStale/PreRemove's RemoveUnlink case: nothing to
// unlink first, just recreate at prevDest).
func TestUndoOneRestoresRelinkWhenTargetAbsent(t *testing.T) {
	dir := realTempDir(t)
	target := filepath.Join(dir, "removed")
	oldDest := realTempDir(t)
	if err := undoOne(undoOp{kind: undoRelinkOld, path: target, prevDest: oldDest}); err != nil {
		t.Fatalf("undoOne: %v", err)
	}
	got, err := os.Readlink(target)
	if err != nil || got != oldDest {
		t.Fatalf("readlink = %q, err %v; want %q", got, err, oldDest)
	}
}

// TestUndoOneRemovesCopy verifies undoRemoveCopy removes a freshly placed copy tree.
func TestUndoOneRemovesCopy(t *testing.T) {
	dir := realTempDir(t)
	target := filepath.Join(dir, "copied")
	if err := os.MkdirAll(filepath.Join(target, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "nested", "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := undoOne(undoOp{kind: undoRemoveCopy, path: target}); err != nil {
		t.Fatalf("undoOne: %v", err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("copy target must be gone after undo, lstat err = %v", err)
	}
}

// TestUndoOneRestoresRename verifies undoRestoreRename discards the freshly recopied content
// and renames the aside file back to its original path (→ ADR-0044 §1, --recopy rename-aside).
func TestUndoOneRestoresRename(t *testing.T) {
	dir := realTempDir(t)
	target := filepath.Join(dir, "tool.conf")
	aside := target + ".nput-recopy-aside"
	if err := os.WriteFile(aside, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("freshly-recopied"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := undoOne(undoOp{kind: undoRestoreRename, path: target, tmpPath: aside}); err != nil {
		t.Fatalf("undoOne: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "original" {
		t.Fatalf("target content = %q, err %v; want %q", data, err, "original")
	}
	if _, err := os.Lstat(aside); !os.IsNotExist(err) {
		t.Fatalf("aside file must be consumed by the rename, lstat err = %v", err)
	}
}

// TestUndoOneRecreatesEmptyDir verifies undoMkdir recreates a directory PreRemove's RemoveRmdir removed.
func TestUndoOneRecreatesEmptyDir(t *testing.T) {
	dir := realTempDir(t)
	target := filepath.Join(dir, "emptied")
	if err := undoOne(undoOp{kind: undoMkdir, path: target}); err != nil {
		t.Fatalf("undoOne: %v", err)
	}
	info, err := os.Lstat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("target must be a recreated directory, lstat = %v, err %v", info, err)
	}
}

// TestUnwindReversesJournalInLIFOOrder verifies unwind restores multiple journal entries in
// last-in-first-out order — the order that correctly reverses a batch like "PreRemove unlinked a
// child then rmdir-ed its now-empty parent": undo must mkdir the parent back before it can
// recreate the child symlink inside it.
func TestUnwindReversesJournalInLIFOOrder(t *testing.T) {
	dir := realTempDir(t)
	parent := filepath.Join(dir, "parent")
	child := filepath.Join(parent, "child")
	childDest := realTempDir(t)

	// Simulate the forward operations PreRemove would have performed: unlink the child symlink,
	// then rmdir the now-empty parent.
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(childDest, child); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(child); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(parent); err != nil {
		t.Fatal(err)
	}

	var warns []string
	a := &applier{opts: Options{Warnf: collectWarnings(&warns)}, result: &Result{}}
	a.journalRelinkedSymlink(child, childDest) // pushed first (child unlinked first)
	a.journalRemovedEmptyDir(parent, 0o755)    // pushed second (parent rmdir-ed second)

	a.unwind(errors.New("simulated failure"))

	if _, err := os.Lstat(parent); err != nil {
		t.Fatalf("parent must be recreated: %v", err)
	}
	got, err := os.Readlink(child)
	if err != nil || got != childDest {
		t.Fatalf("child readlink = %q, err %v; want %q (parent must exist before child recreation)", got, err, childDest)
	}
	if len(a.journal) != 0 {
		t.Errorf("journal must be cleared after unwind, got %d entries", len(a.journal))
	}
}

// TestUnwindBestEffortContinuesPastFailureAndReportsAll verifies unwind's best-effort contract
// (→ ADR-0044 §3): when one journal entry cannot be undone (its path was removed out from under
// it by something else), the rest of the journal is still unwound, and both the original error
// and the specific unrestorable item are reported to Warnf.
func TestUnwindBestEffortContinuesPastFailureAndReportsAll(t *testing.T) {
	dir := realTempDir(t)
	restorable := filepath.Join(dir, "restorable")
	restorableDest := realTempDir(t)
	unrestorable := filepath.Join(dir, "sub", "unrestorable") // parent "sub" does not exist → mkdir/symlink fails

	var warns []string
	a := &applier{opts: Options{Warnf: collectFormatted(&warns)}, result: &Result{}}
	// Pushed in this order; unwind runs LIFO, so "unrestorable" is attempted first, then "restorable".
	a.journalRelinkedSymlink(restorable, restorableDest)
	a.journalRelinkedSymlink(unrestorable, realTempDir(t))

	origErr := errors.New("simulated mid-apply failure")
	a.unwind(origErr)

	got, err := os.Readlink(restorable)
	if err != nil || got != restorableDest {
		t.Errorf("restorable entry must still be undone despite the other failing: readlink=%q, err=%v", got, err)
	}

	var sawOrigErr, sawUnrestorablePath bool
	for _, w := range warns {
		if strings.Contains(w, origErr.Error()) {
			sawOrigErr = true
		}
		if strings.Contains(w, unrestorable) {
			sawUnrestorablePath = true
		}
	}
	if !sawOrigErr {
		t.Errorf("warnings = %v, want the original error reported", warns)
	}
	if !sawUnrestorablePath {
		t.Errorf("warnings = %v, want the unrestorable path named", warns)
	}
	if len(a.journal) != 0 {
		t.Errorf("journal must be cleared even after a partial unwind failure, got %d entries", len(a.journal))
	}
}

// TestUnwindNoJournalReportsOrigErrOnly verifies unwind on an empty journal (a failure before any
// FS write happened yet) just reports the original error without a spurious "N item(s)" line.
func TestUnwindNoJournalReportsOrigErrOnly(t *testing.T) {
	var warns []string
	a := &applier{opts: Options{Warnf: collectFormatted(&warns)}, result: &Result{}}
	origErr := errors.New("failed before any FS write")
	a.unwind(origErr)

	if len(warns) != 1 {
		t.Fatalf("warns = %v, want exactly one line", warns)
	}
	if !strings.Contains(warns[0], origErr.Error()) {
		t.Errorf("warns[0] = %q, want it to mention the original error", warns[0])
	}
}

// TestDiscardJournalRemovesRecopyAsideFiles verifies discardJournal — called after a successful
// commit — cleans up any --recopy rename-aside files left behind, since undo was never triggered
// and the fresh copies already landed successfully (→ ADR-0044).
func TestDiscardJournalRemovesRecopyAsideFiles(t *testing.T) {
	dir := realTempDir(t)
	target := filepath.Join(dir, "tool.conf")
	aside := target + ".nput-recopy-aside"
	if err := os.WriteFile(aside, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	var warns []string
	a := &applier{opts: Options{Warnf: collectWarnings(&warns)}, result: &Result{}}
	a.journalRenamedAside(target, aside)

	a.discardJournal()

	if _, err := os.Lstat(aside); !os.IsNotExist(err) {
		t.Errorf("aside file must be removed by discardJournal, lstat err = %v", err)
	}
	if len(a.journal) != 0 {
		t.Errorf("journal must be cleared, got %d entries", len(a.journal))
	}
}
