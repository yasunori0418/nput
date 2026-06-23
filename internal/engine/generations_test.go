package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/paths"
)

func homeManifest(entries ...manifest.Entry) manifest.Manifest {
	return manifest.Manifest{
		SchemaVersion: 1,
		Root:          manifest.Root{RootKind: manifest.RootKindHome},
		Entries:       entries,
	}
}

func TestParseGenerations(t *testing.T) {
	out := "" +
		"   1   2026-06-01 10:00:00   \n" +
		"   2   2026-06-02 11:30:00   (current)\n"
	gens, err := parseGenerations(out)
	if err != nil {
		t.Fatalf("parseGenerations: %v", err)
	}
	if len(gens) != 2 {
		t.Fatalf("len = %d, want 2", len(gens))
	}
	if gens[0].Number != 1 || gens[0].Current {
		t.Errorf("gen[0] = %+v", gens[0])
	}
	if gens[0].Date != "2026-06-01 10:00:00" {
		t.Errorf("gen[0].Date = %q", gens[0].Date)
	}
	if gens[1].Number != 2 || !gens[1].Current {
		t.Errorf("gen[1] = %+v", gens[1])
	}
	if gens[1].Date != "2026-06-02 11:30:00" {
		t.Errorf("gen[1].Date = %q (should not include (current))", gens[1].Date)
	}
}

func TestParseGenerationsEmpty(t *testing.T) {
	gens, err := parseGenerations("\n  \n")
	if err != nil {
		t.Fatalf("parseGenerations: %v", err)
	}
	if len(gens) != 0 {
		t.Errorf("len = %d, want 0", len(gens))
	}
}

func TestParseGenerationsBadLine(t *testing.T) {
	if _, err := parseGenerations("not-a-number 2026-06-01\n"); err == nil {
		t.Fatal("expected parse error for non-numeric generation, got nil")
	}
}

// TestRollbackReconverges verifies FS re-convergence from current generation N → previous generation N-1.
// gen1 = {a, b}, gen2(current) = {a, c}. Rollback stale-removes c, re-places b, and keeps a.
func TestRollbackReconverges(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	srcA := makeSrc(t, "x")
	srcB := makeSrc(t, "x")
	srcC := makeSrc(t, "x")

	// Use the --root override for home + roothash key to be independent of $HOME.
	prof := paths.Resolve(state, "vim", manifest.RootKindHome, root, true)
	if err := os.MkdirAll(prof.Dir, 0o755); err != nil {
		t.Fatal(err)
	}

	lf1 := writeLinkFarm(t, homeManifest(
		storeEntry(srcA, ".", "a"),
		storeEntry(srcB, ".", "b"),
	))
	lf2 := writeLinkFarm(t, homeManifest(
		storeEntry(srcA, ".", "a"),
		storeEntry(srcC, ".", "c"),
	))

	// Prepare the generation links (profile-N-link) and the current profile (gen2).
	if err := os.Symlink(lf1, paths.GenerationLink(prof.Profile, 1)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(lf2, paths.GenerationLink(prof.Profile, 2)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(lf2, prof.Profile); err != nil {
		t.Fatal(err)
	}

	// Current FS = gen2: place a→srcA, c→srcC.
	if err := os.Symlink(srcA, filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(srcC, filepath.Join(root, "c")); err != nil {
		t.Fatal(err)
	}

	var switched int
	res, err := Rollback(RollbackOptions{
		Name:         "vim",
		RootKind:     manifest.RootKindHome,
		RootOverride: root,
		StateDir:     state,
		ListGenerations: func(string) ([]Generation, error) {
			return []Generation{{Number: 1}, {Number: 2, Current: true}}, nil
		},
		SwitchGeneration: func(_ string, gen int) error { switched = gen; return nil },
	})
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	if res.From != 2 || res.To != 1 {
		t.Errorf("From/To = %d/%d, want 2/1", res.From, res.To)
	}
	if switched != 1 {
		t.Errorf("switched generation = %d, want 1", switched)
	}

	// FS matches gen1: a remains, b new, c removed.
	if dest, err := os.Readlink(filepath.Join(root, "a")); err != nil || dest != srcA {
		t.Errorf("a: dest=%q err=%v, want %q", dest, err, srcA)
	}
	if dest, err := os.Readlink(filepath.Join(root, "b")); err != nil || dest != srcB {
		t.Errorf("b: dest=%q err=%v, want %q", dest, err, srcB)
	}
	if _, err := os.Lstat(filepath.Join(root, "c")); !os.IsNotExist(err) {
		t.Errorf("c should be removed, lstat err = %v", err)
	}
	if len(res.Removed) != 1 || res.Removed[0] != "c" {
		t.Errorf("Removed = %v, want [c]", res.Removed)
	}
}

// TestRollbackNoPreviousErrors verifies that rollback stops with an error when the current generation is the oldest (no previous generation).
func TestRollbackNoPreviousErrors(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	prof := paths.Resolve(state, "vim", manifest.RootKindHome, root, true)
	if err := os.MkdirAll(prof.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	lf := writeLinkFarm(t, homeManifest(storeEntry(makeSrc(t, "x"), ".", "a")))
	if err := os.Symlink(lf, prof.Profile); err != nil {
		t.Fatal(err)
	}

	_, err := Rollback(RollbackOptions{
		Name: "vim", RootKind: manifest.RootKindHome, RootOverride: root, StateDir: state,
		ListGenerations:  func(string) ([]Generation, error) { return []Generation{{Number: 1, Current: true}}, nil },
		SwitchGeneration: func(string, int) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "no previous generation") {
		t.Fatalf("expected no-previous-generation error, got %v", err)
	}
}

// TestRollbackNoProfileErrors verifies that rollback stops with an error when profileDir is absent (never applied).
func TestRollbackNoProfileErrors(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	_, err := Rollback(RollbackOptions{
		Name: "vim", RootKind: manifest.RootKindHome, RootOverride: root, StateDir: state,
		ListGenerations:  func(string) ([]Generation, error) { return nil, nil },
		SwitchGeneration: func(string, int) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected no-profile error, got %v", err)
	}
}

// TestResolveRootHome verifies that rootKind=home returns $HOME (root resolver unit test).
func TestResolveRootHome(t *testing.T) {
	home := realTempDir(t)
	t.Setenv("HOME", home)
	got, err := resolveRoot(manifest.RootKindHome, "", "", "", nil)
	if err != nil {
		t.Fatalf("resolveRoot home: %v", err)
	}
	if got != home {
		t.Errorf("resolveRoot home = %q, want %q", got, home)
	}
}

// TestResolveRootOverrideWins verifies that the --root override takes precedence regardless of rootKind.
func TestResolveRootOverrideWins(t *testing.T) {
	override := realTempDir(t)
	got, err := resolveRoot(manifest.RootKindHome, "", override, "", nil)
	if err != nil {
		t.Fatalf("resolveRoot override: %v", err)
	}
	if got != override {
		t.Errorf("resolveRoot override = %q, want %q", got, override)
	}
}
