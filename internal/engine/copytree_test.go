package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// copytree_test.go directly unit-tests copyTree / copyFile / copySymlink
// (copy.go:84-157), focusing on structural success paths not exercised through
// Apply: deep nesting, directory-mode owner-write, empty dirs, and in-tree
// symlink duplication without deref.
//
// Shared helpers (realTempDir etc.) are reused from engine_test.go; helpers
// added here use the copyTree_ prefix and tests use the TestCopyTree* /
// TestCopyFile* / TestCopySymlink* names to stay collision-free.

// copyTree_chmodRestore registers a cleanup that restores owner-write on a path,
// so that read-only src dirs/files do not break t.TempDir's RemoveAll cleanup.
func copyTree_chmodRestore(t *testing.T, path string) {
	t.Helper()
	t.Cleanup(func() { _ = os.Chmod(path, 0o755) })
}

func TestCopyTreeNestedDirsPreserveStructure(t *testing.T) {
	src := realTempDir(t)
	// Multi-level nested tree: a/b/c/deep.txt plus a sibling at a/top.txt.
	if err := os.MkdirAll(filepath.Join(src, "a", "b", "c"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a", "b", "c", "deep.txt"), []byte("deep"), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a", "top.txt"), []byte("top"), 0o444); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(realTempDir(t), "out")
	if err := copyTree(src, dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}

	// Hierarchy and relative paths are preserved.
	if data, _ := os.ReadFile(filepath.Join(dst, "a", "b", "c", "deep.txt")); string(data) != "deep" {
		t.Errorf("nested file content = %q, want deep", data)
	}
	if data, _ := os.ReadFile(filepath.Join(dst, "a", "top.txt")); string(data) != "top" {
		t.Errorf("sibling file content = %q, want top", data)
	}
	// Intermediate levels are real directories, not flattened.
	for _, rel := range []string{"a", "a/b", "a/b/c"} {
		fi, err := os.Lstat(filepath.Join(dst, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("Lstat %s: %v", rel, err)
		}
		if !fi.IsDir() {
			t.Errorf("%s should be a directory", rel)
		}
	}
}

func TestCopyTreeDirModeOwnerWrite(t *testing.T) {
	src := realTempDir(t)
	roDir := filepath.Join(src, "tree", "ro")
	if err := os.MkdirAll(roDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(roDir, "f.txt"), []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	copyTree_chmodRestore(t, filepath.Join(roDir, "f.txt"))
	// Strip owner-write from the src dir (store dirs are read-only 0555).
	if err := os.Chmod(roDir, 0o555); err != nil {
		t.Fatal(err)
	}
	copyTree_chmodRestore(t, roDir)

	dst := filepath.Join(realTempDir(t), "out")
	if err := copyTree(filepath.Join(src, "tree"), dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}

	// copy.go:117 adds owner-write to the dir mode (0555 → 0755).
	fi, err := os.Lstat(filepath.Join(dst, "ro"))
	if err != nil {
		t.Fatalf("Lstat ro: %v", err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("dir mode = %o, want 0755 (mode-preserve + owner-write)", fi.Mode().Perm())
	}
	if fi.Mode().Perm()&0o200 == 0 {
		t.Error("dir should have owner-write bit set")
	}
}

func TestCopyTreeEmptyDir(t *testing.T) {
	// A standalone empty directory is duplicated as an empty directory.
	src := filepath.Join(realTempDir(t), "empty")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(realTempDir(t), "out")
	if err := copyTree(src, dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("Lstat dst: %v", err)
	}
	if !fi.IsDir() {
		t.Fatal("dst should be a directory")
	}
	entries, err := os.ReadDir(dst)
	if err != nil {
		t.Fatalf("ReadDir dst: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("empty dir copy should be empty, got %d entries", len(entries))
	}
}

func TestCopyTreeNestedEmptyDir(t *testing.T) {
	// An empty subdirectory inside a tree is reproduced (WalkDir MkdirAll path).
	src := realTempDir(t)
	if err := os.MkdirAll(filepath.Join(src, "tree", "emptysub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "tree", "keep.txt"), []byte("k"), 0o444); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(realTempDir(t), "out")
	if err := copyTree(filepath.Join(src, "tree"), dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}

	fi, err := os.Lstat(filepath.Join(dst, "emptysub"))
	if err != nil {
		t.Fatalf("Lstat emptysub: %v", err)
	}
	if !fi.IsDir() {
		t.Error("emptysub should be a directory")
	}
	entries, err := os.ReadDir(filepath.Join(dst, "emptysub"))
	if err != nil {
		t.Fatalf("ReadDir emptysub: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("nested empty dir should stay empty, got %d entries", len(entries))
	}
}

func TestCopyTreeInternalSymlinkNoDeref(t *testing.T) {
	src := realTempDir(t)
	if err := os.MkdirAll(filepath.Join(src, "tree", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "tree", "sub", "f.txt"), []byte("hi"), 0o444); err != nil {
		t.Fatal(err)
	}
	// In-tree relative symlink that must be duplicated as a symlink, not deref'd.
	if err := os.Symlink("sub/f.txt", filepath.Join(src, "tree", "rel")); err != nil {
		t.Fatal(err)
	}
	// Dangling symlink (target absent) must still be duplicated verbatim.
	if err := os.Symlink("nowhere", filepath.Join(src, "tree", "dangling")); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(realTempDir(t), "out")
	if err := copyTree(filepath.Join(src, "tree"), dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}

	relLink := filepath.Join(dst, "rel")
	li, err := os.Lstat(relLink)
	if err != nil {
		t.Fatalf("Lstat rel: %v", err)
	}
	if li.Mode()&os.ModeSymlink == 0 {
		t.Error("in-tree symlink should remain a symlink (no deref)")
	}
	if target, _ := os.Readlink(relLink); target != "sub/f.txt" {
		t.Errorf("rel symlink target = %q, want sub/f.txt", target)
	}

	dl := filepath.Join(dst, "dangling")
	if di, err := os.Lstat(dl); err != nil {
		t.Fatalf("Lstat dangling: %v", err)
	} else if di.Mode()&os.ModeSymlink == 0 {
		t.Error("dangling symlink should remain a symlink")
	}
	if target, _ := os.Readlink(dl); target != "nowhere" {
		t.Errorf("dangling symlink target = %q, want nowhere", target)
	}
}

func TestCopyTreeTopLevelSymlink(t *testing.T) {
	// copyTree dispatches a top-level symlink src straight to copySymlink.
	dir := realTempDir(t)
	link := filepath.Join(dir, "link")
	if err := os.Symlink("some/target", link); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(realTempDir(t), "out")
	if err := copyTree(link, dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}
	li, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("Lstat dst: %v", err)
	}
	if li.Mode()&os.ModeSymlink == 0 {
		t.Error("top-level symlink should be copied as a symlink")
	}
	if target, _ := os.Readlink(dst); target != "some/target" {
		t.Errorf("symlink target = %q, want some/target", target)
	}
}

func TestCopyFilePreservesModeAddsOwnerWrite(t *testing.T) {
	src := realTempDir(t)
	srcFile := filepath.Join(src, "f.txt")
	if err := os.WriteFile(srcFile, []byte("content"), 0o444); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(realTempDir(t), "out.txt")
	if err := copyFile(srcFile, dst, 0o444); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	if data, _ := os.ReadFile(dst); string(data) != "content" {
		t.Errorf("content = %q, want content", data)
	}
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("Lstat dst: %v", err)
	}
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("mode = %o, want 0644 (mode-preserve + owner-write)", fi.Mode().Perm())
	}
}

func TestCopyFileExecBitPreserved(t *testing.T) {
	// 0555 (read+exec, no owner-write) → 0755 keeps the exec bits.
	src := realTempDir(t)
	srcFile := filepath.Join(src, "run.sh")
	if err := os.WriteFile(srcFile, []byte("#!/bin/sh\n"), 0o555); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(realTempDir(t), "run.sh")
	if err := copyFile(srcFile, dst, 0o555); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("Lstat dst: %v", err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("mode = %o, want 0755 (exec preserved + owner-write)", fi.Mode().Perm())
	}
}

func TestCopySymlinkDuplicatesWithoutDeref(t *testing.T) {
	dir := realTempDir(t)
	src := filepath.Join(dir, "link")
	if err := os.Symlink("the/target", src); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(realTempDir(t), "copied")
	if err := copySymlink(src, dst); err != nil {
		t.Fatalf("copySymlink: %v", err)
	}
	li, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("Lstat dst: %v", err)
	}
	if li.Mode()&os.ModeSymlink == 0 {
		t.Error("dst should be a symlink")
	}
	if target, _ := os.Readlink(dst); target != "the/target" {
		t.Errorf("target = %q, want the/target", target)
	}
}

func TestCopySymlinkOverwritesExistingDst(t *testing.T) {
	// copy.go:155 defensively removes a leftover dst before re-linking.
	dir := realTempDir(t)
	src := filepath.Join(dir, "link")
	if err := os.Symlink("new/target", src); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(realTempDir(t), "copied")
	if err := os.WriteFile(dst, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copySymlink(src, dst); err != nil {
		t.Fatalf("copySymlink: %v", err)
	}
	li, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("Lstat dst: %v", err)
	}
	if li.Mode()&os.ModeSymlink == 0 {
		t.Error("existing dst should be replaced by a symlink")
	}
	if target, _ := os.Readlink(dst); target != "new/target" {
		t.Errorf("target = %q, want new/target", target)
	}
}
