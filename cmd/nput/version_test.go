package main

import "testing"

// TestVersionDefault pins the ldflags-unset default. A plain `go build` (no -X main.version=...)
// must leave version at "dev" so the CLI still works out of tree (→ ADR-0042 acceptance criteria).
// The nix build overrides this via ldflags; this test runs without them, so it observes the default.
func TestVersionDefault(t *testing.T) {
	if version != "dev" {
		t.Errorf("version = %q, want %q (ldflags-unset default)", version, "dev")
	}
}

// TestRootCmdVersionWired asserts cobra's Version field is wired to the package version variable,
// so `nput --version` / `nput version` report the embedded value (→ ADR-0042). Guards against the
// field silently drifting from the variable that ldflags targets (main.version, a fixed contract
// for #130's tool.version supply).
func TestRootCmdVersionWired(t *testing.T) {
	root := newRootCmd()
	if root.Version != version {
		t.Errorf("root.Version = %q, want %q (must track main.version)", root.Version, version)
	}
}
