package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// defaultTemplateRef は init が展開するテンプレの固定 flake ref（registry 非依存・ハードコード・
// → ADR-0007）。NPUT_TEMPLATE_REF 環境変数があれば上書きする（E2E / fork 利用者の逃げ道・→ 計画 Q3）。
const defaultTemplateRef = "github:yasunori0418/nput"

// initTemplates は init が受理するテンプレ名（flake.templates の output 名と一致）。
var initTemplates = []string{"standalone", "project"}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <template>",
		Short: "Expand a starter template into the CWD via nix flake init (standalone / project)",
		Long: "nput init <template> is a transparent wrapper around `nix flake init -t <ref>#<template>`. " +
			"It expands a starter flake into the CWD (nput generates no files; the nix templates mechanism handles expansion).\n\n" +
			"template:\n" +
			"  standalone  homeRoot example (places under $HOME)\n" +
			"  project     projectRoot example + devShell + shellHook + .gitignore\n\n" +
			"The template reference is a fixed ref (" + defaultTemplateRef + "). Override it with NPUT_TEMPLATE_REF.\n" +
			"Existing files are not overwritten (inherits nix flake init's behavior).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(args[0])
		},
	}
}

// runInit はテンプレ名を検証し `nix flake init -t <ref>#<template>` を CWD で実行する。
// 新規 flake 生成のため entrypoint 探索（discoverEntrypoint）は通さない（→ 計画 8）。
func runInit(template string) error {
	if !isValidTemplate(template) {
		return fmt.Errorf("nput: unknown template: %q (valid values: %s)", template, strings.Join(initTemplates, " / "))
	}

	ref := defaultTemplateRef
	if env := os.Getenv("NPUT_TEMPLATE_REF"); env != "" {
		ref = env
	}

	args := flakeInitArgs(template, ref)
	if flagDebug {
		fmt.Fprintf(os.Stderr, "nput: + nix %s\n", strings.Join(args, " "))
	}

	// stderr を捕捉する。nix flake init は作成ファイル一覧を stderr に出すため、成功時はそれを転送し、
	// 失敗時は experimental 未有効を分類して案内、他は生 stderr を添える（apply 系と UX 一貫・→ 計画 Q6）。
	cmd := exec.Command("nix", args...)
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nixError(args, stderr.String(), err)
	}

	// 成功時は捕捉した作成ファイル一覧（nix の出力）を stderr へ転送する。
	if s := stderr.String(); s != "" {
		fmt.Fprint(os.Stderr, s)
	}
	return nil
}

// isValidTemplate はテンプレ名が受理対象かを返す。
func isValidTemplate(template string) bool {
	return slices.Contains(initTemplates, template)
}

// flakeInitArgs は `nix flake init` の argv を組む純関数（→ 計画 6・unit test 対象）。
// ref#template の installable で展開元テンプレを指す。
func flakeInitArgs(template, ref string) []string {
	return []string{"flake", "init", "-t", ref + "#" + template}
}
