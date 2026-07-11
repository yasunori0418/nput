package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yasunori0418/nput/internal/manifest"
)

// TestApplyDryRunNoSideEffects verifies that apply --dryrun only returns a plan and
// changes neither the FS nor the profileDir (→ ADR-0006, ADR-0023).
func TestApplyDryRunNoSideEffects(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "sub/file")
	lf := writeLinkFarm(t, homeManifest(storeEntry(src, "sub", ".link")))

	res, err := Apply(Options{
		LinkFarm:     lf,
		Name:         "cfg",
		RootKind:     manifest.RootKindHome,
		RootOverride: root,
		StateDir:     state,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("Apply dryrun: %v", err)
	}

	if !res.DryRun {
		t.Error("Result.DryRun = false, want true")
	}
	if got := res.Placed; len(got) != 1 || got[0] != ".link" {
		t.Errorf("Placed = %v, want [.link]", got)
	}
	// symlink is not created.
	if _, err := os.Lstat(filepath.Join(root, ".link")); !os.IsNotExist(err) {
		t.Errorf(".link should not be created in dryrun, lstat err = %v", err)
	}
	// profileDir is not created either (no mkdir / flock taken).
	if _, err := os.Stat(res.ProfileDir); !os.IsNotExist(err) {
		t.Errorf("profileDir should not be created in dryrun, stat err = %v", err)
	}
}

// TestApplyDryRunConflict verifies that when the target is occupied by a regular file a
// conflict is recorded in the plan and the FS is left unchanged (the CLI decides exit 2 ·
// → docs/spec.md exit code table).
func TestApplyDryRunConflict(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "sub/file")
	lf := writeLinkFarm(t, homeManifest(storeEntry(src, "sub", ".link")))

	// Occupy the target with a regular file (cannot overwrite as symlink → conflict).
	if err := os.WriteFile(filepath.Join(root, ".link"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Apply(Options{
		LinkFarm:     lf,
		Name:         "cfg",
		RootKind:     manifest.RootKindHome,
		RootOverride: root,
		StateDir:     state,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("Apply dryrun: %v", err)
	}
	if len(res.Conflicts) == 0 {
		t.Error("expected a conflict in dryrun plan, got none")
	}
	if len(res.Placed) != 0 {
		t.Errorf("conflicting entry should not be planned for place, Placed = %v", res.Placed)
	}
}

// TestApplyDryRunAncestorMigration verifies that apply --dryrun reports a self-recorded stale
// ancestor migration as a (non-conflict) removal plus child placements and leaves the FS
// untouched (→ ADR-0046).
func TestApplyDryRunAncestorMigration(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)

	// First apply commits a whole-tree symlink at .claude/skills (sets up the previous generation).
	// srcOld deliberately does not contain "foo", so lstat of the child through the still-present
	// ancestor symlink resolves to nothing and the "FS untouched" check below is meaningful.
	srcOld := makeSrc(t, "other")
	lf1 := writeLinkFarm(t, projectManifest(storeEntry(srcOld, ".", ".claude/skills")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// Dryrun the migration to nested children.
	srcNew := makeSrc(t, "foo")
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(srcNew, "foo", ".claude/skills/foo")))
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, DryRun: true,
	})
	if err != nil {
		t.Fatalf("dryrun Apply: %v", err)
	}
	if len(res.Conflicts) != 0 {
		t.Errorf("Conflicts = %v, want none (a self-recorded ancestor is a migration, not a conflict)", res.Conflicts)
	}
	if len(res.Removed) != 1 || res.Removed[0] != ".claude/skills" {
		t.Errorf("Removed = %v, want [.claude/skills]", res.Removed)
	}
	if len(res.Placed) != 1 || res.Placed[0] != ".claude/skills/foo" {
		t.Errorf("Placed = %v, want [.claude/skills/foo]", res.Placed)
	}
	// FS untouched: the ancestor symlink is still present and the child was not created.
	if got, err := os.Readlink(filepath.Join(root, ".claude", "skills")); err != nil || got != srcOld {
		t.Errorf(".claude/skills should be untouched in dryrun = %q (err %v); want symlink to %q", got, err, srcOld)
	}
	if _, err := os.Lstat(filepath.Join(root, ".claude", "skills", "foo")); !os.IsNotExist(err) {
		t.Errorf("child should not be created in dryrun, lstat err = %v", err)
	}
}
