package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeFile creates an empty file at dir/name (content is irrelevant; discoverEntrypoint only checks existence).
func writeFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
		t.Fatalf("writeFile(%s): %v", name, err)
	}
}

// TestDiscoverEntrypoint_FileFlag covers -f pointing directly at a file (→ ADR-0032 discovery order).
func TestDiscoverEntrypoint_FileFlag(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		wantKind entrypointKind
	}{
		{"flake.nix", "flake.nix", entrypointFlake},
		{"shell.nix", "shell.nix", entrypointLegacy},
		{"default.nix", "default.nix", entrypointLegacy},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, c.file)
			path := filepath.Join(dir, c.file)

			ep, err := discoverEntrypoint(path)
			if err != nil {
				t.Fatalf("discoverEntrypoint(%s): %v", path, err)
			}
			if ep.kind != c.wantKind {
				t.Errorf("kind = %v, want %v", ep.kind, c.wantKind)
			}
			switch c.wantKind {
			case entrypointFlake:
				if ep.flakeRef != dir {
					t.Errorf("flakeRef = %q, want %q", ep.flakeRef, dir)
				}
			case entrypointLegacy:
				if ep.legacyPath != path {
					t.Errorf("legacyPath = %q, want %q", ep.legacyPath, path)
				}
			}
		})
	}
}

// TestDiscoverEntrypoint_FileFlagRejectsUnknownName covers -f pointing at a file that is none of the three.
func TestDiscoverEntrypoint_FileFlagRejectsUnknownName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.nix")
	path := filepath.Join(dir, "config.nix")

	if _, err := discoverEntrypoint(path); err == nil {
		t.Fatal("expected an error for an unrecognized -f file name, got nil")
	}
}

// TestDiscoverEntrypoint_FileFlagMissing covers -f pointing at a path that does not exist.
func TestDiscoverEntrypoint_FileFlagMissing(t *testing.T) {
	if _, err := discoverEntrypoint(filepath.Join(t.TempDir(), "nope.nix")); err == nil {
		t.Fatal("expected an error for a missing -f path, got nil")
	}
}

// TestDiscoverEntrypoint_FileFlagDir covers -f pointing at a directory, applying the same
// flake.nix -> shell.nix -> default.nix priority as CWD autodiscovery (→ ADR-0032).
func TestDiscoverEntrypoint_FileFlagDir(t *testing.T) {
	cases := []struct {
		name       string
		files      []string
		wantKind   entrypointKind
		wantLegacy string // expected legacy file name, if wantKind == entrypointLegacy
	}{
		{"flake.nix only", []string{"flake.nix"}, entrypointFlake, ""},
		{"flake.nix wins over shell.nix", []string{"flake.nix", "shell.nix"}, entrypointFlake, ""},
		{"shell.nix only", []string{"shell.nix"}, entrypointLegacy, "shell.nix"},
		{"default.nix only", []string{"default.nix"}, entrypointLegacy, "default.nix"},
		{"shell.nix wins over default.nix", []string{"shell.nix", "default.nix"}, entrypointLegacy, "shell.nix"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range c.files {
				writeFile(t, dir, f)
			}
			ep, err := discoverEntrypoint(dir)
			if err != nil {
				t.Fatalf("discoverEntrypoint(%s): %v", dir, err)
			}
			if ep.kind != c.wantKind {
				t.Errorf("kind = %v, want %v", ep.kind, c.wantKind)
			}
			if c.wantKind == entrypointLegacy {
				want := filepath.Join(dir, c.wantLegacy)
				if ep.legacyPath != want {
					t.Errorf("legacyPath = %q, want %q", ep.legacyPath, want)
				}
			}
		})
	}
}

// TestDiscoverEntrypoint_FileFlagDirEmpty covers -f pointing at a directory with none of the three files.
func TestDiscoverEntrypoint_FileFlagDirEmpty(t *testing.T) {
	if _, err := discoverEntrypoint(t.TempDir()); err == nil {
		t.Fatal("expected an error for a -f directory with no entrypoint file, got nil")
	}
}

// TestDiscoverEntrypoint_CWD covers CWD autodiscovery priority flake.nix -> shell.nix -> default.nix (→ ADR-0032).
func TestDiscoverEntrypoint_CWD(t *testing.T) {
	cases := []struct {
		name       string
		files      []string
		wantKind   entrypointKind
		wantLegacy string
		wantErr    bool
	}{
		{"flake.nix only", []string{"flake.nix"}, entrypointFlake, "", false},
		{"flake.nix wins over shell.nix", []string{"flake.nix", "shell.nix"}, entrypointFlake, "", false},
		{"shell.nix only", []string{"shell.nix"}, entrypointLegacy, "shell.nix", false},
		{"default.nix only", []string{"default.nix"}, entrypointLegacy, "default.nix", false},
		{"shell.nix wins over default.nix", []string{"shell.nix", "default.nix"}, entrypointLegacy, "shell.nix", false},
		{"none", nil, 0, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range c.files {
				writeFile(t, dir, f)
			}
			t.Chdir(dir)

			ep, err := discoverEntrypoint("")
			if c.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("discoverEntrypoint(\"\"): %v", err)
			}
			if ep.kind != c.wantKind {
				t.Errorf("kind = %v, want %v", ep.kind, c.wantKind)
			}
			if c.wantKind == entrypointLegacy {
				want := filepath.Join(dir, c.wantLegacy)
				if ep.legacyPath != want {
					t.Errorf("legacyPath = %q, want %q", ep.legacyPath, want)
				}
			}
		})
	}
}

// TestEntrypointInstallableArgs locks in the flake `<ep>#nput.<system>.<name><suffix>` form vs. the
// legacy `-f <ep> nput.<name><suffix>` form (no per-system dimension; → ADR-0032).
func TestEntrypointInstallableArgs(t *testing.T) {
	flakeEp := &entrypoint{kind: entrypointFlake, flakeRef: "/proj"}
	if got, want := flakeEp.installableArgs("x86_64-linux", "docs", ".rootKind"), []string{"/proj#nput.x86_64-linux.docs.rootKind"}; !reflect.DeepEqual(got, want) {
		t.Errorf("flake installableArgs = %v, want %v", got, want)
	}
	if got, want := flakeEp.namespaceArgs("x86_64-linux"), []string{"/proj#nput.x86_64-linux"}; !reflect.DeepEqual(got, want) {
		t.Errorf("flake namespaceArgs = %v, want %v", got, want)
	}
	if got, want := flakeEp.label("x86_64-linux", "docs"), "nput.x86_64-linux.docs"; got != want {
		t.Errorf("flake label = %q, want %q", got, want)
	}

	legacyEp := &entrypoint{kind: entrypointLegacy, legacyPath: "/proj/shell.nix"}
	if got, want := legacyEp.installableArgs("x86_64-linux", "docs", ".rootKind"), []string{"-f", "/proj/shell.nix", "nput.docs.rootKind"}; !reflect.DeepEqual(got, want) {
		t.Errorf("legacy installableArgs = %v, want %v", got, want)
	}
	if got, want := legacyEp.namespaceArgs("x86_64-linux"), []string{"-f", "/proj/shell.nix", "nput"}; !reflect.DeepEqual(got, want) {
		t.Errorf("legacy namespaceArgs = %v, want %v", got, want)
	}
	if got, want := legacyEp.label("x86_64-linux", "docs"), "nput.docs"; got != want {
		t.Errorf("legacy label = %q, want %q", got, want)
	}
}
