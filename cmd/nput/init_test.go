package main

import (
	"reflect"
	"testing"
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
			name:     "env override ref（path: 局所参照）",
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
