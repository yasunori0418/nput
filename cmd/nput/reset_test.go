package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/engine"
)

// confirmPolicy decides the confirmation policy from --yes / TTY state (→ ADR-0025 §5).
// Regression guard for refusing a destructive reset when non-interactive without --yes (#13 AC-5).
func TestConfirmPolicy(t *testing.T) {
	cases := []struct {
		name        string
		yes         bool
		interactive bool
		wantPrompt  bool
		wantErr     bool
	}{
		{"--yes skips (even non-interactive)", true, false, false, false},
		{"--yes skips (even interactive)", true, true, false, false},
		{"interactive + no --yes prompts", false, true, true, false},
		{"non-interactive + no --yes refuses", false, false, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotPrompt, err := confirmPolicy(c.yes, c.interactive)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if gotPrompt != c.wantPrompt {
				t.Errorf("needPrompt = %v, want %v", gotPrompt, c.wantPrompt)
			}
		})
	}
}

// promptYesNo returns true only for yes-type input (default No). Verified by swapping out stdin.
func TestPromptYesNo(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"yes\n", true},
		{"Y\n", true},
		{"YES\n", true},
		{"n\n", false},
		{"\n", false},      // default No
		{"maybe\n", false}, // unknown input is No
		{"", false},        // EOF (pipe closed) is also No
	}
	for _, c := range cases {
		t.Run(strings.TrimSpace(c.input)+"->"+boolStr(c.want), func(t *testing.T) {
			restore := withStdin(t, c.input)
			defer restore()
			got, err := promptYesNo("Continue?")
			if err != nil {
				t.Fatalf("promptYesNo err = %v", err)
			}
			if got != c.want {
				t.Errorf("promptYesNo(%q) = %v, want %v", c.input, got, c.want)
			}
		})
	}
}

// isInteractive returns false for pipe / redirect stdin (the non-TTY path; → ADR-0025 §5).
// The TTY=true path needs a real terminal and cannot be verified in CI, so only the false path is verified.
func TestIsInteractiveNonTTY(t *testing.T) {
	restore := withStdin(t, "")
	defer restore()
	if isInteractive() {
		t.Error("isInteractive() should be false for piped stdin")
	}
}

// reset's output stream discipline (#13 AC-9): the dryrun plan owns stdout,
// while the planned removals / result report go to stderr.
func TestResetOutputStreams(t *testing.T) {
	res := &engine.ResetResult{
		Root:            "/root",
		RemovedSymlinks: []string{"/root/a"},
		RemovedCopies:   []string{"/root/b"},
		KeptForeign:     []string{"/root/c"},
	}

	t.Run("printResetPlan owns stdout exclusively", func(t *testing.T) {
		out, errOut := captureOutErr(t, func() { printResetPlan(res) })
		if !strings.Contains(out, "remove-symlink\t/root/a") ||
			!strings.Contains(out, "remove-copy\t/root/b") ||
			!strings.Contains(out, "keep-foreign\t/root/c") {
			t.Errorf("plan not emitted to stdout: %q", out)
		}
		if errOut != "" {
			t.Errorf("plan should not write to stderr: %q", errOut)
		}
	})

	t.Run("reportResetResult owns stderr exclusively", func(t *testing.T) {
		out, errOut := captureOutErr(t, func() { reportResetResult(res, "name") })
		if out != "" {
			t.Errorf("report pollutes stdout: %q", out)
		}
		if !strings.Contains(errOut, "removed-symlink") || !strings.Contains(errOut, "/root/a") {
			t.Errorf("report not emitted to stderr: %q", errOut)
		}
	})

	t.Run("reportResetTargets owns stderr exclusively", func(t *testing.T) {
		out, errOut := captureOutErr(t, func() { reportResetTargets(res, "name") })
		if out != "" {
			t.Errorf("confirmation prompt pollutes stdout: %q", out)
		}
		if !strings.Contains(errOut, "removal targets") {
			t.Errorf("confirmation prompt not emitted to stderr: %q", errOut)
		}
	})
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// withStdin replaces os.Stdin with a pipe that returns input, and returns a function that restores it.
func withStdin(t *testing.T, input string) func() {
	t.Helper()
	old := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	go func() {
		_, _ = io.WriteString(w, input)
		_ = w.Close()
	}()
	os.Stdin = r
	return func() { os.Stdin = old; _ = r.Close() }
}

// captureOutErr captures and returns stdout / stderr produced while f runs (for verifying stream discipline).
func captureOutErr(t *testing.T, f func()) (stdout, stderr string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout, os.Stderr = wOut, wErr
	outCh, errCh := make(chan string, 1), make(chan string, 1)
	go func() { b, _ := io.ReadAll(rOut); outCh <- string(b) }()
	go func() { b, _ := io.ReadAll(rErr); errCh <- string(b) }()
	f()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	return <-outCh, <-errCh
}
