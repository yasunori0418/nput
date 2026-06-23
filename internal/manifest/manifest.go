// Package manifest reads and validates the manifest.json contract that
// lib.mkManifest emits and the engine consumes (→ ADR-0006, ADR-0010, ADR-0013).
//
// manifest.json is the sole stable contract between Nix and Go. The engine rejects
// any schemaVersion newer than the version it supports
// (→ ADR-0006, docs/spec.md "manifest.json schema (v1)").
package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrSchemaVersionUnsupported indicates that a manifest newer than the engine's supported version (SchemaVersion) was read.
// A sentinel so the caller (CLI) can detect schemaVersion skew between CLI/flake pin via errors.Is and add guidance (→ ADR-0006).
var ErrSchemaVersionUnsupported = errors.New("nput: schemaVersion が engine の対応版より新しい")

// SchemaVersion is the latest manifest.json version the engine can interpret (→ ADR-0013).
// The MVP accepts only v1 and rejects any newer version (→ ADR-0006, ADR-0015).
const SchemaVersion = 1

// FileName is the fixed manifest name embedded in the link-farm derivation.
const FileName = "manifest.json"

// Clean enums for src kind and placement method (_nputMarker does not leak into the manifest; → ADR-0010).
const (
	SrcKindStore      = "store"
	SrcKindOutOfStore = "outOfStore"

	MethodSymlink = "symlink"
	MethodCopy    = "copy"

	RootKindProject = "project"
	RootKindHome    = "home"
	RootKindSystem  = "system"
	RootKindFixed   = "fixed"
)

// Root is the kind of the placement target base. project / home / system are
// resolved at runtime and carry no path; only fixed holds an absolute path in
// Root determined at evaluation time (→ docs/spec.md).
type Root struct {
	RootKind string `json:"rootKind"`
	Root     string `json:"root,omitempty"`
}

// Entry is a single placement definition. Its identity is Target (derived from the attribute key; the diff key for stale removal; → ADR-0014).
type Entry struct {
	SrcKind string `json:"srcKind"`
	Src     string `json:"src"`
	Subpath string `json:"subpath"`
	Target  string `json:"target"`
	Method  string `json:"method"`
}

// Manifest is the top level of manifest.json.
type Manifest struct {
	SchemaVersion int     `json:"schemaVersion"`
	Root          Root    `json:"root"`
	Entries       []Entry `json:"entries"`
}

// Load reads manifest.json inside the link-farm directory and validates schemaVersion.
func Load(linkFarm string) (*Manifest, error) {
	return LoadFile(filepath.Join(linkFarm, FileName))
}

// LoadFile reads a manifest.json file directly and validates schemaVersion.
func LoadFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("nput: manifest.json を読めません: %w", err)
	}

	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("nput: manifest.json を解析できません (%s): %w", path, err)
	}

	if err := m.validate(); err != nil {
		return nil, fmt.Errorf("nput: manifest.json が不正です (%s): %w", path, err)
	}
	return &m, nil
}

func (m *Manifest) validate() error {
	// The engine rejects any schemaVersion newer than the version it supports (→ ADR-0006).
	if m.SchemaVersion > SchemaVersion {
		return fmt.Errorf("schemaVersion %d は未対応です（この engine は v%d まで対応）: %w", m.SchemaVersion, SchemaVersion, ErrSchemaVersionUnsupported)
	}
	if m.SchemaVersion < 1 {
		return fmt.Errorf("schemaVersion %d は不正です", m.SchemaVersion)
	}
	if m.Root.RootKind == "" {
		return fmt.Errorf("root.rootKind が空です")
	}
	return nil
}
