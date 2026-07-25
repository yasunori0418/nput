package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/yasunori0418/niface/go/conformance"
)

// flakeInitArgs builds the argv for `nix flake init -t <ref>#<template>` (→ plan 6).
func TestFlakeInitArgs(t *testing.T) {
	cases := []struct {
		name     string
		template string
		ref      string
		want     []string
	}{
		{
			name:     "default ref",
			template: "project",
			ref:      defaultTemplateRef,
			want:     []string{"flake", "init", "-t", "github:yasunori0418/nput#project"},
		},
		{
			name:     "standalone",
			template: "standalone",
			ref:      defaultTemplateRef,
			want:     []string{"flake", "init", "-t", "github:yasunori0418/nput#standalone"},
		},
		{
			name:     "env override ref (path: local reference)",
			template: "project",
			ref:      "path:/tmp/nput",
			want:     []string{"flake", "init", "-t", "path:/tmp/nput#project"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := flakeInitArgs(tc.template, tc.ref); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("flakeInitArgs(%q, %q) = %v, want %v", tc.template, tc.ref, got, tc.want)
			}
		})
	}
}

// isValidTemplate is true only for accepted template names (rejects invalid values, sending them to the exit 1 path).
func TestIsValidTemplate(t *testing.T) {
	valid := []string{"standalone", "project"}
	for _, v := range valid {
		if !isValidTemplate(v) {
			t.Errorf("isValidTemplate(%q) = false, want true", v)
		}
	}
	invalid := []string{"", "Project", "home", "standalone ", "default"}
	for _, v := range invalid {
		if isValidTemplate(v) {
			t.Errorf("isValidTemplate(%q) = true, want false", v)
		}
	}
}

// TestInitJSONEnvelopeInfoAbsentOnRejectedTemplate pins the failure boundary that forced
// initInfo to be carried as a pointer (→ issue #196 §4): an unknown template name fails before
// setEnvelopeInfo runs, so the envelope's info must stay absent — as it did while the slot was
// a nil map. A value-struct TEnvInfo would emit "info":{"template":"","ref":""}, which the
// conformance checker would still accept.
func TestInitJSONEnvelopeInfoAbsentOnRejectedTemplate(t *testing.T) {
	r, buf := newInitTestRun()
	if err := r.emit(errors.New(`nput: unknown template: "nosuch"`)); err != nil {
		t.Fatalf("emit: %v", err)
	}
	assertNoInfoKeys(t, decodeEnvelope(t, buf))
}

// TestInitJSONEnvelopeInfo pins init's --json shape (niface ADR-0018 · → issue #132): init has
// no subject, so results stays [] and the run facts (template / ref) ride in the envelope-wide
// top-level info. The envelope is conformant.
func TestInitJSONEnvelopeInfo(t *testing.T) {
	checker, err := conformance.NewDefaultChecker()
	if err != nil {
		t.Fatalf("conformance.NewDefaultChecker: %v", err)
	}

	r, buf := newInitTestRun()
	r.setEnvelopeInfo(&initInfo{Template: "standalone", Ref: defaultTemplateRef})
	if err := r.emit(nil); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if findings := checker.Check(buf.Bytes()); len(findings) > 0 {
		t.Fatalf("conformance findings: %v\ndocument: %s", findings, buf.String())
	}

	doc := decodeEnvelope(t, buf)
	if results := doc["results"].([]any); len(results) != 0 {
		t.Errorf("results = %v, want [] (init has no subject)", results)
	}
	info := doc["info"].(map[string]any)
	if info["template"] != "standalone" || info["ref"] != defaultTemplateRef {
		t.Errorf("info = %v, want template=standalone ref=%s", info, defaultTemplateRef)
	}
	if doc["status"] != "success" || doc["dryRun"] != false {
		t.Errorf("status/dryRun = %v/%v, want success/false", doc["status"], doc["dryRun"])
	}
}
