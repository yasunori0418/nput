package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/manifest"
)

func newGitignoreCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "gitignore [name]",
		Short: "Print placement targets for .gitignore to stdout (project mode only; no writes)",
		Long: "List the placement targets of nput.<name> for .gitignore on stdout (writes no file). " +
			"Output is the root-relative target with a leading / in anchor form (e.g. /.claude/skills/nix), one per line, " +
			"covering every target regardless of method (symlink / copy). project mode only; " +
			"--all sorts and de-duplicates the targets of all projectRoot configs.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nifaceReport.begin(cmd.Name())
			if all {
				if len(args) > 0 {
					return fmt.Errorf("nput: gitignore cannot combine <name> with --all")
				}
				return runGitignoreAll()
			}
			if len(args) != 1 {
				return fmt.Errorf("nput: gitignore requires <name> or --all")
			}
			return runGitignore(args[0])
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Sort and de-duplicate the targets of all projectRoot configs")
	return cmd
}

// runGitignore lists a single config's placement targets. project mode only;
// it errors out if a non-project config (home / fixed) is given (because the anchor form presupposes the git toplevel; → ADR-0023).
func runGitignore(name string) error {
	// The config name is the niface subject; errors from here on are subject-borne (→ issue #130).
	nifaceReport.beginSubject(name)
	ep, err := discoverEntrypoint(flagFile)
	if err != nil {
		return err
	}
	system, err := currentSystem()
	if err != nil {
		return err
	}

	// Confirm project mode via rootKind pre-resolution eval (rejecting cheaply before build).
	rootKind, _, err := evalRoot(ep, system, name)
	if err != nil {
		return err
	}
	if rootKind != manifest.RootKindProject {
		return fmt.Errorf("nput: gitignore is project mode only (nput.%s has rootKind=%q; the .gitignore anchor is meaningless for home / fixed)", name, rootKind)
	}

	targets, err := configTargets(ep, system, name)
	if err != nil {
		return err
	}
	// A read-only enumeration rides in result.info as the anchor-form paths (items stays [] —
	// the listing is not an execution record · → issue #132, ADR-0043 §5). The line-oriented
	// default stdout below is untouched (--json is the opt-in second contract).
	nifaceReport.setPayload(&nifacePayload{info: map[string]any{"paths": gitignoreAnchors(targets)}})
	printGitignore(targets)
	return nil
}

// gitignoreAnchors maps targets into their /-anchor form (non-nil even when empty, so a
// config without entries still lists "paths": []).
func gitignoreAnchors(targets []string) []string {
	anchors := make([]string, 0, len(targets))
	for _, t := range targets {
		anchors = append(anchors, gitignoreAnchor(t))
	}
	return anchors
}

// runGitignoreAll lists the targets of all projectRoot configs, sorted and de-duplicated
// (a repo has a single .gitignore, so listing them together is natural; → docs/spec.md, ADR-0018).
// Non-project configs are excluded (--all picks up only projectRoot configs).
func runGitignoreAll() error {
	ep, err := discoverEntrypoint(flagFile)
	if err != nil {
		return err
	}
	system, err := currentSystem()
	if err != nil {
		return err
	}

	roots, err := evalAllRoots(ep, system)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(roots))
	for name := range roots {
		names = append(names, name)
	}
	sort.Strings(names)

	var all []string
	for _, name := range names {
		if roots[name].RootKind != manifest.RootKindProject {
			continue
		}
		targets, err := configTargets(ep, system, name)
		if err != nil {
			return err
		}
		all = append(all, targets...)
	}
	printGitignore(dedupeSorted(all))
	return nil
}

// configTargets builds the config, reads manifest.json, and lists the placement targets
// (all entries regardless of method; → ADR-0019).
func configTargets(ep *entrypoint, system, name string) ([]string, error) {
	store, err := buildManifestStorePath(ep, system, name)
	if err != nil {
		return nil, err
	}
	m, err := manifest.Load(store)
	if err != nil {
		return nil, err
	}
	targets := make([]string, 0, len(m.Entries))
	for _, e := range m.Entries {
		targets = append(targets, e.Target)
	}
	return targets, nil
}

// dedupeSorted sorts and de-duplicates (for --all's combined listing).
func dedupeSorted(in []string) []string {
	sort.Strings(in)
	out := in[:0]
	var prev string
	for i, s := range in {
		if i == 0 || s != prev {
			out = append(out, s)
		}
		prev = s
	}
	return out
}

// printGitignore prints targets to stdout in /-anchor form (leading /, no trailing /), one per line
// (→ docs/spec.md, ADR-0013). It is pipe-safe by the stdout-ownership principle (`nput gitignore <name> >> .gitignore`).
// Under --json it prints nothing: stdout belongs to the niface envelope alone, gated here — the
// single chokepoint for every call site (→ ADR-0043 §2, issue #130).
func printGitignore(targets []string) {
	if flagJSON {
		return
	}
	for _, t := range targets {
		fmt.Println(gitignoreAnchor(t))
	}
}

// gitignoreAnchor normalizes a root-relative target into /-anchor form. By eval the target has no absolute path
// and no trailing /, but it normalizes defensively (→ ADR-0013: anchor with a leading /, no trailing / for either directory or file).
func gitignoreAnchor(target string) string {
	t := strings.TrimSuffix(target, "/")
	t = strings.TrimPrefix(t, "/")
	return "/" + t
}
