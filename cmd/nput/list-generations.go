package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/engine"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/paths"
)

func newListGenerationsCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "list-generations [name]",
		Short: "List generations (home mode only)",
		Long: "A read-only command that lists the generations of the home mode profile. " +
			"Pass <name> for that config, or --all to list every home mode config.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				if len(args) > 0 {
					return fmt.Errorf("nput: list-generations cannot combine <name> with --all")
				}
				return runListAllGenerations()
			}
			if len(args) != 1 {
				return fmt.Errorf("nput: list-generations requires <name> or --all")
			}
			return runListGenerations(args[0])
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "List generations for every home mode config")
	return cmd
}

// runListGenerations confirms rootKind via eval pre-resolution (home mode only), resolves profileDir, and lists generations.
func runListGenerations(name string) error {
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
		return fmt.Errorf("nput: list-generations is home mode only (nput.%s has rootKind=%q)", name, rootKind)
	}

	prof, _, err := engine.ProfileFor(engine.ProfileOptions{
		Name:         name,
		RootKind:     rootKind,
		FixedRoot:    fixedRoot,
		RootOverride: flagRoot,
	})
	if err != nil {
		return err
	}
	gens, err := engine.ListGenerations(prof.Profile)
	if err != nil {
		return err
	}
	printGenerations(gens)
	return nil
}

// runListAllGenerations scans the home profiles directly under <state>/nix/profiles/nput (the <name>
// directories that hold a profile link directly under them) and lists each config's generations. No entrypoint eval is needed (disk scan only).
// The roothash family (project / fixed / --root) has a <roothash>/<name> structure with no profile directly under it, so it is naturally excluded.
func runListAllGenerations() error {
	stateDir, err := paths.StateDir()
	if err != nil {
		return err
	}
	base := paths.Base(stateDir)
	dents, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no profile created yet = empty listing.
		}
		return fmt.Errorf("nput: cannot read the profile base (%s): %w", base, err)
	}

	var names []string
	for _, d := range dents {
		if !d.IsDir() {
			continue
		}
		prof := paths.Resolve(stateDir, d.Name(), manifest.RootKindHome, "", false)
		if _, err := os.Lstat(prof.Profile); err != nil {
			continue // no profile directly under it = roothash family / empty directory.
		}
		names = append(names, d.Name())
	}
	sort.Strings(names)

	for i, name := range names {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("# %s\n", name)
		prof := paths.Resolve(stateDir, name, manifest.RootKindHome, "", false)
		gens, err := engine.ListGenerations(prof.Profile)
		if err != nil {
			return err
		}
		printGenerations(gens)
	}
	return nil
}

// printGenerations prints the generation list to stdout (the primary output of a read-only command; → ADR-0023).
func printGenerations(gens []engine.Generation) {
	for _, g := range gens {
		marker := ""
		if g.Current {
			marker = "\t(current)"
		}
		fmt.Printf("%d\t%s%s\n", g.Number, g.Date, marker)
	}
}
