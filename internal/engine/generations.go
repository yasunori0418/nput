package engine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/yasunori0418/nput/internal/lock"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/paths"
	"github.com/yasunori0418/nput/internal/planner"
)

// Generation operations are unified under the `nix-env --profile <profileDir>/profile`
// family (→ docs/spec.md generation management spec · ADR-0015, ADR-0025). Commit (--set)
// lives in commit.go; here we handle rollback (--switch-generation) and listing
// (--list-generations). Like --set, these are injectable so tmpdir tests do not call nix.

// Generation is one generation of a profile (one line of nix-env --list-generations).
type Generation struct {
	Number  int    // generation number
	Date    string // creation timestamp (nix-env's display verbatim; carried unparsed)
	Current bool   // whether it is the current generation (the one the profile link points at)
}

// ListGenerationsFunc is the generation-list retrieval point (default nix-env --list-generations). Substituted in tmpdir tests.
type ListGenerationsFunc func(profileLink string) ([]Generation, error)

// SwitchGenerationFunc is the profile pointer move point (default nix-env --switch-generation). Substituted in tmpdir tests.
type SwitchGenerationFunc func(profileLink string, gen int) error

// ProfileOptions is the input for fixing the profileDir (shared by Rollback / list-generations).
type ProfileOptions struct {
	Name         string
	RootKind     string
	FixedRoot    string
	RootOverride string
	WorkDir      string
	StateDir     string
	Git          GitFunc
}

// ProfileFor resolves root and fixes the profileDir layout (same shape as apply's preamble · → ADR-0023, ADR-0024).
// Used in common by non-building rollback / list-generations for flock / generation reads.
func ProfileFor(opts ProfileOptions) (paths.Profile, string, error) {
	root, err := resolveRoot(opts.RootKind, opts.FixedRoot, opts.RootOverride, opts.WorkDir, opts.Git)
	if err != nil {
		return paths.Profile{}, "", err
	}
	stateDir := opts.StateDir
	if stateDir == "" {
		stateDir, err = paths.StateDir()
		if err != nil {
			return paths.Profile{}, "", err
		}
	}
	prof := paths.Resolve(stateDir, opts.Name, opts.RootKind, root, opts.RootOverride != "")
	return prof, root, nil
}

// ListGenerations returns the generation list for profileLink (default nix-env implementation). Used by the CLI's list-generations.
func ListGenerations(profileLink string) ([]Generation, error) {
	return nixEnvListGenerations(profileLink)
}

// RollbackOptions is the input to Rollback. Rollback is home mode only, but that decision is
// the CLI's; the engine resolves the profileDir regardless of rootKind and converges to the previous generation.
type RollbackOptions struct {
	Name         string
	RootKind     string
	FixedRoot    string
	RootOverride string
	WorkDir      string
	StateDir     string

	// ListGenerations / SwitchGeneration substitute the nix-env calls (nil = default nix-env implementation).
	ListGenerations  ListGenerationsFunc
	SwitchGeneration SwitchGenerationFunc
	// Git substitutes git toplevel resolution (nil = gitutil.Toplevel). Unused in home mode.
	Git GitFunc
	// Warnf is the warning output sink (nil = stderr).
	Warnf func(format string, args ...any)
}

// RollbackResult is the result report of Rollback. It augments Result (placement diff) with the generation transition From→To.
type RollbackResult struct {
	Result
	From int // current generation N before rolling back
	To   int // previous generation N-1 rolled back to
}

// Rollback reverts a home-mode profile to one generation earlier. A profile pointer move alone
// does not change the FS at an arbitrary root, so it diffs with the planner using current
// generation N as baseline and previous generation N-1 as target, conservatively stale-removes
// N∖N-1, re-places N-1's entries, and only **last** moves the profile pointer (→ docs/spec.md
// rollback · ADR-0015). Moving the pointer first would shift the baseline to N-2 and corrupt
// stale removal, so FS convergence comes first and the pointer move last.
func Rollback(opts RollbackOptions) (*RollbackResult, error) {
	warnf := opts.Warnf
	if warnf == nil {
		warnf = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}
	listFn := opts.ListGenerations
	if listFn == nil {
		listFn = nixEnvListGenerations
	}
	switchFn := opts.SwitchGeneration
	if switchFn == nil {
		switchFn = nixEnvSwitchGeneration
	}

	// 1. fix profileDir (resolve root → layout · preamble shared with apply).
	prof, root, err := ProfileFor(ProfileOptions{
		Name: opts.Name, RootKind: opts.RootKind, FixedRoot: opts.FixedRoot,
		RootOverride: opts.RootOverride, WorkDir: opts.WorkDir, StateDir: opts.StateDir, Git: opts.Git,
	})
	if err != nil {
		return nil, err
	}

	// If profileDir is absent, apply has never run → no generation to roll back to.
	if _, err := os.Stat(prof.Dir); err != nil {
		return nil, fmt.Errorf("nput: profile がありません（一度も apply していません）: %s", prof.Dir)
	}

	// 2. serialize with concurrent apply / rollback via a blocking flock (→ ADR-0013).
	l, err := lock.Acquire(prof.Dir, true)
	if err != nil {
		return nil, fmt.Errorf("nput: flock の取得に失敗しました (%s): %w", prof.Dir, err)
	}
	defer func() { _ = l.Release() }()

	// 3. identify current generation N and previous generation N-1 from the generation list.
	gens, err := listFn(prof.Profile)
	if err != nil {
		return nil, err
	}
	curIdx := -1
	for i, g := range gens {
		if g.Current {
			curIdx = i
		}
	}
	if curIdx < 0 {
		return nil, fmt.Errorf("nput: 現世代を特定できません（profile: %s）", prof.Profile)
	}
	if curIdx == 0 {
		return nil, fmt.Errorf("nput: 前世代がありません（最古の世代のため rollback できません）")
	}
	cur := gens[curIdx]
	prev := gens[curIdx-1]

	// 4. baseline = current generation N's manifest (current FS state) / target = previous generation N-1's manifest.
	baseline, err := manifest.Load(prof.Profile)
	if err != nil {
		return nil, fmt.Errorf("nput: 現世代 manifest を読めません: %w", err)
	}
	target, err := manifest.Load(paths.GenerationLink(prof.Profile, prev.Number))
	if err != nil {
		return nil, fmt.Errorf("nput: 前世代 manifest を読めません (世代 %d): %w", prev.Number, err)
	}

	// 5. compute the plan for N∖N-1 stale removal · N-1 entry re-placement with the planner (reusing the apply engine with (baseline, target) substituted).
	plan, err := planner.Compute(baseline, target, root, planner.OSFS)
	if err != nil {
		return nil, err
	}
	if len(plan.Conflicts) > 0 {
		c := plan.Conflicts[0]
		return nil, fmt.Errorf("nput: %s (target: %s)", c.Reason, c.Entry.Target)
	}

	// 6. reflect the plan onto the real FS (new/re-link first, stale removal last · → ADR-0006).
	a := &applier{opts: Options{Warnf: warnf}, result: &Result{Root: root, ProfileDir: prof.Dir}}
	a.profile = prof
	a.root = root
	a.emitWarnings(plan.Warnings, false)
	if err := a.place(plan.Place); err != nil {
		return nil, err
	}
	if err := a.removeStale(plan.Remove); err != nil {
		return nil, err
	}

	// 7. finally move the profile pointer to N-1 (→ docs/spec.md rollback step 3).
	if err := switchFn(prof.Profile, prev.Number); err != nil {
		return nil, fmt.Errorf("nput: profile ポインタの移動に失敗しました（--switch-generation %d）: %w", prev.Number, err)
	}

	return &RollbackResult{Result: *a.result, From: cur.Number, To: prev.Number}, nil
}

// nixEnvListGenerations is the default generation-list retrieval (nix-env --profile <p> --list-generations).
// It captures and parses stdout (ingesting machine-readable output).
func nixEnvListGenerations(profileLink string) ([]Generation, error) {
	if _, err := exec.LookPath("nix-env"); err != nil {
		return nil, fmt.Errorf("nix-env が PATH にありません: %w", err)
	}
	cmd := exec.Command("nix-env", "--profile", profileLink, "--list-generations")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		trimmed := strings.TrimSpace(stderr.String())
		if trimmed != "" {
			return nil, fmt.Errorf("nput: nix-env --list-generations に失敗しました: %w\n%s", err, trimmed)
		}
		return nil, fmt.Errorf("nput: nix-env --list-generations に失敗しました: %w", err)
	}
	return parseGenerations(stdout.String())
}

// nixEnvSwitchGeneration is the default profile pointer move (nix-env --profile <p> --switch-generation <gen>).
// nix output is routed to stderr (stdout is reserved for machine-readable output · → docs/spec.md stream discipline).
func nixEnvSwitchGeneration(profileLink string, gen int) error {
	if _, err := exec.LookPath("nix-env"); err != nil {
		return fmt.Errorf("nix-env が PATH にありません: %w", err)
	}
	cmd := exec.Command("nix-env", "--profile", profileLink, "--switch-generation", strconv.Itoa(gen))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// parseGenerations parses the output of `nix-env --list-generations`. Each line is of the
// form "<number>   <timestamp>   [(current)]" (leading and separating whitespace is variable).
func parseGenerations(out string) ([]Generation, error) {
	var gens []Generation
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("nput: 世代一覧の行を解析できません: %q", line)
		}
		g := Generation{Number: n}
		end := len(fields)
		if fields[end-1] == "(current)" {
			g.Current = true
			end--
		}
		g.Date = strings.Join(fields[1:end], " ")
		gens = append(gens, g)
	}
	return gens, nil
}
