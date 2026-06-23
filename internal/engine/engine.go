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
	Skipped    bool     // skipped on try-lock contention (NoWait path)
	DryRun     bool     // read-only preview (Placed etc. are "to be placed" plans · → ADR-0023)
	Conflicts  []string // conflicts detected in dryrun ("target: reason" · used by the CLI to decide exit 2 · → ADR-0006)
	// GenerationSkipped indicates that the project-mode generation skip committed no new
	// generation (omitted --set). The path where the new link-farm equals the previous
	// generation so no commit happens and only drifted entries are lstat-repaired
	// (→ ADR-0005, ADR-0017, docs/spec.md generation skip).
	GenerationSkipped bool
}

// ErrSkipped indicates a skip on the NoWait path because another apply is in progress.
var ErrSkipped = lock.ErrLocked

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
			return nil, err
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
	root, err := a.resolveRoot(rootKind, fixedRoot)
	if err != nil {
		return nil, err
	}
	a.root = root
	a.result.Root = root

	stateDir := opts.StateDir
	if stateDir == "" {
		stateDir, err = paths.StateDir()
		if err != nil {
			return nil, err
		}
	}
	a.profile = paths.Resolve(stateDir, opts.Name, rootKind, root, opts.RootOverride != "")
	a.result.ProfileDir = a.profile.Dir

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
	l, err := lock.Acquire(a.profile.Dir, !opts.NoWait)
	if err != nil {
		if opts.NoWait && err == lock.ErrLocked {
			a.result.Skipped = true
			return a.result, ErrSkipped
		}
		return nil, fmt.Errorf("nput: flock の取得に失敗しました (%s): %w", a.profile.Dir, err)
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
			return nil, err
		}
		a.opts.LinkFarm = linkFarm
		a.manifest = m
	}

	// 5. read the previous generation's manifest (absent = first run = zero stale removals).
	prev := a.loadPrevManifest()

	// 6. compute the place/replace/remove plan with the planner (pure logic · → internal/planner).
	plan, err := planner.Compute(prev, a.manifest, a.root, planner.OSFS)
	if err != nil {
		return nil, err
	}
	if len(plan.Conflicts) > 0 {
		c := plan.Conflicts[0]
		return nil, fmt.Errorf("nput: %s (target: %s)", c.Reason, c.Entry.Target)
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
			a.opts.Warnf("nput: 前世代 link-farm を解決できませんでした。世代スキップせず再コミットします: %v", err)
		} else if same {
			if err := a.repairDrift(plan, opts.Recopy); err != nil {
				return nil, err
			}
			a.result.GenerationSkipped = true
			a.cleanupPending()
			return a.result, nil
		}
	}

	// 8. reflect the plan onto the real FS (new / re-link first, stale removal last · → ADR-0006).
	//    symlinks are plan-driven. copy branches: on --recopy overwrite all copy targets unconditionally,
	//    normally place-once (new copy only when target is absent) (→ ADR-0020).
	if err := a.place(plan.Place); err != nil {
		return nil, err
	}
	if err := a.materializeCopies(plan, opts.Recopy); err != nil {
		return nil, err
	}
	if err := a.removeStale(plan.Remove); err != nil {
		return nil, err
	}

	// 9. generation commit (→ docs/spec.md execution flow 2f).
	commit := opts.Commit
	if commit == nil {
		commit = nixEnvCommit
	}
	if err := commit(a.profile.Profile, a.opts.LinkFarm); err != nil {
		return nil, fmt.Errorf("nput: 世代コミット（nix-env --set）に失敗しました: %w", err)
	}

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
		a.opts.Warnf("nput: .pending out-link を削除できませんでした (%s): %v", a.profile.Pending, err)
	}
}

type applier struct {
	opts     Options
	manifest *manifest.Manifest
	profile  paths.Profile
	root     string
	result   *Result
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
			return nil, err
		}
		a.opts.LinkFarm = linkFarm
		a.manifest = m
	}

	prev := a.loadPrevManifest()
	plan, err := planner.Compute(prev, a.manifest, a.root, planner.OSFS)
	if err != nil {
		return nil, err
	}
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
	for _, r := range plan.Remove {
		a.result.Removed = append(a.result.Removed, r.Entry.Target)
	}
	for _, c := range plan.Conflicts {
		a.result.Conflicts = append(a.result.Conflicts, fmt.Sprintf("%s: %s", c.Entry.Target, c.Reason))
	}
	return a.result, nil
}

func (a *applier) resolveRoot(rootKind, fixedRoot string) (string, error) {
	return resolveRoot(rootKind, fixedRoot, a.opts.RootOverride, a.opts.WorkDir, a.opts.Git)
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
				return "", fmt.Errorf("nput: cwd を取得できません: %w", err)
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
			return "", fmt.Errorf("nput: $HOME を解決できません: %w", err)
		}
		return home, nil
	case manifest.RootKindFixed:
		if fixedRoot == "" {
			return "", fmt.Errorf("nput: rootKind=fixed なのに root パスがありません")
		}
		return filepath.Abs(fixedRoot)
	case manifest.RootKindSystem:
		return "", fmt.Errorf("nput: root = systemRoot (system mode) は未実装です（→ ADR-0013）")
	case "":
		return "", fmt.Errorf("nput: rootKind が未確定です（eval 先取りまたは manifest が必要）")
	default:
		return "", fmt.Errorf("nput: 未知の rootKind: %q", rootKind)
	}
}

func (a *applier) ensureProfileDir() error {
	if err := os.MkdirAll(a.profile.Dir, 0o755); err != nil {
		return fmt.Errorf("nput: profileDir を作成できません (%s): %w", a.profile.Dir, err)
	}
	// Place backref .root at the <roothash> level (reverse-lookup seam for orphan profiles · → ADR-0013).
	if a.profile.Backref != "" {
		if err := os.MkdirAll(a.profile.BackrefDir, 0o755); err != nil {
			return fmt.Errorf("nput: backref ディレクトリを作成できません (%s): %w", a.profile.BackrefDir, err)
		}
		if err := os.WriteFile(a.profile.Backref, []byte(a.root+"\n"), 0o644); err != nil {
			return fmt.Errorf("nput: backref を書けません (%s): %w", a.profile.Backref, err)
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
		a.opts.Warnf("nput: 前世代 manifest を読めませんでした。stale 除去をスキップします: %v", err)
		return nil
	}
	return m
}

// emitWarnings emits the non-fatal warnings computed by the planner to stderr (opts.Warnf).
// Warnings are not silenced even by --quiet (→ docs/spec.md stream discipline · ADR-0015, ADR-0024).
// When recopy=true it suppresses the copy foreign skip warning (recopy overwrites foreign too, so
// "skipped" would be a false report · → ADR-0020).
func (a *applier) emitWarnings(ws []planner.Warning, recopy bool) {
	for _, w := range ws {
		switch w.Kind {
		case planner.WarnForeignReplace:
			a.opts.Warnf("nput: 記録の無い symlink を上書きします (foreign・後勝ち): %s", w.Target)
		case planner.WarnStaleMismatch:
			a.opts.Warnf("nput: stale symlink が記録と不一致のため残します: %s", w.Target)
		case planner.WarnStaleNonSymlink:
			a.opts.Warnf("nput: stale target が symlink ではないため残します: %s", w.Target)
		case planner.WarnCopyOrphan:
			a.opts.Warnf("nput: copy entry が消えましたが target は削除しません（orphan・reset で撤去）: %s", w.Target)
		case planner.WarnCopyForeign:
			if recopy {
				continue
			}
			a.opts.Warnf("nput: copy target に既存の実ファイルがあるため copy をスキップしました（foreign・place-once）: %s", w.Target)
		}
	}
}
