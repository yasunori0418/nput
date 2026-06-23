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
