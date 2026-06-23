package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/engine"
	"github.com/yasunori0418/nput/internal/manifest"
)

func newRollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback <name>",
		Short: "Roll nput.<name> back to the previous generation (home mode only; name required)",
		Long: "Roll the home mode profile back one generation. Because moving the profile pointer alone does not change the FS at an arbitrary root, " +
			"it re-converges the FS (treating current generation N as baseline and previous generation N-1 as target: conservatively stale-removes N∖N-1 and re-places N-1) " +
			"before moving the profile pointer. A name is required (no --all); errors out if there is no previous generation.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRollback(args[0])
		},
	}
}

// runRollback confirms rootKind via eval pre-resolution (home mode only) and drives engine.Rollback.
func runRollback(name string) error {
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
	if rootKind != manifest.RootKindHome {
		return fmt.Errorf("nput: rollback is home mode only (nput.%s has rootKind=%q; project / fixed do not expose generations)", name, rootKind)
	}

	res, err := engine.Rollback(engine.RollbackOptions{
		Name:         name,
		RootKind:     rootKind,
		FixedRoot:    fixedRoot,
		RootOverride: flagRoot,
	})
	if err != nil {
		return err
	}

	if flagVerbose {
		reportRollback(res, name)
	}
	return nil
}

// reportRollback prints the generation transition and placement diff to stderr (stdout is reserved for machine-readable output; → ADR-0023).
func reportRollback(res *engine.RollbackResult, name string) {
	fmt.Fprintf(os.Stderr, "nput: rollback %s done (generation %d → %d, root=%s)\n", name, res.From, res.To, res.Root)
	for _, t := range res.Placed {
		fmt.Fprintf(os.Stderr, "  placed   %s\n", t)
	}
	for _, t := range res.Replaced {
		fmt.Fprintf(os.Stderr, "  replaced %s\n", t)
	}
	for _, t := range res.Removed {
		fmt.Fprintf(os.Stderr, "  removed  %s\n", t)
	}
	if len(res.Placed)+len(res.Replaced)+len(res.Removed) == 0 {
		fmt.Fprintln(os.Stderr, "  no-op")
	}
}
