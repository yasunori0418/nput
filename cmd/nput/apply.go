package main

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/engine"
	"github.com/yasunori0418/nput/internal/manifest"
)

var flagApplyAll bool // --all: nput.* を辞書順に全適用（root filter で絞り込み可）

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply [name]",
		Short: "nput.<name> をビルドし新世代を作って適用（name 省略時は nput.default）",
		Long: "entrypoint の nput.<name> をビルドして配置する。" +
			"name を省略すると nput.default を適用する（flake の default 慣例・未定義ならエラー）。" +
			"--all で nput.* を辞書順に全適用し、--project-root / --home-root / --system-root で root モードを絞れる。",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagApplyAll {
				if len(args) > 0 {
					return fmt.Errorf("nput: apply は <name> と --all を併用できません")
				}
				return runApplyAll()
			}
			if err := ensureNoRootFilter("apply --all"); err != nil {
				return err
			}
			name := "default"
			if len(args) == 1 {
				name = args[0]
			}
			return runApply(name)
		},
	}
	cmd.Flags().BoolVar(&flagApplyAll, "all", false, "nput.* を辞書順に全適用（一部失敗しても続行・1 つでも失敗なら非ゼロ終了）")
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
	res, err := applyOne(ep, system, name, rootKind, fixedRoot)
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

// applyOne は 1 config に engine.Apply を回す（runApply / runApplyAll が共有）。rootKind / fixedRoot は
// 単体は evalRoot 先取り、--all は一括 eval（evalAllRoots）から渡す。build だけは config ごとに行う。
func applyOne(ep *entrypoint, system, name, rootKind, fixedRoot string) (*engine.Result, error) {
	return engine.Apply(engine.Options{
		Name:         name,
		RootKind:     rootKind,
		FixedRoot:    fixedRoot,
		RootOverride: flagRoot,
		NoWait:       flagNoWait,
		Recopy:       flagRecopy,
		Build:        buildFunc(ep, system, name),
	})
}

// runApplyAll は entrypoint の nput.* を辞書順に全適用する（→ docs/spec.md 実行フロー・ADR-0016, ADR-0024）。
// rootKind は 1 回の一括 eval で取り（プロセス起動を N→1）、build は atomic 性のため config ごと。
// 一部失敗しても残りを続行し、最後に集約表示・1 つでも失敗なら非ゼロ終了する。
func runApplyAll() error {
	filter, err := selectedRootFilter()
	if err != nil {
		return err
	}
	ep, err := discoverEntrypoint(flagFile)
	if err != nil {
		return err
	}
	system, err := currentSystem()
	if err != nil {
		return err
	}

	// 1. rootKind を 1 回の一括 eval で取得する（config 名→rootInfo マップ・→ ADR-0024）。
	roots, err := evalAllRoots(ep, system)
	if err != nil {
		return err
	}

	// 2. 辞書順に並べ、root filter（--project-root 等）が指定されていれば該当モードだけに絞る。
	names := make([]string, 0, len(roots))
	for name := range roots {
		names = append(names, name)
	}
	sort.Strings(names)
	var selected []string
	for _, name := range names {
		if filter == "" || roots[name].RootKind == filter {
			selected = append(selected, name)
		}
	}
	if len(selected) == 0 {
		if !flagQuiet {
			fmt.Fprintln(os.Stderr, "nput: apply --all: 対象の config がありません")
		}
		return nil
	}

	// 3. 各 config を独立に適用する。一部失敗しても続行し、失敗を集約する（各 config は独立 atomic）。
	var failures, skipped, applied int
	for _, name := range selected {
		ri := roots[name]
		res, err := applyOne(ep, system, name, ri.RootKind, ri.Root)
		if err != nil {
			if errors.Is(err, engine.ErrSkipped) {
				skipped++
				if !flagQuiet {
					fmt.Fprintf(os.Stderr, "nput: apply %s をスキップしました（別の apply が進行中）\n", name)
				}
				continue
			}
			failures++
			// 部分失敗は握り潰さず stderr に出して続行する（→ docs/spec.md「一部失敗しても続行」）。
			fmt.Fprintf(os.Stderr, "nput: apply %s に失敗しました: %v\n", name, err)
			continue
		}
		applied++
		if !flagQuiet {
			reportResult(res, name)
		}
	}

	// 4. 集約表示と終了コード（error(1) > conflict(2) > 0 の優先・→ docs/spec.md・ADR-0024）。
	if !flagQuiet {
		fmt.Fprintf(os.Stderr, "nput: apply --all 完了 (適用 %d / スキップ %d / 失敗 %d / 対象 %d)\n",
			applied, skipped, failures, len(selected))
	}
	// conflict(2) は --dryrun（#13）の読み取り専用経路でのみ生じる。非 dryrun の --all は error/0 のみ。
	code := applyAllExitCode(failures > 0, false)
	if code == 0 {
		return nil
	}
	return &exitCodeError{code: code, msg: fmt.Sprintf("nput: apply --all: %d 件の config が失敗しました", failures)}
}

// applyAllExitCode は apply --all の終了コードを priority error(1) > conflict(2) > 0 で決める
// （→ docs/spec.md「出力ストリームと終了コード」・ADR-0024）。単純な最大値（2 > 1）は採らない
// （conflict が深刻な eval / engine error を CI で隠すため）。conflict は --dryrun 経路でのみ生じる。
func applyAllExitCode(anyError, anyConflict bool) int {
	switch {
	case anyError:
		return 1
	case anyConflict:
		return 2
	default:
		return 0
	}
}

// selectedRootFilter は --project-root / --home-root / --system-root から root モードフィルタを返す
// （未指定は ""・複数指定はエラー・→ ADR-0017）。返り値は manifest.RootKind* 文字列。
func selectedRootFilter() (string, error) {
	var modes []string
	if flagProjectRoot {
		modes = append(modes, manifest.RootKindProject)
	}
	if flagHomeRoot {
		modes = append(modes, manifest.RootKindHome)
	}
	if flagSystemRoot {
		modes = append(modes, manifest.RootKindSystem)
	}
	if len(modes) > 1 {
		return "", fmt.Errorf("nput: --project-root / --home-root / --system-root は同時に 1 つだけ指定できます")
	}
	if len(modes) == 0 {
		return "", nil
	}
	return modes[0], nil
}

// ensureNoRootFilter は root filter が --all 以外で使われたときにエラーにする
// （フィルタは --all の修飾で、名指し apply では <name> が 1 config を pin するため無意味・→ ADR-0017）。
func ensureNoRootFilter(modifier string) error {
	if flagProjectRoot || flagHomeRoot || flagSystemRoot {
		return fmt.Errorf("nput: --project-root / --home-root / --system-root は %s の修飾です", modifier)
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
