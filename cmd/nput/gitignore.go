package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/manifest"
)

// gitignoreInfo is gitignore's result.info: the anchor-form target enumeration (→ issue #132,
// ADR-0043 §5; typed by #196). The envelope-wide slot stays unused (read-only command), so it is
// an anonymous *struct{} left nil.
//
// Carried as a pointer for the same reason as generationsInfo: gitignore can fail after the
// subject is registered but before the enumeration exists (eval / project-mode rejection /
// build), and only a nil pointer keeps result.info omitted there (→ issue #196 §4).
type gitignoreInfo struct {
	Paths []string `json:"paths"`
}

// gitignoreRun is gitignore's concrete run instantiation, threaded from RunE.
type gitignoreRun = nifaceRun[*gitignoreInfo, *struct{}]

// beginGitignoreRun starts gitignore's run (→ beginNifaceRun, beginApplyRun).
func beginGitignoreRun(command string) *gitignoreRun {
	return beginNifaceRun[*gitignoreInfo, *struct{}](command)
}

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
			// beginGitignoreRun also publishes the run to nifaceReport, so main emits the envelope
			// after Execute returns whichever path below runs.
			run := beginGitignoreRun(cmd.Name())
			if all {
				if len(args) > 0 {
					return fmt.Errorf("nput: gitignore cannot combine <name> with --all")
				}
				return runGitignoreAll(run)
			}
			if len(args) != 1 {
				return fmt.Errorf("nput: gitignore requires <name> or --all")
			}
			return runGitignore(run, args[0])
		},
	}
	cmd.Flags().BoolVar(&all, "all", false,
		"Sort and de-duplicate the targets of all projectRoot configs (under --json each config keeps its own targets instead, un-deduplicated; see ADR-0018)")
	return cmd
}

// runGitignore lists a single config's placement targets. project mode only;
// it errors out if a non-project config (home / fixed) is given (because the anchor form presupposes the git toplevel; → ADR-0023).
func runGitignore(run *gitignoreRun, name string) error {
	// The config name is the niface subject; errors from here on are subject-borne (→ issue #130).
	// A named listing registers exactly one, so the run's results[] holds N=1 (→ issue #164).
	subject := run.beginSubject(name)
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
	subject.setPayload(gitignorePayload(targets))
	printGitignore(targets)
	return nil
}

// gitignorePayload wraps one config's targets as its SubjectResult payload — the anchor-form
// enumeration rides result.info, shared by the named listing and --all so both produce the same
// shape by construction (→ issue #132, #164).
func gitignorePayload(targets []string) *nifacePayload[*gitignoreInfo] {
	return &nifacePayload[*gitignoreInfo]{info: &gitignoreInfo{Paths: gitignoreAnchors(targets)}}
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
//
// The two contracts are deliberately asymmetric here (→ issue #164): the default stdout stays the
// cross-config dedup+sort union (ADR-0018 unchanged — it is meant to be appended to one .gitignore),
// while --json gives each config its own SubjectResult holding that config's own paths, undeduped.
// Attributing a shared path to one arbitrary config would be a lie about which config declares it;
// a consumer that wants the union takes it across the results itself.
func runGitignoreAll(run *gitignoreRun) error {
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
		subject := run.beginSubject(name)
		targets, err := configTargets(ep, system, name)
		if err != nil {
			// The enumeration stops here (unchanged), so this config's subject carries the failure
			// and the ones already listed keep their results (→ issue #164). The same error also
			// returns as the command error, but it lands only here: emit's finish is first-wins,
			// and the top-level errors[] takes a failure only when no subject was registered.
			subject.finish(err)
			return err
		}
		subject.setPayload(gitignorePayload(targets))
		subject.finish(nil)
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
