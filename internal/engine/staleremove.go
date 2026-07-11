package engine

import (
	"fmt"
	"os"

	"github.com/yasunori0418/nput/internal/planner"
)

// removeStale applies the planner's Remove actions, re-verifying the conservative
// invariant against the real FS immediately before each unlink. The plan already
// restricts removals to "symlinks recorded in the previous generation that still
// point at the recorded dest", but placement (and any concurrent change) runs
// between planning and removal, so the stale-remover re-checks lstat/readlink and
// unlinks only when the invariant still holds; drifted targets are kept with a
// warning (→ ADR-0002, ADR-0006, docs/spec.md "stale removal targets and safety invariant").
func (a *applier) removeStale(actions []planner.RemoveAction) error {
	for _, act := range actions {
		if !reverifyStale(act) {
			a.opts.Warnf("nput: keeping stale symlink because it drifted after planning: %s", act.Entry.Target)
			continue
		}
		if err := os.Remove(act.TargetAbs); err != nil {
			return fmt.Errorf("nput: cannot remove stale symlink (%s): %w", act.TargetAbs, err)
		}
		a.result.Removed = append(a.result.Removed, act.Entry.Target)
	}
	return nil
}

// preRemove unlinks the self-recorded stale ancestor symlinks the planner scheduled for
// migration, re-verifying the conservative invariant against the real FS right before each
// unlink (the same check removeStale uses). It runs *before* place so nested children land in
// a real directory instead of resolving through the previous farm's symlink (local ordering
// exception to ADR-0006 · → ADR-0046). Removed ancestors are folded into result.Removed: the
// migration is silent by default and surfaced only under -v, never as a warning (→ ADR-0031).
func (a *applier) preRemove(actions []planner.RemoveAction) error {
	for _, act := range actions {
		if !reverifyStale(act) {
			a.opts.Warnf("nput: skipping ancestor migration because it drifted after planning: %s", act.Entry.Target)
			continue
		}
		if err := os.Remove(act.TargetAbs); err != nil {
			return fmt.Errorf("nput: cannot remove ancestor symlink for migration (%s): %w", act.TargetAbs, err)
		}
		a.result.Removed = append(a.result.Removed, act.Entry.Target)
	}
	return nil
}

// reverifyStale re-checks the conservative invariant on the real FS right before
// unlink: the target must still be a symlink pointing to the recorded dest.
func reverifyStale(act planner.RemoveAction) bool {
	info, err := os.Lstat(act.TargetAbs)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	onDisk, err := os.Readlink(act.TargetAbs)
	if err != nil {
		return false
	}
	return onDisk == planner.LinkDest(act.Entry)
}
