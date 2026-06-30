package engine

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// preflightErr_deniedDir creates a directory with mode 0000 and returns its path.
// Because the directory lacks search (execute) permission, os.Lstat on any path
// *inside* it fails with EACCES (a non-ENOENT error) before existence is checked.
// Cleanup restores 0755 so the surrounding t.TempDir removal succeeds.
func preflightErr_deniedDir(t *testing.T) string {
	t.Helper()
	denied := filepath.Join(realTempDir(t), "denied")
	if err := os.Mkdir(denied, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(denied, 0o755) })
	if err := os.Chmod(denied, 0o000); err != nil {
		t.Fatal(err)
	}
	return denied
}

// TestPreflightOutOfStoreNonENOENTError covers preflight.go:26-30: when os.Lstat on
// an out-of-store link target fails with a non-ENOENT error, checkOutOfStore wraps it
// as a generic runtime error ("cannot check out-of-store link target") and must NOT
// misreport it as a missing target ("does not exist"). The error is induced with EACCES
// from a search-denied parent directory, since there is no DI seam for os.Lstat.
func TestPreflightOutOfStoreNonENOENTError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires non-root to trigger permission error")
	}
	denied := preflightErr_deniedDir(t)
	// LinkDest = <denied>/child; Lstat fails with EACCES because <denied> lacks search permission.
	m := projectManifest(outOfStoreEntry(denied, "child", ".config/nvim"))
	a := &applier{manifest: &m}

	err := a.checkOutOfStore()
	if err == nil {
		t.Fatal("expected non-ENOENT error from Lstat, got nil")
	}
	if !strings.Contains(err.Error(), "cannot check out-of-store link target") {
		t.Errorf("error should be the non-ENOENT branch: %v", err)
	}
	if strings.Contains(err.Error(), "does not exist") {
		t.Errorf("non-ENOENT Lstat error must not be reported as a missing target: %v", err)
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Errorf("wrapped error should be a permission error (EACCES): %v", err)
	}
}
