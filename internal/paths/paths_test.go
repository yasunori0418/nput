package paths

import (
	"path/filepath"
	"testing"

	"github.com/yasunori0418/nput/internal/manifest"
)

func TestRootHashDeterministicAndFixedLen(t *testing.T) {
	a := RootHash("/home/me/proj")
	b := RootHash("/home/me/proj")
	if a != b {
		t.Errorf("RootHash not deterministic: %q != %q", a, b)
	}
	if len(a) != rootHashLen {
		t.Errorf("RootHash len = %d, want %d", len(a), rootHashLen)
	}
	if c := RootHash("/home/me/other"); c == a {
		t.Errorf("distinct roots collided: %q", c)
	}
	// FS 安全（hex のみ）。
	for _, r := range a {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			t.Errorf("RootHash has non-hex char %q in %q", r, a)
		}
	}
}

func TestResolveProjectUsesRootHash(t *testing.T) {
	state := "/state"
	root := "/home/me/proj"
	p := Resolve(state, "skills", manifest.RootKindProject, root, false)

	wantDir := filepath.Join(state, "nix", "profiles", "nput", RootHash(root), "skills")
	if p.Dir != wantDir {
		t.Errorf("Dir = %q, want %q", p.Dir, wantDir)
	}
	if p.Profile != filepath.Join(wantDir, "profile") {
		t.Errorf("Profile = %q", p.Profile)
	}
	if p.Pending != filepath.Join(wantDir, ".pending") {
		t.Errorf("Pending = %q", p.Pending)
	}
	wantBackref := filepath.Join(state, "nix", "profiles", "nput", RootHash(root), ".root")
	if p.Backref != wantBackref {
		t.Errorf("Backref = %q, want %q", p.Backref, wantBackref)
	}
}

func TestResolveHomeUsesNameKey(t *testing.T) {
	state := "/state"
	p := Resolve(state, "vim", manifest.RootKindHome, "/home/me", false)
	wantDir := filepath.Join(state, "nix", "profiles", "nput", "vim")
	if p.Dir != wantDir {
		t.Errorf("Dir = %q, want %q", p.Dir, wantDir)
	}
	if p.Backref != "" {
		t.Errorf("home (no --root) should have no backref, got %q", p.Backref)
	}
}

func TestResolveHomeWithOverrideUsesRootHash(t *testing.T) {
	// --root 明示時は home でも roothash キー（→ ADR-0023）。
	state := "/state"
	root := "/tmp/sandbox"
	p := Resolve(state, "vim", manifest.RootKindHome, root, true)
	wantDir := filepath.Join(state, "nix", "profiles", "nput", RootHash(root), "vim")
	if p.Dir != wantDir {
		t.Errorf("Dir = %q, want %q", p.Dir, wantDir)
	}
	if p.Backref == "" {
		t.Error("override should produce a backref")
	}
}

func TestResolveFixedUsesRootHash(t *testing.T) {
	// fixed root（--root なし）も roothash キー（→ ADR-0024）。
	state := "/state"
	root := "/opt/x"
	p := Resolve(state, "c", manifest.RootKindFixed, root, false)
	if p.Backref == "" {
		t.Error("fixed root should produce a backref")
	}
	wantDir := filepath.Join(state, "nix", "profiles", "nput", RootHash(root), "c")
	if p.Dir != wantDir {
		t.Errorf("Dir = %q, want %q", p.Dir, wantDir)
	}
}
