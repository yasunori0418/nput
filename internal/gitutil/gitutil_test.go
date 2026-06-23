package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	// On macOS /tmp is a symlink to /private/tmp, so compare against the resolved path.
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return dir
}

func TestToplevelFromSubdir(t *testing.T) {
	root := initRepo(t)
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Toplevel(sub)
	if err != nil {
		t.Fatalf("Toplevel: %v", err)
	}
	got, _ = filepath.EvalSymlinks(got)
	if got != root {
		t.Errorf("Toplevel = %q, want %q", got, root)
	}
}

func TestToplevelOutsideRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	// Use an isolated tmpdir, since an ancestor like $HOME being a git repo would cause false positives.
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))

	if _, err := Toplevel(dir); err == nil {
		t.Fatal("expected error outside repo, got nil")
	}
}

func TestToplevelGitNotInPath(t *testing.T) {
	t.Setenv("PATH", "")
	if _, err := Toplevel(t.TempDir()); err != ErrGitNotFound {
		t.Fatalf("git-not-in-PATH: got %v, want ErrGitNotFound", err)
	}
}
