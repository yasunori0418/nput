// Command nput is the primary-UX CLI that drives the placement engine (internal/engine) (→ ADR-0007, ADR-0011).
//
// This slice (#7) is scoped to getting project mode's `nput apply [<name>]` end-to-end through a flake
// entrypoint. The execution order follows docs/spec.md "execution flow":
// "eval first (rootKind) → flock → in-lock build → place → --set → remove .pending"; the engine owns
// flock through build, placement, and commit, while the CLI handles the orchestration of entrypoint
// discovery and nix eval/build (→ ADR-0023, ADR-0025).
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/manifest"
)

// Global flags (→ docs/spec.md "global flags").
var (
	flagFile        string // -f/--file: specify the entrypoint explicitly
	flagRoot        string // --root: explicitly override the resolved root
	flagNoWait      bool   // --no-wait: skip without waiting on flock contention (for shellHook)
	flagVerbose     bool   // -v/--verbose: print the placement report (summary + per-target lines); silent on success by default (→ ADR-0031)
	flagDebug       bool   // --debug: disclose the internally run nix commands on stderr (→ ADR-0031)
	flagRecopy      bool   // --recopy: apply modifier; unconditionally re-copy every copy target from src, overwriting
	flagYes         bool   // -y/--yes: skip reset's confirmation prompt (for scripts / CI)
	flagDryrun      bool   // --dryrun: apply / reset modifier; show the plan with zero side effects
	flagProjectRoot bool   // --project-root: apply --all modifier; apply only projectRoot configs
	flagHomeRoot    bool   // --home-root: apply --all modifier; apply only homeRoot configs
	flagSystemRoot  bool   // --system-root: apply --all modifier; apply only systemRoot configs (future seam)
	flagManifest    string // --manifest: apply a pre-built manifest (link-farm) directly (for module activation)
)

// exitError is an error carrying a specific exit code that cobra RunE returns (→ docs/spec.md exit code table).
// apply --dryrun's conflict is exit 2. If msg is empty, main exits with the code alone and emits no extra output.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

// rootCmdLong discloses the internally run nix commands in --help (for transparency; selectively runnable by hand; → ADR-0007).
const rootCmdLong = `nput places fetched git repositories at arbitrary paths in your environment via symlink or copy.
It does not generate configuration (configuration is written in Nix and evaluated by nix build).

Internal nix commands (disclosed for transparency; you can run them by hand selectively):
  init <template>   nix flake init -t <ref>#<template>
  apply <name>      nix eval <ep>#nput.<system>.<name>.rootKind --raw
                    nix build <ep>#nput.<system>.<name> --out-link <profileDir>/.pending
  apply --all       nix eval <ep>#nput.<system> --apply '<rootKind map>' --json
                    nix build <ep>#nput.<system>.<name> (per config)
  gitignore <name>  nix eval <ep>#nput.<system>.<name>.rootKind --raw
                    nix build <ep>#nput.<system>.<name> --no-link --print-out-paths
  rollback /        nix eval <ep>#nput.<system>.<name>.rootKind --raw
  list-generations

Pass --debug to print the actual nix commands to stderr as they run.`

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "nput",
		Short: "Place fetched git repositories at arbitrary paths via symlink or copy.",
		Long:  rootCmdLong,
		// Errors are printed exactly once in main, so suppress cobra's automatic usage/error display.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	pf := root.PersistentFlags()
	pf.StringVarP(&flagFile, "file", "f", "", "Specify the entrypoint explicitly (overrides autodiscovery)")
	pf.StringVar(&flagRoot, "root", "", "Override the resolved root explicitly (all modes)")
	pf.BoolVar(&flagNoWait, "no-wait", false, "Skip without waiting on flock contention (for shellHook)")
	pf.BoolVarP(&flagVerbose, "verbose", "v", false, "Print the placement report (summary + per-target lines); silent on success by default (see ADR-0031)")
	pf.BoolVar(&flagDebug, "debug", false, "Disclose the internal nix commands on stderr (see ADR-0031)")
	pf.BoolVarP(&flagYes, "yes", "y", false, "Skip reset's confirmation prompt (for scripts / CI)")
	pf.BoolVar(&flagProjectRoot, "project-root", false, "Modifier for apply --all: apply only projectRoot configs")
	pf.BoolVar(&flagHomeRoot, "home-root", false, "Modifier for apply --all: apply only homeRoot configs")
	pf.BoolVar(&flagSystemRoot, "system-root", false, "Modifier for apply --all: apply only systemRoot configs (system mode not yet implemented)")

	root.AddCommand(newInitCmd())
	root.AddCommand(newApplyCmd())
	root.AddCommand(newResetCmd())
	root.AddCommand(newRollbackCmd())
	root.AddCommand(newListGenerationsCmd())
	root.AddCommand(newGitignoreCmd())
	return root
}

// exitCodeX is the interface satisfied by errors that carry an exit code (such as apply --all's aggregate exit code).
type exitCodeX interface{ ExitCode() int }

// exitCodeError is an error that explicitly carries an exit code (→ docs/spec.md "exit codes").
type exitCodeError struct {
	code int
	msg  string
}

func (e *exitCodeError) Error() string { return e.msg }
func (e *exitCodeError) ExitCode() int { return e.code }

func main() {
	if err := newRootCmd().Execute(); err != nil {
		// An error carrying an exit code (such as apply --dryrun conflict=2) exits with the code alone
		// (the plan was already written to stdout; → docs/spec.md exit code table).
		var ee *exitError
		if errors.As(err, &ee) {
			if ee.msg != "" {
				fmt.Fprintln(os.Stderr, ee.msg)
			}
			os.Exit(ee.code)
		}

		fmt.Fprintln(os.Stderr, err)
		// The engine rejects a schemaVersion skew between the CLI and the flake pin (→ manifest.validate).
		// Detect it at the top level and supplement the cause and the fix (→ docs/spec.md "manifest.json schema").
		if errors.Is(err, manifest.ErrSchemaVersionUnsupported) {
			fmt.Fprintln(os.Stderr, "\nnput: the nput version pinned by the CLI (engine) and by the flake may be out of sync.\n"+
				"  The flake's nput input is generating a manifest newer than the CLI.\n"+
				"  Update the CLI, or lower the flake's nput input to match the CLI so both versions align.")
		}
		// An error carrying an exit code (such as apply --all's aggregate) exits with that code (→ docs/spec.md, ADR-0024).
		var ec exitCodeX
		if errors.As(err, &ec) {
			os.Exit(ec.ExitCode())
		}
		os.Exit(1)
	}
}
