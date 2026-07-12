package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/planner"
)

// Tests for PreRemove's generalization from "self-recorded stale ancestor symlink" (ADR-0046)
// to "any self-recorded stale filesystem object occupying a placement target": an occupying real
// directory whose whole tree is recorded-stale-or-empty, and a symlink→copy method change
// (→ ADR-0047, issue #175, #172 D2/D3/D5).

// TestApplyPerFileToDirSymlinkMigratesSameNamedLeaf reproduces the 2026-07-12 real incident: a
// per-file layout `<name>/main.sh` migrating to a whole-tree dir symlink `<name>`, where the new
// generation's dir symlink target shares its leaf name with the old per-file target
// (`.claude/hooks/foo/main.sh` → `.claude/hooks` as a dir symlink). This is exactly the case a
// naive readlink-pattern cleanup (home-manager's) misjudges; nput's manifest-recorded classification
// must migrate it cleanly with a single apply (→ issue #172 background, ADR-0047).
func TestApplyPerFileToDirSymlinkMigratesSameNamedLeaf(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	srcOld := makeSrc(t, "foo/main.sh")

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(srcOld, "foo/main.sh", ".claude/hooks/foo/main.sh")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	srcNew := realTempDir(t)
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(srcNew, ".", ".claude/hooks")))
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("second Apply (dir migration): %v", err)
	}

	got, err := os.Readlink(filepath.Join(root, ".claude", "hooks"))
	if err != nil || got != srcNew {
		t.Fatalf("readlink(.claude/hooks) = %q, err %v; want %q", got, err, srcNew)
	}
	if len(res.Conflicts) != 0 {
		t.Errorf("Conflicts = %v, want none", res.Conflicts)
	}
	// The leaf must be reported exactly once as Removed (Unlink). Both the intermediate
	// ".claude/hooks/foo" directory AND the placement target ".claude/hooks" itself are
	// legitimately Rmdir-ed and reported in Pruned exactly once each — not twice each. A prior
	// bug double-reported them: preRemove's own pruneEmptyAncestors walk (triggered by the
	// leaf's Unlink, or by an inner Rmdir's success) raced the planner's own explicit,
	// already-ordered RemoveRmdir actions for the same directories, recording each directory
	// from two paths (→ ADR-0047, issue #175). preRemove no longer walks pruneEmptyAncestors at
	// all, since every directory a migration needs cleared is already an explicit RemoveRmdir
	// planner action and the placement step that follows recreates the target immediately.
	if len(res.Removed) != 1 || res.Removed[0] != ".claude/hooks/foo/main.sh" {
		t.Errorf("Removed = %v, want exactly [.claude/hooks/foo/main.sh]", res.Removed)
	}
	wantPruned := []string{
		filepath.Join(root, ".claude", "hooks", "foo"),
		filepath.Join(root, ".claude", "hooks"),
	}
	if len(res.Pruned) != len(wantPruned) {
		t.Errorf("Pruned = %v, want exactly %v (no double-report)", res.Pruned, wantPruned)
	} else {
		seen := map[string]bool{}
		for _, p := range res.Pruned {
			if seen[p] {
				t.Errorf("Pruned = %v contains a duplicate: %s", res.Pruned, p)
			}
			seen[p] = true
		}
		for _, want := range wantPruned {
			if !seen[want] {
				t.Errorf("Pruned = %v, missing %s", res.Pruned, want)
			}
		}
	}
}

// TestApplyDirSymlinkRoundTripsThroughPerFile verifies the reverse and back: dir symlink →
// per-file → dir symlink again converges cleanly across three generations, exercising
// ADR-0046's ancestor migration and ADR-0047's dir migration in sequence.
func TestApplyDirSymlinkRoundTripsThroughPerFile(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	opts := func(lf string) Options {
		return Options{LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil)}
	}

	src1 := realTempDir(t)
	lf1 := writeLinkFarm(t, projectManifest(storeEntry(src1, ".", ".claude/skills")))
	if _, err := Apply(opts(lf1)); err != nil {
		t.Fatalf("gen1 (dir symlink): %v", err)
	}

	src2 := makeSrc(t, "nix")
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(src2, "nix", ".claude/skills/nix")))
	if _, err := Apply(opts(lf2)); err != nil {
		t.Fatalf("gen2 (per-file): %v", err)
	}
	if got, err := os.Readlink(filepath.Join(root, ".claude", "skills", "nix")); err != nil || got != filepath.Join(src2, "nix") {
		t.Fatalf("gen2 readlink = %q, err %v", got, err)
	}

	src3 := realTempDir(t)
	lf3 := writeLinkFarm(t, projectManifest(storeEntry(src3, ".", ".claude/skills")))
	if _, err := Apply(opts(lf3)); err != nil {
		t.Fatalf("gen3 (dir symlink again): %v", err)
	}
	got, err := os.Readlink(filepath.Join(root, ".claude", "skills"))
	if err != nil || got != src3 {
		t.Fatalf("gen3 readlink = %q, err %v; want %q", got, err, src3)
	}
}

// TestApplyDirMigrationConflictLeavesSiblingsUntouched verifies D2's "one real file blocks the
// whole directory" rule end-to-end: a real file mixed among otherwise-migratable recorded-stale
// symlinks makes Apply stop with a conflict, and NONE of the migratable siblings are removed —
// no partial removal (→ ADR-0047 D2, issue #175 §8).
func TestApplyDirMigrationConflictLeavesSiblingsUntouched(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	srcOld := realTempDir(t)

	lf1 := writeLinkFarm(t, projectManifest(
		storeEntry(srcOld, ".", ".claude/hooks/foo"),
		storeEntry(srcOld, ".", ".claude/hooks/bar"),
	))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// A foreign real file appears alongside the recorded-stale symlinks.
	if err := os.WriteFile(filepath.Join(root, ".claude", "hooks", "README"), []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}

	srcNew := realTempDir(t)
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(srcNew, ".", ".claude/hooks")))
	_, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil {
		t.Fatal("expected a conflict error, got nil")
	}

	// Both recorded-stale siblings must still exist: no partial removal.
	for _, leaf := range []string{"foo", "bar"} {
		if _, err := os.Lstat(filepath.Join(root, ".claude", "hooks", leaf)); err != nil {
			t.Errorf("sibling %q must survive the conflict untouched: %v", leaf, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, ".claude", "hooks", "README")); err != nil {
		t.Errorf("foreign file must survive: %v", err)
	}
}

// TestApplyDirMigrationEmptySubdirsAtMultipleDepths verifies D2's "empty dirs are migratable
// regardless of provenance" rule across a multi-level nested empty subtree, and that a root-level
// (direct child of root) real-dir target migrates too.
func TestApplyDirMigrationEmptySubdirsAtMultipleDepths(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)

	// A multi-level empty subtree nput never created, occupying a root-level target directly.
	if err := os.MkdirAll(filepath.Join(root, "hooks", "a", "b", "c"), 0o755); err != nil {
		t.Fatal(err)
	}

	srcNew := realTempDir(t)
	lf := writeLinkFarm(t, projectManifest(storeEntry(srcNew, ".", "hooks")))
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(res.Conflicts) != 0 {
		t.Fatalf("Conflicts = %v, want none", res.Conflicts)
	}
	got, err := os.Readlink(filepath.Join(root, "hooks"))
	if err != nil || got != srcNew {
		t.Fatalf("readlink(hooks) = %q, err %v; want %q", got, err, srcNew)
	}
}

// TestApplyMethodChangeSymlinkToCopyMigrates verifies D5: a target whose method changes from
// symlink (previous generation, recorded, on-disk matches) to copy (new generation) is
// automatically migrated — the recorded symlink is pre-removed and a fresh place-once copy lands
// in its place, with zero data loss (the symlink carried no user data · → ADR-0047 D5).
func TestApplyMethodChangeSymlinkToCopyMigrates(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	symSrc := realTempDir(t)

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(symSrc, ".", ".config/tool.conf")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply (symlink): %v", err)
	}

	copySrc := makeSrc(t, "tool.conf")
	lf2 := writeLinkFarm(t, projectManifest(copyEntry(copySrc, "tool.conf", ".config/tool.conf")))
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("second Apply (method change → copy): %v", err)
	}

	info, err := os.Lstat(filepath.Join(root, ".config", "tool.conf"))
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("target must be a real copied file, not a symlink")
	}
	data, err := os.ReadFile(filepath.Join(root, ".config", "tool.conf"))
	if err != nil || string(data) != "content" {
		t.Errorf("copied content = %q, err %v; want %q", data, err, "content")
	}
	if len(res.Conflicts) != 0 {
		t.Errorf("Conflicts = %v, want none", res.Conflicts)
	}
}

// TestApplyMethodChangeCopyToSymlinkStaysConflict verifies D5's asymmetry: copy→symlink is NOT
// automated (a copy may hold user edits), so it stays the ordinary no-overwrite conflict.
func TestApplyMethodChangeCopyToSymlinkStaysConflict(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	copySrc := makeSrc(t, "tool.conf")

	lf1 := writeLinkFarm(t, projectManifest(copyEntry(copySrc, "tool.conf", ".config/tool.conf")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply (copy): %v", err)
	}

	symSrc := realTempDir(t)
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(symSrc, ".", ".config/tool.conf")))
	_, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil {
		t.Fatal("expected a conflict error for copy→symlink, got nil")
	}

	// The copy target must survive untouched (not migrated, not deleted).
	data, rerr := os.ReadFile(filepath.Join(root, ".config", "tool.conf"))
	if rerr != nil || string(data) != "content" {
		t.Errorf("copy target must survive untouched: data=%q, err=%v", data, rerr)
	}
}

// TestApplyMethodChangeSymlinkToCopyDriftFallsBackToForeign verifies D5's fallback: if the
// on-disk symlink drifted from the previous generation's record (readlink mismatch) at the
// moment of planning, the method-change migration does not fire; it falls through to the
// ordinary copy-foreign-file handling (skip + warning, not an overwrite).
func TestApplyMethodChangeSymlinkToCopyDriftFallsBackToForeign(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	symSrc := realTempDir(t)

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(symSrc, ".", ".config/tool.conf")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply (symlink): %v", err)
	}

	// Drift the on-disk symlink away from the recorded dest before the method-changing apply.
	targetAbs := filepath.Join(root, ".config", "tool.conf")
	if err := os.Remove(targetAbs); err != nil {
		t.Fatal(err)
	}
	foreign := realTempDir(t)
	if err := os.Symlink(foreign, targetAbs); err != nil {
		t.Fatal(err)
	}

	copySrc := makeSrc(t, "tool.conf")
	var warns []string
	lf2 := writeLinkFarm(t, projectManifest(copyEntry(copySrc, "tool.conf", ".config/tool.conf")))
	_, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
		Warnf: collectWarnings(&warns),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// The drifted symlink must survive untouched (not migrated, not overwritten).
	got, rerr := os.Readlink(targetAbs)
	if rerr != nil || got != foreign {
		t.Errorf("drifted symlink must survive untouched: readlink=%q, err=%v, want %q", got, rerr, foreign)
	}
	found := false
	for _, w := range warns {
		if strings.Contains(w, "skipped copy") {
			found = true
		}
	}
	if !found {
		t.Errorf("warnings = %v, want a copy-foreign-file skip warning", warns)
	}
}

// TestApplyDirMigrationNonEmptySubdirIsConflictAtPlanTime verifies the plan-time half of D3's
// safety net: a subdirectory that is non-empty *before Apply even plans* makes the planner
// classify the whole occupying directory as non-migratable (a real-file leaf), so Apply reports
// a conflict rather than ever scheduling an Rmdir for it. The complementary runtime half — an
// Rmdir action scheduled at plan time whose target gains content in the window before preRemove
// executes it (true TOCTOU) — is exercised directly against applier.preRemove in
// TestPreRemoveRmdirDriftErrorsDirectly, since Apply's own planning step cannot be paused
// mid-flight to inject content after a snapshot it already took.
func TestApplyDirMigrationNonEmptySubdirIsConflictAtPlanTime(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)

	if err := os.MkdirAll(filepath.Join(root, "hooks", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hooks", "empty", "surprise"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	srcNew := realTempDir(t)
	lf := writeLinkFarm(t, projectManifest(storeEntry(srcNew, ".", "hooks")))
	_, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err == nil {
		t.Fatal("expected a conflict/error, got nil")
	}
}

// TestPreRemoveRmdirDriftErrorsDirectly drives applier.preRemove directly with a RemoveRmdir
// action whose target directory has gained content since planning, verifying the ENOTEMPTY drift
// is surfaced as a loud error (not skipped) with a message naming the target (→ ADR-0047 D3).
func TestPreRemoveRmdirDriftErrorsDirectly(t *testing.T) {
	dir := realTempDir(t)
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// Drift: something added content after the plan believed sub/ was empty.
	if err := os.WriteFile(filepath.Join(sub, "surprise"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var warns []string
	a := staleErr_applier(&warns)
	act := planner.RemoveAction{Kind: planner.RemoveRmdir, TargetAbs: sub}
	err := a.preRemove([]planner.RemoveAction{act})
	if err == nil {
		t.Fatal("expected an error for a non-empty Rmdir target, got nil")
	}
	if !strings.Contains(err.Error(), "cannot migrate this placement target safely") {
		t.Errorf("error = %q, want it to mention the safe-migration abort message", err.Error())
	}
	if _, statErr := os.Lstat(sub); statErr != nil {
		t.Errorf("sub must survive the aborted rmdir: %v", statErr)
	}
}

// TestPreRemoveUnlinkDriftErrorsDirectly drives applier.preRemove directly with a RemoveUnlink
// action whose recorded symlink drifted (foreign rewrite) since planning, verifying it errors
// loudly instead of skipping (→ ADR-0047 D3, same asymmetry as ADR-0046 §3).
func TestPreRemoveUnlinkDriftErrorsDirectly(t *testing.T) {
	dir := realTempDir(t)
	targetAbs := filepath.Join(dir, "ancestor")
	foreign := realTempDir(t)
	if err := os.Symlink(foreign, targetAbs); err != nil {
		t.Fatal(err)
	}

	var warns []string
	a := staleErr_applier(&warns)
	act := staleErr_action(realTempDir(t), "ancestor", targetAbs)
	err := a.preRemove([]planner.RemoveAction{act})
	if err == nil {
		t.Fatal("expected an error for a drifted recorded symlink, got nil")
	}
	if !strings.Contains(err.Error(), "cannot migrate this placement target safely") {
		t.Errorf("error = %q, want it to mention the safe-migration abort message", err.Error())
	}
	got, rerr := os.Readlink(targetAbs)
	if rerr != nil || got != foreign {
		t.Errorf("drifted symlink must survive the aborted unlink: readlink=%q, err=%v", got, rerr)
	}
}

// TestApplyDirMigrationInterruptedAfterPreRemoveReRunConverges verifies ADR-0017 idempotence for
// a genuinely partial-migration crash window: PreRemove runs (the occupying directory's
// recorded-stale child is unlinked and the now-empty directory rmdir-ed) but the process stops
// there, before place/commit — the real crash window the generalized PreRemove opens (→ ADR-0047,
// issue #175 §8). A subsequent ordinary Apply must re-plan against that partially-migrated FS
// (the target now absent, not a real directory) and converge to the same fully-migrated state as
// an uninterrupted apply.
func TestApplyDirMigrationInterruptedAfterPreRemoveReRunConverges(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	srcOld := realTempDir(t)

	lf1 := writeLinkFarm(t, projectManifest(storeEntry(srcOld, ".", ".claude/hooks/foo")))
	if _, err := Apply(Options{
		LinkFarm: lf1, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	srcNew := realTempDir(t)
	next := projectManifest(storeEntry(srcNew, ".", ".claude/hooks"))
	prev := projectManifest(storeEntry(srcOld, ".", ".claude/hooks/foo"))
	plan, err := planner.Compute(&prev, &next, root, planner.OSFS, planner.Options{})
	if err != nil {
		t.Fatalf("planner.Compute: %v", err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("plan.Conflicts = %v, want none", plan.Conflicts)
	}

	// Simulate a crash: run only PreRemove (unlink the recorded-stale child, rmdir the now-empty
	// dirs including the placement target itself), then stop — no place, no commit. The target is
	// now absent from the FS, and the previous generation's manifest.json (the profile link) still
	// points at generation 1 since --set never ran.
	a := &applier{opts: Options{Warnf: func(string, ...any) {}}, result: &Result{}}
	a.root = root
	if err := a.preRemove(plan.PreRemove); err != nil {
		t.Fatalf("simulated partial preRemove: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".claude", "hooks")); !os.IsNotExist(err) {
		t.Fatalf("setup: .claude/hooks must be absent after the simulated crash, lstat err = %v", err)
	}

	// An ordinary re-run apply must re-plan against this partially-migrated FS (target absent, no
	// PreRemove needed this time) and converge to the fully-migrated state without erroring.
	lf2 := writeLinkFarm(t, projectManifest(storeEntry(srcNew, ".", ".claude/hooks")))
	res, err := Apply(Options{
		LinkFarm: lf2, Name: "c", RootOverride: root, StateDir: state, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("re-run Apply after simulated crash: %v", err)
	}
	got, rerr := os.Readlink(filepath.Join(root, ".claude", "hooks"))
	if rerr != nil || got != srcNew {
		t.Fatalf("re-run readlink = %q, err %v; want %q", got, rerr, srcNew)
	}
	// The target was already absent going into the re-run (the simulated crash's PreRemove already
	// cleared it), so this second Apply must take the plain PlaceNew path, not PreRemove again —
	// confirming the crash window truly left nothing further for PreRemove to do.
	if len(res.Removed) != 0 || len(res.Pruned) != 0 {
		t.Errorf("re-run Removed/Pruned = %v/%v, want both empty (no PreRemove needed against an already-absent target)", res.Removed, res.Pruned)
	}
}
