package planner

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/yasunori0418/nput/internal/manifest"
)

// --- fake FS（純関数 planner を実 FS なしで table-test するための偽 lstat/readlink）---

type fakeEntry struct {
	mode os.FileMode // ModeSymlink / ModeDir / 0(regular) を立てる
	dest string      // symlink のとき readlink が返す先
}

func sym(dest string) fakeEntry { return fakeEntry{mode: os.ModeSymlink, dest: dest} }
func dir() fakeEntry            { return fakeEntry{mode: os.ModeDir} }
func reg() fakeEntry            { return fakeEntry{mode: 0} }

type fakeFS map[string]fakeEntry

func (f fakeFS) Lstat(path string) (os.FileInfo, error) {
	e, ok := f[path]
	if !ok {
		return nil, os.ErrNotExist // os.IsNotExist が true になる
	}
	return fakeInfo{name: filepath.Base(path), mode: e.mode}, nil
}

func (f fakeFS) Readlink(path string) (string, error) {
	e, ok := f[path]
	if !ok || e.mode&os.ModeSymlink == 0 {
		return "", os.ErrInvalid
	}
	return e.dest, nil
}

type fakeInfo struct {
	name string
	mode os.FileMode
}

func (i fakeInfo) Name() string       { return i.name }
func (i fakeInfo) Size() int64        { return 0 }
func (i fakeInfo) Mode() os.FileMode  { return i.mode }
func (i fakeInfo) ModTime() time.Time { return time.Time{} }
func (i fakeInfo) IsDir() bool        { return i.mode&os.ModeDir != 0 }
func (i fakeInfo) Sys() any           { return nil }

// --- manifest helpers -------------------------------------------------------

const root = "/root"

func entry(src, subpath, target, method string) manifest.Entry {
	return manifest.Entry{
		SrcKind: manifest.SrcKindStore,
		Src:     src,
		Subpath: subpath,
		Target:  target,
		Method:  method,
	}
}

func sl(src, target string) manifest.Entry { return entry(src, ".", target, manifest.MethodSymlink) }
func cp(src, target string) manifest.Entry { return entry(src, ".", target, manifest.MethodCopy) }

func mani(entries ...manifest.Entry) *manifest.Manifest {
	return &manifest.Manifest{
		SchemaVersion: 1,
		Root:          manifest.Root{RootKind: manifest.RootKindProject},
		Entries:       entries,
	}
}

func abs(target string) string { return filepath.Join(root, filepath.Clean(target)) }

// --- expectations -----------------------------------------------------------

type want struct {
	placeNew     []string
	placeReplace []string
	placeForeign []string
	remove       []string
	warns        []WarnKind
	conflicts    int
}

func placeTargets(p Plan, kind PlaceKind) []string {
	var out []string
	for _, a := range p.Place {
		if a.Kind == kind {
			out = append(out, a.Entry.Target)
		}
	}
	return out
}

func removeTargets(p Plan) []string {
	var out []string
	for _, a := range p.Remove {
		out = append(out, a.Entry.Target)
	}
	return out
}

func warnKinds(p Plan) []WarnKind {
	var out []WarnKind
	for _, w := range p.Warnings {
		out = append(out, w.Kind)
	}
	return out
}

func sortedEq(t *testing.T, label string, got, want []string) {
	t.Helper()
	g := append([]string(nil), got...)
	w := append([]string(nil), want...)
	sort.Strings(g)
	sort.Strings(w)
	if len(g) != len(w) {
		t.Errorf("%s = %v, want %v", label, got, want)
		return
	}
	for i := range g {
		if g[i] != w[i] {
			t.Errorf("%s = %v, want %v", label, got, want)
			return
		}
	}
}

func warnEq(t *testing.T, got, want []WarnKind) {
	t.Helper()
	g := append([]WarnKind(nil), got...)
	w := append([]WarnKind(nil), want...)
	sort.Slice(g, func(i, j int) bool { return g[i] < g[j] })
	sort.Slice(w, func(i, j int) bool { return w[i] < w[j] })
	if len(g) != len(w) {
		t.Errorf("warnings = %v, want %v", got, want)
		return
	}
	for i := range g {
		if g[i] != w[i] {
			t.Errorf("warnings = %v, want %v", got, want)
			return
		}
	}
}

func TestComputeTableDriven(t *testing.T) {
	const srcA, srcB = "/nix/store/aaa-src", "/nix/store/bbb-src"

	tests := []struct {
		name string
		prev *manifest.Manifest
		next *manifest.Manifest
		fs   fakeFS
		want want
	}{
		{
			// 初回（前世代マニフェストなし）: 何も削除しない・新規配置のみ。
			name: "first apply: prev nil, remove zero",
			prev: nil,
			next: mani(sl(srcA, ".config/foo")),
			fs:   fakeFS{},
			want: want{placeNew: []string{".config/foo"}},
		},
		{
			// 記録あり×記録通り: 自身の前世代 symlink を silent 張替（foreign 警告なし）。
			name: "recorded link → silent replace",
			prev: mani(sl(srcA, ".config/foo")),
			next: mani(sl(srcB, ".config/foo")),
			fs:   fakeFS{abs(".config/foo"): sym(srcA)},
			want: want{placeReplace: []string{".config/foo"}},
		},
		{
			// foreign symlink（記録なし）: warning を出して後勝ち置換。
			name: "foreign symlink → warn + replace",
			prev: nil,
			next: mani(sl(srcB, ".config/foo")),
			fs:   fakeFS{abs(".config/foo"): sym("/somewhere/foreign")},
			want: want{placeForeign: []string{".config/foo"}, warns: []WarnKind{WarnForeignReplace}},
		},
		{
			// 通常ファイルが target に既存: 上書きしない conflict。
			name: "regular file at place target → conflict",
			prev: nil,
			next: mani(sl(srcB, ".config/foo")),
			fs:   fakeFS{abs(".config/foo"): reg()},
			want: want{conflicts: 1},
		},
		{
			// 祖先 component が symlink: 配下にネストできない conflict（→ ADR-0015）。
			name: "ancestor symlink → conflict",
			prev: nil,
			next: mani(sl(srcB, ".claude/skills/nix")),
			fs:   fakeFS{abs(".claude"): sym("/some/store")},
			want: want{conflicts: 1},
		},
		{
			// stale 記録通り: 保守的不変条件を満たすので remove。
			name: "stale recorded link → remove",
			prev: mani(sl(srcA, ".config/keep"), sl(srcA, ".config/drop")),
			next: mani(sl(srcA, ".config/keep")),
			fs: fakeFS{
				abs(".config/keep"): sym(srcA),
				abs(".config/drop"): sym(srcA),
			},
			want: want{
				placeReplace: []string{".config/keep"},
				remove:       []string{".config/drop"},
			},
		},
		{
			// entries = {}（空 manifest）: 前世代の全 nput symlink を保守的に除去（警告なし）。
			name: "empty manifest → remove all recorded (no warning)",
			prev: mani(sl(srcA, "a"), sl(srcA, "b")),
			next: mani(),
			fs: fakeFS{
				abs("a"): sym(srcA),
				abs("b"): sym(srcA),
			},
			want: want{remove: []string{"a", "b"}},
		},
		{
			// stale 記録あり but 実体が別先を指す（不一致）: 削除せず warning。
			name: "stale mismatch (recorded but points elsewhere) → keep + warn",
			prev: mani(sl(srcA, ".config/foo")),
			next: mani(),
			fs:   fakeFS{abs(".config/foo"): sym("/other/place")},
			want: want{warns: []WarnKind{WarnStaleMismatch}},
		},
		{
			// stale target が通常ファイル: nput 非管理として残し warning。
			name: "stale non-symlink (regular file) → keep + warn",
			prev: mani(sl(srcA, ".config/foo")),
			next: mani(),
			fs:   fakeFS{abs(".config/foo"): reg()},
			want: want{warns: []WarnKind{WarnStaleNonSymlink}},
		},
		{
			// stale target が既に無い: no-op（警告なし）。
			name: "stale already gone → no-op",
			prev: mani(sl(srcA, ".config/foo")),
			next: mani(),
			fs:   fakeFS{},
			want: want{},
		},
		{
			// copy entry が消えた: 削除せず orphan を警告（FS 状態に依らない）。
			name: "copy orphan → keep + warn",
			prev: mani(cp(srcA, ".config/foo")),
			next: mani(),
			fs:   fakeFS{abs(".config/foo"): reg()},
			want: want{warns: []WarnKind{WarnCopyOrphan}},
		},
		{
			// 複合: 新規 + silent 張替 + foreign 警告 + stale 除去 + mismatch 残し。
			name: "mixed plan",
			prev: mani(sl(srcA, "keep"), sl(srcA, "drop"), sl(srcA, "mism")),
			next: mani(sl(srcB, "keep"), sl(srcB, "fresh"), sl(srcB, "foreign")),
			fs: fakeFS{
				abs("keep"):    sym(srcA),          // 記録通り → silent replace
				abs("drop"):    sym(srcA),          // stale 記録通り → remove
				abs("mism"):    sym("/elsewhere"),  // stale 不一致 → keep + warn
				abs("foreign"): sym("/foreign/ln"), // 記録なし symlink → foreign warn + replace
				// fresh は不在 → PlaceNew
			},
			want: want{
				placeNew:     []string{"fresh"},
				placeReplace: []string{"keep"},
				placeForeign: []string{"foreign"},
				remove:       []string{"drop"},
				warns:        []WarnKind{WarnForeignReplace, WarnStaleMismatch},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := Compute(tt.prev, tt.next, root, tt.fs)
			if err != nil {
				t.Fatalf("Compute: unexpected error: %v", err)
			}
			sortedEq(t, "placeNew", placeTargets(plan, PlaceNew), tt.want.placeNew)
			sortedEq(t, "placeReplace", placeTargets(plan, PlaceReplace), tt.want.placeReplace)
			sortedEq(t, "placeForeign", placeTargets(plan, PlaceForeign), tt.want.placeForeign)
			sortedEq(t, "remove", removeTargets(plan), tt.want.remove)
			warnEq(t, warnKinds(plan), tt.want.warns)
			if len(plan.Conflicts) != tt.want.conflicts {
				t.Errorf("conflicts = %d, want %d (%v)", len(plan.Conflicts), tt.want.conflicts, plan.Conflicts)
			}
		})
	}
}

// TestComputeCopyInNextErrors は新世代に copy entry があると未実装エラーになることを検証する
// （本スライスは symlink のみ・→ Issue #6）。
func TestComputeCopyInNextErrors(t *testing.T) {
	_, err := Compute(nil, mani(cp("/nix/store/x", ".config/foo")), root, fakeFS{})
	if err == nil {
		t.Fatal("expected error for copy entry in next manifest, got nil")
	}
}

// TestComputeUnknownMethodErrors は未知 method を弾くことを検証する。
func TestComputeUnknownMethodErrors(t *testing.T) {
	e := sl("/nix/store/x", ".config/foo")
	e.Method = "bogus"
	_, err := Compute(nil, mani(e), root, fakeFS{})
	if err == nil {
		t.Fatal("expected error for unknown method, got nil")
	}
}

// TestComputeAncestorDirNotSymlink は祖先が通常ディレクトリなら conflict にならないことを確認する。
func TestComputeAncestorDirNotSymlink(t *testing.T) {
	plan, err := Compute(nil, mani(sl("/nix/store/x", ".config/foo")), root,
		fakeFS{abs(".config"): dir()})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(plan.Conflicts) != 0 {
		t.Errorf("conflicts = %v, want none (ancestor is a dir)", plan.Conflicts)
	}
	if len(plan.Place) != 1 || plan.Place[0].Kind != PlaceNew {
		t.Errorf("Place = %v, want one PlaceNew", plan.Place)
	}
}
