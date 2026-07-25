package main

import (
	"errors"
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

	r, buf := newGitignoreTestRun()
	r.beginSubject("docs")
	r.setPayload(&nifacePayload[*gitignoreInfo]{info: &gitignoreInfo{
		Paths: gitignoreAnchors([]string{".claude/skills/nix", ".nput-out/docs"}),
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

// TestGitignoreJSONInfoAbsentWithoutEnumeration pins the failure boundary that forced
// gitignoreInfo to be carried as a pointer (→ issue #196 §4): gitignore can fail after the
// subject is registered but before the enumeration exists (entrypoint discovery, the
// project-mode rejection, the manifest build), and result.info must stay absent there — as it
// did while the slot was a nil map. A value-struct TInfo would emit "info":{"paths":null},
// which the conformance checker would still accept.
func TestGitignoreJSONInfoAbsentWithoutEnumeration(t *testing.T) {
	r, buf := newGitignoreTestRun()
	r.beginSubject("docs")
	if err := r.emit(errors.New("nput: gitignore is project mode only")); err != nil {
		t.Fatalf("emit: %v", err)
	}
	assertNoInfoKeys(t, decodeEnvelope(t, buf))
}

// TestGitignoreJSONEmptyPathsStaysArray pins the zero-entry boundary at the emit level
// (spec: entry 0 件でも "paths": [] を明示): the emitted document must carry the paths key as
// an empty array — a nil slice would marshal the key away.
func TestGitignoreJSONEmptyPathsStaysArray(t *testing.T) {
	r, buf := newGitignoreTestRun()
	r.beginSubject("empty")
	r.setPayload(&nifacePayload[*gitignoreInfo]{info: &gitignoreInfo{Paths: gitignoreAnchors(nil)}})
	if err := r.emit(nil); err != nil {
		t.Fatalf("emit: %v", err)
	}
	info := decodeEnvelope(t, buf)["results"].([]any)[0].(map[string]any)["result"].(map[string]any)["info"].(map[string]any)
	paths, ok := info["paths"].([]any)
	if !ok {
		t.Fatalf("info = %v, want the paths key present as an array", info)
	}
	if len(paths) != 0 {
		t.Errorf("info.paths = %v, want []", paths)
	}
}
