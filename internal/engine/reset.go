package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yasunori0418/nput/internal/lock"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/planner"
)

// reset is an FS-only teardown that reverts placed entities to a not-placed state (→ ADR-0020,
// ADR-0021, ADR-0025, docs/spec.md "recopy · reset").
//
//   - symlink: the same conservative invariant as stale removal (delete only symlinks that the
//     previous generation's manifest recorded and that currently still point at the recorded dest.
//     foreign / record mismatches are kept with a warning). Reuses planner.Compute (next=nil) +
//     staleremove (finished modules · → planner, staleremove.go).
//   - copy target: deleted (the only explicit means to remove a copy · the risk of deleting a
//     pre-existing file is guarded by the CLI's confirmation).
//   - profile / generations are untouched. As long as the config keeps the entry it is re-placed
//     on the next apply (transient).
//
// The source of teardown-target entries is the **previous generation's manifest (the manifest.json
// of the link-farm that profileDir/profile points at)**. This is "the truth of what nput actually
// placed (recorded)" and matches the conservative invariant's "recorded dest" (rebuilding the
// config would diverge from the recorded dest under src drift and misjudge, so the recorded
// previous generation is used). The CLI handles the rootKind pre-resolution eval (fixing profileDir),
// and entries are read from this previous generation's manifest.

// ResetOptions is the input to Reset. Reset does not build (it reads the previous generation's manifest).
type ResetOptions struct {
	Name         string
	RootKind     string
	FixedRoot    string
	RootOverride string
	WorkDir      string
	StateDir     string
	Git          GitFunc

	// Targets narrows the teardown-target targets (root-relative · empty = all entries).
	// Specifying a target not present in the previous generation's manifest is an error.
	Targets []string
	// DryRun is a side-effect-free preview (just computes and returns the removal targets · no flock / confirm / FS deletion · → ADR-0021).
	DryRun bool
	// Confirm is the confirmation callback before performing deletion (nil = run without confirmation · --yes path / dryrun).
	// It is passed the computed plan; returning false aborts (Result.Aborted = true). The CLI handles the TTY prompt.
	Confirm func(*ResetResult) (bool, error)
	// Warnf is the warning output sink (nil = stderr). Used to surface kept foreign symlinks etc.
	Warnf func(format string, args ...any)
}

// ResetResult is the result of Reset (dryrun is a preview · at run time the actual deletion result).
type ResetResult struct {
	Root            string   // resolved absolute root
	ProfileDir      string   // the fixed profileDir
	RemovedSymlinks []string // removed (in dryrun, to-be-removed) symlink targets
	RemovedCopies   []string // removed (in dryrun, to-be-removed) copy targets
	KeptForeign     []string // symlink targets kept for not satisfying the conservative invariant (foreign / record mismatch)
	DryRun          bool     // was a read-only preview
	Aborted         bool     // aborted at the confirmation prompt
}

// Reset reverts the placed entities of the target entries to a not-placed state. It shares with the
// CLI the non-build command preamble of docs/spec.md "execution flow" (rootKind pre-resolution eval →
// root resolution → fixing profileDir), and the engine side owns profileDir resolution · blocking
// flock · reading the previous generation's manifest · conservative symlink removal + copy deletion
// (→ ADR-0021, ADR-0024).
func Reset(opts ResetOptions) (*ResetResult, error) {
	warnf := opts.Warnf
	if warnf == nil {
		warnf = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}

	// 1. fix profileDir (resolve root → layout · preamble shared with apply / rollback · → ADR-0024).
	prof, root, err := ProfileFor(ProfileOptions{
		Name: opts.Name, RootKind: opts.RootKind, FixedRoot: opts.FixedRoot,
		RootOverride: opts.RootOverride, WorkDir: opts.WorkDir, StateDir: opts.StateDir, Git: opts.Git,
	})
	if err != nil {
		return nil, err
	}
	res := &ResetResult{Root: root, ProfileDir: prof.Dir, DryRun: opts.DryRun}

	// If profile (the previous generation link) is absent, apply has never run = a no-op with zero teardown targets.
	if _, err := os.Stat(prof.Profile); err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return nil, fmt.Errorf("nput: cannot check profile (%s): %w", prof.Profile, err)
	}

	// 2. at run time, serialize with concurrent apply / reset via a blocking flock (→ ADR-0013, ADR-0021).
	//    dryrun is read-only, so it does not take a flock.
	if !opts.DryRun {
		l, err := lock.Acquire(prof.Dir, true)
		if err != nil {
			return nil, fmt.Errorf("nput: failed to acquire flock (%s): %w", prof.Dir, err)
		}
		defer func() { _ = l.Release() }()
	}

	// 3. read the previous generation's manifest (the recorded truth) and narrow the target entries.
	prev, err := manifest.Load(prof.Profile)
	if err != nil {
		return nil, err
	}
	entries, err := selectResetEntries(prev.Entries, opts.Targets)
	if err != nil {
		return nil, err
	}

	// 4. for symlinks, compute a conservative-invariant removal plan with the planner (next=nil makes all targets remove candidates).
	//    copy is a region the planner never removes, so handle it separately.
	var symEntries, copyEntries []manifest.Entry
	for _, e := range entries {
		if e.Method == manifest.MethodCopy {
			copyEntries = append(copyEntries, e)
		} else {
			symEntries = append(symEntries, e)
		}
	}
	symManifest := &manifest.Manifest{SchemaVersion: prev.SchemaVersion, Root: prev.Root, Entries: symEntries}
	plan, err := planner.Compute(symManifest, nil, root, planner.OSFS)
	if err != nil {
		return nil, err
	}

	// 5. make the existing copy targets removal candidates (absent ones are a no-op · → docs/spec.md error spec).
	copyTargets := make([]string, 0, len(copyEntries))
	for _, e := range copyEntries {
		targetAbs := filepath.Join(root, filepath.Clean(e.Target))
		if _, err := os.Lstat(targetAbs); err == nil {
			copyTargets = append(copyTargets, targetAbs)
			res.RemovedCopies = append(res.RemovedCopies, e.Target)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("nput: cannot lstat copy target (%s): %w", targetAbs, err)
		}
	}

	// For the preview (dryrun / confirm display), pack the to-be-removed symlinks and the kept foreign.
	for _, a := range plan.Remove {
		res.RemovedSymlinks = append(res.RemovedSymlinks, a.Entry.Target)
	}
	for _, w := range plan.Warnings {
		if w.Kind == planner.WarnStaleMismatch || w.Kind == planner.WarnStaleNonSymlink {
			res.KeptForeign = append(res.KeptForeign, w.Target)
		}
	}

	// 6. dryrun returns the computed plan and finishes (no FS deletion · no confirm · → ADR-0021).
	if opts.DryRun {
		return res, nil
	}

	// 7. confirmation (data-loss risk · → ADR-0020, ADR-0025). The CLI handles the TTY prompt / --yes.
	if opts.Confirm != nil {
		proceed, err := opts.Confirm(res)
		if err != nil {
			return nil, err
		}
		if !proceed {
			res.Aborted = true
			return res, nil
		}
	}

	// 8. reflect onto the real FS. For symlinks reuse staleremove (with post-plan drift re-verification),
	//    and delete copy targets. Emit warnings for the kept foreign.
	a := &applier{opts: Options{Warnf: warnf}, result: &Result{Root: root, ProfileDir: prof.Dir}}
	a.profile = prof
	a.root = root
	a.emitWarnings(plan.Warnings, false)
	if err := a.removeStale(plan.Remove); err != nil {
		return nil, err
	}
	res.RemovedSymlinks = a.result.Removed // actually removed (excludes those kept due to drift)

	removedCopies := make([]string, 0, len(copyTargets))
	for i, targetAbs := range copyTargets {
		if err := os.RemoveAll(targetAbs); err != nil {
			return nil, fmt.Errorf("nput: cannot remove copy target (%s): %w", targetAbs, err)
		}
		removedCopies = append(removedCopies, res.RemovedCopies[i])
	}
	res.RemovedCopies = removedCopies

	return res, nil
}

// selectResetEntries narrows the previous generation's manifest entries by Targets (empty = all entries).
// If a specified target does not exist in the previous generation, it is an error (a target nput did not place is not a teardown target).
func selectResetEntries(entries []manifest.Entry, targets []string) ([]manifest.Entry, error) {
	if len(targets) == 0 {
		return entries, nil
	}
	byTarget := make(map[string]manifest.Entry, len(entries))
	for _, e := range entries {
		byTarget[e.Target] = e
	}
	out := make([]manifest.Entry, 0, len(targets))
	var unknown []string
	for _, t := range targets {
		key := filepath.Clean(t)
		e, ok := byTarget[key]
		if !ok {
			unknown = append(unknown, t)
			continue
		}
		out = append(out, e)
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("nput: reset target not found in the previous generation's manifest (not a target nput placed): %v", unknown)
	}
	return out, nil
}
