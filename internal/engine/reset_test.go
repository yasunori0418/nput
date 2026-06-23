package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yasunori0418/nput/internal/manifest"
)

// applyForReset commits one generation as a precondition for reset tests (prof.Profile points at the link-farm).
// The --root override (RootOverride=root) fixes root without depending on git.
func applyForReset(t *testing.T, root, state string, m manifest.Manifest) {
	t.Helper()
	lf := writeLinkFarm(t, m)
	if _, err := Apply(Options{
		LinkFarm:     lf,
		Name:         "cfg",
		RootKind:     manifest.RootKindHome,
		RootOverride: root,
		StateDir:     state,
		Commit:       fakeCommit(nil),
	}); err != nil {
		t.Fatalf("setup Apply: %v", err)
	}
}

func resetOpts(root, state string, targets []string, dryrun bool, confirm func(*ResetResult) (bool, error)) ResetOptions {
	return ResetOptions{
		Name:         "cfg",
		RootKind:     manifest.RootKindHome,
		RootOverride: root,
		StateDir:     state,
		Targets:      targets,
		DryRun:       dryrun,
		Confirm:      confirm,
	}
}

func TestResetRemovesSymlinkAndCopy(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	symSrc := makeSrc(t, "sub/file")
	copySrc := makeSrc(t, "data/x")

	applyForReset(t, root, state, homeManifest(
		storeEntry(symSrc, "sub", ".link"),
		copyEntry(copySrc, "data", ".copied"),
	))

	linkAbs := filepath.Join(root, ".link")
	copyAbs := filepath.Join(root, ".copied")
	if fi, err := os.Lstat(linkAbs); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("setup: .link not a symlink: %v", err)
	}
	if _, err := os.Stat(copyAbs); err != nil {
		t.Fatalf("setup: .copied missing: %v", err)
	}

	res, err := Reset(resetOpts(root, state, nil, false, nil))
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if _, err := os.Lstat(linkAbs); !os.IsNotExist(err) {
		t.Errorf(".link should be removed, lstat err = %v", err)
	}
	if _, err := os.Lstat(copyAbs); !os.IsNotExist(err) {
		t.Errorf(".copied should be removed, lstat err = %v", err)
	}
	if got := res.RemovedSymlinks; len(got) != 1 || got[0] != ".link" {
		t.Errorf("RemovedSymlinks = %v, want [.link]", got)
	}
	if got := res.RemovedCopies; len(got) != 1 || got[0] != ".copied" {
		t.Errorf("RemovedCopies = %v, want [.copied]", got)
	}
}

func TestResetKeepsForeignSymlink(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	symSrc := makeSrc(t, "sub/file")

	applyForReset(t, root, state, homeManifest(storeEntry(symSrc, "sub", ".link")))

	// Swap the placed symlink to point elsewhere (simulate foreign / record mismatch).
	linkAbs := filepath.Join(root, ".link")
	if err := os.Remove(linkAbs); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/somewhere/else", linkAbs); err != nil {
		t.Fatal(err)
	}

	res, err := Reset(resetOpts(root, state, nil, false, nil))
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}

	// Kept because it does not satisfy the conservative invariant.
	if _, err := os.Lstat(linkAbs); err != nil {
		t.Errorf("foreign symlink should be kept, lstat err = %v", err)
	}
	if len(res.RemovedSymlinks) != 0 {
		t.Errorf("RemovedSymlinks = %v, want empty", res.RemovedSymlinks)
	}
	if got := res.KeptForeign; len(got) != 1 || got[0] != ".link" {
		t.Errorf("KeptForeign = %v, want [.link]", got)
	}
}

func TestResetDryRunNoSideEffects(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	symSrc := makeSrc(t, "sub/file")
	copySrc := makeSrc(t, "data/x")

	applyForReset(t, root, state, homeManifest(
		storeEntry(symSrc, "sub", ".link"),
		copyEntry(copySrc, "data", ".copied"),
	))

	res, err := Reset(resetOpts(root, state, nil, true, nil))
	if err != nil {
		t.Fatalf("Reset dryrun: %v", err)
	}
	if !res.DryRun {
		t.Error("Result.DryRun = false, want true")
	}

	// FS unchanged (preview only).
	if _, err := os.Lstat(filepath.Join(root, ".link")); err != nil {
		t.Errorf(".link should still exist after dryrun: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".copied")); err != nil {
		t.Errorf(".copied should still exist after dryrun: %v", err)
	}
	// The plan enumerates the to-be-removed entries.
	if len(res.RemovedSymlinks) != 1 || len(res.RemovedCopies) != 1 {
		t.Errorf("dryrun plan = sym%v copy%v, want one each", res.RemovedSymlinks, res.RemovedCopies)
	}
}

func TestResetTargetFilter(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "sub/file")

	applyForReset(t, root, state, homeManifest(
		storeEntry(src, "sub", ".a"),
		storeEntry(src, "sub", ".b"),
	))

	if _, err := Reset(resetOpts(root, state, []string{".a"}, false, nil)); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if _, err := os.Lstat(filepath.Join(root, ".a")); !os.IsNotExist(err) {
		t.Errorf(".a should be removed, lstat err = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".b")); err != nil {
		t.Errorf(".b should remain (not targeted): %v", err)
	}
}

func TestResetUnknownTargetErrors(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "sub/file")
	applyForReset(t, root, state, homeManifest(storeEntry(src, "sub", ".a")))

	_, err := Reset(resetOpts(root, state, []string{".nope"}, false, nil))
	if err == nil {
		t.Fatal("Reset with unknown target should error")
	}
}

func TestResetNoProfileIsNoop(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)

	// Never applied (profile absent) → no-op, no error.
	res, err := Reset(resetOpts(root, state, nil, false, nil))
	if err != nil {
		t.Fatalf("Reset no-profile: %v", err)
	}
	if len(res.RemovedSymlinks) != 0 || len(res.RemovedCopies) != 0 {
		t.Errorf("expected no-op, got %+v", res)
	}
}

func TestResetConfirmAbort(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "sub/file")
	applyForReset(t, root, state, homeManifest(storeEntry(src, "sub", ".a")))

	// If confirm returns false, abort and leave the FS unchanged.
	res, err := Reset(resetOpts(root, state, nil, false, func(*ResetResult) (bool, error) {
		return false, nil
	}))
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if !res.Aborted {
		t.Error("Result.Aborted = false, want true")
	}
	if _, err := os.Lstat(filepath.Join(root, ".a")); err != nil {
		t.Errorf(".a should remain after abort: %v", err)
	}
}
