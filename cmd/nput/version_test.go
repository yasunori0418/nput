package main

import "testing"

// TestVersionDefault pins the ldflags-unset default. A plain `go build` (no -X main.version=...)
// must leave version at "dev" so the CLI still works out of tree (→ ADR-0042 acceptance criteria).
// The nix build overrides this via ldflags; go test runs without them (the flake's custom checkPhase
// deliberately omits ldflags — see flake.nix), so this test observes the default.
func TestVersionDefault(t *testing.T) {
	if version != "dev" {
		t.Errorf("version = %q, want %q (ldflags-unset default)", version, "dev")
	}
}

// TestRootCmdVersionWired asserts cobra's Version field is wired to the package version variable.
// Guards against the field silently drifting from the variable that ldflags targets (main.version,
// a fixed contract for #130's tool.version supply).
func TestRootCmdVersionWired(t *testing.T) {
	root := newRootCmd()
	if root.Version != version {
		t.Errorf("root.Version = %q, want %q (must track main.version)", root.Version, version)
	}
}

// TestVersionFlagOutput drives `nput --version` end-to-end and observes the actual stdout, not just
// the wired field. ADR-0042 requires cobra's default template ("nput version X.Y.Z\n") unchanged, so
// this catches drift the field-equality check can't — e.g. an errant SetVersionTemplate. cobra prints
// the version via OutOrStdout(), which falls back to os.Stdout, so captureStdout observes it.
func TestVersionFlagOutput(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"--version"})
	out := captureStdout(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("Execute(--version): %v", err)
		}
	})
	want := "nput version " + version + "\n"
	if out != want {
		t.Errorf("`nput --version` output = %q, want %q (cobra default template)", out, want)
	}
}

// TestVersionSubcommandAbsent locks in that `nput version` is NOT a command: cobra's Version field
// adds a --version flag only, never a `version` subcommand. This pins the actual UX so a comment or
// doc claiming otherwise can't drift back in (→ diff-review must finding).
func TestVersionSubcommandAbsent(t *testing.T) {
	root := newRootCmd()
	cmd, _, err := root.Find([]string{"version"})
	if err == nil && cmd != nil && cmd != root {
		t.Errorf("`version` resolved to subcommand %q, want no such subcommand (only --version flag)", cmd.Name())
	}
}
