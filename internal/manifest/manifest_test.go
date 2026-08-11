package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, FileName)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadValid(t *testing.T) {
	dir := writeManifest(t, `{
	  "schemaVersion": 1,
	  "root": { "rootKind": "project" },
	  "entries": [
	    { "srcKind": "store", "src": "/nix/store/aaa-source", "subpath": "skills/nix", "target": ".claude/skills/nix", "method": "symlink" }
	  ]
	}`)

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.SchemaVersion != 1 {
		t.Errorf("schemaVersion = %d, want 1", m.SchemaVersion)
	}
	if m.Root.RootKind != RootKindProject {
		t.Errorf("rootKind = %q, want project", m.Root.RootKind)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(m.Entries))
	}
	e := m.Entries[0]
	if e.SrcKind != SrcKindStore || e.Src != "/nix/store/aaa-source" || e.Subpath != "skills/nix" || e.Target != ".claude/skills/nix" || e.Method != MethodSymlink {
		t.Errorf("entry mismatch: %+v", e)
	}
}

func TestLoadRejectsNewerSchema(t *testing.T) {
	dir := writeManifest(t, `{ "schemaVersion": 2, "root": { "rootKind": "project" }, "entries": [] }`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for schemaVersion 2, got nil")
	}
	// The sentinel must be detectable via errors.Is so the CLI can add skew guidance (→ docs/spec.md).
	if !errors.Is(err, ErrSchemaVersionUnsupported) {
		t.Errorf("error should wrap ErrSchemaVersionUnsupported, got %v", err)
	}
}

func TestLoadRejectsBelowMinimumSchema(t *testing.T) {
	for _, tt := range []struct {
		name    string
		version string
	}{
		{"zero", "0"},
		{"negative", "-1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeManifest(t, `{ "schemaVersion": `+tt.version+`, "root": { "rootKind": "project" }, "entries": [] }`)
			_, err := Load(dir)
			if err == nil {
				t.Fatalf("expected error for schemaVersion %s, got nil", tt.version)
			}
			// Below the minimum is not version skew, so it must not wrap the skew sentinel (→ ADR-0006).
			if errors.Is(err, ErrSchemaVersionUnsupported) {
				t.Errorf("error should not wrap ErrSchemaVersionUnsupported, got %v", err)
			}
		})
	}
}

func TestLoadRejectsEmptyRootKind(t *testing.T) {
	dir := writeManifest(t, `{ "schemaVersion": 1, "root": { "rootKind": "" }, "entries": [] }`)
	if _, err := Load(dir); err == nil {
		t.Fatal("expected error for empty rootKind, got nil")
	}
}

func TestLoadRejectsMissingRoot(t *testing.T) {
	dir := writeManifest(t, `{ "schemaVersion": 1, "entries": [] }`)
	if _, err := Load(dir); err == nil {
		t.Fatal("expected error for missing root, got nil")
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	dir := writeManifest(t, `{ "schemaVersion": 1, "root": { "rootKind": "project" }, "entries": [], "bogus": true }`)
	if _, err := Load(dir); err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("expected error for missing manifest.json, got nil")
	}
}

func TestLoadFixedRootHasPath(t *testing.T) {
	dir := writeManifest(t, `{ "schemaVersion": 1, "root": { "rootKind": "fixed", "root": "/opt/x" }, "entries": [] }`)
	m, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Root.Root != "/opt/x" {
		t.Errorf("fixed root = %q, want /opt/x", m.Root.Root)
	}
}
