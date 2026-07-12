package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/planner"
)

// Tests for apply --backup (→ ADR-0045, issue #169): renaming an occupying foreign filesystem
// object aside to "<target>.<suffix>" before placement, instead of stopping on conflict.

// TestApplyBackupRenamesForeignFileAndPlaces verifies the basic flow: a regular file occupying a
// symlink placement target is renamed aside to "<target>.nput-backup" (the default suffix) and the
// entry is placed fresh, with the target reported in Result.BackedUp.
func TestApplyBackupRenamesForeignFileAndPlaces(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "foo")

	target := filepath.Join(root, "foo")
	if err := os.WriteFile(target, []byte("pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	lf := writeLinkFarm(t, projectManifest(storeEntry(src, "foo", "foo")))
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil), Backup: true,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, rerr := os.Readlink(target)
	if rerr != nil || got != filepath.Join(src, "foo") {
		t.Errorf("readlink(foo) = %q, err %v; want %q", got, rerr, filepath.Join(src, "foo"))
	}
	backupPath := target + ".nput-backup"
	data, berr := os.ReadFile(backupPath)
	if berr != nil || string(data) != "pre-existing" {
		t.Errorf("backup file content = %q, err = %v; want %q", data, berr, "pre-existing")
	}
	if len(res.BackedUp) != 1 || res.BackedUp[0] != "foo" {
		t.Errorf("Result.BackedUp = %v, want [foo]", res.BackedUp)
	}
}

// TestApplyBackupCustomSuffix verifies apply --backup=<suffix> uses the given suffix instead of
// the default "nput-backup".
func TestApplyBackupCustomSuffix(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "foo")

	target := filepath.Join(root, "foo")
	if err := os.WriteFile(target, []byte("pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	lf := writeLinkFarm(t, projectManifest(storeEntry(src, "foo", "foo")))
	if _, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
		Backup: true, BackupSuffix: "bak",
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if _, err := os.Lstat(target + ".bak"); err != nil {
		t.Errorf("backup with custom suffix must exist at foo.bak: %v", err)
	}
	if _, err := os.Lstat(target + ".nput-backup"); !os.IsNotExist(err) {
		t.Errorf("default-suffix path must not exist when a custom suffix is given, lstat err = %v", err)
	}
}

// TestApplyBackupSurvivesCommit verifies that unlike --recopy's rename-aside (cleaned up on
// success), a --backup aside is never removed once the run commits — it is the user's backup, kept
// indefinitely (→ ADR-0045 "reset は復元しない").
func TestApplyBackupSurvivesCommit(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "foo")

	target := filepath.Join(root, "foo")
	if err := os.WriteFile(target, []byte("pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	lf := writeLinkFarm(t, projectManifest(storeEntry(src, "foo", "foo")))
	if _, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil), Backup: true,
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if _, err := os.Lstat(target + ".nput-backup"); err != nil {
		t.Errorf("backup file must survive a successful commit: %v", err)
	}
}

// TestApplyBackupMidBatchFailureRestoresBackup verifies that a later placement failure in the same
// batch rolls the backup back: the target is restored to its pre-apply content and the aside path
// is gone, mirroring --recopy's rollback shape but keeping the destination on rollback failure too
// (→ ADR-0044, ADR-0045).
func TestApplyBackupMidBatchFailureRestoresBackup(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "foo")

	target := filepath.Join(root, "foo")
	if err := os.WriteFile(target, []byte("pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	blockWrite(t, filepath.Join(root, "ro"))

	lf := writeLinkFarm(t, projectManifest(
		storeEntry(src, "foo", "foo"),
		storeEntry(realTempDir(t), ".", "ro/leaf"),
	))
	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil), Backup: true,
	})
	if err == nil {
		t.Fatal("expected an error from the blocked mid-batch placement, got nil")
	}

	data, rerr := os.ReadFile(target)
	if rerr != nil || string(data) != "pre-existing" {
		t.Errorf("target must be restored to its pre-apply content: data=%q, err=%v", data, rerr)
	}
	if _, lerr := os.Lstat(target + ".nput-backup"); !os.IsNotExist(lerr) {
		t.Errorf("backup aside path must not be left behind after a rolled-back backup, lstat err = %v", lerr)
	}
}

// TestApplyBackupDestinationExistsConflict verifies that a pre-existing "<target>.<suffix>" (a
// leftover from an earlier backup) stops apply with a conflict instead of being silently
// overwritten (→ ADR-0045).
func TestApplyBackupDestinationExistsConflict(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "foo")

	target := filepath.Join(root, "foo")
	if err := os.WriteFile(target, []byte("pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target+".nput-backup", []byte("earlier backup"), 0o644); err != nil {
		t.Fatal(err)
	}

	lf := writeLinkFarm(t, projectManifest(storeEntry(src, "foo", "foo")))
	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil), Backup: true,
	})
	if err == nil {
		t.Fatal("expected a conflict error when the backup destination already exists, got nil")
	}

	data, rerr := os.ReadFile(target)
	if rerr != nil || string(data) != "pre-existing" {
		t.Errorf("target must be untouched on conflict: data=%q, err=%v", data, rerr)
	}
	data, rerr = os.ReadFile(target + ".nput-backup")
	if rerr != nil || string(data) != "earlier backup" {
		t.Errorf("earlier backup must be untouched on conflict: data=%q, err=%v", data, rerr)
	}
}

// TestApplyDryRunBackupNoConflictNoSideEffects verifies apply --dryrun --backup reports the
// occupying target as a planned backup (not a conflict) with zero exit-2-worthy conflicts and
// leaves the FS untouched (→ ADR-0045 "--dryrun --backup は conflict ではなく backup + 配置予定").
func TestApplyDryRunBackupNoConflictNoSideEffects(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "foo")

	target := filepath.Join(root, "foo")
	if err := os.WriteFile(target, []byte("pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	lf := writeLinkFarm(t, projectManifest(storeEntry(src, "foo", "foo")))
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, DryRun: true, Backup: true,
	})
	if err != nil {
		t.Fatalf("Apply dryrun: %v", err)
	}
	if len(res.Conflicts) != 0 {
		t.Errorf("Conflicts = %v, want none (--backup turns this into a planned backup)", res.Conflicts)
	}
	if len(res.BackedUp) != 1 || res.BackedUp[0] != "foo" {
		t.Errorf("Result.BackedUp = %v, want [foo]", res.BackedUp)
	}
	if len(res.Placed) != 1 || res.Placed[0] != "foo" {
		t.Errorf("Result.Placed = %v, want [foo]", res.Placed)
	}
	// dryrun must not touch the FS at all.
	data, rerr := os.ReadFile(target)
	if rerr != nil || string(data) != "pre-existing" {
		t.Errorf("target must be untouched by dryrun: data=%q, err=%v", data, rerr)
	}
	if _, lerr := os.Lstat(target + ".nput-backup"); !os.IsNotExist(lerr) {
		t.Errorf("no backup file must be created by dryrun, lstat err = %v", lerr)
	}
}

// TestApplyBackupDirTargetDoesNotWarnAboutSiblingRecordedLeaf verifies that when a real-dir target
// migration fails (a foreign leaf mixed in) and --backup renames the whole directory aside, a
// sibling leaf that prev recorded as its own stale symlink is NOT also reported by removeStale as
// "drifted after planning" — it left with the directory in the single rename, which is expected,
// not drift (→ ADR-0045, planner.markDirEntriesPreRemoved). Regression test for a diff-review
// finding: the remove-side loop used to independently schedule such a leaf as a Remove candidate
// (never marked preRemoved), so at execution time removeStale's reverifyStale would find it already
// gone (renamed away with the parent) and misreport a drift warning.
func TestApplyBackupDirTargetDoesNotWarnAboutSiblingRecordedLeaf(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	srcOld := makeSrc(t, "x")

	// First apply: place a per-file symlink at .claude/hooks/safe.sh (this generation's own record).
	lf1 := writeLinkFarm(t, projectManifest(storeEntry(srcOld, ".", ".claude/hooks/safe.sh")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// A foreign tool drops a real file next to it, occupying .claude/hooks as a real dir with a
	// foreign leaf — a whole-dir migration that would normally fail (foo.txt is not
	// recorded/migratable), forcing the --backup whole-dir rename path.
	if err := os.WriteFile(filepath.Join(root, ".claude", "hooks", "foo.txt"), []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second apply: new generation wants .claude/hooks as a single dir symlink (drops safe.sh's
	// entry). Without --backup this would conflict; with --backup the whole dir is renamed aside.
	srcNew := realTempDir(t)
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(srcNew, ".", ".claude/hooks")))
	var warns []string
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
		Backup: true, Warnf: collectFormatted(&warns),
	})
	if err != nil {
		t.Fatalf("second Apply (dir backup): %v", err)
	}
	if len(res.Conflicts) != 0 {
		t.Errorf("Conflicts = %v, want none", res.Conflicts)
	}
	if len(res.BackedUp) != 1 || res.BackedUp[0] != ".claude/hooks" {
		t.Errorf("Result.BackedUp = %v, want [.claude/hooks]", res.BackedUp)
	}
	if len(res.Removed) != 0 {
		t.Errorf("Result.Removed = %v, want none (safe.sh left with the whole-dir rename, not independently removed)", res.Removed)
	}
	for _, w := range warns {
		if strings.Contains(w, "drifted after planning") {
			t.Errorf("unexpected drift warning for a leaf that was backed up with its parent dir, not drifted: %q (all warnings: %v)", w, warns)
		}
	}

	got, rerr := os.Readlink(filepath.Join(root, ".claude", "hooks"))
	if rerr != nil || got != srcNew {
		t.Errorf("readlink(.claude/hooks) = %q, err %v; want %q", got, rerr, srcNew)
	}
	if _, lerr := os.Lstat(filepath.Join(root, ".claude", "hooks.nput-backup", "safe.sh")); lerr != nil {
		t.Errorf("backed-up dir must retain safe.sh: %v", lerr)
	}
}

// TestApplyGenerationSkipBackupRepairsForeignFile verifies that apply --backup also fires on the
// project-mode generation-skip (drift repair) path, not just normal apply: unlike PreRemove, Backup
// has no "derivation unchanged ⇒ never fires" invariant, since a foreign entity can appear at a
// target purely from an FS-level event between shell re-entries, independent of config content
// (→ ADR-0045, docs/spec.md "途中失敗時の巻き戻し" drift 修復の扱い).
func TestApplyGenerationSkipBackupRepairsForeignFile(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	var commits [][2]string
	applyOnce(t, lf, "c", root, state, &commits, nil)

	// foreign tool replaces the target with a regular file between shell re-entries.
	tgt := filepath.Join(root, ".config", "foo")
	if err := os.Remove(tgt); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tgt, []byte("foreign content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second apply: same link-farm (generation skip candidate) with --backup enabled.
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits), Backup: true,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !res.GenerationSkipped {
		t.Errorf("GenerationSkipped = false, want true (same derivation)")
	}
	if len(commits) != 1 {
		t.Errorf("commit calls = %d, want 1 (no new generation)", len(commits))
	}
	if len(res.BackedUp) != 1 || res.BackedUp[0] != ".config/foo" {
		t.Errorf("Result.BackedUp = %v, want [.config/foo]", res.BackedUp)
	}
	got, rerr := os.Readlink(tgt)
	if rerr != nil || got != src {
		t.Errorf("readlink(.config/foo) = %q, err %v; want %q", got, rerr, src)
	}
	data, berr := os.ReadFile(tgt + ".nput-backup")
	if berr != nil || string(data) != "foreign content" {
		t.Errorf("backup file content = %q, err = %v; want %q", data, berr, "foreign content")
	}
}

// TestApplyWithoutBackupStillConflicts verifies the default (Backup: false) behavior is unchanged:
// a foreign regular file at a placement target still stops apply with a conflict, not a backup.
func TestApplyWithoutBackupStillConflicts(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "foo")

	target := filepath.Join(root, "foo")
	if err := os.WriteFile(target, []byte("pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	lf := writeLinkFarm(t, projectManifest(storeEntry(src, "foo", "foo")))
	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil {
		t.Fatal("expected a conflict error without --backup, got nil")
	}
	if _, lerr := os.Lstat(target + ".nput-backup"); !os.IsNotExist(lerr) {
		t.Errorf("no backup file must be created without --backup, lstat err = %v", lerr)
	}
}

// TestApplierBackupReverifiesDestinationImmediatelyBeforeRename directly exercises applier.backup
// with a BackupAction whose destination did NOT exist when the plan was computed but exists by the
// time backup() executes (simulating the plan/execute TOCTOU window backup.go's doc comment
// describes) — mirroring TestPreRemoveJournalRmdirThenUnlinkOrder's pattern of driving an applier
// stage directly, without a full Apply. backup() must abort loudly rather than clobber the
// concurrently-created destination (→ ADR-0045, ADR-0017).
func TestApplierBackupReverifiesDestinationImmediatelyBeforeRename(t *testing.T) {
	root := realTempDir(t)
	target := filepath.Join(root, "foo")
	backupAbs := target + ".nput-backup"
	if err := os.WriteFile(target, []byte("pre-existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Simulates a concurrent writer creating the destination after Compute planned this
	// BackupAction (which found it absent) but before backup() reached it.
	if err := os.WriteFile(backupAbs, []byte("concurrently created"), 0o644); err != nil {
		t.Fatal(err)
	}

	var warns []string
	a := &applier{opts: Options{Warnf: collectWarnings(&warns)}, result: &Result{}}
	a.root = root
	actions := []planner.BackupAction{
		{Entry: storeEntry(root, ".", "foo"), TargetAbs: target, BackupAbs: backupAbs},
	}
	err := a.backup(actions)
	if err == nil {
		t.Fatal("expected an error when the backup destination appears between planning and execution, got nil")
	}

	data, rerr := os.ReadFile(target)
	if rerr != nil || string(data) != "pre-existing" {
		t.Errorf("target must be untouched: data=%q, err=%v", data, rerr)
	}
	data, rerr = os.ReadFile(backupAbs)
	if rerr != nil || string(data) != "concurrently created" {
		t.Errorf("concurrently-created destination must not be clobbered: data=%q, err=%v", data, rerr)
	}
	if len(a.result.BackedUp) != 0 {
		t.Errorf("Result.BackedUp = %v, want none (nothing was actually backed up)", a.result.BackedUp)
	}
}
