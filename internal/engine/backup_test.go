package engine

import (
	"os"
	"path/filepath"
	"testing"
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
