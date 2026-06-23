package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/engine"
	"github.com/yasunori0418/nput/internal/manifest"
)

var flagApplyAll bool // --all: apply all of nput.* in lexical order (narrowable by root filter)

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply [name]",
		Short: "Build nput.<name>, create a new generation, and apply it (defaults to nput.default)",
		Long: "Build and place the entrypoint's nput.<name>. " +
			"Omitting name applies nput.default (the flake default convention; an error if undefined). " +
			"--all applies all of nput.* in lexical order; --project-root / --home-root / --system-root narrow by root mode.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagApplyAll {
				if len(args) > 0 {
					return fmt.Errorf("nput: apply cannot combine <name> with --all")
				}
				return runApplyAll()
			}
			if err := ensureNoRootFilter("apply --all"); err != nil {
				return err
			}
			name := "default"
			if len(args) == 1 {
				name = args[0]
			}
			return runApply(name)
		},
	}
	cmd.Flags().BoolVar(&flagApplyAll, "all", false, "Apply all of nput.* in lexical order (continues on partial failure; exits non-zero if any fails)")
	cmd.Flags().BoolVar(&flagRecopy, "recopy", false,
		"Unconditionally re-copy every copy target from src, overwriting (discards local edits; see ADR-0020)")
	cmd.Flags().BoolVar(&flagDryrun, "dryrun", false,
		"Show place/replace/remove/conflict/no-op with zero side effects (exit 2 on conflict; see ADR-0006)")
	cmd.Flags().StringVar(&flagManifest, "manifest", "",
		"Apply a pre-built manifest (link-farm path) directly (host/module activation seam; no entrypoint discovery or nix eval/build; see ADR-0026)")
	return cmd
}

// runApplyManifest applies the pre-built link-farm passed via --manifest directly to the engine
// (the module activation path). It does no entrypoint discovery, no rootKind pre-resolution eval,
// and no nix build; the engine reads rootKind from manifest.json (an HM module pins homeRoot, so home).
// It drives engine.Apply's Build=nil path (a pre-built LinkFarm) from the CLI (→ engine.Options).
func runApplyManifest(name string) error {
	linkFarm, err := filepath.Abs(flagManifest)
	if err != nil {
		return fmt.Errorf("nput: cannot resolve the --manifest path (%s): %w", flagManifest, err)
	}

	res, err := engine.Apply(engine.Options{
		Name:         name,
		LinkFarm:     linkFarm,
		RootOverride: flagRoot,
		NoWait:       flagNoWait,
		Recopy:       flagRecopy,
	})
	if err != nil {
		if errors.Is(err, engine.ErrSkipped) {
			if flagVerbose {
				fmt.Fprintln(os.Stderr, "nput: skipped because another apply is in progress (run nput apply manually)")
			}
			return nil
		}
		return err
	}

	if flagVerbose {
		reportResult(res, name)
	}
	return nil
}

// runApply drives the "execution flow" in docs/spec.md:
// entrypoint discovery → rootKind pre-resolution eval → engine.Apply (flock → in-lock build → place → --set → remove .pending).
// When --manifest is given it does no entrypoint discovery and no nix eval/build, passing the pre-built link-farm
// directly to the engine (the module activation path; → docs/spec.md "per-module behavior spec", ADR-0003, ADR-0007).
func runApply(name string) error {
	if flagManifest != "" {
		// --manifest fixes the source to a link-farm, so it conflicts in meaning with the
		// entrypoint discovery flags (the positional name is orthogonal as a profile selector and coexists; → ADR-0026).
		if flagFile != "" {
			return errors.New("nput: --manifest cannot be combined with -f (--manifest fixes the source to a pre-built link-farm)")
		}
		return runApplyManifest(name)
	}

	ep, err := discoverEntrypoint(flagFile)
	if err != nil {
		return err
	}
	system, err := currentSystem()
	if err != nil {
		return err
	}

	// 1. Pre-resolve rootKind before build (to establish the order profileDir resolution → flock → build; → ADR-0023).
	rootKind, fixedRoot, err := evalRoot(ep, system, name)
	if err != nil {
		return err
	}

	// 1.5 --dryrun is a side-effect-free preview (takes no flock / pending gcroot; runs only build read-only; → ADR-0023).
	if flagDryrun {
		res, err := engine.Apply(engine.Options{
			Name:         name,
			RootKind:     rootKind,
			FixedRoot:    fixedRoot,
			RootOverride: flagRoot,
			Recopy:       flagRecopy,
			DryRun:       true,
			Build:        dryBuildFunc(ep, system, name),
		})
		if err != nil {
			return err
		}
		printApplyPlan(res)
		// exit 2 if there are conflicts (a pre-gate for CI; → docs/spec.md exit code table).
		if len(res.Conflicts) > 0 {
			return &exitError{code: 2}
		}
		return nil
	}

	// 2. Drive the engine (flock acquisition, in-lock build, placement, commit, and .pending removal are owned by the engine).
	res, err := applyOne(ep, system, name, rootKind, fixedRoot)
	if err != nil {
		if errors.Is(err, engine.ErrSkipped) {
			// A try-lock skip is a normal skip (exit 0; → docs/spec.md exit code table).
			if flagVerbose {
				fmt.Fprintln(os.Stderr, "nput: skipped because another apply is in progress (run nput apply manually)")
			}
			return nil
		}
		return err
	}

	if flagVerbose {
		reportResult(res, name)
	}
	return nil
}

// printApplyPlan prints the apply --dryrun plan to stdout (it owns the machine-readable output; one action per line;
// → docs/spec.md stream discipline, ADR-0023, ADR-0024). It is not suppressed even under silent-on-success (the stdout-ownership principle; → ADR-0031).
// conflict lines are also put on stdout as part of the plan, with the exit code (exit 2) complementing machine discrimination.
func printApplyPlan(res *engine.Result) {
	for _, t := range res.Placed {
		fmt.Printf("place\t%s\n", t)
	}
	for _, t := range res.Replaced {
		fmt.Printf("replace\t%s\n", t)
	}
	for _, t := range res.Copied {
		fmt.Printf("copy\t%s\n", t)
	}
	for _, t := range res.Removed {
		fmt.Printf("remove\t%s\n", t)
	}
	for _, c := range res.Conflicts {
		fmt.Printf("conflict\t%s\n", c)
	}
}

// applyOne runs engine.Apply for one config (shared by runApply / runApplyAll). rootKind / fixedRoot
// come from evalRoot pre-resolution for the single case and from the batch eval (evalAllRoots) for --all. Only build is done per config.
func applyOne(ep *entrypoint, system, name, rootKind, fixedRoot string) (*engine.Result, error) {
	return engine.Apply(engine.Options{
		Name:         name,
		RootKind:     rootKind,
		FixedRoot:    fixedRoot,
		RootOverride: flagRoot,
		NoWait:       flagNoWait,
		Recopy:       flagRecopy,
		Build:        buildFunc(ep, system, name),
	})
}

// runApplyAll applies all of the entrypoint's nput.* in lexical order (→ docs/spec.md execution flow, ADR-0016, ADR-0024).
// rootKind is taken in a single batch eval (collapsing process launches N→1); build is per config for atomicity.
// It continues with the rest on a partial failure, shows an aggregate at the end, and exits non-zero if any one fails.
func runApplyAll() error {
	filter, err := selectedRootFilter()
	if err != nil {
		return err
	}
	ep, err := discoverEntrypoint(flagFile)
	if err != nil {
		return err
	}
	system, err := currentSystem()
	if err != nil {
		return err
	}

	// 1. Get rootKind in a single batch eval (config name → rootInfo map; → ADR-0024).
	roots, err := evalAllRoots(ep, system)
	if err != nil {
		return err
	}

	// 2. Sort lexically and, if a root filter (--project-root etc.) is given, narrow to that mode only.
	names := make([]string, 0, len(roots))
	for name := range roots {
		names = append(names, name)
	}
	sort.Strings(names)
	var selected []string
	for _, name := range names {
		if filter == "" || roots[name].RootKind == filter {
			selected = append(selected, name)
		}
	}
	if len(selected) == 0 {
		if flagVerbose {
			fmt.Fprintln(os.Stderr, "nput: apply --all: no matching configs")
		}
		return nil
	}

	// 2.5 --dryrun is a side-effect-free preview (takes no flock / --set / pending gcroot; runs only build
	//     read-only). It aggregates each selected config's plan to stdout and decides the exit code by
	//     priority error(1) > conflict(2) > 0 (→ docs/spec.md, ADR-0024).
	if flagDryrun {
		return runApplyAllDryRun(ep, system, selected, roots)
	}

	// 3. Apply each config independently. Continue on partial failure and aggregate failures (each config is independently atomic).
	var failures, skipped, applied int
	for _, name := range selected {
		ri := roots[name]
		res, err := applyOne(ep, system, name, ri.RootKind, ri.Root)
		if err != nil {
			if errors.Is(err, engine.ErrSkipped) {
				skipped++
				if flagVerbose {
					fmt.Fprintf(os.Stderr, "nput: skipped apply %s (another apply is in progress)\n", name)
				}
				continue
			}
			failures++
			// Do not swallow partial failures; print to stderr and continue (→ docs/spec.md "continue on partial failure").
			fmt.Fprintf(os.Stderr, "nput: apply %s failed: %v\n", name, err)
			continue
		}
		applied++
		if flagVerbose {
			reportResult(res, name)
		}
	}

	// 4. Aggregate report and exit code (priority error(1) > conflict(2) > 0; → docs/spec.md, ADR-0024).
	if flagVerbose {
		fmt.Fprintf(os.Stderr, "nput: apply --all done (applied %d / skipped %d / failed %d / selected %d)\n",
			applied, skipped, failures, len(selected))
	}
	// conflict(2) arises only on the --dryrun (#13) read-only path. Non-dryrun --all yields only error/0.
	code := applyAllExitCode(failures > 0, false)
	if code == 0 {
		return nil
	}
	return &exitCodeError{code: code, msg: fmt.Sprintf("nput: apply --all: %d config(s) failed", failures)}
}

// runApplyAllDryRun drives apply --all --dryrun. It builds each selected config read-only and
// aggregates the plan to stdout (taking none of FS writes / flock / --set / pending gcroot; → ADR-0023).
// It decides the exit code by priority error(1) > conflict(2) > 0 (→ docs/spec.md, ADR-0024) and carries it in an
// empty-msg exitError (symmetric with the single apply --dryrun conflict=2; main exits with the code alone).
func runApplyAllDryRun(ep *entrypoint, system string, selected []string, roots map[string]rootInfo) error {
	code := aggregateDryRun(selected, func(name string) (*engine.Result, error) {
		ri := roots[name]
		return engine.Apply(engine.Options{
			Name:         name,
			RootKind:     ri.RootKind,
			FixedRoot:    ri.Root,
			RootOverride: flagRoot,
			Recopy:       flagRecopy,
			DryRun:       true,
			Build:        dryBuildFunc(ep, system, name),
		})
	})
	if code == 0 {
		return nil
	}
	return &exitError{code: code}
}

// aggregateDryRun runs each selected config read-only via applyDry, prints the plan to stdout,
// aggregates error / conflict, and returns the exit code (a seam that injects the apply implementation for testability).
// It does not swallow a config's build / eval failure (error); it prints to stderr, continues, and reflects it in the final code.
func aggregateDryRun(selected []string, applyDry func(name string) (*engine.Result, error)) int {
	var anyError, anyConflict bool
	for _, name := range selected {
		res, err := applyDry(name)
		if err != nil {
			anyError = true
			// Do not swallow partial failures; print to stderr and continue (→ docs/spec.md "continue on partial failure").
			fmt.Fprintf(os.Stderr, "nput: apply %s --dryrun failed: %v\n", name, err)
			continue
		}
		printApplyPlan(res)
		if len(res.Conflicts) > 0 {
			anyConflict = true
		}
	}
	return applyAllExitCode(anyError, anyConflict)
}

// applyAllExitCode decides apply --all's exit code by priority error(1) > conflict(2) > 0
// (→ docs/spec.md "output streams and exit codes", ADR-0024). It does not take the plain maximum (2 > 1)
// (because a conflict would hide serious eval / engine errors in CI). conflict arises only on the --dryrun path.
func applyAllExitCode(anyError, anyConflict bool) int {
	switch {
	case anyError:
		return 1
	case anyConflict:
		return 2
	default:
		return 0
	}
}

// selectedRootFilter returns the root mode filter from --project-root / --home-root / --system-root
// (none gives ""; specifying more than one is an error; → ADR-0017). The return value is a manifest.RootKind* string.
func selectedRootFilter() (string, error) {
	var modes []string
	if flagProjectRoot {
		modes = append(modes, manifest.RootKindProject)
	}
	if flagHomeRoot {
		modes = append(modes, manifest.RootKindHome)
	}
	if flagSystemRoot {
		modes = append(modes, manifest.RootKindSystem)
	}
	if len(modes) > 1 {
		return "", fmt.Errorf("nput: --project-root / --home-root / --system-root may be specified only one at a time")
	}
	if len(modes) == 0 {
		return "", nil
	}
	return modes[0], nil
}

// ensureNoRootFilter errors when a root filter is used outside --all
// (the filter is a modifier for --all; in a named apply <name> pins a single config, so it is meaningless; → ADR-0017).
func ensureNoRootFilter(modifier string) error {
	if flagProjectRoot || flagHomeRoot || flagSystemRoot {
		return fmt.Errorf("nput: --project-root / --home-root / --system-root are modifiers for %s", modifier)
	}
	return nil
}

// reportResult prints the placement report to stderr (stdout is reserved for machine-readable output; → ADR-0023).
func reportResult(res *engine.Result, name string) {
	fmt.Fprintf(os.Stderr, "nput: apply %s done (root=%s)\n", name, res.Root)
	for _, t := range res.Placed {
		fmt.Fprintf(os.Stderr, "  placed   %s\n", t)
	}
	for _, t := range res.Replaced {
		fmt.Fprintf(os.Stderr, "  replaced %s\n", t)
	}
	for _, t := range res.Copied {
		fmt.Fprintf(os.Stderr, "  copied   %s\n", t)
	}
	for _, t := range res.Recopied {
		fmt.Fprintf(os.Stderr, "  recopied %s\n", t)
	}
	for _, t := range res.Removed {
		fmt.Fprintf(os.Stderr, "  removed  %s\n", t)
	}
	if len(res.Placed)+len(res.Replaced)+len(res.Copied)+len(res.Recopied)+len(res.Removed) == 0 {
		fmt.Fprintln(os.Stderr, "  no-op")
	}
}
