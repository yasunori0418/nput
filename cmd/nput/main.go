// Command nput は配置エンジン（internal/engine）を駆動する一次 UX の CLI（→ ADR-0007, ADR-0011）。
//
// 本スライス（#7）のスコープは project mode の `nput apply [<name>]` を flake entrypoint で
// end-to-end に通すことに絞る。実行順は docs/spec.md「実行フロー」の
// 「eval 先行（rootKind） → flock → ロック内 build → 配置 → --set → .pending 削除」に従い、
// flock〜build〜配置〜commit は engine が所有し、CLI は entrypoint 発見・nix eval/build の
// オーケストレーションを担う（→ ADR-0023, ADR-0025）。
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/manifest"
)

// グローバルフラグ（→ docs/spec.md「グローバルフラグ」）。
var (
	flagFile        string // -f/--file: entrypoint 明示
	flagRoot        string // --root: 解決 root の明示上書き
	flagNoWait      bool   // --no-wait: flock 競合時に待たず skip（shellHook 用）
	flagVerbose     bool   // -v/--verbose: 配置レポート（サマリ + per-target 行）を出力（既定は成功時沈黙・→ ADR-0031）
	flagDebug       bool   // --debug: 内部実行する nix コマンドを stderr に開示（→ ADR-0031）
	flagRecopy      bool   // --recopy: apply 修飾。全 copy target を src から無条件上書き再コピー
	flagYes         bool   // -y/--yes: reset の確認プロンプトをスキップ（スクリプト / CI 用）
	flagDryrun      bool   // --dryrun: apply / reset 修飾。副作用ゼロで plan を表示
	flagProjectRoot bool   // --project-root: apply --all の修飾。projectRoot の config のみ適用
	flagHomeRoot    bool   // --home-root: apply --all の修飾。homeRoot の config のみ適用
	flagSystemRoot  bool   // --system-root: apply --all の修飾。systemRoot の config のみ適用（将来 seam）
	flagManifest    string // --manifest: ビルド済み manifest（link-farm）を直接適用（module activation 用）
)

// exitError は cobra RunE が返す、特定の終了コードを伴うエラー（→ docs/spec.md 終了コード表）。
// apply --dryrun の conflict は exit 2。msg が空なら main は追加出力せず code だけで終了する。
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

// rootCmdLong は --help で内部実行する nix コマンドを開示する（透明性・選択的に手で実行可能・→ ADR-0007）。
const rootCmdLong = `nput places fetched git repositories at arbitrary paths in your environment via symlink or copy.
It does not generate configuration (configuration is written in Nix and evaluated by nix build).

Internal nix commands (disclosed for transparency; you can run them by hand selectively):
  init <template>   nix flake init -t <ref>#<template>
  apply <name>      nix eval <ep>#nput.<system>.<name>.rootKind --raw
                    nix build <ep>#nput.<system>.<name> --out-link <profileDir>/.pending
  apply --all       nix eval <ep>#nput.<system> --apply '<rootKind map>' --json
                    nix build <ep>#nput.<system>.<name> (per config)
  gitignore <name>  nix eval <ep>#nput.<system>.<name>.rootKind --raw
                    nix build <ep>#nput.<system>.<name> --no-link --print-out-paths
  rollback /        nix eval <ep>#nput.<system>.<name>.rootKind --raw
  list-generations

Pass --debug to print the actual nix commands to stderr as they run.`

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "nput",
		Short: "Place fetched git repositories at arbitrary paths via symlink or copy.",
		Long:  rootCmdLong,
		// エラーは main で 1 度だけ出すため cobra の usage/error 自動表示は抑制する。
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	pf := root.PersistentFlags()
	pf.StringVarP(&flagFile, "file", "f", "", "Specify the entrypoint explicitly (overrides autodiscovery)")
	pf.StringVar(&flagRoot, "root", "", "Override the resolved root explicitly (all modes)")
	pf.BoolVar(&flagNoWait, "no-wait", false, "Skip without waiting on flock contention (for shellHook)")
	pf.BoolVarP(&flagVerbose, "verbose", "v", false, "Print the placement report (summary + per-target lines); silent on success by default (see ADR-0031)")
	pf.BoolVar(&flagDebug, "debug", false, "Disclose the internal nix commands on stderr (see ADR-0031)")
	pf.BoolVarP(&flagYes, "yes", "y", false, "Skip reset's confirmation prompt (for scripts / CI)")
	pf.BoolVar(&flagProjectRoot, "project-root", false, "Modifier for apply --all: apply only projectRoot configs")
	pf.BoolVar(&flagHomeRoot, "home-root", false, "Modifier for apply --all: apply only homeRoot configs")
	pf.BoolVar(&flagSystemRoot, "system-root", false, "Modifier for apply --all: apply only systemRoot configs (system mode not yet implemented)")

	root.AddCommand(newInitCmd())
	root.AddCommand(newApplyCmd())
	root.AddCommand(newResetCmd())
	root.AddCommand(newRollbackCmd())
	root.AddCommand(newListGenerationsCmd())
	root.AddCommand(newGitignoreCmd())
	return root
}

// exitCodeX は終了コードを運ぶエラーが満たすインターフェース（apply --all の集約終了コード等）。
type exitCodeX interface{ ExitCode() int }

// exitCodeError は終了コードを明示的に運ぶエラー（→ docs/spec.md「終了コード」）。
type exitCodeError struct {
	code int
	msg  string
}

func (e *exitCodeError) Error() string { return e.msg }
func (e *exitCodeError) ExitCode() int { return e.code }

func main() {
	if err := newRootCmd().Execute(); err != nil {
		// 終了コードを伴うエラー（apply --dryrun conflict=2 等）は code だけで終了する
		// （plan は既に stdout に出力済み・→ docs/spec.md 終了コード表）。
		var ee *exitError
		if errors.As(err, &ee) {
			if ee.msg != "" {
				fmt.Fprintln(os.Stderr, ee.msg)
			}
			os.Exit(ee.code)
		}

		fmt.Fprintln(os.Stderr, err)
		// CLI/flake pin 間の schemaVersion skew は engine が拒否する（→ manifest.validate）。
		// 上位で検知して原因と解消策を補う（→ docs/spec.md「manifest.json スキーマ」）。
		if errors.Is(err, manifest.ErrSchemaVersionUnsupported) {
			fmt.Fprintln(os.Stderr, "\nnput: the nput version pinned by the CLI (engine) and by the flake may be out of sync.\n"+
				"  The flake's nput input is generating a manifest newer than the CLI.\n"+
				"  Update the CLI, or lower the flake's nput input to match the CLI so both versions align.")
		}
		// 終了コードを運ぶエラー（apply --all の集約等）はその code で終了する（→ docs/spec.md・ADR-0024）。
		var ec exitCodeX
		if errors.As(err, &ec) {
			os.Exit(ec.ExitCode())
		}
		os.Exit(1)
	}
}
