package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/engine"
)

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
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
	cmd.Flags().BoolVar(&flagRecopy, "recopy", false,
		"config 内の全 copy target を src から無条件上書き再コピー（ローカル編集は破棄・→ ADR-0020）")
	cmd.Flags().BoolVar(&flagDryrun, "dryrun", false,
		"副作用ゼロで place/replace/remove/conflict/no-op を表示（conflict 検出時 exit 2・→ ADR-0006）")
	return cmd
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

	// 1.5 --dryrun は副作用ゼロのプレビュー（flock / pending gcroot を取らず build だけ read-only で回す・→ ADR-0023）。
	if flagDryrun {
		res, err := engine.Apply(engine.Options{
			Name:         name,
			RootKind:     rootKind,
			FixedRoot:    fixedRoot,
			RootOverride: flagRoot,
			Recopy:       flagRecopy,
			DryRun:       true,
			Build:        dryBuildFunc(ep, system, name),
		})
		if err != nil {
			return err
		}
		printApplyPlan(res)
		// conflict があれば exit 2（CI の事前 gate・→ docs/spec.md 終了コード表）。
		if len(res.Conflicts) > 0 {
			return &exitError{code: 2}
		}
		return nil
	}

	// 2. engine を駆動する（flock 取得・ロック内 build・配置・コミット・.pending 削除は engine が所有）。
	res, err := engine.Apply(engine.Options{
		Name:         name,
		RootKind:     rootKind,
		FixedRoot:    fixedRoot,
		RootOverride: flagRoot,
		NoWait:       flagNoWait,
		Recopy:       flagRecopy,
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

// printApplyPlan は apply --dryrun のプランを stdout に出す（機械可読出力を専有・1 行 1 アクション・
// → docs/spec.md ストリーム規律・ADR-0023, ADR-0024）。--quiet 下でも抑制しない（stdout 専有原則）。
// conflict 行も plan の一部として stdout に載せ、終了コード（exit 2）が機械判別を補う。
func printApplyPlan(res *engine.Result) {
	for _, t := range res.Placed {
		fmt.Printf("place\t%s\n", t)
	}
	for _, t := range res.Replaced {
		fmt.Printf("replace\t%s\n", t)
	}
	for _, t := range res.Copied {
		fmt.Printf("copy\t%s\n", t)
	}
	for _, t := range res.Removed {
		fmt.Printf("remove\t%s\n", t)
	}
	for _, c := range res.Conflicts {
		fmt.Printf("conflict\t%s\n", c)
	}
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
	for _, t := range res.Copied {
		fmt.Fprintf(os.Stderr, "  copied   %s\n", t)
	}
	for _, t := range res.Recopied {
		fmt.Fprintf(os.Stderr, "  recopied %s\n", t)
	}
	for _, t := range res.Removed {
		fmt.Fprintf(os.Stderr, "  removed  %s\n", t)
	}
	if len(res.Placed)+len(res.Replaced)+len(res.Copied)+len(res.Recopied)+len(res.Removed) == 0 {
		fmt.Fprintln(os.Stderr, "  no-op")
	}
}
