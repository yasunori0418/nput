package main

import (
	"reflect"
	"testing"
)

// gitignoreAnchor normalizes a root-relative target into /-anchor form (leading /, no trailing /; → ADR-0013).
func TestGitignoreAnchor(t *testing.T) {
	cases := map[string]string{
		".claude/skills/nix": "/.claude/skills/nix",
		".config/nvim":       "/.config/nvim",
		"dir/":               "/dir", // no trailing / is added
		"/already/anchored":  "/already/anchored",
		"file.txt":           "/file.txt",
	}
	for in, want := range cases {
		if got := gitignoreAnchor(in); got != want {
			t.Errorf("gitignoreAnchor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDedupeSorted(t *testing.T) {
	in := []string{".b", ".a", ".b", ".c", ".a"}
	want := []string{".a", ".b", ".c"}
	if got := dedupeSorted(in); !reflect.DeepEqual(got, want) {
		t.Errorf("dedupeSorted = %v, want %v", got, want)
	}
	if got := dedupeSorted(nil); len(got) != 0 {
		t.Errorf("dedupeSorted(nil) = %v, want empty", got)
	}
}
