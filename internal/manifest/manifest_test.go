package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	// project omits the path in the manifest, so it decodes to the empty string.
	// This fixes the decoded shape only; rejecting a project document that does carry
	// a path is not implemented and therefore not asserted here (→ REQ-dd10d820).
	if m.Root.Root != "" {
		t.Errorf("project root = %q, want empty", m.Root.Root)
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

// The lowest schemaVersion the engine accepts. Pinning the valid edge next to the
// invalid ones keeps the boundary visible if SchemaVersion is ever raised.
const minSchemaVersion = 1

func TestLoadSchemaVersionBoundary(t *testing.T) {
	for _, tt := range []struct {
		name    string
		version string
		wantErr bool
	}{
		{"negative", "-1", true},
		{"zero", "0", true},
		{"minimum", strconv.Itoa(minSchemaVersion), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeManifest(t, `{ "schemaVersion": `+tt.version+`, "root": { "rootKind": "project" }, "entries": [] }`)
			_, err := Load(dir)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("schemaVersion %s should be accepted, got %v", tt.version, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for schemaVersion %s, got nil", tt.version)
			}
			// Pin the rejection to the below-minimum check rather than any other failure.
			if !strings.Contains(err.Error(), "is invalid") {
				t.Errorf("error should report an invalid schemaVersion, got %v", err)
			}
			// Below the minimum is a broken document, not version skew, so it must not wrap
			// the skew sentinel — otherwise the CLI emits a misleading "flake and CLI differ"
			// hint for it (→ TC-172548ea, cmd/nput: ErrSchemaVersionUnsupported branch).
			if errors.Is(err, ErrSchemaVersionUnsupported) {
				t.Errorf("error should not wrap ErrSchemaVersionUnsupported, got %v", err)
			}
		})
	}
}

// A missing root object decodes to the zero value, so it reaches the very same
// emptiness check as an explicitly empty rootKind. Both spellings are pinned to
// document that no separate guard covers the missing key (→ TC-172548ea).
func TestLoadRejectsRootKindEmptiedByEitherSpelling(t *testing.T) {
	for _, tt := range []struct {
		name string
		doc  string
	}{
		{"explicitly empty", `{ "schemaVersion": 1, "root": { "rootKind": "" }, "entries": [] }`},
		{"root object omitted", `{ "schemaVersion": 1, "entries": [] }`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(writeManifest(t, tt.doc))
			if err == nil {
				t.Fatal("expected error for empty rootKind, got nil")
			}
			// Pin the rejection to the rootKind check, not to decoding or any later guard.
			if !strings.Contains(err.Error(), "root.rootKind") {
				t.Errorf("error should report an empty root.rootKind, got %v", err)
			}
		})
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
