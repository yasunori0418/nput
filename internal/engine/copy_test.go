package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/planner"
)

// copy_test.go covers the error-return paths of copy.go that the high-level Apply
// tests in engine_test.go do not reach (placeCopies/recopyAll/copyTree/copyFile/copySymlink).
//
// All failures here are induced through root-proof FS conditions rather than permission
// denial (a regular file standing in for a directory → ENOTDIR, an opened directory fd →
// EISDIR on read, a non-symlink → EINVAL on readlink). They therefore need no EUID guard
// and stay valid when the suite runs as root in CI.

// copyErr_blockerFile creates a regular file at <dir>/<name> and returns its path.
// Using it as a path *component* makes any MkdirAll / Lstat / OpenFile that must
// traverse "through" it fail with ENOTDIR for every uid (root included).
func copyErr_blockerFile(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("blocker"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestCopyPlaceMkdirError exercises placeCopies' parent-directory creation failure
// (copy.go:31-32). The target's parent path runs through a regular file, so MkdirAll
// returns ENOTDIR before copyTree is ever reached.
func TestCopyPlaceMkdirError(t *testing.T) {
	root := realTempDir(t)
	blocker := copyErr_blockerFile(t, root, "blocker")
	src := makeROSrc(t, "file.txt", "store-content")

	a := &applier{result: &Result{}}
	act := planner.CopyAction{
		Entry:     copyEntry(src, "file.txt", "blocker/sub/foo"),
		TargetAbs: filepath.Join(blocker, "sub", "foo"), // parent = <root>/blocker/sub, blocker is a file
		Src:       filepath.Join(src, "file.txt"),
	}

	err := a.placeCopies([]planner.CopyAction{act})
	if err == nil {
		t.Fatal("expected mkdir parent error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot create parent directory") {
		t.Errorf("error = %v, want a parent-directory message", err)
	}
	if len(a.result.Copied) != 0 {
		t.Errorf("Copied = %v, want none on mkdir failure", a.result.Copied)
	}
}

// TestCopyTreeDirMkdirError exercises copyTree's directory branch error
// (copy.go:113-117). With a directory src, WalkDir's first node is the tree root, whose
// MkdirAll(dst) fails because dst's parent is a regular file (ENOTDIR). This is the only
// owner-inducible failure in that block: the chmod at line 117 cannot be isolated without
// a DI seam (copy.go always re-adds owner-write, so a subdir mkdir never hits a perm wall),
// and adding a seam is out of scope for this file-restricted task.
func TestCopyTreeDirMkdirError(t *testing.T) {
	root := realTempDir(t)
	blocker := copyErr_blockerFile(t, root, "blocker")

	// src is a real directory tree.
	src := filepath.Join(realTempDir(t), "tree")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	// dst's parent is a regular file → MkdirAll(dst) inside the walk returns ENOTDIR.
	dst := filepath.Join(blocker, "dst")
	if err := copyTree(src, dst); err == nil {
		t.Fatal("expected copyTree directory-branch mkdir error, got nil")
	}
}

// TestCopyFileOpenError exercises copyFile's os.Open failure (copy.go:127-128):
// a non-existent src fails to open for every uid.
func TestCopyFileOpenError(t *testing.T) {
	dst := filepath.Join(realTempDir(t), "out")
	missing := filepath.Join(realTempDir(t), "does-not-exist")

	if err := copyFile(missing, dst, 0o644); err == nil {
		t.Fatal("expected open(src) error, got nil")
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Errorf("dst should not be created on open failure: lstat err = %v", err)
	}
}

// TestCopyFileOpenFileError exercises copyFile's os.OpenFile(dst) failure
// (copy.go:134-135): the dst parent runs through a regular file (ENOTDIR).
func TestCopyFileOpenFileError(t *testing.T) {
	dir := realTempDir(t)
	src := filepath.Join(dir, "in")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	blocker := copyErr_blockerFile(t, dir, "blocker")
	dst := filepath.Join(blocker, "out") // parent is a file → OpenFile fails with ENOTDIR

	if err := copyFile(src, dst, 0o644); err == nil {
		t.Fatal("expected open(dst) error, got nil")
	}
}

// TestCopyFileCopyError exercises copyFile's io.Copy failure (copy.go:137-139):
// opening a directory as src succeeds, but reading from a directory fd yields EISDIR.
func TestCopyFileCopyError(t *testing.T) {
	srcDir := realTempDir(t) // a directory; os.Open succeeds, read fails (EISDIR)
	dst := filepath.Join(realTempDir(t), "out")

	if err := copyFile(srcDir, dst, 0o644); err == nil {
		t.Fatal("expected io.Copy error reading a directory, got nil")
	}
}

// TestCopySymlinkReadlinkError exercises copySymlink's os.Readlink failure
// (copy.go:151-152): a regular (non-symlink) file makes Readlink return EINVAL.
func TestCopySymlinkReadlinkError(t *testing.T) {
	dir := realTempDir(t)
	notALink := filepath.Join(dir, "regular")
	if err := os.WriteFile(notALink, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst")

	if err := copySymlink(notALink, dst); err == nil {
		t.Fatal("expected readlink error on a non-symlink, got nil")
	}
}

// TestCopyRecopyLstatError exercises recopyAll's non-ENOENT Lstat branch
// (copy.go:59-60): the recopy target path runs through a regular file, so Lstat returns
// ENOTDIR — distinct from "absent" (ENOENT), so recopyAll must surface it as an error.
func TestCopyRecopyLstatError(t *testing.T) {
	root := realTempDir(t)
	copyErr_blockerFile(t, root, "blocker")
	src := makeROSrc(t, "file.txt", "store-content")

	a := &applier{
		root:   root,
		result: &Result{},
		manifest: &manifest.Manifest{
			SchemaVersion: 1,
			Root:          manifest.Root{RootKind: manifest.RootKindProject},
			// Target traverses the regular file "blocker" → Lstat → ENOTDIR (not ENOENT).
			Entries: []manifest.Entry{copyEntry(src, "file.txt", "blocker/foo")},
		},
	}

	err := a.recopyAll()
	if err == nil {
		t.Fatal("expected non-ENOENT lstat error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot lstat recopy target") {
		t.Errorf("error = %v, want a lstat-recopy message", err)
	}
	if len(a.result.Recopied) != 0 || len(a.result.Copied) != 0 {
		t.Errorf("nothing should be recorded on lstat failure: Recopied=%v Copied=%v",
			a.result.Recopied, a.result.Copied)
	}
}
