package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/engine"
)

// resetResultInfo / resetEnvInfo are reset's niface info slots (→ issue #196): empty seat types
// held as nil pointers, so both info keys stay out of the document exactly as before. Reset's
// record lives in items / changes; the seats are here so later mutation run facts arrive as
// field additions alone (→ applyResultInfo in apply.go for the full rationale).
type (
	resetResultInfo struct{}
	resetEnvInfo    struct{}
)

// resetRun is reset's concrete run instantiation, threaded from RunE into runReset.
type resetRun = nifaceRun[*resetResultInfo, *resetEnvInfo]

func newResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset <name> [target...]",
		Short: "Tear placements back down to nothing (FS-only teardown; name required; no --all)",
		Long: "Teardown that returns nput.<name>'s placements to nothing. Omitting target tears down every entry; specifying targets tears down only those entries. " +
			"Symlinks are removed under the conservative invariant (only nput-managed, only as recorded) and foreign ones are kept. copy targets are deleted (confirmed due to the data-loss risk). " +
			"It does not touch the profile or generations (FS-only). A name is required (no --all). " +
			"--dryrun shows the removal targets with zero side effects and exits (no confirm / flock).",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			run := beginNifaceRun[*resetResultInfo, *resetEnvInfo](cmd.Name())
			return runReset(run, args[0], args[1:], flagDryrun)
		},
	}
	cmd.Flags().BoolVar(&flagDryrun, "dryrun", false,
		"Show the removal targets (symlink / copy target) with zero side effects and exit (no confirm / flock; see ADR-0021)")
	return cmd
}

// runReset resolves rootKind (→ profileDir) via eval pre-resolution and drives engine.Reset.
// --dryrun prints the plan read-only to stdout and exits 0. Non-dryrun requires TTY confirmation / --yes.
func runReset(run *resetRun, name string, targets []string, dryrun bool) error {
	// The config name is the niface subject; errors from here on are subject-borne (→ issue #130).
	run.beginSubject(name)
	ep, err := discoverEntrypoint(flagFile)
	if err != nil {
		return err
	}
	system, err := currentSystem()
	if err != nil {
		return err
	}
	rootKind, fixedRoot, err := evalRoot(ep, system, name)
	if err != nil {
		return err
	}

	// --dryrun: a side-effect-free preview (no flock / confirm; exit code is 0 regardless of whether there are targets; → ADR-0021).
	if dryrun {
		res, err := engine.Reset(engine.ResetOptions{
			Name:         name,
			RootKind:     rootKind,
			FixedRoot:    fixedRoot,
			RootOverride: flagRoot,
			Targets:      targets,
			DryRun:       true,
		})
		if err != nil {
			return err
		}
		printResetPlan(res)
		return nil
	}

	// Non-dryrun is a destructive operation. Decide the confirmation policy (skip / prompt / refuse) from --yes and TTY state.
	needPrompt, err := confirmPolicy(flagYes, resetPromptAllowed(isInteractive(), flagJSON))
	if err != nil {
		return err
	}

	// Pass confirm only when a prompt is needed. The prompt shows the computed plan.
	var confirm func(*engine.ResetResult) (bool, error)
	if needPrompt {
		confirm = func(res *engine.ResetResult) (bool, error) {
			reportResetTargets(res, name)
			return promptYesNo("This will remove the above. Continue?")
		}
	}

	res, err := engine.Reset(engine.ResetOptions{
		Name:         name,
		RootKind:     rootKind,
		FixedRoot:    fixedRoot,
		RootOverride: flagRoot,
		Targets:      targets,
		Confirm:      confirm,
	})
	if res != nil {
		// Also on a mid-teardown failure: the partial result keeps the changes complete up to
		// the failure point (→ issue #131, niface ADR-0020). No generation slot — reset never
		// moves the profile pointer (FS-only teardown).
		attachResetPayload(run, res, err)
	}
	if err != nil {
		return err
	}
	if res.Aborted {
		fmt.Fprintln(os.Stderr, "nput: reset aborted")
		return nil
	}
	if flagVerbose {
		reportResetResult(res, name)
	}
	return nil
}

// printResetPlan prints reset --dryrun's removal targets to stdout (it owns the machine-readable output; one per line;
// → docs/spec.md stream discipline, ADR-0023). It is not suppressed even under silent-on-success (the stdout-ownership principle; → ADR-0031).
// Under --json the stdout lines are suppressed at this single chokepoint (the envelope owns
// stdout); the stderr nothing-to-remove notice stays — human diagnostics coexist (→ ADR-0043 §2, issue #130).
func printResetPlan(res *engine.ResetResult) {
	if !flagJSON {
		for _, t := range res.RemovedSymlinks {
			fmt.Printf("remove-symlink\t%s\n", t)
		}
		for _, t := range res.RemovedCopies {
			fmt.Printf("remove-copy\t%s\n", t)
		}
		for _, t := range res.KeptForeign {
			fmt.Printf("keep-foreign\t%s\n", t)
		}
	}
	if len(res.RemovedSymlinks)+len(res.RemovedCopies)+len(res.KeptForeign) == 0 {
		fmt.Fprintln(os.Stderr, "nput: reset --dryrun: nothing to remove")
	}
}

// reportResetTargets prints the planned removals to stderr before the confirmation prompt (treated as progress; stdout is reserved for machine-readable output).
func reportResetTargets(res *engine.ResetResult, name string) {
	fmt.Fprintf(os.Stderr, "nput: reset %s removal targets (root=%s):\n", name, res.Root)
	for _, t := range res.RemovedSymlinks {
		fmt.Fprintf(os.Stderr, "  symlink %s\n", t)
	}
	for _, t := range res.RemovedCopies {
		fmt.Fprintf(os.Stderr, "  copy    %s\n", t)
	}
	for _, t := range res.KeptForeign {
		fmt.Fprintf(os.Stderr, "  keep    %s (foreign / record mismatch; kept)\n", t)
	}
	if len(res.RemovedSymlinks)+len(res.RemovedCopies) == 0 {
		fmt.Fprintln(os.Stderr, "  (nothing to remove)")
	}
}

// reportResetResult prints the actual removal result to stderr (stdout is reserved for machine-readable output; → ADR-0023).
func reportResetResult(res *engine.ResetResult, name string) {
	fmt.Fprintf(os.Stderr, "nput: reset %s done (root=%s)\n", name, res.Root)
	for _, t := range res.RemovedSymlinks {
		fmt.Fprintf(os.Stderr, "  removed-symlink %s\n", t)
	}
	for _, t := range res.RemovedCopies {
		fmt.Fprintf(os.Stderr, "  removed-copy    %s\n", t)
	}
	for _, t := range res.KeptForeign {
		fmt.Fprintf(os.Stderr, "  kept            %s (foreign / record mismatch)\n", t)
	}
	for _, t := range res.Pruned {
		fmt.Fprintf(os.Stderr, "  pruned          %s\n", t)
	}
	if len(res.RemovedSymlinks)+len(res.RemovedCopies) == 0 {
		fmt.Fprintln(os.Stderr, "  no-op")
	}
}

// resetPromptAllowed reports whether reset may prompt interactively: it requires a TTY, and
// --json unconditionally forbids it — machine consumption never prompts, so without --yes the
// confirmPolicy refuse path fails fast exactly like the non-interactive case (→ ADR-0043 §8,
// docs/spec.md "reset --json は --yes 必須").
func resetPromptAllowed(interactive, jsonMode bool) bool {
	return interactive && !jsonMode
}

// confirmPolicy decides the confirmation policy for a destructive reset from --yes and TTY state (→ ADR-0025 §5).
//   - --yes: skip confirmation (needPrompt=false, err=nil)
//   - no --yes + interactive environment: require a confirmation prompt (needPrompt=true)
//   - no --yes + non-interactive environment: error immediately to prevent a hang / accidental deletion on empty input (refuse)
//
// runReset passes in the result of isInteractive() to use it (a nix-independent, unit-testable seam).
func confirmPolicy(yes, interactive bool) (needPrompt bool, err error) {
	if yes {
		return false, nil
	}
	if !interactive {
		return false, errors.New("nput: refusing destructive reset without --yes in a non-interactive context")
	}
	return true, nil
}

// isInteractive returns whether stdin is a TTY (attached to a terminal) (→ ADR-0025 §5).
// It decides using stdlib only (os.ModeCharDevice). false under pipe / redirect / CI.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// promptYesNo reads y/N from stdin and returns true only for yes-type input (default No).
func promptYesNo(msg string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", msg)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
