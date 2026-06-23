// Package paths computes the on-disk profile layout the engine operates on
// (→ ADR-0005, ADR-0013, ADR-0022, ADR-0024, ADR-0025).
//
// profileDir is the per-config dedicated directory (= flock key). The profile
// link sits inside it as profile, generations as profile-N-link, the build
// out-link as .pending, and the backref .root at the <roothash> level
// (→ docs/spec.md "on-disk layout of the profile").
package paths

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yasunori0418/nput/internal/manifest"
)

// rootHashLen is the hex character count of the roothash (128 bit; fixed length;
// FS-safe; → ADR-0013). It matches the digit count of lib.mkManifest's anchorName
// (the first 32 hex of sha256).
const rootHashLen = 32

// StateDir returns the base <state> for the profiles. $XDG_STATE_HOME if set,
// otherwise $HOME/.local/state (consistent with nix's own profile default; → ADR-0022).
func StateDir() (string, error) {
	if s := os.Getenv("XDG_STATE_HOME"); s != "" {
		return s, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("nput: $HOME を解決できません: %w", err)
	}
	return filepath.Join(home, ".local", "state"), nil
}

// RootHash returns the truncated hex of the sha256 of the resolved absolute root path (→ ADR-0013).
func RootHash(absRoot string) string {
	sum := sha256.Sum256([]byte(absRoot))
	return hex.EncodeToString(sum[:])[:rootHashLen]
}

// Base returns the base <state>/nix/profiles/nput for the profiles. The home
// (no --root) profileDir is <name> directly under it; the roothash series is
// <roothash>/<name> (→ ADR-0024).
func Base(stateDir string) string {
	return filepath.Join(stateDir, "nix", "profiles", "nput")
}

// GenerationLink returns the path of the generation link
// <profileDir>/profile-<gen>-link that nix-env creates as a sibling of the
// profile link (→ docs/spec.md on-disk layout; ADR-0025).
func GenerationLink(profileLink string, gen int) string {
	return fmt.Sprintf("%s-%d-link", profileLink, gen)
}

// Profile is the full set of profile-layout paths for one config.
type Profile struct {
	// Dir is the profileDir (per-config directory; flock key).
	Dir string
	// Profile is the profile link (the target of nix-env --profile).
	Profile string
	// Pending is the output of nix build --out-link (a sibling that does not pass through the profile).
	Pending string
	// BackrefDir is the level where the backref .root is placed. Empty for home (no --root).
	BackrefDir string
	// Backref is the backref file recording the original root's absolute path. Empty for home.
	Backref string
}

// Resolve determines the profile layout from the state base, config name,
// rootKind, resolved absolute root, and whether --root was overridden
// (→ docs/spec.md "root resolution" table; ADR-0024, ADR-0025).
//
//   - home (no --root)               : <state>/nix/profiles/nput/<name> (no backref)
//   - project / fixed / --root override : <state>/nix/profiles/nput/<roothash>/<name>
//     (backref .root at the <roothash> level)
func Resolve(stateDir, name, rootKind, absRoot string, rootOverride bool) Profile {
	base := Base(stateDir)

	// Only home (no --root) keys directly on <name>. Otherwise separate into an independent series per root.
	if rootKind == manifest.RootKindHome && !rootOverride {
		dir := filepath.Join(base, name)
		return Profile{
			Dir:     dir,
			Profile: filepath.Join(dir, "profile"),
			Pending: filepath.Join(dir, ".pending"),
		}
	}

	hashDir := filepath.Join(base, RootHash(absRoot))
	dir := filepath.Join(hashDir, name)
	return Profile{
		Dir:        dir,
		Profile:    filepath.Join(dir, "profile"),
		Pending:    filepath.Join(dir, ".pending"),
		BackrefDir: hashDir,
		Backref:    filepath.Join(hashDir, ".root"),
	}
}
