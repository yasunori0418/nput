package engine

import (
	"fmt"
	"os"

	"github.com/yasunori0418/nput/internal/planner"
)

// backup applies the planner's Backup actions: rename each occupying foreign filesystem object
// aside (TargetAbs → BackupAbs = "<target>.<suffix>") so the placement stage that follows lands
// on an absent target (→ ADR-0045, issue #169). It runs after PreRemove and before Place /
// materializeCopies, mirroring recopyAll's rename-aside (asidePath / journalRenamedAside): the
// rename is a same-parent-directory metadata-only operation that cannot fail partway under
// ENOSPC. Undo restores it with the undoRestoreBackup inverse (remove whatever this run placed at
// TargetAbs, then rename BackupAbs back) — unlike recopy's aside, a successful commit does NOT
// clean up BackupAbs; it is the user's backup and stays on disk (→ ADR-0044, ADR-0045).
//
// The backup destination's absence was already checked at plan time (→ planner.appendBackup), but
// a re-check immediately before the rename guards the same plan/execute TOCTOU window PreRemove's
// reverifyStale guards for stale removal: a concurrent writer could have created BackupAbs between
// planning and this run reaching it. Unlike PreRemove's drift handling, there is nothing here to
// "keep as-is" on drift — the destination existing at all means a prior backup would be silently
// clobbered, so this aborts loudly (→ ADR-0017 idempotent re-run convergence).
func (a *applier) backup(actions []planner.BackupAction) error {
	for _, act := range actions {
		if _, err := os.Lstat(act.BackupAbs); err == nil {
			return fmt.Errorf("nput: backup destination already exists (%s); cannot safely back up %s; re-run apply to converge", act.BackupAbs, act.TargetAbs)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("nput: cannot lstat backup destination (%s): %w", act.BackupAbs, err)
		}
		if err := os.Rename(act.TargetAbs, act.BackupAbs); err != nil {
			return fmt.Errorf("nput: cannot rename aside for backup (%s -> %s): %w", act.TargetAbs, act.BackupAbs, err)
		}
		// Journaled immediately after the rename, before placement lands: if a later stage fails,
		// undoRestoreBackup's os.RemoveAll tolerates TargetAbs being absent/partial and still
		// renames BackupAbs back, restoring the pre-apply content (→ ADR-0044).
		a.journalBackedUp(act.TargetAbs, act.BackupAbs)
		a.result.BackedUp = append(a.result.BackedUp, act.Entry.Target)
		a.opts.Warnf("nput: backed up existing target before placement: %s -> %s", act.TargetAbs, act.BackupAbs)
	}
	return nil
}
