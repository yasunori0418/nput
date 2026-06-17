package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/engine"
)

// applyAllExitCode は priority error(1) > conflict(2) > 0（単純な最大値ではない・→ ADR-0024）。
func TestApplyAllExitCode(t *testing.T) {
	cases := []struct {
		name              string
		anyError, anyConf bool
		want              int
	}{
		{"none", false, false, 0},
		{"error only", true, false, 1},
		{"conflict only", false, true, 2},
		{"error wins over conflict", true, true, 1}, // 最大値なら 2 だが error を隠さない
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := applyAllExitCode(c.anyError, c.anyConf); got != c.want {
				t.Errorf("applyAllExitCode(%v, %v) = %d, want %d", c.anyError, c.anyConf, got, c.want)
			}
		})
	}
}

// captureStdout は f 実行中の stdout を捕捉して返す（dryrun plan が stdout 専有か検証する）。
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

// aggregateDryRun の集約終了コードと plan 出力（stdout 専有）を、apply 実体を注入して
// nix 無しで検証する。これが #14 AC-4「error(1) > conflict(2) > 0」の実経路の回帰防止。
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
			// 成功 config の plan は stdout に出る（機械可読出力を専有・→ ADR-0023）。
			if strings.Contains(c.name, "all clean") && !strings.Contains(out, "place\t/p/a") {
				t.Errorf("stdout に plan が出ていない: %q", out)
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
			t.Fatal("複数指定はエラーになるべき")
		}
	})
}
