package engine

import (
	"fmt"
	"os"

	"github.com/yasunori0418/nput/internal/planner"
)

// removeStale applies the planner's Remove actions, re-verifying the conservative
// invariant against the real FS immediately before each unlink. The plan already
// restricts removals to「前世代記録あり かつ 現状も記録通りの先を指す symlink」,
// but placement (and any concurrent change) runs between planning and removal, so
// the stale-remover re-checks lstat/readlink and unlinks only when the invariant
// still holds; drifted targets are kept with a warning (→ ADR-0002, ADR-0006,
// docs/spec.md「stale 除去の対象と安全不変条件」).
func (a *applier) removeStale(actions []planner.RemoveAction) error {
	for _, act := range actions {
		if !reverifyStale(act) {
			a.opts.Warnf("nput: stale symlink が plan 後にドリフトしたため残します: %s", act.Entry.Target)
			continue
		}
		if err := os.Remove(act.TargetAbs); err != nil {
			return fmt.Errorf("nput: stale symlink を除去できません (%s): %w", act.TargetAbs, err)
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
