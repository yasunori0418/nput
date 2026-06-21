package main

import (
	"reflect"
	"testing"
)

// flakeInitArgs は `nix flake init -t <ref>#<template>` の argv を組む（→ 計画 6）。
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

// isValidTemplate は受理するテンプレ名のみ true（不正値は拒否して exit 1 経路へ）。
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
