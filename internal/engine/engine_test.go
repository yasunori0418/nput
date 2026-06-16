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

// profileDirFor は --root 上書き（roothash キー）時の profileDir を返す。
func profileDirFor(t *testing.T, state, root, name string) string {
	t.Helper()
	return paths.Resolve(state, name, manifest.RootKindProject, root, true).Dir
}

// --- test helpers ----------------------------------------------------------

// realTempDir は EvalSymlinks 済みの tmpdir を返す（macOS の /var → /private/var 対策）。
func realTempDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// writeLinkFarm は手書き manifest.json を持つ link-farm ディレクトリを作って返す。
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

// makeSrc は store src 相当の tmpdir に <subpath> のファイルを作って src dir を返す。
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

// fakeCommit は nix-env --set を模し、profile リンクを link-farm へ張る
// （前世代 manifest の読み取りを後続 apply で成立させる）。
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

// outOfStoreEntry は marker のローカル絶対パス（store 外）を指す out-of-store symlink entry。
func outOfStoreEntry(src, subpath, target string) manifest.Entry {
	return manifest.Entry{
		SrcKind: manifest.SrcKindOutOfStore,
		Src:     src,
		Subpath: subpath,
		Target:  target,
		Method:  manifest.MethodSymlink,
	}
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

	// symlink が <root>/<target> -> <src>/<subpath> で張られる。
	target := filepath.Join(root, ".claude", "skills", "nix")
	dest, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink target: %v", err)
	}
	if want := filepath.Join(src, "skills/nix"); dest != want {
		t.Errorf("symlink dest = %q, want %q", dest, want)
	}

	// profileDir レイアウトが <state>/nix/profiles/nput/<roothash>/<name> に作られる。
	if !strings.HasPrefix(res.ProfileDir, filepath.Join(state, "nix", "profiles", "nput")) {
		t.Errorf("ProfileDir = %q, not under state base", res.ProfileDir)
	}
	if fi, err := os.Stat(res.ProfileDir); err != nil || !fi.IsDir() {
		t.Errorf("profileDir not created: %v", err)
	}

	// backref .root が <roothash> 階層に root の絶対パスを記録する。
	backref := filepath.Join(filepath.Dir(res.ProfileDir), ".root")
	data, err := os.ReadFile(backref)
	if err != nil {
		t.Fatalf("read backref: %v", err)
	}
	if strings.TrimSpace(string(data)) != root {
		t.Errorf("backref = %q, want %q", strings.TrimSpace(string(data)), root)
	}

	// commit が profileDir/profile と link-farm で 1 回呼ばれる。
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
	if dest != src { // subpath="." → src そのもの
		t.Errorf("dest = %q, want %q", dest, src)
	}
}

func TestApplyAncestorSymlinkError(t *testing.T) {
	root := realTempDir(t)
	src := makeSrc(t, "x")
	state := realTempDir(t)

	// <root>/.claude を symlink 配置済みにする → 配下に nix をネストできない。
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

	// 記録の無い foreign symlink を target に置く。
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

	// 2 回目: 同 target・別 src。前世代記録と一致するので foreign 警告を出さず張替。
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

	// 1 回目: 2 entry。
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

	// 2 回目: drop を外す → stale 除去される。keep は残る。
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

	// ユーザーが target を別の先へ差し替える（記録と不一致）→ stale 除去しない。
	tgt := filepath.Join(root, ".config", "drop")
	_ = os.Remove(tgt)
	if err := os.Symlink(realTempDir(t), tgt); err != nil {
		t.Fatal(err)
	}

	lf2 := writeLinkFarm(t, projectManifest()) // 空 manifest = 全クリア。
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

	// profileDir を先に作って flock を握っておく。
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
	// skip したので配置はされない。
	if _, err := os.Lstat(filepath.Join(root, ".config", "foo")); !os.IsNotExist(err) {
		t.Errorf("skip should not place: lstat err = %v", err)
	}
}

func TestApplyOutOfStorePlacesLiveSymlink(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	// store 外のローカル実体を out-of-store のリンク先に使う。
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
	// live symlink は marker の絶対パス（<src>/<subpath>）を直接指す。
	if want := filepath.Join(src, "lua"); dest != want {
		t.Errorf("symlink dest = %q, want %q (local abs path)", dest, want)
	}
	// dangling でなく実体に解決できる。
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

	// 1 回目: out-of-store entry を配置（manifest に marker の絶対パスが記録される）。
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

	// 2 回目: 空 manifest。記録された out-of-store パスを指す link なので保守的不変条件を満たし除去。
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

	// ユーザーが target を別のローカル先へ差し替える（記録と不一致）→ stale 除去しない。
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
	// 存在しないローカルパスを out-of-store のリンク先にする。
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
	// 不在検査は配置前に閉じるため target は作られない。
	if _, err := os.Lstat(filepath.Join(root, ".config", "nvim")); !os.IsNotExist(err) {
		t.Errorf("should not place on missing path: lstat err = %v", err)
	}
}

func TestApplyCopyMethodUnimplemented(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")
	e := storeEntry(src, ".", ".config/foo")
	e.Method = manifest.MethodCopy
	lf := writeLinkFarm(t, projectManifest(e))

	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil {
		t.Fatal("expected copy-unimplemented error, got nil")
	}
}
