package engine

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/lock"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/paths"
)

// profileDirFor returns the profileDir for the --root override (roothash key) case.
func profileDirFor(t *testing.T, state, root, name string) string {
	t.Helper()
	return paths.Resolve(state, name, manifest.RootKindProject, root, true).Dir
}

// --- test helpers ----------------------------------------------------------

// realTempDir returns an EvalSymlinks'd tmpdir (works around macOS /var → /private/var).
func realTempDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// writeLinkFarm creates and returns a link-farm directory with a hand-written manifest.json.
func writeLinkFarm(t *testing.T, m manifest.Manifest) string {
	t.Helper()
	dir := realTempDir(t)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, manifest.FileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// makeSrc creates a file at <subpath> in a store-src-equivalent tmpdir and returns the src dir.
func makeSrc(t *testing.T, subpath string) string {
	t.Helper()
	src := realTempDir(t)
	full := filepath.Join(src, subpath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	return src
}

// fakeCommit mimics nix-env --set and links the profile link to the link-farm
// (so that reading the previous generation's manifest works on a subsequent apply).
func fakeCommit(captured *[][2]string) CommitFunc {
	return func(profileLink, linkFarm string) error {
		if captured != nil {
			*captured = append(*captured, [2]string{profileLink, linkFarm})
		}
		_ = os.Remove(profileLink)
		return os.Symlink(linkFarm, profileLink)
	}
}

func collectWarnings(buf *[]string) func(string, ...any) {
	return func(format string, args ...any) {
		*buf = append(*buf, format)
	}
}

func projectManifest(entries ...manifest.Entry) manifest.Manifest {
	return manifest.Manifest{
		SchemaVersion: 1,
		Root:          manifest.Root{RootKind: manifest.RootKindProject},
		Entries:       entries,
	}
}

func storeEntry(src, subpath, target string) manifest.Entry {
	return manifest.Entry{
		SrcKind: manifest.SrcKindStore,
		Src:     src,
		Subpath: subpath,
		Target:  target,
		Method:  manifest.MethodSymlink,
	}
}

// outOfStoreEntry is an out-of-store symlink entry pointing at the marker's local absolute path (outside the store).
func outOfStoreEntry(src, subpath, target string) manifest.Entry {
	return manifest.Entry{
		SrcKind: manifest.SrcKindOutOfStore,
		Src:     src,
		Subpath: subpath,
		Target:  target,
		Method:  manifest.MethodSymlink,
	}
}

func copyEntry(src, subpath, target string) manifest.Entry {
	e := storeEntry(src, subpath, target)
	e.Method = manifest.MethodCopy
	return e
}

// makeROSrc creates a store-equivalent read-only file (0o444) at <subpath> and returns the src dir.
// A src for verifying copy's mode preservation + owner-write addition (0444 → 0644).
func makeROSrc(t *testing.T, subpath, content string) string {
	t.Helper()
	src := realTempDir(t)
	full := filepath.Join(src, subpath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o444); err != nil {
		t.Fatal(err)
	}
	return src
}

// --- tests -----------------------------------------------------------------

func TestApplyFirstPlacementProjectMode(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := realTempDir(t)
	if out, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	src := makeSrc(t, "skills/nix/init.lua")
	state := realTempDir(t)
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, "skills/nix", ".claude/skills/nix")))

	var commits [][2]string
	res, err := Apply(Options{
		LinkFarm: lf,
		Name:     "skills",
		WorkDir:  root, // project mode: git toplevel = root
		StateDir: state,
		Commit:   fakeCommit(&commits),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// symlink is created as <root>/<target> -> <src>/<subpath>.
	target := filepath.Join(root, ".claude", "skills", "nix")
	dest, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink target: %v", err)
	}
	if want := filepath.Join(src, "skills/nix"); dest != want {
		t.Errorf("symlink dest = %q, want %q", dest, want)
	}

	// The profileDir layout is created at <state>/nix/profiles/nput/<roothash>/<name>.
	if !strings.HasPrefix(res.ProfileDir, filepath.Join(state, "nix", "profiles", "nput")) {
		t.Errorf("ProfileDir = %q, not under state base", res.ProfileDir)
	}
	if fi, err := os.Stat(res.ProfileDir); err != nil || !fi.IsDir() {
		t.Errorf("profileDir not created: %v", err)
	}

	// backref .root records root's absolute path at the <roothash> level.
	backref := filepath.Join(filepath.Dir(res.ProfileDir), ".root")
	data, err := os.ReadFile(backref)
	if err != nil {
		t.Fatalf("read backref: %v", err)
	}
	if strings.TrimSpace(string(data)) != root {
		t.Errorf("backref = %q, want %q", strings.TrimSpace(string(data)), root)
	}

	// commit is called once with profileDir/profile and the link-farm.
	if len(commits) != 1 {
		t.Fatalf("commit calls = %d, want 1", len(commits))
	}
	if commits[0][0] != filepath.Join(res.ProfileDir, "profile") || commits[0][1] != lf {
		t.Errorf("commit args = %v", commits[0])
	}
	if len(res.Placed) != 1 || res.Placed[0] != ".claude/skills/nix" {
		t.Errorf("Placed = %v", res.Placed)
	}
}

func TestApplySubpathDot(t *testing.T) {
	root := realTempDir(t)
	src := makeSrc(t, "file.txt")
	state := realTempDir(t)
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/whole")))

	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	dest, err := os.Readlink(filepath.Join(root, ".config", "whole"))
	if err != nil {
		t.Fatal(err)
	}
	if dest != src { // subpath="." → src itself
		t.Errorf("dest = %q, want %q", dest, src)
	}
}

func TestApplyAncestorSymlinkError(t *testing.T) {
	root := realTempDir(t)
	src := makeSrc(t, "x")
	state := realTempDir(t)

	// Make <root>/.claude an already-placed symlink → cannot nest nix beneath it.
	if err := os.Symlink(realTempDir(t), filepath.Join(root, ".claude")); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".claude/skills/nix")))

	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected ancestor symlink error, got %v", err)
	}
}

func TestApplyExistingRegularFileError(t *testing.T) {
	root := realTempDir(t)
	src := makeSrc(t, "x")
	state := realTempDir(t)

	if err := os.MkdirAll(filepath.Join(root, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".config", "foo"), []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil {
		t.Fatal("expected error for existing regular file, got nil")
	}
}

func TestApplyForeignSymlinkWarns(t *testing.T) {
	root := realTempDir(t)
	src := makeSrc(t, "x")
	state := realTempDir(t)

	// Place an unrecorded foreign symlink at the target.
	foreign := realTempDir(t)
	if err := os.MkdirAll(filepath.Join(root, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, filepath.Join(root, ".config", "foo")); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	var warns []string
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state,
		Commit: fakeCommit(nil), Warnf: collectWarnings(&warns),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(warns) == 0 {
		t.Error("expected a foreign symlink warning")
	}
	if len(res.Replaced) != 1 {
		t.Errorf("Replaced = %v, want 1", res.Replaced)
	}
	dest, _ := os.Readlink(filepath.Join(root, ".config", "foo"))
	if dest != src {
		t.Errorf("dest = %q, want %q (last-wins)", dest, src)
	}
}

func TestApplyRecordedSymlinkReplacedSilently(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src1 := makeSrc(t, "x")

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(src1, ".", ".config/foo")))
	var commits [][2]string
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// Second apply: same target, different src. Matches the previous generation's record, so re-link without a foreign warning.
	src2 := makeSrc(t, "x")
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(src2, ".", ".config/foo")))
	var warns []string
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state,
		Commit: fakeCommit(&commits), Warnf: collectWarnings(&warns),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	for _, w := range warns {
		if strings.Contains(w, "foreign") {
			t.Errorf("unexpected foreign warning on recorded replace: %q", w)
		}
	}
	if len(res.Replaced) != 1 {
		t.Errorf("Replaced = %v, want 1", res.Replaced)
	}
	dest, _ := os.Readlink(filepath.Join(root, ".config", "foo"))
	if dest != src2 {
		t.Errorf("dest = %q, want %q", dest, src2)
	}
}

func TestApplyStaleRemoval(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	// First apply: 2 entries.
	lf1 := writeLinkFarm(t, projectManifest(
		storeEntry(src, ".", ".config/keep"),
		storeEntry(src, ".", ".config/drop"),
	))
	var commits [][2]string
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".config", "drop")); err != nil {
		t.Fatalf("drop should exist after first apply: %v", err)
	}

	// Second apply: drop drop → stale-removed. keep remains.
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/keep")))
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".config", "drop")); !os.IsNotExist(err) {
		t.Errorf("drop should be removed, lstat err = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".config", "keep")); err != nil {
		t.Errorf("keep should remain: %v", err)
	}
	if len(res.Removed) != 1 || res.Removed[0] != ".config/drop" {
		t.Errorf("Removed = %v, want [.config/drop]", res.Removed)
	}
}

func TestApplyStaleRemovalKeepsForeign(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/drop")))
	var commits [][2]string
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// User swaps the target to point elsewhere (mismatch with the record) → no stale removal.
	tgt := filepath.Join(root, ".config", "drop")
	_ = os.Remove(tgt)
	if err := os.Symlink(realTempDir(t), tgt); err != nil {
		t.Fatal(err)
	}

	lf2 := writeLinkFarm(t, projectManifest()) // empty manifest = clear everything.
	var warns []string
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state,
		Commit: fakeCommit(&commits), Warnf: collectWarnings(&warns),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if len(res.Removed) != 0 {
		t.Errorf("Removed = %v, want none (mismatch kept)", res.Removed)
	}
	if _, err := os.Lstat(tgt); err != nil {
		t.Errorf("mismatched symlink should be kept: %v", err)
	}
	if len(warns) == 0 {
		t.Error("expected a mismatch warning")
	}
}

func TestApplyGitNotInRepoError(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	work := realTempDir(t)
	t.Setenv("HOME", work)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(work))
	src := makeSrc(t, "x")
	state := realTempDir(t)
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", WorkDir: work, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil {
		t.Fatal("expected git-not-in-repo error, got nil")
	}
}

func TestApplyNoWaitSkipsWhenLocked(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	// Create profileDir first and hold the flock.
	prof := profileDirFor(t, state, root, "c")
	if err := os.MkdirAll(prof, 0o755); err != nil {
		t.Fatal(err)
	}
	held, err := lock.Acquire(prof, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Release() }()

	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, NoWait: true, Commit: fakeCommit(nil),
	})
	if err != ErrSkipped {
		t.Fatalf("NoWait while locked: err = %v, want ErrSkipped", err)
	}
	if res == nil || !res.Skipped {
		t.Errorf("expected res.Skipped = true, got %+v", res)
	}
	// Skipped, so nothing is placed.
	if _, err := os.Lstat(filepath.Join(root, ".config", "foo")); !os.IsNotExist(err) {
		t.Errorf("skip should not place: lstat err = %v", err)
	}
}

func TestApplyOutOfStorePlacesLiveSymlink(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	// Use a local entity outside the store as the out-of-store link target.
	src := makeSrc(t, "lua/init.lua")
	lf := writeLinkFarm(t, projectManifest(outOfStoreEntry(src, "lua", ".config/nvim")))

	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	target := filepath.Join(root, ".config", "nvim")
	dest, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink target: %v", err)
	}
	// The live symlink points directly at the marker's absolute path (<src>/<subpath>).
	if want := filepath.Join(src, "lua"); dest != want {
		t.Errorf("symlink dest = %q, want %q (local abs path)", dest, want)
	}
	// Resolves to a live entity, not dangling.
	if _, err := os.Stat(target); err != nil {
		t.Errorf("symlink should resolve to a live path: %v", err)
	}
	if len(res.Placed) != 1 || res.Placed[0] != ".config/nvim" {
		t.Errorf("Placed = %v", res.Placed)
	}
}

func TestApplyOutOfStoreStaleRemoval(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	// First apply: place the out-of-store entry (the marker's absolute path is recorded in the manifest).
	lf1 := writeLinkFarm(t, projectManifest(outOfStoreEntry(src, ".", ".config/nvim")))
	var commits [][2]string
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	tgt := filepath.Join(root, ".config", "nvim")
	if dest, _ := os.Readlink(tgt); dest != src {
		t.Fatalf("first apply dest = %q, want %q", dest, src)
	}

	// Second apply: empty manifest. The link points at the recorded out-of-store path, so it satisfies the conservative invariant and is removed.
	lf2 := writeLinkFarm(t, projectManifest())
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if _, err := os.Lstat(tgt); !os.IsNotExist(err) {
		t.Errorf("recorded out-of-store link should be removed: lstat err = %v", err)
	}
	if len(res.Removed) != 1 || res.Removed[0] != ".config/nvim" {
		t.Errorf("Removed = %v, want [.config/nvim]", res.Removed)
	}
}

func TestApplyOutOfStoreStaleKeepsMismatch(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")

	lf1 := writeLinkFarm(t, projectManifest(outOfStoreEntry(src, ".", ".config/nvim")))
	var commits [][2]string
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// User swaps the target to a different local destination (mismatch with the record) → no stale removal.
	tgt := filepath.Join(root, ".config", "nvim")
	_ = os.Remove(tgt)
	if err := os.Symlink(realTempDir(t), tgt); err != nil {
		t.Fatal(err)
	}

	lf2 := writeLinkFarm(t, projectManifest())
	var warns []string
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state,
		Commit: fakeCommit(&commits), Warnf: collectWarnings(&warns),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if len(res.Removed) != 0 {
		t.Errorf("Removed = %v, want none (mismatch kept)", res.Removed)
	}
	if _, err := os.Lstat(tgt); err != nil {
		t.Errorf("mismatched out-of-store link should be kept: %v", err)
	}
	if len(warns) == 0 {
		t.Error("expected a mismatch warning")
	}
}

func TestApplyOutOfStoreMissingPathError(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	// Use a non-existent local path as the out-of-store link target.
	missing := filepath.Join(realTempDir(t), "does-not-exist")
	lf := writeLinkFarm(t, projectManifest(outOfStoreEntry(missing, ".", ".config/nvim")))

	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil {
		t.Fatal("expected error for missing out-of-store path, got nil")
	}
	if !strings.Contains(err.Error(), "out-of-store") {
		t.Errorf("error should mention out-of-store: %v", err)
	}
	// The absence check is closed before placement, so the target is not created.
	if _, err := os.Lstat(filepath.Join(root, ".config", "nvim")); !os.IsNotExist(err) {
		t.Errorf("should not place on missing path: lstat err = %v", err)
	}
}

func TestApplyCopyPlaceOnceFile(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeROSrc(t, "file.txt", "store-content")
	lf := writeLinkFarm(t, projectManifest(copyEntry(src, "file.txt", ".config/foo")))

	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	target := filepath.Join(root, ".config", "foo")
	// Content is copied (a regular file, not a symlink).
	fi, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat target: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Errorf("copy target should be a regular file, got symlink")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "store-content" {
		t.Errorf("content = %q (err %v), want store-content", data, err)
	}
	// Preserve mode while adding owner-write (0444 → 0644).
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("mode = %o, want 0644 (mode-preserve + owner-write)", fi.Mode().Perm())
	}
	if len(res.Copied) != 1 || res.Copied[0] != ".config/foo" {
		t.Errorf("Copied = %v, want [.config/foo]", res.Copied)
	}
}

func TestApplyCopyPlaceOnceKeepsExisting(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeROSrc(t, "file.txt", "store-content")
	lf := writeLinkFarm(t, projectManifest(copyEntry(src, "file.txt", ".config/foo")))

	// First apply: new copy.
	var commits [][2]string
	if _, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	// User edits it.
	target := filepath.Join(root, ".config", "foo")
	if err := os.WriteFile(target, []byte("user-edit"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second apply: place-once leaves the target untouched (the edit remains).
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
	})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	data, _ := os.ReadFile(target)
	if string(data) != "user-edit" {
		t.Errorf("place-once should keep user edit, content = %q", data)
	}
	if len(res.Copied) != 0 {
		t.Errorf("Copied = %v, want none (recorded place-once no-op)", res.Copied)
	}
}

func TestApplyCopyForeignFileSkipsWithWarn(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeROSrc(t, "file.txt", "store-content")

	// Place an unrecorded foreign regular file at the target.
	target := filepath.Join(root, ".config", "foo")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, projectManifest(copyEntry(src, "file.txt", ".config/foo")))

	var warns []string
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state,
		Commit: fakeCommit(nil), Warnf: collectWarnings(&warns),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// place-once skip: the foreign file is not overwritten.
	if data, _ := os.ReadFile(target); string(data) != "foreign" {
		t.Errorf("foreign file should be kept, content = %q", data)
	}
	if len(res.Copied) != 0 {
		t.Errorf("Copied = %v, want none (foreign skipped)", res.Copied)
	}
	if len(warns) == 0 {
		t.Error("expected a copy foreign warning")
	}
}

func TestApplyCopyDirRecursivePreservesSymlinks(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)

	// src tree: directory + file + internal symlink.
	src := realTempDir(t)
	if err := os.MkdirAll(filepath.Join(src, "tree", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "tree", "sub", "f.txt"), []byte("hi"), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("sub/f.txt", filepath.Join(src, "tree", "link")); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, projectManifest(copyEntry(src, "tree", ".config/tree")))

	if _, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	dst := filepath.Join(root, ".config", "tree")
	if data, _ := os.ReadFile(filepath.Join(dst, "sub", "f.txt")); string(data) != "hi" {
		t.Errorf("nested file content = %q, want hi", data)
	}
	// The internal symlink is duplicated as a symlink without deref.
	li, err := os.Lstat(filepath.Join(dst, "link"))
	if err != nil {
		t.Fatalf("Lstat link: %v", err)
	}
	if li.Mode()&os.ModeSymlink == 0 {
		t.Error("internal symlink should remain a symlink (no deref)")
	}
	if dest, _ := os.Readlink(filepath.Join(dst, "link")); dest != "sub/f.txt" {
		t.Errorf("symlink dest = %q, want sub/f.txt", dest)
	}
}

func TestApplyRecopyOverwrites(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeROSrc(t, "file.txt", "store-content")
	lf := writeLinkFarm(t, projectManifest(copyEntry(src, "file.txt", ".config/foo")))

	// First apply: new copy.
	var commits [][2]string
	if _, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(&commits),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	target := filepath.Join(root, ".config", "foo")
	if err := os.WriteFile(target, []byte("user-edit"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second apply: --recopy overwrites unconditionally → reverts to src content.
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state,
		Recopy: true, Commit: fakeCommit(&commits),
	})
	if err != nil {
		t.Fatalf("recopy Apply: %v", err)
	}
	if data, _ := os.ReadFile(target); string(data) != "store-content" {
		t.Errorf("recopy should restore src content, got %q", data)
	}
	if len(res.Recopied) != 1 || res.Recopied[0] != ".config/foo" {
		t.Errorf("Recopied = %v, want [.config/foo]", res.Recopied)
	}
}

func TestApplyRecopyForeignFileOverwrites(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeROSrc(t, "file.txt", "store-content")

	// Unrecorded foreign file: normal apply would skip, but --recopy overwrites unconditionally.
	target := filepath.Join(root, ".config", "foo")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, projectManifest(copyEntry(src, "file.txt", ".config/foo")))

	var warns []string
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state,
		Recopy: true, Commit: fakeCommit(nil), Warnf: collectWarnings(&warns),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if data, _ := os.ReadFile(target); string(data) != "store-content" {
		t.Errorf("recopy should overwrite foreign file, got %q", data)
	}
	if len(res.Recopied) != 1 {
		t.Errorf("Recopied = %v, want 1", res.Recopied)
	}
	// On the recopy path no foreign skip warning is emitted (it overwrites, so that would be a false report).
	for _, w := range warns {
		if strings.Contains(w, "skipped copy") {
			t.Errorf("unexpected copy foreign skip warning during recopy: %q", w)
		}
	}
}

// --- error-path coverage (engine.go:262-264, 359, 369-382) ------------------
// These exercise the under-covered failure branches of resolveRoot (fixed-root
// Abs), ensureProfileDir (mkdir / backref write), and cleanupPending (warn-only
// Remove). Failures are induced by file-type conflicts (ENOTDIR / EISDIR /
// ENOTEMPTY), not permission bits, so they reproduce under root as well and need
// no os.Geteuid()==0 skip guard.

// TestResolveRootFixedAbsFailure covers engine.go:359 (filepath.Abs(fixedRoot)).
// filepath.Abs only errors when the path is relative and os.Getwd fails, which is
// inherently environment-dependent: we induce it by chdir'ing into a directory and
// removing it out from under the process (cwd no longer resolves → Getwd errors).
// On platforms where a removed cwd still resolves, the branch cannot be reached and
// the test skips with that reason recorded rather than silently passing.
func TestResolveRootFixedAbsFailure(t *testing.T) {
	gone := filepath.Join(realTempDir(t), "gone")
	if err := os.Mkdir(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	// t.Chdir restores the original cwd at cleanup regardless of removal.
	t.Chdir(gone)
	if err := os.Remove(gone); err != nil {
		t.Skipf("cannot remove cwd to induce Getwd failure: %v", err)
	}
	if _, err := os.Getwd(); err == nil {
		t.Skip("removed cwd still resolves on this platform; Abs failure branch unreachable")
	}

	// rootKind=fixed with a relative fixedRoot routes into filepath.Abs, which now
	// fails because Getwd fails. resolveRoot is unexported but reachable in-package.
	_, err := resolveRoot(manifest.RootKindFixed, "relative/path", "", "", nil)
	if err == nil {
		t.Fatal("expected Abs failure for relative fixed root with broken cwd, got nil")
	}
}

// TestEnsureProfileDirMkdirFailure covers engine.go:370-372 (MkdirAll(profile.Dir)).
// A regular file planted at <state>/nix forces MkdirAll under it to fail with ENOTDIR.
func TestEnsureProfileDirMkdirFailure(t *testing.T) {
	root := realTempDir(t)
	src := makeSrc(t, "x")
	state := realTempDir(t)
	// <state>/nix is a file, so <state>/nix/profiles/... cannot be created.
	if err := os.WriteFile(filepath.Join(state, "nix"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil {
		t.Fatal("expected profileDir mkdir failure, got nil")
	}
	if !strings.Contains(err.Error(), "cannot create profileDir") {
		t.Errorf("error = %v, want mention of profileDir creation", err)
	}
}

// TestEnsureProfileDirBackrefWriteFailure covers engine.go:378-380 (WriteFile backref).
// The backref path <roothash>/.root is pre-created as a directory so WriteFile fails
// with EISDIR while the two preceding MkdirAll calls still succeed.
func TestEnsureProfileDirBackrefWriteFailure(t *testing.T) {
	root := realTempDir(t)
	src := makeSrc(t, "x")
	state := realTempDir(t)

	// Resolve the same layout Apply will compute (RootOverride → roothash key, abs root == root).
	prof := paths.Resolve(state, "c", manifest.RootKindProject, root, true)
	if prof.Backref == "" {
		t.Fatal("expected a backref path for the roothash-keyed layout")
	}
	// Plant a directory where the backref file should be written.
	if err := os.MkdirAll(prof.Backref, 0o755); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil {
		t.Fatal("expected backref write failure, got nil")
	}
	if !strings.Contains(err.Error(), "cannot write backref") {
		t.Errorf("error = %v, want mention of backref write", err)
	}
}

// TestApplyCleanupPendingRemoveFailureWarns covers engine.go:262-264 (warn-only Remove).
// cleanupPending runs only on the Build path. The injected Build leaves a non-empty
// directory at the .pending path, so os.Remove fails with ENOTEMPTY; the failure must
// be surfaced as a warning only and must not fail Apply or undo the placement.
func TestApplyCleanupPendingRemoveFailureWarns(t *testing.T) {
	root := realTempDir(t)
	src := makeSrc(t, "x")
	state := realTempDir(t)
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	build := func(pending string) (string, error) {
		// A non-empty directory at the pending out-link makes os.Remove fail (ENOTEMPTY).
		if err := os.MkdirAll(filepath.Join(pending, "child"), 0o755); err != nil {
			return "", err
		}
		return lf, nil
	}

	var warns []string
	res, err := Apply(Options{
		Name: "c", RootOverride: root, StateDir: state,
		RootKind: manifest.RootKindProject, Build: build,
		Commit: fakeCommit(nil), Warnf: collectWarnings(&warns),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// Placement is unaffected by the pending cleanup failure.
	if len(res.Placed) != 1 || res.Placed[0] != ".config/foo" {
		t.Errorf("Placed = %v, want [.config/foo]", res.Placed)
	}
	if _, err := os.Readlink(filepath.Join(root, ".config", "foo")); err != nil {
		t.Errorf("symlink should be placed despite pending cleanup failure: %v", err)
	}
	found := false
	for _, w := range warns {
		if strings.Contains(w, "could not remove the .pending out-link") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a .pending cleanup warning, warns = %v", warns)
	}
}
