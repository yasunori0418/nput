// Package manifest reads and validates the manifest.json contract that
// lib.mkManifest emits and the engine consumes (→ ADR-0006, ADR-0010, ADR-0013).
//
// manifest.json は Nix↔Go の唯一の安定契約。engine は自身の対応版より新しい
// schemaVersion を拒否する（→ ADR-0006, docs/spec.md「manifest.json スキーマ（v1）」）。
package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SchemaVersion は engine が解釈できる manifest.json の最新版（→ ADR-0013）。
// MVP は v1 のみを受理し、これより新しい版は拒否する（→ ADR-0006, ADR-0015）。
const SchemaVersion = 1

// FileName は link-farm derivation 内に埋め込まれる manifest の固定名。
const FileName = "manifest.json"

// 配置元・配置方法の clean enum（_nputMarker は manifest に漏れない・→ ADR-0010）。
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

// Root は配置先基準の kind。project / home / system は実行時解決のためパスを持たず、
// fixed のみ評価時確定の絶対パスを Root に持つ（→ docs/spec.md）。
type Root struct {
	RootKind string `json:"rootKind"`
	Root     string `json:"root,omitempty"`
}

// Entry は 1 配置定義。identity は Target（属性キー由来・stale 除去の diff キー・→ ADR-0014）。
type Entry struct {
	SrcKind string `json:"srcKind"`
	Src     string `json:"src"`
	Subpath string `json:"subpath"`
	Target  string `json:"target"`
	Method  string `json:"method"`
}

// Manifest は manifest.json トップレベル。
type Manifest struct {
	SchemaVersion int     `json:"schemaVersion"`
	Root          Root    `json:"root"`
	Entries       []Entry `json:"entries"`
}

// Load は link-farm ディレクトリ内の manifest.json を読み、schemaVersion を検証する。
func Load(linkFarm string) (*Manifest, error) {
	return LoadFile(filepath.Join(linkFarm, FileName))
}

// LoadFile は manifest.json ファイルを直接読み、schemaVersion を検証する。
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
	// engine は自身の対応版より新しい schemaVersion を拒否する（→ ADR-0006）。
	if m.SchemaVersion > SchemaVersion {
		return fmt.Errorf("schemaVersion %d は未対応です（この engine は v%d まで対応）", m.SchemaVersion, SchemaVersion)
	}
	if m.SchemaVersion < 1 {
		return fmt.Errorf("schemaVersion %d は不正です", m.SchemaVersion)
	}
	if m.Root.RootKind == "" {
		return fmt.Errorf("root.rootKind が空です")
	}
	return nil
}
