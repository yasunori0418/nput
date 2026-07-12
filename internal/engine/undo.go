package engine

import (
	"fmt"
	"os"
)

// Package-level undo journal (→ ADR-0044, issue #168): apply's FS-mutating stages — PreRemove,
// place, materializeCopies, removeStale, and the generation-skip drift repair — each push one
// undoOp per successful FS write onto applier.journal. If any stage in the same Apply/Rollback
// call later fails, the journal accumulated so far is unwound in reverse (LIFO) order, restoring
// the pre-apply state before the error is returned. Once `nix-env --set` (commit) succeeds the
// journal is discarded by the caller: FS changes are then part of the new generation, and undoing
// them is rollback's job, not this run's.
//
// Process crashes (SIGKILL, power loss) are out of scope: the journal lives only in memory. That
// gap is intentionally left to the existing backstop — an uncommitted generation plus idempotent
// re-run convergence (→ ADR-0006, ADR-0017) — rather than a persisted WAL (→ ADR-0044 §4).

// undoKind identifies which inverse operation an undoOp performs.
type undoKind int

const (
	// undoUnlinkNew removes a symlink this run newly created at path (PlaceNew / PlaceAction
	// appendAbsentPlacement's absent case).
	undoUnlinkNew undoKind = iota
	// undoRelinkOld removes the symlink now at path and recreates it pointing at prevDest (a
	// re-link: PlaceReplace / PlaceForeign, or a stale-removed entry — → ADR-0006, ADR-0015).
	undoRelinkOld
	// undoRemoveCopy removes a copy tree this run newly placed at path (placeCopies /
	// recopyAll's "target was absent" branch).
	undoRemoveCopy
	// undoRestoreRename renames tmpPath back to path (recopyAll's rename-aside of an existing
	// copy target before overwriting it — → ADR-0044 §1, ADR-0047 D5 boundary).
	undoRestoreRename
	// undoRestoreBackup renames tmpPath (the apply --backup aside) back to path, identically to
	// undoRestoreRename's inverse. Kept as a distinct kind solely so discardJournal does not sweep
	// it up on success: a backup aside is user-owned, kept as-is, not transient overwrite debris
	// (→ ADR-0045, issue #169).
	undoRestoreBackup
	// undoMkdir recreates the empty directory removed at path (PreRemove's RemoveRmdir → ADR-0047).
	undoMkdir
)

// undoOp is one entry in applier.journal: enough information to invert a single FS write that
// already succeeded (→ ADR-0044).
type undoOp struct {
	kind     undoKind
	path     string      // the target path the forward operation wrote to
	prevDest string      // undoRelinkOld: the symlink destination to restore
	tmpPath  string      // undoRestoreRename: the aside path holding the pre-overwrite content
	mode     os.FileMode // undoMkdir: the removed directory's mode, to recreate it identically
}

// journalPlacedSymlink records a freshly created symlink (target was absent) for undo (→ ADR-0044).
func (a *applier) journalPlacedSymlink(path string) {
	a.journal = append(a.journal, undoOp{kind: undoUnlinkNew, path: path})
}

// journalRelinkedSymlink records a symlink this run removed and (for a re-link) replaced,
// capturing the destination it pointed at before this run touched it, so undo can recreate it.
// Covers place's PlaceReplace/PlaceForeign re-link, and removeStale/PreRemove's RemoveUnlink
// (which removes without replacing — the same inverse applies: recreate at prevDest → ADR-0044).
func (a *applier) journalRelinkedSymlink(path, prevDest string) {
	a.journal = append(a.journal, undoOp{kind: undoRelinkOld, path: path, prevDest: prevDest})
}

// journalPlacedCopy records a freshly placed copy tree (target was absent) for undo (→ ADR-0044).
func (a *applier) journalPlacedCopy(path string) {
	a.journal = append(a.journal, undoOp{kind: undoRemoveCopy, path: path})
}

// journalRenamedAside records a --recopy overwrite's rename-aside: path's pre-overwrite content
// was moved to tmpPath before the fresh copy landed at path (→ ADR-0044 §1, ADR-0047 D5).
func (a *applier) journalRenamedAside(path, tmpPath string) {
	a.journal = append(a.journal, undoOp{kind: undoRestoreRename, path: path, tmpPath: tmpPath})
}

// journalBackedUp records an apply --backup rename-aside: path's pre-placement occupant was moved
// to tmpPath (= "<path>.<suffix>") before the fresh placement landed at path (→ ADR-0045, issue
// #169). Unlike journalRenamedAside, the aside is never cleaned up on success — it is the user's
// backup, not overwrite debris (→ discardJournal).
func (a *applier) journalBackedUp(path, tmpPath string) {
	a.journal = append(a.journal, undoOp{kind: undoRestoreBackup, path: path, tmpPath: tmpPath})
}

// journalRemovedEmptyDir records an empty directory this run rmdir-ed (PreRemove's RemoveRmdir →
// ADR-0047), capturing its mode so undo recreates it identically rather than with a fixed default.
func (a *applier) journalRemovedEmptyDir(path string, mode os.FileMode) {
	a.journal = append(a.journal, undoOp{kind: undoMkdir, path: path, mode: mode})
}

// discardJournal drops the accumulated journal once it no longer needs undoing: after a
// successful commit (the FS changes are now part of the new generation — undoing them is
// rollback's job, not this run's · → ADR-0044 §2). A recopy rename-aside entry's tmpPath still
// holds the pre-overwrite content at this point (undo was never triggered), so it is cleaned up
// here rather than left to linger — a warning-only best-effort, since the fresh copy already
// landed successfully and a leftover aside file is cosmetic, not a correctness issue. An
// apply --backup aside (undoRestoreBackup) is deliberately excluded from this sweep: the backup is
// user-owned and stays on disk indefinitely, not cleaned up by nput (reset does not restore it
// either · → ADR-0045, issue #169).
func (a *applier) discardJournal() {
	for _, op := range a.journal {
		if op.kind != undoRestoreRename {
			continue
		}
		if err := os.RemoveAll(op.tmpPath); err != nil && !os.IsNotExist(err) {
			a.opts.Warnf("nput: could not remove recopy aside file (%s): %v", op.tmpPath, err)
		}
	}
	a.journal = nil
}

// unwind reverses applier.journal in LIFO order, restoring the pre-apply state after origErr
// aborted an Apply/Rollback call mid-flight (→ ADR-0044). Each inverse operation is attempted
// independently (best-effort): a failure is recorded but does not stop the rest of the unwind, so
// one unrestorable item never leaves other, restorable items undone (→ ADR-0044 §3). When done,
// it reports origErr plus every item that could not be restored to opts.Warnf, mirroring
// reportConflicts's "list everything, then one aggregate error" shape. The journal is cleared
// regardless of outcome — a partially-unwound run has nothing left it can safely retry the same
// way, and the failure list has already been surfaced.
func (a *applier) unwind(origErr error) {
	journal := a.journal
	a.journal = nil
	if len(journal) == 0 {
		a.opts.Warnf("nput: apply failed; no filesystem changes had been made yet: %v", origErr)
		return
	}

	var failed []string
	for i := len(journal) - 1; i >= 0; i-- {
		if err := undoOne(journal[i]); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", journal[i].path, err))
		}
	}

	a.opts.Warnf("nput: apply failed; rolled back this run's filesystem changes: %v", origErr)
	if len(failed) == 0 {
		return
	}
	a.opts.Warnf("nput: %d item(s) could not be restored during rollback:", len(failed))
	for _, f := range failed {
		a.opts.Warnf("nput:   → %s", f)
	}
}

// undoOne performs the single inverse FS operation described by op.
func undoOne(op undoOp) error {
	switch op.kind {
	case undoUnlinkNew:
		if err := os.Remove(op.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	case undoRelinkOld:
		if err := os.Remove(op.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return os.Symlink(op.prevDest, op.path)
	case undoRemoveCopy:
		return os.RemoveAll(op.path)
	case undoRestoreRename, undoRestoreBackup:
		if err := os.RemoveAll(op.path); err != nil {
			return err
		}
		return os.Rename(op.tmpPath, op.path)
	case undoMkdir:
		mode := op.mode
		if mode == 0 {
			// Defensive fallback for the case the removal site's Lstat failed to capture a mode
			// (see journalRemovedEmptyDir's callers). A genuinely mode-0o000 directory would also
			// hit this branch and come back as 0o755 instead — an accepted, vanishingly rare edge
			// case, not a real invariant violation.
			mode = 0o755
		}
		return os.Mkdir(op.path, mode)
	default:
		return fmt.Errorf("nput: internal error: unknown undo kind %d", op.kind)
	}
}
