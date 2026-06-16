package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yasunori0418/nput/internal/manifest"
)

// TestApplyDryRunNoSideEffects は apply --dryrun が plan を返すだけで FS / profileDir を一切
// 変えないことを検証する（→ ADR-0006, ADR-0023）。
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
	// symlink は張られない。
	if _, err := os.Lstat(filepath.Join(root, ".link")); !os.IsNotExist(err) {
		t.Errorf(".link should not be created in dryrun, lstat err = %v", err)
	}
	// profileDir も作られない（mkdir / flock を取らない）。
	if _, err := os.Stat(res.ProfileDir); !os.IsNotExist(err) {
		t.Errorf("profileDir should not be created in dryrun, stat err = %v", err)
	}
}

// TestApplyDryRunConflict は target が通常ファイルで占有されているとき conflict を plan に載せ、
// FS を変えないことを検証する（CLI が exit 2 を判定する・→ docs/spec.md 終了コード表）。
func TestApplyDryRunConflict(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "sub/file")
	lf := writeLinkFarm(t, homeManifest(storeEntry(src, "sub", ".link")))

	// target を通常ファイルで占有する（symlink 上書き不可 → conflict）。
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
