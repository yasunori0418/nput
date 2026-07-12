package planner

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/yasunori0418/nput/internal/manifest"
)

// --- fake FS (fake lstat/readlink to table-test the pure planner without a real FS) ---

type fakeEntry struct {
	mode os.FileMode // set ModeSymlink / ModeDir / 0 (regular)
	dest string      // destination readlink returns when a symlink
}

func sym(dest string) fakeEntry { return fakeEntry{mode: os.ModeSymlink, dest: dest} }
func dir() fakeEntry            { return fakeEntry{mode: os.ModeDir} }
func reg() fakeEntry            { return fakeEntry{mode: 0} }

type fakeFS map[string]fakeEntry

func (f fakeFS) Lstat(path string) (os.FileInfo, error) {
	e, ok := f[path]
	if !ok {
		return nil, os.ErrNotExist // makes os.IsNotExist true
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

// ReadDir lists path's immediate children by scanning the flat fakeFS map for keys one
// path component below path (path itself need not exist as a "dir" entry in the map;
// only its children's presence matters, matching how the table-driven tests populate fs).
func (f fakeFS) ReadDir(path string) ([]os.DirEntry, error) {
	prefix := path + string(os.PathSeparator)
	seen := map[string]fakeEntry{}
	for p, e := range f {
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		rest := p[len(prefix):]
		if idx := strings.IndexRune(rest, os.PathSeparator); idx >= 0 {
			rest = rest[:idx]
		}
		if rest == "" {
			continue
		}
		if _, ok := seen[rest]; !ok {
			seen[rest] = e
		}
	}
	out := make([]os.DirEntry, 0, len(seen))
	for name, e := range seen {
		out = append(out, fakeDirEntry{name: name, mode: e.mode})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
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

type fakeDirEntry struct {
	name string
	mode os.FileMode
}

func (e fakeDirEntry) Name() string      { return e.name }
func (e fakeDirEntry) IsDir() bool       { return e.mode&os.ModeDir != 0 }
func (e fakeDirEntry) Type() os.FileMode { return e.mode.Type() }
func (e fakeDirEntry) Info() (os.FileInfo, error) {
	return fakeInfo(e), nil
}

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
	copies       []string
	remove       []string
	preRemove    []string // RemoveUnlink actions, by Entry.Target
	preRemoveDir []string // RemoveRmdir actions, by root-relative TargetAbs
	warns        []WarnKind
	conflicts    int
	// conflictKinds asserts plan.Conflicts[i].Kind in order (nil = skip; the conflicts count
	// alone does not catch a Kind mix-up such as ForeignAncestor ↔ SelfContradictoryAncestor,
	// both assigned from the same keptInNext branch · → #176).
	conflictKinds []ConflictKind
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

func preRemoveTargets(p Plan) []string {
	var out []string
	for _, a := range p.PreRemove {
		if a.Kind == RemoveUnlink {
			out = append(out, a.Entry.Target)
		}
	}
	return out
}

func preRemoveDirTargets(p Plan) []string {
	var out []string
	for _, a := range p.PreRemove {
		if a.Kind == RemoveRmdir {
			rel, err := filepath.Rel(root, a.TargetAbs)
			if err != nil {
				rel = a.TargetAbs
			}
			out = append(out, rel)
		}
	}
	return out
}

func copyTargets(p Plan) []string {
	var out []string
	for _, a := range p.Copies {
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
			// First apply (no previous-generation manifest): remove nothing, place only.
			name: "first apply: prev nil, remove zero",
			prev: nil,
			next: mani(sl(srcA, ".config/foo")),
			fs:   fakeFS{},
			want: want{placeNew: []string{".config/foo"}},
		},
		{
			// Recorded × matches record: silently re-link this profile's own previous-generation symlink (no foreign warning).
			name: "recorded link → silent replace",
			prev: mani(sl(srcA, ".config/foo")),
			next: mani(sl(srcB, ".config/foo")),
			fs:   fakeFS{abs(".config/foo"): sym(srcA)},
			want: want{placeReplace: []string{".config/foo"}},
		},
		{
			// foreign symlink (unrecorded): emit a warning and last-wins replace.
			name: "foreign symlink → warn + replace",
			prev: nil,
			next: mani(sl(srcB, ".config/foo")),
			fs:   fakeFS{abs(".config/foo"): sym("/somewhere/foreign")},
			want: want{placeForeign: []string{".config/foo"}, warns: []WarnKind{WarnForeignReplace}},
		},
		{
			// A regular file already at target: no-overwrite conflict.
			name: "regular file at place target → conflict",
			prev: nil,
			next: mani(sl(srcB, ".config/foo")),
			fs:   fakeFS{abs(".config/foo"): reg()},
			want: want{conflicts: 1, conflictKinds: []ConflictKind{ConflictForeignEntity}},
		},
		{
			// An ancestor component is a symlink: cannot nest under it, conflict (→ ADR-0015).
			name: "ancestor symlink → conflict",
			prev: nil,
			next: mani(sl(srcB, ".claude/skills/nix")),
			fs:   fakeFS{abs(".claude"): sym("/some/store")},
			want: want{conflicts: 1, conflictKinds: []ConflictKind{ConflictForeignAncestor}},
		},
		{
			// Self-recorded stale ancestor symlink (prev recorded .claude/skills, on-disk matches, next
			// drops it for children): migrate — pre-remove the ancestor and place children as new,
			// deduping the ancestor across multiple children (→ ADR-0046).
			name: "self-recorded stale ancestor → migrate (preRemove + child PlaceNew)",
			prev: mani(sl(srcA, ".claude/skills")),
			next: mani(sl(srcB, ".claude/skills/foo"), sl(srcB, ".claude/skills/bar")),
			fs: fakeFS{
				abs(".claude"):        dir(),
				abs(".claude/skills"): sym(srcA),
				// The child keys stand in for the files a real lstat would find by resolving through the
				// ancestor symlink into the previous farm. The relaxation must place children as new
				// WITHOUT probing them; if the code regressed to normal lstat classification it would see
				// these regular files and emit conflicts, so their presence keeps this case honest (→ ADR-0046).
				abs(".claude/skills/foo"): reg(),
				abs(".claude/skills/bar"): reg(),
			},
			want: want{
				placeNew:  []string{".claude/skills/foo", ".claude/skills/bar"},
				preRemove: []string{".claude/skills"},
			},
		},
		{
			// Same migration but the nested child is a copy entry: it becomes a place-once CopyAction
			// (the "target absent" copy arm), not a symlink placement (→ ADR-0046).
			name: "self-recorded stale ancestor, copy child → migrate (preRemove + copy)",
			prev: mani(sl(srcA, ".claude/skills")),
			next: mani(cp(srcB, ".claude/skills/foo")),
			fs: fakeFS{
				abs(".claude"):        dir(),
				abs(".claude/skills"): sym(srcA),
				// Stand-in for the file a real lstat would resolve through the ancestor symlink; the copy
				// child must be planned as a place-once new copy without probing it (→ ADR-0046).
				abs(".claude/skills/foo"): reg(),
			},
			want: want{
				copies:    []string{".claude/skills/foo"},
				preRemove: []string{".claude/skills"},
			},
		},
		{
			// Two distinct stale ancestors dropped and re-nested in the same generation: PreRemove
			// accumulates both (the slice/dedup map grow across multiple keys, not just repeat one) (→ ADR-0046).
			name: "two distinct self-recorded stale ancestors → migrate both",
			prev: mani(sl(srcA, ".claude/skills"), sl(srcA, ".config/nvim")),
			next: mani(sl(srcB, ".claude/skills/foo"), sl(srcB, ".config/nvim/init.lua")),
			fs: fakeFS{
				abs(".claude"):        dir(),
				abs(".claude/skills"): sym(srcA),
				abs(".config"):        dir(),
				abs(".config/nvim"):   sym(srcA),
			},
			want: want{
				placeNew:  []string{".claude/skills/foo", ".config/nvim/init.lua"},
				preRemove: []string{".claude/skills", ".config/nvim"},
			},
		},
		{
			// Child nested two levels below the stale ancestor symlink (.claude/skills/sub/foo): the walk
			// stops at the ancestor and the deeper child is placed new via appendAbsentPlacement (→ ADR-0046).
			name: "self-recorded stale ancestor, deep child → migrate",
			prev: mani(sl(srcA, ".claude/skills")),
			next: mani(sl(srcB, ".claude/skills/sub/foo")),
			fs: fakeFS{
				abs(".claude"):        dir(),
				abs(".claude/skills"): sym(srcA),
			},
			want: want{
				placeNew:  []string{".claude/skills/sub/foo"},
				preRemove: []string{".claude/skills"},
			},
		},
		{
			// Recorded ancestor but the on-disk symlink points elsewhere (mismatch = foreign / user-swapped):
			// not eligible for migration, the child stays a conflict; the ancestor is kept with a stale-mismatch
			// warning by the remove side (→ ADR-0046).
			name: "foreign ancestor (recorded mismatch) → conflict",
			prev: mani(sl(srcA, ".claude/skills")),
			next: mani(sl(srcB, ".claude/skills/foo")),
			fs: fakeFS{
				abs(".claude"):        dir(),
				abs(".claude/skills"): sym("/foreign"),
			},
			want: want{conflicts: 1, conflictKinds: []ConflictKind{ConflictForeignAncestor}, warns: []WarnKind{WarnStaleMismatch}},
		},
		{
			// The new generation keeps the ancestor whole-tree symlink AND a nested child
			// (self-contradictory): the ancestor cannot be removed, so the child stays a conflict while
			// the ancestor entry itself re-links as a recorded replace (→ ADR-0046).
			name: "self-contradictory ancestor (kept in next) → conflict",
			prev: mani(sl(srcA, ".claude/skills")),
			next: mani(sl(srcB, ".claude/skills"), sl(srcB, ".claude/skills/foo")),
			fs: fakeFS{
				abs(".claude"):        dir(),
				abs(".claude/skills"): sym(srcA),
			},
			want: want{
				placeReplace:  []string{".claude/skills"},
				conflicts:     1,
				conflictKinds: []ConflictKind{ConflictSelfContradictoryAncestor},
			},
		},
		{
			// Recorded stale ancestor but already gone on disk: ancestorSymlink stops at the missing
			// component, so the child classifies normally as a new placement and nothing is pre-removed.
			name: "recorded ancestor already gone → plain place",
			prev: mani(sl(srcA, ".claude/skills")),
			next: mani(sl(srcB, ".claude/skills/foo")),
			fs:   fakeFS{abs(".claude"): dir()},
			want: want{placeNew: []string{".claude/skills/foo"}},
		},
		{
			// stale matches record: satisfies the conservative invariant, so remove.
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
			// entries = {} (empty manifest): conservatively remove all previous-generation nput symlinks (no warning).
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
			// stale recorded but reality points elsewhere (mismatch): not removed, warning.
			name: "stale mismatch (recorded but points elsewhere) → keep + warn",
			prev: mani(sl(srcA, ".config/foo")),
			next: mani(),
			fs:   fakeFS{abs(".config/foo"): sym("/other/place")},
			want: want{warns: []WarnKind{WarnStaleMismatch}},
		},
		{
			// stale target is a regular file: kept as non-nput-managed, warning.
			name: "stale non-symlink (regular file) → keep + warn",
			prev: mani(sl(srcA, ".config/foo")),
			next: mani(),
			fs:   fakeFS{abs(".config/foo"): reg()},
			want: want{warns: []WarnKind{WarnStaleNonSymlink}},
		},
		{
			// stale target already gone: no-op (no warning).
			name: "stale already gone → no-op",
			prev: mani(sl(srcA, ".config/foo")),
			next: mani(),
			fs:   fakeFS{},
			want: want{},
		},
		{
			// copy entry vanished: not removed, warn as orphan (independent of FS state).
			name: "copy orphan → keep + warn",
			prev: mani(cp(srcA, ".config/foo")),
			next: mani(),
			fs:   fakeFS{abs(".config/foo"): reg()},
			want: want{warns: []WarnKind{WarnCopyOrphan}},
		},
		{
			// copy target absent: new copy under place-once.
			name: "copy target absent → place-once copy",
			prev: nil,
			next: mani(cp(srcA, ".config/foo")),
			fs:   fakeFS{},
			want: want{copies: []string{".config/foo"}},
		},
		{
			// copy exists, recorded (previous generation was also copy): no-op under place-once.
			name: "copy recorded → no-op",
			prev: mani(cp(srcA, ".config/foo")),
			next: mani(cp(srcA, ".config/foo")),
			fs:   fakeFS{srcA: reg(), abs(".config/foo"): reg()},
			want: want{},
		},
		{
			// copy exists, unrecorded (foreign real file): skip + warning.
			name: "copy foreign file → skip + warn",
			prev: nil,
			next: mani(cp(srcA, ".config/foo")),
			fs:   fakeFS{srcA: reg(), abs(".config/foo"): reg()},
			want: want{warns: []WarnKind{WarnCopyForeign}},
		},
		{
			// structure mismatch (src dir × target file): conflict.
			name: "copy structure mismatch (src dir, target file) → conflict",
			prev: nil,
			next: mani(cp(srcA, ".config/foo")),
			fs:   fakeFS{srcA: dir(), abs(".config/foo"): reg()},
			want: want{conflicts: 1, conflictKinds: []ConflictKind{ConflictCopyStructureMismatch}},
		},
		{
			// structure mismatch (src file × target dir): conflict.
			name: "copy structure mismatch (src file, target dir) → conflict",
			prev: nil,
			next: mani(cp(srcA, ".config/foo")),
			fs:   fakeFS{srcA: reg(), abs(".config/foo"): dir()},
			want: want{conflicts: 1, conflictKinds: []ConflictKind{ConflictCopyStructureMismatch}},
		},
		{
			// copy exists, recorded, dir/dir match: no-op.
			name: "copy recorded dir → no-op",
			prev: mani(cp(srcA, ".config/foo")),
			next: mani(cp(srcA, ".config/foo")),
			fs:   fakeFS{srcA: dir(), abs(".config/foo"): dir()},
			want: want{},
		},
		{
			// Real directory occupying the target, all children recorded ∧ stale (self-recorded by
			// this profile's previous generation, dropped by the new one): fully migratable — every
			// child is scheduled RemoveUnlink, the now-empty dir itself RemoveRmdir, and the entry
			// places as a new symlink (→ ADR-0047, issue #175).
			name: "real dir target, all children recorded+stale → migrate (preRemove unlink+rmdir)",
			prev: mani(sl(srcA, ".claude/hooks/foo/main.sh"), sl(srcA, ".claude/hooks/bar/main.sh")),
			next: mani(sl(srcB, ".claude/hooks")),
			fs: fakeFS{
				abs(".claude"):                   dir(),
				abs(".claude/hooks"):             dir(),
				abs(".claude/hooks/foo"):         dir(),
				abs(".claude/hooks/foo/main.sh"): sym(srcA),
				abs(".claude/hooks/bar"):         dir(),
				abs(".claude/hooks/bar/main.sh"): sym(srcA),
			},
			want: want{
				placeNew: []string{".claude/hooks"},
				preRemove: []string{
					".claude/hooks/foo/main.sh",
					".claude/hooks/bar/main.sh",
				},
				preRemoveDir: []string{
					".claude/hooks/foo",
					".claude/hooks/bar",
					".claude/hooks",
				},
			},
		},
		{
			// Real directory containing only an empty subdirectory (no leaf entries at all): empty
			// dirs are migratable regardless of provenance, since rmdir only ever succeeds when empty
			// (data-loss-free even for dirs nput never created · → ADR-0047 D2).
			name: "real dir target, empty subdir of unknown provenance → migrate (rmdir only)",
			prev: nil,
			next: mani(sl(srcB, ".claude/hooks")),
			fs: fakeFS{
				abs(".claude"):             dir(),
				abs(".claude/hooks"):       dir(),
				abs(".claude/hooks/empty"): dir(),
			},
			want: want{
				placeNew: []string{".claude/hooks"},
				preRemoveDir: []string{
					".claude/hooks/empty",
					".claude/hooks",
				},
			},
		},
		{
			// Real directory with one foreign real file mixed among otherwise-migratable children:
			// the whole directory is a conflict, and critically NONE of the migratable siblings are
			// partially removed (no dirActions leak into the plan on failure · → ADR-0047).
			name: "real dir target, one real file mixed in → conflict (no partial removal)",
			prev: mani(sl(srcA, ".claude/hooks/foo/main.sh")),
			next: mani(sl(srcB, ".claude/hooks")),
			fs: fakeFS{
				abs(".claude"):                   dir(),
				abs(".claude/hooks"):             dir(),
				abs(".claude/hooks/foo"):         dir(),
				abs(".claude/hooks/foo/main.sh"): sym(srcA),
				abs(".claude/hooks/README"):      reg(),
			},
			// The dir-migration classification for ".claude/hooks" discards its dirActions on
			// conflict (no partial removal from *that* walk), but ".claude/hooks/foo/main.sh" is
			// independently a stale entry in prev.Entries under the ordinary remove-side loop (it
			// was never added to preRemoved, since the dir classification failed before dedup-ing
			// it). This is harmless: engine.Apply stops before removeStale on any conflict, so this
			// planned removal never executes (→ ADR-0047).
			want: want{conflicts: 1, remove: []string{".claude/hooks/foo/main.sh"}},
		},
		{
			// Real directory whose new generation still records a nested child at the same relative
			// target as a dir-child symlink (self-contradictory manifest): the dir-migration
			// classification for ".claude/hooks" itself conflicts (child kept in next), but
			// ".claude/hooks/foo" is also independently present in next.Entries and gets classified
			// on its own merits by the ordinary per-entry loop (a recorded symlink, unaffected by
			// its ancestor still being a real, unmigrated directory) → PlaceReplace. Harmless for the
			// same reason as above: the conflict blocks engine.Apply before any placement runs.
			name: "real dir target, child kept in next → conflict (self-contradictory)",
			prev: mani(sl(srcA, ".claude/hooks/foo")),
			next: mani(sl(srcB, ".claude/hooks"), sl(srcB, ".claude/hooks/foo")),
			fs: fakeFS{
				abs(".claude"):           dir(),
				abs(".claude/hooks"):     dir(),
				abs(".claude/hooks/foo"): sym(srcA),
			},
			want: want{conflicts: 1, placeReplace: []string{".claude/hooks/foo"}},
		},
		{
			// Real directory with a foreign (record-mismatched) symlink child: conflict.
			name: "real dir target, foreign symlink child → conflict",
			prev: nil,
			next: mani(sl(srcB, ".claude/hooks")),
			fs: fakeFS{
				abs(".claude"):           dir(),
				abs(".claude/hooks"):     dir(),
				abs(".claude/hooks/foo"): sym("/foreign"),
			},
			want: want{conflicts: 1},
		},
		{
			// Method changed symlink→copy at the same target: the previous generation's recorded
			// symlink is pre-removed (Unlink) and a fresh place-once copy is scheduled — zero data
			// loss since the symlink carried no user data (→ ADR-0047 D5).
			name: "method change symlink→copy, recorded → migrate (preRemove unlink + copy)",
			prev: mani(sl(srcA, ".config/tool.conf")),
			next: mani(cp(srcB, ".config/tool.conf")),
			fs: fakeFS{
				srcB:                     reg(),
				abs(".config/tool.conf"): sym(srcA),
			},
			want: want{
				copies:    []string{".config/tool.conf"},
				preRemove: []string{".config/tool.conf"},
			},
		},
		{
			// Method changed symlink→copy but the on-disk symlink drifted from the record (foreign /
			// user-swapped): not eligible for the method-change migration, falls through to the
			// ordinary foreign handling (→ ADR-0047 D5 fallback).
			name: "method change symlink→copy, drifted record → foreign (no migration)",
			prev: mani(sl(srcA, ".config/tool.conf")),
			next: mani(cp(srcB, ".config/tool.conf")),
			fs: fakeFS{
				srcB:                     reg(),
				abs(".config/tool.conf"): sym("/elsewhere"),
			},
			want: want{warns: []WarnKind{WarnCopyForeign}},
		},
		{
			// Method changed copy→symlink at the same target: NOT migrated (D5 keeps copy→symlink a
			// conflict to protect potentially user-edited copy data; ADR-0047 only automates the
			// zero-data-loss symlink→copy direction). The existing copy target is a real file
			// occupying a symlink placement target, so it is the ordinary no-overwrite conflict.
			name: "method change copy→symlink, recorded copy → conflict (not migrated)",
			prev: mani(cp(srcA, ".config/tool.conf")),
			next: mani(sl(srcB, ".config/tool.conf")),
			fs: fakeFS{
				abs(".config/tool.conf"): reg(),
			},
			want: want{conflicts: 1},
		},
		{
			// mixed: new + silent re-link + foreign warning + stale removal + mismatch kept.
			name: "mixed plan",
			prev: mani(sl(srcA, "keep"), sl(srcA, "drop"), sl(srcA, "mism")),
			next: mani(sl(srcB, "keep"), sl(srcB, "fresh"), sl(srcB, "foreign")),
			fs: fakeFS{
				abs("keep"):    sym(srcA),          // matches record → silent replace
				abs("drop"):    sym(srcA),          // stale matches record → remove
				abs("mism"):    sym("/elsewhere"),  // stale mismatch → keep + warn
				abs("foreign"): sym("/foreign/ln"), // unrecorded symlink → foreign warn + replace
				// fresh is absent → PlaceNew
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
			sortedEq(t, "copies", copyTargets(plan), tt.want.copies)
			sortedEq(t, "remove", removeTargets(plan), tt.want.remove)
			sortedEq(t, "preRemove", preRemoveTargets(plan), tt.want.preRemove)
			sortedEq(t, "preRemoveDir", preRemoveDirTargets(plan), tt.want.preRemoveDir)
			warnEq(t, warnKinds(plan), tt.want.warns)
			if len(plan.Conflicts) != tt.want.conflicts {
				t.Errorf("conflicts = %d, want %d (%v)", len(plan.Conflicts), tt.want.conflicts, plan.Conflicts)
			}
			if tt.want.conflictKinds != nil {
				got := make([]ConflictKind, len(plan.Conflicts))
				for i, c := range plan.Conflicts {
					got[i] = c.Kind
				}
				if len(got) != len(tt.want.conflictKinds) {
					t.Errorf("conflict kinds length = %d, want %d (got %v, want %v)", len(got), len(tt.want.conflictKinds), got, tt.want.conflictKinds)
				} else {
					for i := range got {
						if got[i] != tt.want.conflictKinds[i] {
							t.Errorf("conflict kinds[%d] = %v, want %v (full: got %v, want %v)", i, got[i], tt.want.conflictKinds[i], got, tt.want.conflictKinds)
							break
						}
					}
				}
			}
		})
	}
}

// TestComputeDirMigrationPreRemoveOrderIsBottomUp verifies the ordering invariant
// classifyDirMigration's doc comment promises (→ ADR-0047, issue #175 §8): plan.PreRemove lists
// children before parents, so a consumer walking the slice front-to-back naturally unlinks
// leaves first and rmdirs from the deepest directory upward — never rmdir-ing a directory before
// its own contents are cleared. TestComputeTableDriven's sortedEq comparisons cannot catch an
// order regression (they sort before comparing), so this test asserts slice order directly on a
// three-level-deep occupying directory (.claude/hooks/foo/sub, with a leaf at each level).
func TestComputeDirMigrationPreRemoveOrderIsBottomUp(t *testing.T) {
	const src = "/nix/store/aaa-src"
	prev := mani(
		sl(src, ".claude/hooks/top.sh"),
		sl(src, ".claude/hooks/foo/mid.sh"),
		sl(src, ".claude/hooks/foo/sub/leaf.sh"),
	)
	next := mani(sl(src, ".claude/hooks"))
	fs := fakeFS{
		abs(".claude"):                       dir(),
		abs(".claude/hooks"):                 dir(),
		abs(".claude/hooks/top.sh"):          sym(src),
		abs(".claude/hooks/foo"):             dir(),
		abs(".claude/hooks/foo/mid.sh"):      sym(src),
		abs(".claude/hooks/foo/sub"):         dir(),
		abs(".claude/hooks/foo/sub/leaf.sh"): sym(src),
	}

	plan, err := Compute(prev, next, root, fs)
	if err != nil {
		t.Fatalf("Compute: unexpected error: %v", err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("Conflicts = %v, want none", plan.Conflicts)
	}

	// Build a position index over the actual PreRemove slice order (both Unlink and Rmdir kinds,
	// keyed by their target — Entry.Target for Unlink, the root-relative dir for Rmdir).
	pos := map[string]int{}
	for i, a := range plan.PreRemove {
		key := a.Entry.Target
		if a.Kind == RemoveRmdir {
			rel, err := filepath.Rel(root, a.TargetAbs)
			if err != nil {
				t.Fatalf("filepath.Rel: %v", err)
			}
			key = rel
		}
		pos[key] = i
	}

	// Every ordering constraint classifyDirMigration's bottom-up contract implies: a leaf's Unlink
	// must precede the Rmdir of the directory that directly contains it, at every nesting level.
	constraints := [][2]string{
		{".claude/hooks/foo/sub/leaf.sh", ".claude/hooks/foo/sub"}, // deepest leaf before its dir
		{".claude/hooks/foo/sub", ".claude/hooks/foo"},             // deepest dir before its parent dir
		{".claude/hooks/foo/mid.sh", ".claude/hooks/foo"},          // mid-level leaf before its dir
		{".claude/hooks/foo", ".claude/hooks"},                     // that dir before the placement target
		{".claude/hooks/top.sh", ".claude/hooks"},                  // top-level leaf before the placement target
	}
	for _, c := range constraints {
		before, after := c[0], c[1]
		bi, ok := pos[before]
		if !ok {
			t.Fatalf("PreRemove missing expected action for %q; got %v", before, plan.PreRemove)
		}
		ai, ok := pos[after]
		if !ok {
			t.Fatalf("PreRemove missing expected action for %q; got %v", after, plan.PreRemove)
		}
		if bi >= ai {
			t.Errorf("PreRemove order: %q (index %d) must come before %q (index %d); got %v", before, bi, after, ai, plan.PreRemove)
		}
	}
}

// TestComputeUnknownMethodErrors verifies that an unknown method is rejected.
func TestComputeUnknownMethodErrors(t *testing.T) {
	e := sl("/nix/store/x", ".config/foo")
	e.Method = "bogus"
	_, err := Compute(nil, mani(e), root, fakeFS{})
	if err == nil {
		t.Fatal("expected error for unknown method, got nil")
	}
}

// TestComputeAncestorDirNotSymlink confirms that no conflict arises when the ancestor is a regular directory.
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
