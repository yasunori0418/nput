package paths

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/yasunori0418/nput/internal/manifest"
)

func TestStateDirUsesXDGStateHome(t *testing.T) {
	// $XDG_STATE_HOME takes precedence regardless of $HOME.
	t.Setenv("XDG_STATE_HOME", "/xdg/state")
	t.Setenv("HOME", "/home/me")
	got, err := StateDir()
	if err != nil {
		t.Fatalf("StateDir() error = %v", err)
	}
	if got != "/xdg/state" {
		t.Errorf("StateDir() = %q, want %q", got, "/xdg/state")
	}
}

func TestStateDirFallsBackToHome(t *testing.T) {
	// Without $XDG_STATE_HOME, fall back to $HOME/.local/state.
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/me")
	got, err := StateDir()
	if err != nil {
		t.Fatalf("StateDir() error = %v", err)
	}
	want := filepath.Join("/home/me", ".local", "state")
	if got != want {
		t.Errorf("StateDir() = %q, want %q", got, want)
	}
}

func TestStateDirErrorsWhenHomeUnresolvable(t *testing.T) {
	// With neither $XDG_STATE_HOME nor $HOME set, os.UserHomeDir fails and the
	// error is surfaced (no fallback path is returned).
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")
	got, err := StateDir()
	if err == nil {
		t.Fatalf("StateDir() error = nil, want non-nil (got %q)", got)
	}
	if got != "" {
		t.Errorf("StateDir() = %q on error, want empty string", got)
	}
}

func TestGenerationLinkFormat(t *testing.T) {
	profileLink := filepath.Join("/state", "nix", "profiles", "nput", "vim", "profile")
	for _, gen := range []int{0, 1, 42} {
		got := GenerationLink(profileLink, gen)
		want := fmt.Sprintf("%s-%d-link", profileLink, gen)
		if got != want {
			t.Errorf("GenerationLink(%q, %d) = %q, want %q", profileLink, gen, got, want)
		}
	}
}

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
	// FS-safe (hex only).
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
	// With --root explicit, even home uses the roothash key (→ ADR-0023).
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
	// A fixed root (no --root) also uses the roothash key (→ ADR-0024).
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
