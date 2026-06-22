package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/engine"
	"github.com/yasunori0418/nput/internal/manifest"
)

func newRollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback <name>",
		Short: "nput.<name> を前世代へ戻す（home mode 限定・名指し必須）",
		Long: "home mode の profile を 1 世代前へ戻す。profile ポインタ移動だけでは任意 root の FS は変わらないため、" +
			"現世代 N を baseline・前世代 N-1 を target として FS を再収束（N∖N-1 を保守的 stale 除去・N-1 を再配置）してから" +
			"profile ポインタを移す。名指し必須（--all 非対応）・前世代が無ければエラー停止。",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRollback(args[0])
		},
	}
}

// runRollback は eval 先取りで rootKind を確認（home mode 限定）し、engine.Rollback を駆動する。
func runRollback(name string) error {
	ep, err := discoverEntrypoint(flagFile)
	if err != nil {
		return err
	}
	system, err := currentSystem()
	if err != nil {
		return err
	}

	rootKind, fixedRoot, err := evalRoot(ep, system, name)
	if err != nil {
		return err
	}
	if rootKind != manifest.RootKindHome {
		return fmt.Errorf("nput: rollback は home mode 限定です（nput.%s は rootKind=%q・project / fixed は世代を公開しません）", name, rootKind)
	}

	res, err := engine.Rollback(engine.RollbackOptions{
		Name:         name,
		RootKind:     rootKind,
		FixedRoot:    fixedRoot,
		RootOverride: flagRoot,
	})
	if err != nil {
		return err
	}

	if flagVerbose {
		reportRollback(res, name)
	}
	return nil
}

// reportRollback は世代遷移と配置差分を stderr に出す（stdout は機械可読出力に専有・→ ADR-0023）。
func reportRollback(res *engine.RollbackResult, name string) {
	fmt.Fprintf(os.Stderr, "nput: rollback %s 完了 (世代 %d → %d, root=%s)\n", name, res.From, res.To, res.Root)
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
