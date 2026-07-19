// Package engine is the placement core: it resolves root, fixes the profileDir
// layout, takes a flock, places store-symlinks via native FS ops, removes stale
// links conservatively, and commits a generation with `nix-env --set`
// (→ ADR-0002, ADR-0005, ADR-0006, ADR-0011, ADR-0013, ADR-0015, ADR-0025).
//
// The minimal core of this slice (#6) is limited to store-symlink placement in
// project mode. Placement through stale removal (native FS) is made unit/integration
// testable without nix, and commit (nix-env --set) is injectable so tmpdir tests do
// not call nix (→ ADR-0006).
package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yasunori0418/nput/internal/gitutil"
	"github.com/yasunori0418/nput/internal/lock"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/paths"
	"github.com/yasunori0418/nput/internal/planner"
)

// CommitFunc is the commit point that records a generation after a successful placement
// (→ ADR-0006, docs/spec.md execution flow f). The default is
// nix-env --profile <profileLink> --set <linkFarm>. tmpdir tests substitute this to
// verify without calling nix.
type CommitFunc func(profileLink, linkFarm string) error

// BuildFunc is the callback that builds the link-farm in-lock after the flock is taken
// (→ docs/spec.md execution flow 2b · ADR-0011, ADR-0023). The pending argument is the
// out-link destination (<profileDir>/.pending). The return value is the built link-farm's
// store path (after os.Readlink resolution). The CLI injects
// `nix build <ep>#nput.<system>.<name> --out-link <pending>`. When nil, opts.LinkFarm is
// used as pre-built (tmpdir test path).
type BuildFunc func(pending string) (linkFarm string, err error)

// GitFunc resolves the git toplevel in project mode (→ ADR-0005). The default is gitutil.Toplevel.
type GitFunc func(dir string) (string, error)

// Options is the input to Apply.
type Options struct {
	// LinkFarm is the link-farm directory containing manifest.json and the GC anchor symlink farm.
	// Pre-built link-farm used only on the path that does not pass Build (tmpdir tests) (→ ADR-0011).
	LinkFarm string
	// Name is the config name (uniquely identifies a profile; derived from the entrypoint's nput.<name>).
	Name string
	// RootKind is the root kind obtained via eval pre-resolution (→ docs/spec.md execution flow 1 · ADR-0023).
	// Required on the Build path since the manifest is not yet built. When empty, obtained from LinkFarm's manifest.
	RootKind string
	// FixedRoot is the absolute path when rootKind=fixed (from eval pre-resolution's passthru.root).
	// When empty and Build=nil, LinkFarm's manifest.Root.Root is used.
	FixedRoot string
	// RootOverride is the --root override (empty = none). When set, uses the roothash key in all modes (→ ADR-0023).
	RootOverride string
	// WorkDir is the starting point for project-mode git toplevel resolution (empty = os.Getwd).
	WorkDir string
	// StateDir overrides the profile base <state> (empty = resolved via paths.StateDir · mainly for tests).
	StateDir string
	// NoWait makes the flock a try-lock (shellHook path · ErrSkipped if held · → ADR-0013).
	NoWait bool
	// Recopy is the apply --recopy modifier (unconditionally overwrite/re-copy all copy targets in the config from src · → ADR-0020).
	// An opt-in path that breaks place-once. The normal apply of the symlink part (stale removal + generation commit) is unchanged.
	Recopy bool
	// Backup is the apply --backup modifier: a foreign occupant that would otherwise be a conflict
	// (or a copy foreign skip) is renamed aside to "<target>.<BackupSuffix>" and the entry placed
	// fresh, instead of stopping (→ ADR-0045, issue #169).
	Backup bool
	// BackupSuffix is the apply --backup rename suffix. Empty defaults to "nput-backup" (→ ADR-0045).
	BackupSuffix string
	// DryRun is a side-effect-free read-only preview (apply --dryrun · → ADR-0006, ADR-0023).
	// When true it runs the planner read-only, packs the plan into Result and returns,
	// taking none of FS writes / --set / flock / pending gcroot. It builds (src resolution) but does not place.
	DryRun bool

	// Build substitutes the in-lock build (nil = use opts.LinkFarm as pre-built).
	Build BuildFunc
	// Git substitutes git toplevel resolution (nil = gitutil.Toplevel).
	Git GitFunc
	// Commit substitutes the generation commit (nil = nix-env --set).
	Commit CommitFunc
	// Warnf is the warning output sink (nil = stderr). Used to surface foreign symlinks etc. (→ ADR-0015).
	Warnf func(format string, args ...any)
}

// Result is the result report of Apply (for dryrun / report display · test verification).
type Result struct {
	Root       string   // resolved absolute root path
	ProfileDir string   // the fixed profileDir
	Placed     []string // newly placed symlink targets
	Replaced   []string // targets whose existing symlink was re-linked
	Copied     []string // copy targets newly copied via place-once
	Recopied   []string // existing copy targets overwritten/re-copied by --recopy (→ ADR-0020)
	Removed    []string // stale-removed targets
	Pruned     []string // empty ancestor directories rmdir-ed after a removal (→ Issue #174, #172 (D4))
	BackedUp   []string // targets renamed aside to "<target>.<suffix>" under apply --backup (→ ADR-0045)
	Skipped    bool     // skipped on try-lock contention (NoWait path)
	DryRun     bool     // read-only preview (Placed etc. are "to be placed" plans · → ADR-0023)
	// Conflicts are the planner-detected conflicts, in structured form. Populated on the dryrun
	// path (the CLI decides exit 2 · → ADR-0006) and on the non-dryrun conflict stop, where the
	// partial Result is returned alongside the aggregate error so the CLI can map each conflict
	// onto a failed niface item (E_NPUT_COLLISION · → issue #131, ADR-0043 §6).
	Conflicts []planner.Conflict
	// GenerationSkipped indicates that the project-mode generation skip committed no new
	// generation (omitted --set). The path where the new link-farm equals the previous
	// generation so no commit happens and only drifted entries are lstat-repaired
	// (→ ADR-0005, ADR-0017, docs/spec.md generation skip).
	GenerationSkipped bool

	// Entries is the new manifest's full entry inventory, exposed regardless of whether an
	// entry produced any FS action, so the CLI can list every entry — not just the diff —
	// as niface items (full-inventory · → issue #130, niface ADR-0016).
	Entries []manifest.Entry
	// RemovalEntries are the previous-generation manifest entries behind this run's planned
	// symlink removals (pre-removal migration + stale removal), recorded at plan time regardless
	// of completion — completion is what the Removed list says. They are the old-entry half of
	// the full inventory: the CLI renders items (target/method/subpath) and the recorded old
	// dest for remove changes from them, including entries whose removal failed or was never
	// reached (→ issue #131). A method-change target also present in Entries is shadowed there.
	RemovalEntries []manifest.Entry
	// ReplacedDests records, for each re-linked target in Replaced, the symlink destination
	// that was actually on disk immediately before the re-link (the pre-removal readlink —
	// for a foreign replace this is the foreign dest, not a recorded one), so the CLI can
	// carry the old→new transition in change.info (→ issue #131).
	ReplacedDests map[string]string
	// Warnings are the planner's entry-scoped warnings in structured form (kind + target),
	// for the CLI to map onto niface item/subject warnings. The human-readable stderr text
	// is still emitted through Warnf alongside (→ issue #130, niface ADR-0019).
	Warnings []planner.Warning
	// FailedTarget is the root-relative target of the entry whose FS action failed, "" when
	// the failure was not entry-scoped (build / lock / commit ...). When set, this target's
	// presence in an op list above means the action was attempted, not completed
	// (→ issue #130 到達状態, niface ADR-0016 / ADR-0020).
	FailedTarget string
	// Unreached lists the root-relative targets of planned actions never attempted because
	// an earlier failure stopped the run (the niface "skipped" partition · → issue #130,
	// niface ADR-0020). Empty on success.
	Unreached []string
	// Unwound reports that the undo journal rolled this run's FS writes back after a failure:
	// the op lists above then describe performed-then-reverted actions, not surviving state
	// (→ ADR-0044, issue #130).
	Unwound bool
	// GenBefore / GenAfter are the profile generation numbers observed at run start / end.
	// nil when unobservable — no profile yet (first apply's before), or a profile whose link
	// does not parse as a generation link (→ issue #130, niface ADR-0015 Generation.Before/After).
	// Dryrun observes the same untouched pointer twice, so before == after.
	GenBefore *int
	GenAfter  *int
}

// ErrSkipped indicates a skip on the NoWait path because another apply is in progress.
var ErrSkipped = lock.ErrLocked

// acquireProfileLock takes the profileDir flock and wraps the error, deduplicating the
// acquire→wrap step shared by Apply / Rollback / Reset (→ ADR-0013). The NoWait/ErrSkipped
// branch and defer Release stay with each caller since they differ (Apply short-circuits with
// a Result; Rollback/Reset always block).
func acquireProfileLock(dir string, wait bool) (*lock.Lock, error) {
	l, err := lock.Acquire(dir, wait)
	if err != nil {
		return nil, fmt.Errorf("nput: failed to acquire flock (%s): %w", dir, err)
	}
	return l, nil
}

// Apply places store-symlinks in project mode and commits a generation on success.
// It corresponds to the engine-driven part of docs/spec.md "execution flow" (2. drive the
// engine), and the engine owns the order "flock → in-lock build → placement → --set →
// .pending removal". The build is delegated in-lock to opts.Build (the CLI injects nix
// build); when unspecified, opts.LinkFarm is used as pre-built (tmpdir test path).
func Apply(opts Options) (*Result, error) {
	a := &applier{opts: opts, result: &Result{}}
	if a.opts.Warnf == nil {
		a.opts.Warnf = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}

	// root kind: on the Build path the manifest is not yet built, so use eval-pre-resolved opts.RootKind.
	// On the pre-built LinkFarm path (tests), read the manifest first to obtain rootKind.
	rootKind := opts.RootKind
	fixedRoot := opts.FixedRoot
	if opts.Build == nil {
		m, err := manifest.Load(opts.LinkFarm)
		if err != nil {
			return nil, fmt.Errorf("nput: cannot read the link-farm's manifest (%s): %w", opts.LinkFarm, err)
		}
		a.manifest = m
		if rootKind == "" {
			rootKind = m.Root.RootKind
		}
		if fixedRoot == "" {
			fixedRoot = m.Root.Root
		}
	}

	// 1. resolve root → fix profileDir (→ docs/spec.md "root resolution").
	prof, root, err := ProfileFor(ProfileOptions{
		Name: opts.Name, RootKind: rootKind, FixedRoot: fixedRoot,
		RootOverride: opts.RootOverride, WorkDir: opts.WorkDir, StateDir: opts.StateDir, Git: opts.Git,
	})
	if err != nil {
		return nil, err
	}
	a.root = root
	a.result.Root = root
	a.profile = prof
	a.result.ProfileDir = a.profile.Dir

	// 1.2 observe the profile generation at run start, and again on every return path (deferred),
	//     so Result carries the before/after generation numbers (nil when unobservable — first
	//     apply, or a test-substituted commit whose profile link is not a generation link ·
	//     → issue #130, niface ADR-0015). The dryrun / generation-skip / failure paths never move
	//     the pointer, so they observe before == after without extra branching.
	a.result.GenBefore = observeGeneration(a.profile.Profile)
	defer func() { a.result.GenAfter = observeGeneration(a.profile.Profile) }()

	// 1.5 dryrun is a side-effect-free read-only short-circuit (→ ADR-0006, ADR-0023, docs/spec.md execution flow).
	//     Up to fixing profileDir it is common with apply, but from here on (mkdir / flock / placement / --set /
	//     pending gcroot) nothing is done; the planner is run read-only and the plan is packed into Result and returned.
	if opts.DryRun {
		return a.dryRun()
	}

	// 2. prepare profileDir / backref (the flock opens profileDir, so create it first).
	if err := a.ensureProfileDir(); err != nil {
		return nil, err
	}

	// 3. acquire a flock per resolved profileDir and serialize (→ ADR-0013).
	l, err := acquireProfileLock(a.profile.Dir, !opts.NoWait)
	if err != nil {
		if opts.NoWait && errors.Is(err, lock.ErrLocked) {
			a.result.Skipped = true
			return a.result, ErrSkipped
		}
		return nil, err
	}
	defer func() { _ = l.Release() }()

	// 4. build the link-farm in-lock (→ docs/spec.md execution flow 2b · ADR-0023).
	//    Closing the build inside the lock structurally removes .pending contention among concurrent applies.
	if opts.Build != nil {
		linkFarm, err := opts.Build(a.profile.Pending)
		if err != nil {
			return nil, err
		}
		m, err := manifest.Load(linkFarm)
		if err != nil {
			return nil, fmt.Errorf("nput: cannot read the built link-farm's manifest (%s): %w", linkFarm, err)
		}
		a.opts.LinkFarm = linkFarm
		a.manifest = m
	}

	// 5. read the previous generation's manifest (absent = first run = zero stale removals).
	prev := a.loadPrevManifest()

	// 5.5 expose the new manifest's full entry inventory (not just the diff · → issue #130).
	a.result.Entries = a.manifest.Entries

	// 6. compute the place/replace/remove plan with the planner (pure logic · → internal/planner).
	plan, err := planner.Compute(prev, a.manifest, a.root, planner.OSFS, a.plannerOptions())
	if err != nil {
		return nil, err
	}
	a.recordRemovalPlan(plan)
	if len(plan.Conflicts) > 0 {
		// Return the partial Result (full inventory + structured conflicts + everything-unreached
		// partition via fail), not nil: the CLI needs it to report conflicted entries as failed
		// items and the rest as skipped (→ issue #131, niface ADR-0016 / ADR-0020).
		a.result.Conflicts = plan.Conflicts
		return a.fail(plan, reportConflicts(a.opts.Warnf, plan.Conflicts))
	}
	a.emitWarnings(plan.Warnings, opts.Recopy)

	// 6.5 check out-of-store link target existence just before placement (no dangling · → ADR-0001, ADR-0013).
	//     Closed before any FS change, so on absence it places nothing and stops with an error.
	if err := a.checkOutOfStore(); err != nil {
		return nil, err
	}

	// 7. project-mode generation-skip decision (is the new link-farm derivation the same as the previous generation?).
	//    If the same, commit no new generation (omit --set) and return after lstat-repairing only drifted entries
	//    (not a full no-op). home / fixed / system are excluded and commit a new generation every time
	//    (generation skip is project mode only · → ADR-0005, ADR-0017, docs/spec.md generation skip).
	if rootKind == manifest.RootKindProject && prev != nil {
		same, err := generationUnchanged(a.profile.Profile, a.opts.LinkFarm)
		if err != nil {
			// When the previous generation's link-farm cannot be resolved, fall back to the safe side: normal apply (commit a new generation).
			a.opts.Warnf("nput: could not resolve the previous generation's link-farm; recommitting without a generation skip: %v", err)
		} else if same {
			// The drift repair's re-links are journaled the same as normal placement (→ ADR-0044); this
			// path commits no generation, so there is no commit success to discard the journal on —
			// discard immediately once the repair itself succeeds (nothing here to roll back further).
			if err := a.runJournaled(func() error { return a.repairDrift(plan, opts.Recopy) }); err != nil {
				return nil, err
			}
			a.discardJournal()
			a.result.GenerationSkipped = true
			a.cleanupPending()
			return a.result, nil
		}
	}

	// 8. reflect the plan onto the real FS. PreRemove first (clear whatever self-recorded stale
	//    filesystem object occupies a placement target — an ancestor symlink, a real directory
	//    fully migratable, or a symlink replaced by a symlink→copy method change — so placement
	//    lands on an empty/absent target · local exception to ADR-0006 · → ADR-0046, ADR-0047),
	//    then Backup (apply --backup: rename a foreign occupant aside so placement lands on an
	//    absent target · → ADR-0045), then new / re-link, then stale removal last (→ ADR-0006).
	//    copy branches: on --recopy overwrite all copy targets unconditionally, normally
	//    place-once (new copy only when target is absent) (→ ADR-0020).
	//    Each stage journals its own FS writes; a failure in any of the five unwinds everything this
	//    run has done so far (across all five, not just the failing stage) before returning (→ ADR-0044).
	//    On a stage failure the partial Result is returned alongside the error, carrying the
	//    reached/unreached partition (FailedTarget / Unreached / Unwound) so the CLI can report
	//    how far the run got (→ issue #130 到達状態, niface ADR-0016 / ADR-0020).
	if err := a.runJournaled(func() error { return a.preRemove(plan.PreRemove) }); err != nil {
		return a.fail(plan, err)
	}
	if err := a.runJournaled(func() error { return a.backup(plan.Backup) }); err != nil {
		return a.fail(plan, err)
	}
	if err := a.runJournaled(func() error { return a.place(plan.Place) }); err != nil {
		return a.fail(plan, err)
	}
	if err := a.runJournaled(func() error { return a.materializeCopies(plan, opts.Recopy) }); err != nil {
		return a.fail(plan, err)
	}
	if err := a.runJournaled(func() error { return a.removeStale(plan.Remove) }); err != nil {
		return a.fail(plan, err)
	}

	// 9. generation commit (→ docs/spec.md execution flow 2f). A commit failure is NOT unwound: every
	//    FS write up to this point already succeeded, so there is nothing wrong to roll back — the run
	//    simply fails to advance the generation, and idempotent re-apply converges (→ ADR-0006, ADR-0017,
	//    ADR-0044 §2).
	commit := opts.Commit
	if commit == nil {
		commit = nixEnvCommit
	}
	if err := commit(a.profile.Profile, a.opts.LinkFarm); err != nil {
		// Not entry-scoped (every planned FS action already succeeded), so no FailedTarget /
		// Unreached — but the partial Result is still returned so the CLI can see what landed
		// without a generation advancing (→ issue #130 到達状態).
		return a.result, fmt.Errorf("nput: generation commit (nix-env --set) failed: %w", err)
	}
	a.discardJournal()

	// 10. remove .pending after --set succeeds (the generation link inherits the gcroot · → ADR-0011, ADR-0025).
	a.cleanupPending()

	return a.result, nil
}

// cleanupPending removes the .pending out-link after --set succeeds (or after a generation skip).
// pending is only created on the build path, so it is removed only on that path (→ ADR-0011, ADR-0025).
func (a *applier) cleanupPending() {
	if a.opts.Build == nil {
		return
	}
	if err := os.Remove(a.profile.Pending); err != nil && !os.IsNotExist(err) {
		a.opts.Warnf("nput: could not remove the .pending out-link (%s): %v", a.profile.Pending, err)
	}
}

type applier struct {
	opts     Options
	manifest *manifest.Manifest
	profile  paths.Profile
	root     string
	result   *Result
	journal  []undoOp
}

// plannerOptions translates the apply --backup modifier (opts.Backup / opts.BackupSuffix) into
// planner.Options for planner.Compute (→ ADR-0045, issue #169).
func (a *applier) plannerOptions() planner.Options {
	return planner.Options{Backup: a.opts.Backup, Suffix: a.opts.BackupSuffix}
}

// runJournaled runs an FS-mutating stage and unwinds the journal recorded so far — by this
// stage and any earlier ones in the same Apply/Rollback call — if it fails (→ ADR-0044). Stages
// covered: preRemove, place, materializeCopies, removeStale, repairDrift. A stage succeeding
// simply leaves its journal entries in place for the next stage (or for the final discard on
// commit success); nothing here decides when the journal is discarded — that is the caller's
// job once every stage in the call has succeeded and (for Apply) commit has landed.
func (a *applier) runJournaled(stage func() error) error {
	if err := stage(); err != nil {
		a.unwind(err)
		a.result.Unwound = true
		return err
	}
	return nil
}

// recordRemovalPlan exposes the previous-generation entries behind the plan's symlink
// removals on the Result (RemovalEntries), at plan time so the record covers removals that
// later fail or are never reached (→ issue #131). Rmdir actions carry no manifest entry and
// are skipped. Shared by Apply (normal + dryrun) and Rollback right after planner.Compute.
func (a *applier) recordRemovalPlan(plan planner.Plan) {
	for _, acts := range [][]planner.RemoveAction{plan.PreRemove, plan.Remove} {
		for _, act := range acts {
			if act.Kind != planner.RemoveUnlink {
				continue
			}
			a.result.RemovalEntries = append(a.result.RemovalEntries, act.Entry)
		}
	}
}

// recordReplacedDest records the on-disk symlink destination a re-linked target pointed at
// immediately before this run replaced it (→ Result.ReplacedDests, issue #131).
func (a *applier) recordReplacedDest(target, prevDest string) {
	if a.result.ReplacedDests == nil {
		a.result.ReplacedDests = map[string]string{}
	}
	a.result.ReplacedDests[target] = prevDest
}

// entryFailed records the entry-scoped failure position (result.FailedTarget) when err is
// non-nil, passing err through unchanged so the error-wrap convention (wrap once at the source)
// is untouched (→ issue #130 到達状態). target == "" (an action with no manifest entry, such as
// a RemoveRmdir) records nothing. Only the first failure is recorded; a run stops at it anyway.
func (a *applier) entryFailed(target string, err error) error {
	if err != nil && target != "" && a.result.FailedTarget == "" {
		a.result.FailedTarget = target
	}
	return err
}

// fail finalizes a mid-run stage failure: it fills result.Unreached with the planned-but-never-
// attempted targets (everything in the plan that is neither in a completed-op list nor the
// FailedTarget) and returns the partial Result alongside err, so the CLI can derive the niface
// success/failed/skipped item partition (→ issue #130 到達状態, niface ADR-0016 / ADR-0020).
// Under --recopy the copy execution source is the manifest, not plan.Copies (→ recopyAll), so
// the manifest's copy entries are walked as well.
func (a *applier) fail(plan planner.Plan, err error) (*Result, error) {
	done := map[string]bool{}
	if a.result.FailedTarget != "" {
		done[a.result.FailedTarget] = true
	}
	for _, list := range [][]string{
		a.result.Placed, a.result.Replaced, a.result.Copied, a.result.Recopied,
		a.result.Removed, a.result.BackedUp,
	} {
		for _, t := range list {
			done[t] = true
		}
	}
	unreached := func(target string) {
		if target == "" || done[target] {
			return
		}
		done[target] = true
		a.result.Unreached = append(a.result.Unreached, target)
	}
	for _, r := range plan.PreRemove {
		unreached(r.Entry.Target)
	}
	for _, b := range plan.Backup {
		unreached(b.Entry.Target)
	}
	for _, p := range plan.Place {
		unreached(p.Entry.Target)
	}
	for _, c := range plan.Copies {
		unreached(c.Entry.Target)
	}
	if a.opts.Recopy && a.manifest != nil {
		for _, e := range a.manifest.Entries {
			if e.Method == manifest.MethodCopy {
				unreached(e.Target)
			}
		}
	}
	for _, r := range plan.Remove {
		unreached(r.Entry.Target)
	}
	return a.result, err
}

// dryRun is the read-only short-circuit of apply --dryrun (→ ADR-0006, ADR-0023). It resolves
// the manifest via build (it builds for src resolution but does not place · does not create a
// pending gcroot), computes place/replace/remove/conflict via planner.Compute against the
// previous generation's manifest, and packs them into Result before returning. It performs none
// of flock / FS writes / --set. Even on a conflict it does not error but records it in
// Result.Conflicts, and the CLI decides exit 2 (→ docs/spec.md exit code table).
func (a *applier) dryRun() (*Result, error) {
	// On the build path (CLI) the manifest is not yet obtained, so resolve src via a read-only build.
	// In dryrun the CLI injects `nix build --no-link --print-out-paths` (no gcroot).
	if a.opts.Build != nil {
		linkFarm, err := a.opts.Build(a.profile.Pending)
		if err != nil {
			return nil, err
		}
		m, err := manifest.Load(linkFarm)
		if err != nil {
			return nil, fmt.Errorf("nput: cannot read the built link-farm's manifest (%s): %w", linkFarm, err)
		}
		a.opts.LinkFarm = linkFarm
		a.manifest = m
	}

	prev := a.loadPrevManifest()
	a.result.Entries = a.manifest.Entries
	plan, err := planner.Compute(prev, a.manifest, a.root, planner.OSFS, a.plannerOptions())
	if err != nil {
		return nil, err
	}
	a.recordRemovalPlan(plan)
	a.emitWarnings(plan.Warnings, a.opts.Recopy)

	a.result.DryRun = true
	for _, p := range plan.Place {
		if p.Kind == planner.PlaceNew {
			a.result.Placed = append(a.result.Placed, p.Entry.Target)
		} else {
			a.result.Replaced = append(a.result.Replaced, p.Entry.Target)
		}
	}
	for _, c := range plan.Copies {
		a.result.Copied = append(a.result.Copied, c.Entry.Target)
	}
	for _, r := range plan.PreRemove {
		if r.Kind == planner.RemoveRmdir {
			// Rmdir actions carry no manifest Entry (nothing was ever recorded for a bare
			// directory); report the absolute directory path instead, matching the real-run
			// convention (result.Pruned is always absolute — staleremove.go's preRemove /
			// pruneEmptyAncestors · → ADR-0047, issue #175).
			a.result.Pruned = append(a.result.Pruned, r.TargetAbs)
			continue
		}
		a.result.Removed = append(a.result.Removed, r.Entry.Target)
	}
	for _, r := range plan.Remove {
		a.result.Removed = append(a.result.Removed, r.Entry.Target)
	}
	for _, b := range plan.Backup {
		a.result.BackedUp = append(a.result.BackedUp, b.Entry.Target)
	}
	a.result.Conflicts = plan.Conflicts
	return a.result, nil
}

// resolveRoot resolves the absolute placement root from rootKind (+ the absolute path when
// fixed root) (→ docs/spec.md "root resolution"). Pure resolution logic shared by Apply /
// Rollback / ProfileFor; on `--root` override it uses the override path regardless of kind.
func resolveRoot(rootKind, fixedRoot, rootOverride, workDir string, git GitFunc) (string, error) {
	if rootOverride != "" {
		return filepath.Abs(rootOverride)
	}
	switch rootKind {
	case manifest.RootKindProject:
		dir := workDir
		if dir == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("nput: cannot get cwd: %w", err)
			}
			dir = cwd
		}
		if git == nil {
			git = gitutil.Toplevel
		}
		return git(dir)
	case manifest.RootKindHome:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("nput: cannot resolve $HOME: %w", err)
		}
		return home, nil
	case manifest.RootKindFixed:
		if fixedRoot == "" {
			return "", fmt.Errorf("nput: rootKind=fixed but no root path provided")
		}
		return filepath.Abs(fixedRoot)
	case manifest.RootKindSystem:
		return "", fmt.Errorf("nput: root = systemRoot (system mode) is not implemented (→ ADR-0013)")
	case "":
		return "", fmt.Errorf("nput: rootKind is undetermined (eval prefetch or a manifest is required)")
	default:
		return "", fmt.Errorf("nput: unknown rootKind: %q", rootKind)
	}
}

func (a *applier) ensureProfileDir() error {
	if err := os.MkdirAll(a.profile.Dir, 0o755); err != nil {
		return fmt.Errorf("nput: cannot create profileDir (%s): %w", a.profile.Dir, err)
	}
	// Place backref .root at the <roothash> level (reverse-lookup seam for orphan profiles · → ADR-0013).
	if a.profile.Backref != "" {
		if err := os.MkdirAll(a.profile.BackrefDir, 0o755); err != nil {
			return fmt.Errorf("nput: cannot create backref directory (%s): %w", a.profile.BackrefDir, err)
		}
		if err := os.WriteFile(a.profile.Backref, []byte(a.root+"\n"), 0o644); err != nil {
			return fmt.Errorf("nput: cannot write backref (%s): %w", a.profile.Backref, err)
		}
	}
	return nil
}

// loadPrevManifest reads the manifest.json pointed at by profileDir/profile (the symlink to
// the previous generation's link-farm). On the first run (profile absent) it returns nil
// (zero removal targets · → ADR-0006).
func (a *applier) loadPrevManifest() *manifest.Manifest {
	if _, err := os.Stat(a.profile.Profile); err != nil {
		return nil
	}
	m, err := manifest.Load(a.profile.Profile)
	if err != nil {
		// Even if the previous generation cannot be read, do not block new placement (just give up stale removal).
		a.opts.Warnf("nput: could not read the previous generation's manifest; skipping stale removal: %v", err)
		return nil
	}
	return m
}

// reportConflicts lists every planner-detected conflict to stderr (warnf), each followed by a
// one-line guidance for that conflict's kind, then returns a single count-bearing aggregate error
// (exit code stays 1 · → docs/spec.md エラー仕様, grilling 2026-07-12 D6). Mirrors the dryrun path
// (a.dryRun packs the same plan.Conflicts into Result.Conflicts in full), so apply / Rollback no
// longer stop at the first conflict only.
func reportConflicts(warnf func(format string, args ...any), conflicts []planner.Conflict) error {
	for _, c := range conflicts {
		warnf("nput: conflict: %s (target: %s)", c.Reason, c.Entry.Target)
		warnf("nput:   → %s", conflictGuidance(c.Kind))
	}
	return fmt.Errorf("nput: %d conflict(s) detected; stopped without placing (see above)", len(conflicts))
}

// conflictGuidance returns the one-line remediation hint for a conflict kind (→ docs/spec.md
// エラー仕様, grilling 2026-07-12 D6; HM checkLinkTargets に倣う).
func conflictGuidance(kind planner.ConflictKind) string {
	switch kind {
	case planner.ConflictForeignEntity:
		return "move or remove the existing file/directory manually, then re-apply"
	case planner.ConflictForeignAncestor:
		return "check what created this symlink (another config / tool / manual placement)"
	case planner.ConflictSelfContradictoryAncestor:
		return "the manifest keeps both this ancestor and an entry nested beneath it; fix the entry definitions"
	case planner.ConflictCopyStructureMismatch:
		return "the copy entry's structure no longer matches the existing target; fix the entry definition"
	case planner.ConflictDirMigrationFailed:
		return "move or remove the directory's non-migratable contents manually (or use --backup to back up the whole directory), then re-apply"
	case planner.ConflictBackupTargetExists:
		return "a previous --backup was left at \"<target>.<suffix>\"; move or remove it manually, then re-run --backup"
	default:
		return "review the entry definition and the existing target"
	}
}

// emitWarnings emits the non-fatal warnings computed by the planner to stderr (opts.Warnf) and
// records them in structured form on the Result (kind + target · → issue #130, niface ADR-0019),
// so the CLI has the same warnings as data for the machine channel while the human text keeps
// streaming. Warnings are always emitted, regardless of the silent-on-success default or -v
// (→ docs/spec.md stream discipline · ADR-0015, ADR-0024, ADR-0031).
// When recopy=true it suppresses the copy foreign skip warning on both channels (recopy
// overwrites foreign too, so "skipped" would be a false report · → ADR-0020).
func (a *applier) emitWarnings(ws []planner.Warning, recopy bool) {
	for _, w := range ws {
		switch w.Kind {
		case planner.WarnForeignReplace:
			a.opts.Warnf("nput: overwriting an unrecorded symlink (foreign; last-wins): %s", w.Target)
		case planner.WarnStaleMismatch:
			a.opts.Warnf("nput: keeping stale symlink because it mismatches the record: %s", w.Target)
		case planner.WarnStaleNonSymlink:
			a.opts.Warnf("nput: keeping stale target because it is not a symlink: %s", w.Target)
		case planner.WarnCopyOrphan:
			a.opts.Warnf("nput: copy entry vanished but the target is not removed (orphan; clear it with reset): %s", w.Target)
		case planner.WarnCopyForeign:
			if recopy {
				continue
			}
			a.opts.Warnf("nput: skipped copy because a real file already exists at the copy target (foreign; place-once): %s", w.Target)
		}
		a.result.Warnings = append(a.result.Warnings, w)
	}
}
