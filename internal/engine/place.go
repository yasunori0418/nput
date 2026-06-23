package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yasunori0418/nput/internal/planner"
)

// place materializes the planner's Place actions as native symlinks
// (new / re-link before stale removal · → ADR-0006). The plan is already computed by
// planner.Compute from the current FS state, so this stays a thin executor that reflects
// the plan onto the real FS. This slice covers only store / out-of-store symlink placement
// (copy is a future slice · → Issue #6).
func (a *applier) place(actions []planner.PlaceAction) error {
	for _, act := range actions {
		// Create the parent directory (ancestor symlinks are already rejected as conflicts by the planner · → ADR-0015).
		if err := os.MkdirAll(filepath.Dir(act.TargetAbs), 0o755); err != nil {
			return fmt.Errorf("nput: 親ディレクトリを作成できません (%s): %w", filepath.Dir(act.TargetAbs), err)
		}

		switch act.Kind {
		case planner.PlaceNew:
			a.result.Placed = append(a.result.Placed, act.Entry.Target)
		case planner.PlaceReplace, planner.PlaceForeign:
			// Re-link is unlink + symlink (no rename-based atomic swap · → ADR-0017).
			// The foreign-overwrite warning is already emitted via planner.Warnings by emitWarnings (→ ADR-0015).
			if err := os.Remove(act.TargetAbs); err != nil {
				return fmt.Errorf("nput: 既存 symlink を除去できません (%s): %w", act.TargetAbs, err)
			}
			a.result.Replaced = append(a.result.Replaced, act.Entry.Target)
		}

		if err := os.Symlink(act.Dest, act.TargetAbs); err != nil {
			return fmt.Errorf("nput: symlink を作成できません (%s -> %s): %w", act.TargetAbs, act.Dest, err)
		}
	}
	return nil
}
