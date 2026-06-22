package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/engine"
)

func newResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset <name> [target...]",
		Short: "配置物を無い状態へ戻す（FS-only teardown・名指し必須・--all 非対応）",
		Long: "nput.<name> の配置物を無い状態へ戻す teardown。target 省略で全 entry、target 指定でその entry のみ撤去する。" +
			"symlink は保守的不変条件（nput 管理・記録通りのみ）で除去し foreign は残す。copy target は削除する（データ損失リスクのため確認）。" +
			"profile / 世代は触らない（FS-only）。名指し必須（--all 非対応）。" +
			"--dryrun は副作用ゼロで削除対象を表示して exit する（confirm / flock なし）。",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReset(args[0], args[1:], flagDryrun)
		},
	}
	cmd.Flags().BoolVar(&flagDryrun, "dryrun", false,
		"副作用ゼロで削除対象（symlink / copy target）を表示して exit（confirm / flock なし・→ ADR-0021）")
	return cmd
}

// runReset は eval 先取りで rootKind（→ profileDir）を確定し、engine.Reset を駆動する。
// --dryrun は読み取り専用でプランを stdout に出して exit 0。非 dryrun は TTY 確認 / --yes を要求する。
func runReset(name string, targets []string, dryrun bool) error {
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

	// --dryrun: 副作用ゼロのプレビュー（flock / confirm なし・終了コードは対象有無に依らず 0・→ ADR-0021）。
	if dryrun {
		res, err := engine.Reset(engine.ResetOptions{
			Name:         name,
			RootKind:     rootKind,
			FixedRoot:    fixedRoot,
			RootOverride: flagRoot,
			Targets:      targets,
			DryRun:       true,
		})
		if err != nil {
			return err
		}
		printResetPlan(res)
		return nil
	}

	// 非 dryrun は破壊的操作。確認方針（スキップ / プロンプト / 拒否）を --yes と TTY 状態から決める。
	needPrompt, err := confirmPolicy(flagYes, isInteractive())
	if err != nil {
		return err
	}

	// プロンプトが要るときだけ confirm を渡す。プロンプトは算出済みプランを表示する。
	var confirm func(*engine.ResetResult) (bool, error)
	if needPrompt {
		confirm = func(res *engine.ResetResult) (bool, error) {
			reportResetTargets(res, name)
			return promptYesNo("上記を削除します。続行しますか？")
		}
	}

	res, err := engine.Reset(engine.ResetOptions{
		Name:         name,
		RootKind:     rootKind,
		FixedRoot:    fixedRoot,
		RootOverride: flagRoot,
		Targets:      targets,
		Confirm:      confirm,
	})
	if err != nil {
		return err
	}
	if res.Aborted {
		fmt.Fprintln(os.Stderr, "nput: reset を中止しました")
		return nil
	}
	if flagVerbose {
		reportResetResult(res, name)
	}
	return nil
}

// printResetPlan は reset --dryrun の削除対象を stdout に出す（機械可読出力を専有・1 行 1 件・
// → docs/spec.md ストリーム規律・ADR-0023）。成功時沈黙でも抑制しない（stdout 専有原則・→ ADR-0031）。
func printResetPlan(res *engine.ResetResult) {
	for _, t := range res.RemovedSymlinks {
		fmt.Printf("remove-symlink\t%s\n", t)
	}
	for _, t := range res.RemovedCopies {
		fmt.Printf("remove-copy\t%s\n", t)
	}
	for _, t := range res.KeptForeign {
		fmt.Printf("keep-foreign\t%s\n", t)
	}
	if len(res.RemovedSymlinks)+len(res.RemovedCopies)+len(res.KeptForeign) == 0 {
		fmt.Fprintln(os.Stderr, "nput: reset --dryrun: 削除対象はありません")
	}
}

// reportResetTargets は確認プロンプト前に削除予定を stderr に出す（進捗扱い・stdout は機械可読専有）。
func reportResetTargets(res *engine.ResetResult, name string) {
	fmt.Fprintf(os.Stderr, "nput: reset %s 削除対象 (root=%s):\n", name, res.Root)
	for _, t := range res.RemovedSymlinks {
		fmt.Fprintf(os.Stderr, "  symlink %s\n", t)
	}
	for _, t := range res.RemovedCopies {
		fmt.Fprintf(os.Stderr, "  copy    %s\n", t)
	}
	for _, t := range res.KeptForeign {
		fmt.Fprintf(os.Stderr, "  keep    %s (foreign / 記録不一致のため残します)\n", t)
	}
	if len(res.RemovedSymlinks)+len(res.RemovedCopies) == 0 {
		fmt.Fprintln(os.Stderr, "  （削除対象なし）")
	}
}

// reportResetResult は実削除結果を stderr に出す（stdout は機械可読出力に専有・→ ADR-0023）。
func reportResetResult(res *engine.ResetResult, name string) {
	fmt.Fprintf(os.Stderr, "nput: reset %s 完了 (root=%s)\n", name, res.Root)
	for _, t := range res.RemovedSymlinks {
		fmt.Fprintf(os.Stderr, "  removed-symlink %s\n", t)
	}
	for _, t := range res.RemovedCopies {
		fmt.Fprintf(os.Stderr, "  removed-copy    %s\n", t)
	}
	for _, t := range res.KeptForeign {
		fmt.Fprintf(os.Stderr, "  kept            %s (foreign / 記録不一致)\n", t)
	}
	if len(res.RemovedSymlinks)+len(res.RemovedCopies) == 0 {
		fmt.Fprintln(os.Stderr, "  no-op")
	}
}

// confirmPolicy は破壊的 reset の確認方針を --yes と TTY 状態から決める（→ ADR-0025 §5）。
//   - --yes: 確認スキップ（needPrompt=false・err=nil）
//   - --yes 無し + 対話環境: 確認プロンプト要求（needPrompt=true）
//   - --yes 無し + 非対話環境: ハング / 空入力誤削除を防ぐため即エラー（refuse）
//
// runReset から isInteractive() の結果を渡して使う（nix 非依存で単体テスト可能な seam）。
func confirmPolicy(yes, interactive bool) (needPrompt bool, err error) {
	if yes {
		return false, nil
	}
	if !interactive {
		return false, errors.New("nput: 非対話環境では --yes なしの破壊的 reset を拒否します " +
			"(refusing destructive reset without --yes in non-interactive context)")
	}
	return true, nil
}

// isInteractive は stdin が TTY か（端末に接続されているか）を返す（→ ADR-0025 §5）。
// stdlib のみで判定する（os.ModeCharDevice）。パイプ / リダイレクト / CI では false。
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// promptYesNo は stdin から y/N を読み、yes 系の入力のときだけ true を返す（既定 No）。
func promptYesNo(msg string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", msg)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
