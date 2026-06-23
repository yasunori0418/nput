package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// defaultTemplateRef is the fixed flake ref of the template that init expands (registry-independent, hardcoded;
// → ADR-0007). The NPUT_TEMPLATE_REF environment variable overrides it (an escape hatch for E2E / fork users; → plan Q3).
const defaultTemplateRef = "github:yasunori0418/nput"

// initTemplates is the template names that init accepts (matching the output names in flake.templates).
var initTemplates = []string{"standalone", "project"}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <template>",
		Short: "Expand a starter template into the CWD via nix flake init (standalone / project)",
		Long: "nput init <template> is a transparent wrapper around `nix flake init -t <ref>#<template>`. " +
			"It expands a starter flake into the CWD (nput generates no files; the nix templates mechanism handles expansion).\n\n" +
			"template:\n" +
			"  standalone  homeRoot example (places under $HOME)\n" +
			"  project     projectRoot example + devShell + shellHook + .gitignore\n\n" +
			"The template reference is a fixed ref (" + defaultTemplateRef + "). Override it with NPUT_TEMPLATE_REF.\n" +
			"Existing files are not overwritten (inherits nix flake init's behavior).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(args[0])
		},
	}
}

// runInit validates the template name and runs `nix flake init -t <ref>#<template>` in the CWD.
// Because it generates a new flake, it does not go through entrypoint discovery (discoverEntrypoint; → plan 8).
func runInit(template string) error {
	if !isValidTemplate(template) {
		return fmt.Errorf("nput: unknown template: %q (valid values: %s)", template, strings.Join(initTemplates, " / "))
	}

	ref := defaultTemplateRef
	if env := os.Getenv("NPUT_TEMPLATE_REF"); env != "" {
		ref = env
	}

	args := flakeInitArgs(template, ref)
	if flagDebug {
		fmt.Fprintf(os.Stderr, "nput: + nix %s\n", strings.Join(args, " "))
	}

	// Capture stderr. nix flake init prints the list of created files to stderr, so on success forward it,
	// and on failure classify and guide the experimental-not-enabled case, attaching raw stderr otherwise (UX consistent with apply; → plan Q6).
	cmd := exec.Command("nix", args...)
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nixError(args, stderr.String(), err)
	}

	// On success, forward the captured list of created files (nix's output) to stderr.
	if s := stderr.String(); s != "" {
		fmt.Fprint(os.Stderr, s)
	}
	return nil
}

// isValidTemplate reports whether the template name is accepted.
func isValidTemplate(template string) bool {
	return slices.Contains(initTemplates, template)
}

// flakeInitArgs is a pure function that builds the argv for `nix flake init` (→ plan 6; a unit-test target).
// It points to the source template via the ref#template installable.
func flakeInitArgs(template, ref string) []string {
	return []string{"flake", "init", "-t", ref + "#" + template}
}
