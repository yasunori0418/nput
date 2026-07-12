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

// preRemove removes the self-recorded stale filesystem objects the planner scheduled to clear a
// placement target: ancestor symlinks (→ ADR-0046), and — generalized under ADR-0047 — leaf
// symlinks and now-empty directories beneath an occupying real directory, and a symlink replaced
// by method change (symlink→copy). It runs *before* place so the target lands empty/absent
// instead of resolving through stale content (local ordering exception to ADR-0006). Removed
// objects are folded into result.Removed (Unlink) / result.Pruned (Rmdir): the migration is
// silent by default and surfaced only under -v, never as a warning (→ ADR-0031).
//
// Unlike removeStale — the last stage, where keeping a drifted link is harmless — drift here is
// *not* safe to skip: placement was planned assuming these objects are gone (children as
// unconditional new placements, or the target itself as absent), so continuing on drift would
// nest/place through it (e.g. a symlink swapped to a foreign, writable dir) and re-open the
// ADR-0015 §4 pollution that place/ensureParentDir do not re-guard. So on drift it aborts loudly
// instead of skipping for both Unlink and Rmdir (→ ADR-0017, ADR-0046, ADR-0047 D3); an idempotent
// re-run re-plans against the current FS and converges.
//
// It also prunes ancestors left empty by an Unlink (→ Issue #174); unlike the drift check above,
// a prune failure does not abort — the removal itself already succeeded, so a leftover empty dir
// is a cosmetic miss, not a correctness hazard for the child placements that follow.
func (a *applier) preRemove(actions []planner.RemoveAction) error {
	for _, act := range actions {
		switch act.Kind {
		case planner.RemoveRmdir:
			// No pre-check of emptiness: os.Remove on a non-empty dir simply fails with ENOTEMPTY,
			// which IS the re-verification (TOCTOU-safe without a separate stat+readdir race · → ADR-0047 D3).
			if err := os.Remove(act.TargetAbs); err != nil {
				if errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST) {
					return fmt.Errorf("nput: directory gained content after planning; cannot migrate this placement target safely (%s); re-run apply to converge", act.TargetAbs)
				}
				return fmt.Errorf("nput: cannot remove empty directory for migration (%s): %w", act.TargetAbs, err)
			}
			a.result.Pruned = append(a.result.Pruned, act.TargetAbs)
			if err := a.pruneEmptyAncestors(act.TargetAbs); err != nil {
				a.opts.Warnf("nput: could not prune an empty ancestor directory: %v", err)
			}
		default: // planner.RemoveUnlink
			if !reverifyStale(act) {
				return fmt.Errorf("nput: recorded symlink changed after planning; cannot migrate this placement target safely (%s); re-run apply to converge", act.Entry.Target)
			}
			if err := os.Remove(act.TargetAbs); err != nil {
				return fmt.Errorf("nput: cannot remove recorded symlink for migration (%s): %w", act.TargetAbs, err)
			}
			a.result.Removed = append(a.result.Removed, act.Entry.Target)
			if err := a.pruneEmptyAncestors(act.TargetAbs); err != nil {
				a.opts.Warnf("nput: could not prune an empty ancestor directory: %v", err)
			}
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
// removal did not empty); a symlink *anywhere in dir's path from root* stops the walk
// without touching it. This must be a full re-check from root on every iteration, not
// just an Lstat of dir itself: Lstat resolves intermediate path components, so once any
// ancestor component is a symlink, "dir" as a path silently resolves through it into a
// foreign subtree — Remove(dir) would then rmdir *through* the symlink and delete real
// directories outside root's tree that this removal never emptied (the ADR-0015 style
// pollution this walk must not reopen). Pruned dirs are folded into result.Pruned,
// surfaced only under -v like the rest of the placement report (→ ADR-0031).
func (a *applier) pruneEmptyAncestors(removedAbs string) error {
	root := filepath.Clean(a.root)
	dir := filepath.Dir(removedAbs)
	for {
		dir = filepath.Clean(dir)
		if dir == root || !isWithinRoot(root, dir) {
			return nil
		}
		hasSymlink, err := pathHasSymlinkComponent(root, dir)
		if err != nil {
			return err
		}
		if hasSymlink {
			return nil
		}
		if err := os.Remove(dir); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
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

// pathHasSymlinkComponent reports whether any path component strictly between root and
// dir (dir itself included) is a symlink, lstat-ing each component from root downward
// rather than the resolved dir path so an ancestor symlink is caught before Remove(dir)
// would silently resolve through it (→ pruneEmptyAncestors).
func pathHasSymlinkComponent(root, dir string) (bool, error) {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return false, fmt.Errorf("nput: cannot resolve %q relative to root (%s): %w", dir, root, err)
	}
	cur := root
	for _, comp := range strings.Split(rel, string(filepath.Separator)) {
		if comp == "" || comp == "." {
			continue
		}
		cur = filepath.Join(cur, comp)
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, fmt.Errorf("nput: cannot lstat ancestor directory (%s): %w", cur, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true, nil
		}
	}
	return false, nil
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
