package main

import (
	"reflect"
	"testing"

	"github.com/yasunori0418/niface/go/conformance"
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

// TestGitignoreJSONInfoPaths pins gitignore's --json shape (→ issue #132, ADR-0043 §5): the
// enumeration rides in result.info.paths in anchor form, items stays an empty array (not an
// id-derived item per path), no generation slot appears, and the envelope is conformant.
func TestGitignoreJSONInfoPaths(t *testing.T) {
	checker, err := conformance.NewDefaultChecker()
	if err != nil {
		t.Fatalf("conformance.NewDefaultChecker: %v", err)
	}

	r, buf := newTestRun("gitignore")
	r.beginSubject("docs")
	r.setPayload(&nifacePayload{info: map[string]any{
		"paths": gitignoreAnchors([]string{".claude/skills/nix", ".nput-out/docs"}),
	}})
	if err := r.emit(nil); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if findings := checker.Check(buf.Bytes()); len(findings) > 0 {
		t.Fatalf("conformance findings: %v\ndocument: %s", findings, buf.String())
	}

	doc := decodeEnvelope(t, buf)
	if doc["dryRun"] != false {
		t.Errorf("dryRun = %v, want false (a read-only command reflects only the flag)", doc["dryRun"])
	}
	sr := doc["results"].([]any)[0].(map[string]any)
	if _, ok := sr["generation"]; ok {
		t.Errorf("generation slot present, want absent for gitignore: %v", sr["generation"])
	}
	res := sr["result"].(map[string]any)
	if items := res["items"].([]any); len(items) != 0 {
		t.Errorf("items = %v, want empty (the enumeration is info, not items)", items)
	}
	paths := res["info"].(map[string]any)["paths"].([]any)
	if len(paths) != 2 || paths[0] != "/.claude/skills/nix" || paths[1] != "/.nput-out/docs" {
		t.Errorf("info.paths = %v, want the anchor-form targets", paths)
	}
}

// TestGitignoreAnchorsEmptyStaysArray pins that a config without entries still lists
// "paths": [] (a non-nil slice; nil would marshal the key away).
func TestGitignoreAnchorsEmptyStaysArray(t *testing.T) {
	got := gitignoreAnchors(nil)
	if got == nil || len(got) != 0 {
		t.Errorf("gitignoreAnchors(nil) = %#v, want a non-nil empty slice", got)
	}
}
