package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/yasunori0418/nput/internal/planner"
)

// removeStale applies the planner's Remove actions, re-verifying the conservative
// invariant against the real FS immediately before each unlink. The plan already
// restricts removals to "symlinks recorded in the previous generation that still
// point at the recorded dest", but placement (and any concurrent change) runs
// between planning and removal, so the stale-remover re-checks lstat/readlink and
// unlinks only when the invariant still holds; drifted targets are kept with a
// warning (→ ADR-0002, ADR-0006, docs/spec.md "stale removal targets and safety invariant").
// After each unlink it walks the parent chain toward root, pruning ancestors left
// empty by the removal (→ Issue #174, #172 (D4)); a prune failure is folded
// into the same keep+warn policy as a drifted target, since the target itself is
// already gone and nothing is lost by leaving a leftover empty dir.
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
		if err := a.pruneEmptyAncestors(act.TargetAbs); err != nil {
			a.opts.Warnf("nput: could not prune an empty ancestor directory: %v", err)
		}
	}
	return nil
}

// preRemove unlinks the self-recorded stale ancestor symlinks the planner scheduled for
// migration, re-verifying the conservative invariant against the real FS right before each
// unlink. It runs *before* place so nested children land in a real directory instead of
// resolving through the previous farm's symlink (local ordering exception to ADR-0006 · → ADR-0046).
// Removed ancestors are folded into result.Removed: the migration is silent by default and
// surfaced only under -v, never as a warning (→ ADR-0031).
//
// Unlike removeStale — the last stage, where keeping a drifted link is harmless — a drifted
// ancestor here is *not* safe to skip: the children under it were planned as unconditional new
// placements that assume the ancestor is gone, so continuing would nest them through the drifted
// symlink (e.g. one swapped to a foreign, writable dir) and re-open the ADR-0015 §4 pollution that
// place/ensureParentDir do not re-guard. So on drift it aborts loudly instead of skipping; an
// idempotent re-run re-plans against the current FS and converges (→ ADR-0017, ADR-0046).
//
// It also prunes ancestors left empty by the unlink (→ Issue #174); unlike the drift check
// above, a prune failure does not abort — the migration itself already succeeded, so a
// leftover empty dir is a cosmetic miss, not a correctness hazard for the child placements
// that follow.
func (a *applier) preRemove(actions []planner.RemoveAction) error {
	for _, act := range actions {
		if !reverifyStale(act) {
			return fmt.Errorf("nput: ancestor symlink changed after planning; cannot migrate its nesting safely (%s); re-run apply to converge", act.Entry.Target)
		}
		if err := os.Remove(act.TargetAbs); err != nil {
			return fmt.Errorf("nput: cannot remove ancestor symlink for migration (%s): %w", act.TargetAbs, err)
		}
		a.result.Removed = append(a.result.Removed, act.Entry.Target)
		if err := a.pruneEmptyAncestors(act.TargetAbs); err != nil {
			a.opts.Warnf("nput: could not prune an empty ancestor directory: %v", err)
		}
	}
	return nil
}

// pruneEmptyAncestors walks removedAbs's parent chain toward a.root, rmdir-ing each
// ancestor left empty by a removal (→ Issue #174, #172 (D4) · HM `rmdir -p
// --ignore-fail-on-non-empty` counterpart). rmdir only ever succeeds on an empty
// directory, so this is TOCTOU-safe without extra locking: a concurrent writer simply
// makes the dir non-empty again and the walk stops there.
//
// Stop conditions: root itself is never removed; a non-empty ancestor (ENOTEMPTY) is
// left in place and the walk stops there (conservative — never touches directories the
// removal did not empty); a symlink ancestor stops the walk without touching it (rmdir
// on a symlink targets the link itself, not what it points at, and a symlink ancestor
// means the walk has left the tree this removal actually emptied). Pruned dirs are
// folded into result.Pruned, surfaced only under -v like the rest of the placement
// report (→ ADR-0031).
func (a *applier) pruneEmptyAncestors(removedAbs string) error {
	root := filepath.Clean(a.root)
	dir := filepath.Dir(removedAbs)
	for {
		dir = filepath.Clean(dir)
		if dir == root || !isWithinRoot(root, dir) {
			return nil
		}
		info, err := os.Lstat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("nput: cannot lstat ancestor directory (%s): %w", dir, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if err := os.Remove(dir); err != nil {
			// non-empty dir → ENOTEMPTY on Linux, EEXIST on some BSD/Darwin rmdir(2) implementations;
			// both mean "left something behind, stop here" rather than a real failure.
			if errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST) {
				return nil
			}
			return fmt.Errorf("nput: cannot remove empty ancestor directory (%s): %w", dir, err)
		}
		a.result.Pruned = append(a.result.Pruned, dir)
		dir = filepath.Dir(dir)
	}
}

// isWithinRoot reports whether dir is root or a descendant of root, guarding the
// upward walk from ever stepping outside root's subtree (defense in depth alongside
// the dir == root stop condition above).
func isWithinRoot(root, dir string) bool {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
