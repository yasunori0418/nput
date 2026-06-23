package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/engine"
)

// applyAllExitCode follows priority error(1) > conflict(2) > 0 (not the plain maximum; → ADR-0024).
func TestApplyAllExitCode(t *testing.T) {
	cases := []struct {
		name              string
		anyError, anyConf bool
		want              int
	}{
		{"none", false, false, 0},
		{"error only", true, false, 1},
		{"conflict only", false, true, 2},
		{"error wins over conflict", true, true, 1}, // the maximum would be 2, but do not hide error
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := applyAllExitCode(c.anyError, c.anyConf); got != c.want {
				t.Errorf("applyAllExitCode(%v, %v) = %d, want %d", c.anyError, c.anyConf, got, c.want)
			}
		})
	}
}

// captureStdout captures and returns stdout produced while f runs (verifies that the dryrun plan owns stdout).
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	f()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

// Verifies aggregateDryRun's aggregate exit code and plan output (stdout ownership) by injecting the apply
// implementation, without nix. This is the regression guard for the real path of #14 AC-4 "error(1) > conflict(2) > 0".
func TestAggregateDryRun(t *testing.T) {
	clean := func(name string) (*engine.Result, error) {
		return &engine.Result{Placed: []string{"/p/" + name}}, nil
	}
	conflict := func(name string) (*engine.Result, error) {
		return &engine.Result{Conflicts: []string{"/c/" + name}}, nil
	}
	fail := func(name string) (*engine.Result, error) {
		return nil, io.EOF
	}

	cases := []struct {
		name     string
		apply    func(string) (*engine.Result, error)
		selected []string
		wantCode int
	}{
		{"all clean → 0", clean, []string{"a", "b"}, 0},
		{"conflict → 2", func(n string) (*engine.Result, error) {
			if n == "b" {
				return conflict(n)
			}
			return clean(n)
		}, []string{"a", "b"}, 2},
		{"error → 1", func(n string) (*engine.Result, error) {
			if n == "b" {
				return fail(n)
			}
			return clean(n)
		}, []string{"a", "b"}, 1},
		{"error wins over conflict → 1", func(n string) (*engine.Result, error) {
			switch n {
			case "a":
				return conflict(n)
			case "b":
				return fail(n)
			}
			return clean(n)
		}, []string{"a", "b", "c"}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got int
			out := captureStdout(t, func() { got = aggregateDryRun(c.selected, c.apply) })
			if got != c.wantCode {
				t.Errorf("aggregateDryRun code = %d, want %d", got, c.wantCode)
			}
			// A successful config's plan goes to stdout (owns the machine-readable output; → ADR-0023).
			if strings.Contains(c.name, "all clean") && !strings.Contains(out, "place\t/p/a") {
				t.Errorf("plan not emitted to stdout: %q", out)
			}
		})
	}
}

func TestSelectedRootFilter(t *testing.T) {
	defer func() { flagProjectRoot, flagHomeRoot, flagSystemRoot = false, false, false }()

	t.Run("none", func(t *testing.T) {
		flagProjectRoot, flagHomeRoot, flagSystemRoot = false, false, false
		got, err := selectedRootFilter()
		if err != nil || got != "" {
			t.Fatalf("got (%q, %v), want (\"\", nil)", got, err)
		}
	})
	t.Run("project", func(t *testing.T) {
		flagProjectRoot, flagHomeRoot, flagSystemRoot = true, false, false
		got, err := selectedRootFilter()
		if err != nil || got != "project" {
			t.Fatalf("got (%q, %v), want (\"project\", nil)", got, err)
		}
	})
	t.Run("multiple → error", func(t *testing.T) {
		flagProjectRoot, flagHomeRoot, flagSystemRoot = true, true, false
		if _, err := selectedRootFilter(); err == nil {
			t.Fatal("specifying multiple should be an error")
		}
	})
}
