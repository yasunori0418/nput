package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/engine"
)

func newApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "apply [name]",
		Short: "nput.<name> をビルドし新世代を作って適用（name 省略時は nput.default）",
		Long: "entrypoint の nput.<name> をビルドして配置する。" +
			"name を省略すると nput.default を適用する（flake の default 慣例・未定義ならエラー）。",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "default"
			if len(args) == 1 {
				name = args[0]
			}
			return runApply(name)
		},
	}
}

// runApply は docs/spec.md「実行フロー」を駆動する:
// entrypoint 発見 → rootKind 先取り eval → engine.Apply（flock → ロック内 build → 配置 → --set → .pending 削除）。
func runApply(name string) error {
	ep, err := discoverEntrypoint(flagFile)
	if err != nil {
		return err
	}
	system, err := currentSystem()
	if err != nil {
		return err
	}

	// 1. rootKind を build 前に先取りする（profileDir 確定 → flock → build の順を成立させる・→ ADR-0023）。
	rootKind, fixedRoot, err := evalRoot(ep, system, name)
	if err != nil {
		return err
	}

	// 2. engine を駆動する（flock 取得・ロック内 build・配置・コミット・.pending 削除は engine が所有）。
	res, err := engine.Apply(engine.Options{
		Name:         name,
		RootKind:     rootKind,
		FixedRoot:    fixedRoot,
		RootOverride: flagRoot,
		NoWait:       flagNoWait,
		Build:        buildFunc(ep, system, name),
	})
	if err != nil {
		if errors.Is(err, engine.ErrSkipped) {
			// try-lock skip は正常スキップ（exit 0・→ docs/spec.md 終了コード表）。
			if !flagQuiet {
				fmt.Fprintln(os.Stderr, "nput: 別の apply が進行中のためスキップしました（手動で nput apply を実行してください）")
			}
			return nil
		}
		return err
	}

	if !flagQuiet {
		reportResult(res, name)
	}
	return nil
}

// reportResult は配置レポートを stderr に出す（stdout は機械可読出力に専有・→ ADR-0023）。
func reportResult(res *engine.Result, name string) {
	fmt.Fprintf(os.Stderr, "nput: apply %s 完了 (root=%s)\n", name, res.Root)
	for _, t := range res.Placed {
		fmt.Fprintf(os.Stderr, "  placed   %s\n", t)
	}
	for _, t := range res.Replaced {
		fmt.Fprintf(os.Stderr, "  replaced %s\n", t)
	}
	for _, t := range res.Removed {
		fmt.Fprintf(os.Stderr, "  removed  %s\n", t)
	}
	if len(res.Placed)+len(res.Replaced)+len(res.Removed) == 0 {
		fmt.Fprintln(os.Stderr, "  no-op")
	}
}
