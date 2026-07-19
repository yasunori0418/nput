package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yasunori0418/nput/internal/planner"
)

// ensureParentDir creates targetAbs's parent directory (ancestor symlinks are already
// rejected as conflicts by the planner · → ADR-0015), wrapping any failure with the
// target path for diagnosability. Shared by all place-execution entry points that write
// to targetAbs (place, placeCopies, recopyAll).
//
// The directories it creates are NOT journaled for undo (→ ADR-0044 §1 scope note): unwind
// removing the leaf symlink/copy it made room for already returns the target to "absent" from
// the caller's perspective, and any now-empty intermediate directories left behind are
// indistinguishable from the plain mkdir -p residue apply has always left after non-rollback
// runs (e.g. an entry deleted from config the next run down) — a case the existing
// pruneEmptyAncestors backstop, not the undo journal, already exists to sweep up on the next
// removal/apply. Adding mkdir/rmdir entries here would track a directory that may be shared by
// several journal entries (multiple leaves under the same fresh parent), complicating dedup for
// a cosmetic leftover with no data-loss risk.
func ensureParentDir(targetAbs string) error {
	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return fmt.Errorf("nput: cannot create parent directory (%s): %w", filepath.Dir(targetAbs), err)
	}
	return nil
}

// place materializes the planner's Place actions as native symlinks
// (new / re-link before stale removal · → ADR-0006). The plan is already computed by
// planner.Compute from the current FS state, so this stays a thin executor that reflects
// the plan onto the real FS. This slice covers only store / out-of-store symlink placement
// (copy is a future slice · → Issue #6).
// Result op lists are appended only after an action's final FS write succeeds, and a failing
// action records its entry as result.FailedTarget instead, so the lists stay a faithful
// "completed" record for the reached/unreached partition (→ issue #130 到達状態).
func (a *applier) place(actions []planner.PlaceAction) error {
	for _, act := range actions {
		if err := ensureParentDir(act.TargetAbs); err != nil {
			return a.entryFailed(act.Entry.Target, err)
		}

		if act.Kind == planner.PlaceReplace || act.Kind == planner.PlaceForeign {
			// Re-link is unlink + symlink (no rename-based atomic swap · → ADR-0017).
			// The foreign-overwrite warning is already emitted via planner.Warnings by emitWarnings (→ ADR-0015).
			prevDest, err := os.Readlink(act.TargetAbs)
			if err != nil {
				return a.entryFailed(act.Entry.Target, fmt.Errorf("nput: cannot read existing symlink before re-link (%s): %w", act.TargetAbs, err))
			}
			if err := os.Remove(act.TargetAbs); err != nil {
				return a.entryFailed(act.Entry.Target, fmt.Errorf("nput: cannot remove existing symlink (%s): %w", act.TargetAbs, err))
			}
			// Journaled immediately after the unlink, before the re-symlink: if the symlink
			// creation below fails, undoRelinkOld's own os.Remove tolerates the target already
			// being absent and still recreates it at prevDest — so this target is restorable even
			// when this run never got as far as writing the new symlink (→ ADR-0044).
			a.journalRelinkedSymlink(act.TargetAbs, prevDest)
			if err := os.Symlink(act.Dest, act.TargetAbs); err != nil {
				return a.entryFailed(act.Entry.Target, fmt.Errorf("nput: cannot create symlink (%s -> %s): %w", act.TargetAbs, act.Dest, err))
			}
			a.result.Replaced = append(a.result.Replaced, act.Entry.Target)
			a.recordReplacedDest(act.Entry.Target, prevDest)
			continue
		}

		// Only PlaceNew reaches here; assert it so a future PlaceKind cannot silently fall
		// through to a fresh-symlink creation it was never classified for.
		if act.Kind != planner.PlaceNew {
			return a.entryFailed(act.Entry.Target, fmt.Errorf("nput: internal: unhandled place kind %d (target: %s)", act.Kind, act.Entry.Target))
		}
		if err := os.Symlink(act.Dest, act.TargetAbs); err != nil {
			return a.entryFailed(act.Entry.Target, fmt.Errorf("nput: cannot create symlink (%s -> %s): %w", act.TargetAbs, act.Dest, err))
		}
		a.journalPlacedSymlink(act.TargetAbs)
		a.result.Placed = append(a.result.Placed, act.Entry.Target)
	}
	return nil
}
