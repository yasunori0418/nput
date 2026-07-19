package engine

// Tests for the Result extensions issue #130 wires for the niface envelope (#131 / #132 share
// them): full-inventory (Entries), reached state on a mid-run failure (FailedTarget / Unreached /
// Unwound + partial Result), generation observation (GenBefore / GenAfter, nil-able), and
// structured planner warnings (Warnings).

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/paths"
	"github.com/yasunori0418/nput/internal/planner"
)

// fixedManifest is a rootKind=fixed manifest (no git dependency, root passed explicitly).
func fixedManifest(root string, entries ...manifest.Entry) manifest.Manifest {
	return manifest.Manifest{
		SchemaVersion: 1,
		Root:          manifest.Root{RootKind: manifest.RootKindFixed, Root: root},
		Entries:       entries,
	}
}

// fakeGenCommit mimics nix-env --set's generation bookkeeping: each commit creates the sibling
// generation link <profile>-<N>-link -> linkFarm and re-points the profile link at it
// (relative, like nix-env), so observeGeneration can read N back.
func fakeGenCommit(gen *int) CommitFunc {
	return func(profileLink, linkFarm string) error {
		*gen++
		genLink := paths.GenerationLink(profileLink, *gen)
		if err := os.Symlink(linkFarm, genLink); err != nil {
			return err
		}
		_ = os.Remove(profileLink)
		return os.Symlink(filepath.Base(genLink), profileLink)
	}
}

func TestObserveGeneration(t *testing.T) {
	dir := realTempDir(t)
	profile := filepath.Join(dir, "profile")

	if got := observeGeneration(profile); got != nil {
		t.Errorf("absent profile: got %v, want nil", *got)
	}

	// A generation link "profile-3-link" (relative sibling, as nix-env creates it) parses as 3.
	if err := os.Symlink("profile-3-link", profile); err != nil {
		t.Fatal(err)
	}
	if got := observeGeneration(profile); got == nil || *got != 3 {
		t.Errorf("profile-3-link: got %v, want 3", got)
	}

	// A profile link pointing at anything else (e.g. a store path in tests) is unobservable.
	_ = os.Remove(profile)
	if err := os.Symlink("/nix/store/abc-link-farm", profile); err != nil {
		t.Fatal(err)
	}
	if got := observeGeneration(profile); got != nil {
		t.Errorf("non-generation dest: got %v, want nil", *got)
	}

	// A malformed generation number is unobservable.
	_ = os.Remove(profile)
	if err := os.Symlink("profile-x-link", profile); err != nil {
		t.Fatal(err)
	}
	if got := observeGeneration(profile); got != nil {
		t.Errorf("malformed number: got %v, want nil", *got)
	}
}

// TestApplyResultInventoryAndGeneration verifies Entries carries the full manifest inventory and
// GenBefore/GenAfter track the observed generation numbers across first apply (nil → 1), second
// apply (1 → 2), and dryrun (before == after, untouched pointer).
func TestApplyResultInventoryAndGeneration(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "conf/rc")
	e1 := storeEntry(src, "conf", ".config/one")
	lf := writeLinkFarm(t, fixedManifest(root, e1))

	gen := 0
	res, err := Apply(Options{LinkFarm: lf, Name: "c", StateDir: state, Commit: fakeGenCommit(&gen)})
	if err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	if len(res.Entries) != 1 || res.Entries[0].Target != ".config/one" {
		t.Errorf("Entries = %+v, want the full manifest inventory", res.Entries)
	}
	if res.GenBefore != nil {
		t.Errorf("first apply GenBefore = %v, want nil (no profile yet)", *res.GenBefore)
	}
	if res.GenAfter == nil || *res.GenAfter != 1 {
		t.Errorf("first apply GenAfter = %v, want 1", res.GenAfter)
	}

	// Second apply: a changed manifest (extra entry) commits generation 2.
	src2 := makeSrc(t, "conf/rc")
	e2 := storeEntry(src2, "conf", ".config/two")
	lf2 := writeLinkFarm(t, fixedManifest(root, e1, e2))
	res, err = Apply(Options{LinkFarm: lf2, Name: "c", StateDir: state, Commit: fakeGenCommit(&gen)})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if len(res.Entries) != 2 {
		t.Errorf("Entries = %+v, want 2 entries", res.Entries)
	}
	if res.GenBefore == nil || *res.GenBefore != 1 {
		t.Errorf("second apply GenBefore = %v, want 1", res.GenBefore)
	}
	if res.GenAfter == nil || *res.GenAfter != 2 {
		t.Errorf("second apply GenAfter = %v, want 2", res.GenAfter)
	}

	// Dryrun never moves the pointer: before == after == 2, and Entries is still the inventory.
	res, err = Apply(Options{LinkFarm: lf2, Name: "c", StateDir: state, DryRun: true})
	if err != nil {
		t.Fatalf("dryrun Apply: %v", err)
	}
	if res.GenBefore == nil || *res.GenBefore != 2 || res.GenAfter == nil || *res.GenAfter != 2 {
		t.Errorf("dryrun GenBefore/GenAfter = %v/%v, want 2/2", res.GenBefore, res.GenAfter)
	}
	if len(res.Entries) != 2 {
		t.Errorf("dryrun Entries = %+v, want 2 entries", res.Entries)
	}
}

// TestApplyResultReachedStateOnFailure verifies the reached/unreached partition on a mid-place
// failure: the completed op lists stay a "completed" record, the failing entry lands in
// FailedTarget (not in Placed), later planned targets land in Unreached, the partial Result is
// returned alongside the error, and Unwound reports the journal rollback.
func TestApplyResultReachedStateOnFailure(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "conf/rc")

	// "blocked/child" plans as an ordinary new placement (absent target) but fails at execution:
	// its parent directory exists read-only, so os.Symlink EACCESes mid-place.
	if os.Getuid() == 0 {
		t.Skip("running as root; read-only directory does not block")
	}
	if err := os.Mkdir(filepath.Join(root, "blocked"), 0o555); err != nil {
		t.Fatal(err)
	}
	ok := storeEntry(src, "conf", "aa-ok")
	bad := storeEntry(src, "conf", "blocked/child")
	later := storeEntry(src, "conf", "zz-later")
	lf := writeLinkFarm(t, fixedManifest(root, ok, bad, later))

	res, err := Apply(Options{LinkFarm: lf, Name: "c", StateDir: state, Commit: fakeCommit(nil)})
	if err == nil {
		t.Fatal("Apply should fail on the blocked parent")
	}
	if res == nil {
		t.Fatal("Apply should return the partial Result alongside the error")
	}
	if res.FailedTarget != "blocked/child" {
		t.Errorf("FailedTarget = %q, want %q", res.FailedTarget, "blocked/child")
	}
	if len(res.Placed) != 1 || res.Placed[0] != "aa-ok" {
		t.Errorf("Placed = %v, want the completed action only", res.Placed)
	}
	if len(res.Unreached) != 1 || res.Unreached[0] != "zz-later" {
		t.Errorf("Unreached = %v, want [zz-later]", res.Unreached)
	}
	if !res.Unwound {
		t.Error("Unwound = false, want true (journal rolled the run back)")
	}
	// The unwind reverted the completed placement, so the op list records performed-then-reverted work.
	if _, err := os.Lstat(filepath.Join(root, "aa-ok")); !os.IsNotExist(err) {
		t.Errorf("aa-ok should have been unwound, lstat err = %v", err)
	}
}

// TestApplyResultCommitFailurePartialResult verifies the commit-failure branch: every planned
// FS action already succeeded, so the partial Result is returned with the inventory filled and
// the reached-state fields empty (not entry-scoped), nothing is unwound (→ ADR-0044 §2), and the
// unmoved profile pointer is observed as GenAfter == GenBefore.
func TestApplyResultCommitFailurePartialResult(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "conf/rc")
	lf := writeLinkFarm(t, fixedManifest(root, storeEntry(src, "conf", "link")))

	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", StateDir: state,
		Commit: func(string, string) error { return os.ErrPermission },
	})
	if err == nil {
		t.Fatal("Apply must fail when the commit fails")
	}
	if res == nil {
		t.Fatal("Apply must return the partial Result alongside a commit failure")
	}
	if len(res.Entries) != 1 || res.Entries[0].Target != "link" {
		t.Errorf("Entries = %+v, want the full inventory", res.Entries)
	}
	if res.FailedTarget != "" || len(res.Unreached) != 0 {
		t.Errorf("FailedTarget/Unreached = %q/%v, want empty (commit failure is not entry-scoped)", res.FailedTarget, res.Unreached)
	}
	if res.Unwound {
		t.Error("Unwound = true, want false (a commit failure is not unwound; → ADR-0044 §2)")
	}
	if res.GenBefore != nil || res.GenAfter != nil {
		t.Errorf("GenBefore/GenAfter = %v/%v, want nil/nil (no generation was ever committed)", res.GenBefore, res.GenAfter)
	}
	// The placement itself survives (idempotent re-apply converges; only the commit failed).
	if _, err := os.Readlink(filepath.Join(root, "link")); err != nil {
		t.Errorf("placed symlink must survive a commit failure: %v", err)
	}
	if len(res.Placed) != 1 || res.Placed[0] != "link" {
		t.Errorf("Placed = %v, want [link]", res.Placed)
	}
}

// TestApplyResultRecopyUnreached verifies fail()'s --recopy branch: recopy's copy execution
// source is the manifest (not plan.Copies), so a copy entry whose place-once classification
// produced no CopyAction (an existing foreign target recopy would overwrite) must still land in
// Unreached when an earlier stage failure stops the run before materializeCopies.
func TestApplyResultRecopyUnreached(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; read-only directory does not block")
	}
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "conf/rc")

	// The symlink entry fails at place (read-only parent); the copy entry's target already
	// exists as a foreign real file, so place-once emits no CopyAction — only recopy would
	// overwrite it, and only the manifest walk can report it as unreached.
	if err := os.Mkdir(filepath.Join(root, "blocked"), 0o555); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "copytarget"), []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := storeEntry(src, "conf", "blocked/child")
	cp := copyEntry(src, "conf/rc", "copytarget") // file×file: place-once skips it as foreign, no structure conflict
	lf := writeLinkFarm(t, fixedManifest(root, bad, cp))

	res, err := Apply(Options{LinkFarm: lf, Name: "c", StateDir: state, Recopy: true, Commit: fakeCommit(nil)})
	if err == nil {
		t.Fatal("Apply should fail on the blocked parent")
	}
	if res == nil {
		t.Fatal("Apply should return the partial Result alongside the error")
	}
	if res.FailedTarget != "blocked/child" {
		t.Errorf("FailedTarget = %q, want %q", res.FailedTarget, "blocked/child")
	}
	if len(res.Unreached) != 1 || res.Unreached[0] != "copytarget" {
		t.Errorf("Unreached = %v, want [copytarget] via the manifest walk", res.Unreached)
	}
}

// TestApplyResultStructuredWarnings verifies the planner's entry warnings are recorded on the
// Result in structured form (kind + target) while the Warnf text keeps streaming.
func TestApplyResultStructuredWarnings(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "conf/rc")

	// A pre-existing unrecorded symlink at the target → WarnForeignReplace (last-wins).
	if err := os.Symlink("/somewhere/else", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, fixedManifest(root, storeEntry(src, "conf", "link")))

	var warned []string
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", StateDir: state,
		Commit: fakeCommit(nil), Warnf: collectFormatted(&warned),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Kind != planner.WarnForeignReplace || res.Warnings[0].Target != "link" {
		t.Errorf("Warnings = %+v, want [{WarnForeignReplace link}]", res.Warnings)
	}
	if len(warned) != 1 {
		t.Errorf("Warnf stream = %v, want the human-readable text alongside", warned)
	}
}

// TestResetResultGenerationUnobservable pins the nil branch of Reset's generation observation:
// a profile link that is not a generation link (test-substituted commit) observes neither
// before nor after (→ niface ADR-0015's nil-able Generation).
func TestResetResultGenerationUnobservable(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "conf/rc")
	lf := writeLinkFarm(t, fixedManifest(root, storeEntry(src, "conf", "link")))

	if _, err := Apply(Options{LinkFarm: lf, Name: "c", StateDir: state, Commit: fakeCommit(nil)}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	res, err := Reset(ResetOptions{Name: "c", RootKind: manifest.RootKindFixed, FixedRoot: root, StateDir: state})
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if res.GenBefore != nil || res.GenAfter != nil {
		t.Errorf("GenBefore/GenAfter = %v/%v, want nil/nil (profile link is not a generation link)", res.GenBefore, res.GenAfter)
	}
}

// TestResetResultGenerationAndWarnings verifies Reset observes the (unmoved) profile generation
// as before == after and exposes the planner warnings in structured form.
func TestResetResultGenerationAndWarnings(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "conf/rc")
	e := storeEntry(src, "conf", "link")
	lf := writeLinkFarm(t, fixedManifest(root, e))

	gen := 0
	if _, err := Apply(Options{LinkFarm: lf, Name: "c", StateDir: state, Commit: fakeGenCommit(&gen)}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Drift the placed symlink so reset keeps it (record mismatch → structured warning).
	target := filepath.Join(root, "link")
	_ = os.Remove(target)
	if err := os.Symlink("/somewhere/else", target); err != nil {
		t.Fatal(err)
	}

	var warned []string
	res, err := Reset(ResetOptions{
		Name: "c", RootKind: manifest.RootKindFixed, FixedRoot: root, StateDir: state,
		Warnf: collectFormatted(&warned),
	})
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if res.GenBefore == nil || *res.GenBefore != 1 || res.GenAfter == nil || *res.GenAfter != 1 {
		t.Errorf("GenBefore/GenAfter = %v/%v, want 1/1 (FS-only teardown)", res.GenBefore, res.GenAfter)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Kind != planner.WarnStaleMismatch || res.Warnings[0].Target != "link" {
		t.Errorf("Warnings = %+v, want [{WarnStaleMismatch link}]", res.Warnings)
	}
	if len(res.KeptForeign) != 1 || res.KeptForeign[0] != "link" {
		t.Errorf("KeptForeign = %v, want [link]", res.KeptForeign)
	}
}

// --- issue #131 extensions: structured conflicts, removal-plan entries, replaced dests,
// reset partial failure ---

// TestApplyConflictPartialResult verifies the non-dryrun conflict stop returns the partial
// Result (not nil): the full inventory, the structured conflicts, and every planned action in
// Unreached, so the CLI can map failed (E_NPUT_COLLISION) vs skipped items (→ issue #131).
func TestApplyConflictPartialResult(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "conf/rc")

	// "busy" is occupied by a regular file (ConflictForeignEntity); "free" would place fine.
	if err := os.WriteFile(filepath.Join(root, "busy"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, fixedManifest(root, storeEntry(src, "conf", "busy"), storeEntry(src, "conf", "free")))

	var warned []string
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", StateDir: state, Commit: fakeCommit(nil),
		Warnf: collectFormatted(&warned),
	})
	if err == nil {
		t.Fatal("Apply must fail on the conflict")
	}
	if res == nil {
		t.Fatal("Apply must return the partial Result alongside the conflict error")
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0].Entry.Target != "busy" || res.Conflicts[0].Kind != planner.ConflictForeignEntity {
		t.Errorf("Conflicts = %+v, want the structured busy conflict", res.Conflicts)
	}
	if len(res.Entries) != 2 {
		t.Errorf("Entries = %+v, want the full inventory", res.Entries)
	}
	if len(res.Unreached) != 1 || res.Unreached[0] != "free" {
		t.Errorf("Unreached = %v, want [free] (nothing ran)", res.Unreached)
	}
	if res.FailedTarget != "" {
		t.Errorf("FailedTarget = %q, want empty (the conflict rides Conflicts, not FailedTarget)", res.FailedTarget)
	}
	// Nothing was placed: the conflict stops the run before any FS action.
	if _, err := os.Lstat(filepath.Join(root, "free")); !os.IsNotExist(err) {
		t.Errorf("free must not be placed on a conflict stop, lstat err = %v", err)
	}
}

// TestApplyRecordsReplacedDestsAndRemovalEntries verifies the #131 change-info data sources:
// a re-linked target records the dest it pointed at before this run (ReplacedDests), and a
// dropped entry's previous-generation record lands in RemovalEntries.
func TestApplyRecordsReplacedDestsAndRemovalEntries(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src1 := makeSrc(t, "conf/rc")
	src2 := makeSrc(t, "conf/rc")

	lf1 := writeLinkFarm(t, fixedManifest(root, storeEntry(src1, "conf", "link"), storeEntry(src1, "conf", "drop")))
	gen := 0
	if _, err := Apply(Options{LinkFarm: lf1, Name: "c", StateDir: state, Commit: fakeGenCommit(&gen)}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	lf2 := writeLinkFarm(t, fixedManifest(root, storeEntry(src2, "conf", "link")))
	res, err := Apply(Options{LinkFarm: lf2, Name: "c", StateDir: state, Commit: fakeGenCommit(&gen)})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	wantOld := filepath.Join(src1, "conf")
	if got := res.ReplacedDests["link"]; got != wantOld {
		t.Errorf("ReplacedDests[link] = %q, want %q (the pre-re-link dest)", got, wantOld)
	}
	if len(res.RemovalEntries) != 1 || res.RemovalEntries[0].Target != "drop" || res.RemovalEntries[0].Src != src1 {
		t.Errorf("RemovalEntries = %+v, want the dropped entry's previous-generation record", res.RemovalEntries)
	}
	if res.Profile == "" {
		t.Error("Profile = empty, want the profile link path for generation.profile")
	}
}

// TestResetPartialFailureReturnsPartial verifies Reset's mid-teardown failure contract: the
// symlink half already removed keeps its record, the failing copy target is FailedTarget, the
// never-attempted copy is Unreached, and Entries carries the selected inventory (→ issue #131).
func TestResetPartialFailureReturnsPartial(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; read-only directory does not block")
	}
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "conf/rc")

	lf := writeLinkFarm(t, fixedManifest(root,
		storeEntry(src, "conf", "s"),
		copyEntry(src, "conf", "boxed/c1"),
		copyEntry(src, "conf", "later/c2"),
	))
	gen := 0
	if _, err := Apply(Options{LinkFarm: lf, Name: "c", StateDir: state, Commit: fakeGenCommit(&gen)}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Freeze c1's parent so its RemoveAll fails; c2 must then never be attempted.
	boxed := filepath.Join(root, "boxed")
	if err := os.Chmod(boxed, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(boxed, 0o755) })

	var warned []string
	res, err := Reset(ResetOptions{
		Name: "c", RootKind: manifest.RootKindFixed, FixedRoot: root, StateDir: state,
		Warnf: collectFormatted(&warned),
	})
	if err == nil {
		t.Fatal("Reset must fail on the frozen copy target")
	}
	if res == nil {
		t.Fatal("Reset must return the partial ResetResult alongside the error")
	}
	if len(res.Entries) != 3 {
		t.Errorf("Entries = %+v, want the selected inventory", res.Entries)
	}
	if len(res.RemovedSymlinks) != 1 || res.RemovedSymlinks[0] != "s" {
		t.Errorf("RemovedSymlinks = %v, want [s] (completed before the failure)", res.RemovedSymlinks)
	}
	if len(res.RemovedCopies) != 0 {
		t.Errorf("RemovedCopies = %v, want none (the first copy removal failed)", res.RemovedCopies)
	}
	if res.FailedTarget != "boxed/c1" {
		t.Errorf("FailedTarget = %q, want boxed/c1", res.FailedTarget)
	}
	if len(res.Unreached) != 1 || res.Unreached[0] != "later/c2" {
		t.Errorf("Unreached = %v, want [later/c2]", res.Unreached)
	}
}
