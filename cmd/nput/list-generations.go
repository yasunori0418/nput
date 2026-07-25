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

// generationsInfo is list-generations' result.info: the read-only generation inventory
// (→ issue #132, ADR-0043 §5; typed by #196). The envelope-wide slot stays unused — a read-only
// command records no run-scoped state — so it is an anonymous *struct{} left nil (omitted).
//
// It is carried as a *generationsInfo, not a value: the run can fail after the subject is
// registered but before the listing exists (eval / rootKind rejection / profile resolution),
// and only a nil pointer keeps omitempty effective there. A value struct would newly emit
// "info":{"generations":null} on those paths — the same output-invariance trap as the mutation
// seats (→ issue #196 §4).
type generationsInfo struct {
	Generations []generationRow `json:"generations"`
}

// listGenerationsRun is list-generations' concrete run instantiation, threaded from RunE.
type listGenerationsRun = nifaceRun[*generationsInfo, *struct{}]

// beginListGenerationsRun starts list-generations' run (→ beginNifaceRun, beginApplyRun).
func beginListGenerationsRun(command string) *listGenerationsRun {
	return beginNifaceRun[*generationsInfo, *struct{}](command)
}

func newListGenerationsCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "list-generations [name]",
		Short: "List generations (home mode only)",
		Long: "A read-only command that lists the generations of the home mode profile. " +
			"Pass <name> for that config, or --all to list every home mode config.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			run := beginListGenerationsRun(cmd.Name())
			if all {
				if len(args) > 0 {
					return fmt.Errorf("nput: list-generations cannot combine <name> with --all")
				}
				return runListAllGenerations()
			}
			if len(args) != 1 {
				return fmt.Errorf("nput: list-generations requires <name> or --all")
			}
			return runListGenerations(run, args[0])
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "List generations for every home mode config")
	return cmd
}

// runListGenerations confirms rootKind via eval pre-resolution (home mode only), resolves profileDir, and lists generations.
func runListGenerations(run *listGenerationsRun, name string) error {
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
	// A read-only enumeration rides in result.info (items stays [] — generations are not
	// id-derived items, and the SubjectResult.generation slot stays absent to avoid encoding
	// the same numbers twice · → issue #132, ADR-0043 §5).
	run.setPayload(&nifacePayload[*generationsInfo]{info: &generationsInfo{Generations: generationRows(gens)}})
	printGenerations(gens)
	return nil
}

// generationRow is one generation of the --json inventory (result.info.generations · → issue
// #132, ADR-0043 §5). date carries nix-env's display timestamp verbatim, like the text output.
type generationRow struct {
	Number  int    `json:"number"`
	Date    string `json:"date"`
	Current bool   `json:"current"`
}

// generationRows converts the engine listing into the info inventory (non-nil even when empty,
// so an empty profile still lists "generations": []).
func generationRows(gens []engine.Generation) []generationRow {
	rows := make([]generationRow, 0, len(gens))
	for _, g := range gens {
		rows = append(rows, generationRow{Number: g.Number, Date: g.Date, Current: g.Current})
	}
	return rows
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
		prof := paths.Resolve(stateDir, name, manifest.RootKindHome, "", false)
		gens, err := engine.ListGenerations(prof.Profile)
		if err != nil {
			return err
		}
		// Under --json stdout belongs to the envelope alone (→ ADR-0043 §2, issue #130); the
		// listing itself still runs so read failures keep the same exit behavior.
		if flagJSON {
			continue
		}
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("# %s\n", name)
		printGenerations(gens)
	}
	return nil
}

// printGenerations prints the generation list to stdout (the primary output of a read-only command; → ADR-0023).
// Under --json it prints nothing: stdout belongs to the niface envelope alone, gated here — the
// single chokepoint for every call site (→ ADR-0043 §2, issue #130). The --all path additionally
// gates its own per-config header lines at the call site.
func printGenerations(gens []engine.Generation) {
	if flagJSON {
		return
	}
	for _, g := range gens {
		marker := ""
		if g.Current {
			marker = "\t(current)"
		}
		fmt.Printf("%d\t%s%s\n", g.Number, g.Date, marker)
	}
}
