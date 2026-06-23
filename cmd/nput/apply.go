package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/engine"
	"github.com/yasunori0418/nput/internal/manifest"
)

var flagApplyAll bool // --all: nput.* を辞書順に全適用（root filter で絞り込み可）

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply [name]",
		Short: "Build nput.<name>, create a new generation, and apply it (defaults to nput.default)",
		Long: "Build and place the entrypoint's nput.<name>. " +
			"Omitting name applies nput.default (the flake default convention; an error if undefined). " +
			"--all applies all of nput.* in lexical order; --project-root / --home-root / --system-root narrow by root mode.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagApplyAll {
				if len(args) > 0 {
					return fmt.Errorf("nput: apply cannot combine <name> with --all")
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
	cmd.Flags().BoolVar(&flagApplyAll, "all", false, "Apply all of nput.* in lexical order (continues on partial failure; exits non-zero if any fails)")
	cmd.Flags().BoolVar(&flagRecopy, "recopy", false,
		"Unconditionally re-copy every copy target from src, overwriting (discards local edits; see ADR-0020)")
	cmd.Flags().BoolVar(&flagDryrun, "dryrun", false,
		"Show place/replace/remove/conflict/no-op with zero side effects (exit 2 on conflict; see ADR-0006)")
	cmd.Flags().StringVar(&flagManifest, "manifest", "",
		"Apply a pre-built manifest (link-farm path) directly (host/module activation seam; no entrypoint discovery or nix eval/build; see ADR-0026)")
	return cmd
}

// runApplyManifest は --manifest で渡されたビルド済み link-farm を engine へ直接適用する
// （module activation 経路）。entrypoint 発見・rootKind 先取り eval・nix build は行わず、
// rootKind は manifest.json から engine が読む（HM モジュールは homeRoot を pin するため home）。
// engine.Apply の Build=nil 経路（既ビルド済み LinkFarm）を CLI から駆動する（→ engine.Options）。
func runApplyManifest(name string) error {
	linkFarm, err := filepath.Abs(flagManifest)
	if err != nil {
		return fmt.Errorf("nput: cannot resolve the --manifest path (%s): %w", flagManifest, err)
	}

	res, err := engine.Apply(engine.Options{
		Name:         name,
		LinkFarm:     linkFarm,
		RootOverride: flagRoot,
		NoWait:       flagNoWait,
		Recopy:       flagRecopy,
	})
	if err != nil {
		if errors.Is(err, engine.ErrSkipped) {
			if flagVerbose {
				fmt.Fprintln(os.Stderr, "nput: skipped because another apply is in progress (run nput apply manually)")
			}
			return nil
		}
		return err
	}

	if flagVerbose {
		reportResult(res, name)
	}
	return nil
}

// runApply は docs/spec.md「実行フロー」を駆動する:
// entrypoint 発見 → rootKind 先取り eval → engine.Apply（flock → ロック内 build → 配置 → --set → .pending 削除）。
// --manifest 指定時は entrypoint 発見・nix eval/build を行わず、ビルド済み link-farm を engine へ直接渡す
// （module activation 経路・→ docs/spec.md「モジュール別動作仕様」・ADR-0003, ADR-0007）。
func runApply(name string) error {
	if flagManifest != "" {
		// --manifest は取得元を link-farm に固定するため、entrypoint 発見系フラグとは
		// 意味が衝突する（位置引数 name は profile 選択として直交し両立する・→ ADR-0026）。
		if flagFile != "" {
			return errors.New("nput: --manifest cannot be combined with -f (--manifest fixes the source to a pre-built link-farm)")
		}
		return runApplyManifest(name)
	}

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
			if flagVerbose {
				fmt.Fprintln(os.Stderr, "nput: skipped because another apply is in progress (run nput apply manually)")
			}
			return nil
		}
		return err
	}

	if flagVerbose {
		reportResult(res, name)
	}
	return nil
}

// printApplyPlan は apply --dryrun のプランを stdout に出す（機械可読出力を専有・1 行 1 アクション・
// → docs/spec.md ストリーム規律・ADR-0023, ADR-0024）。成功時沈黙でも抑制しない（stdout 専有原則・→ ADR-0031）。
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
		if flagVerbose {
			fmt.Fprintln(os.Stderr, "nput: apply --all: no matching configs")
		}
		return nil
	}

	// 2.5 --dryrun は副作用ゼロのプレビュー（flock / --set / pending gcroot を取らず build だけ
	//     read-only で回す）。選んだ各 config の plan を stdout へ集約し、終了コードは
	//     error(1) > conflict(2) > 0 の優先で決める（→ docs/spec.md・ADR-0024）。
	if flagDryrun {
		return runApplyAllDryRun(ep, system, selected, roots)
	}

	// 3. 各 config を独立に適用する。一部失敗しても続行し、失敗を集約する（各 config は独立 atomic）。
	var failures, skipped, applied int
	for _, name := range selected {
		ri := roots[name]
		res, err := applyOne(ep, system, name, ri.RootKind, ri.Root)
		if err != nil {
			if errors.Is(err, engine.ErrSkipped) {
				skipped++
				if flagVerbose {
					fmt.Fprintf(os.Stderr, "nput: skipped apply %s (another apply is in progress)\n", name)
				}
				continue
			}
			failures++
			// 部分失敗は握り潰さず stderr に出して続行する（→ docs/spec.md「一部失敗しても続行」）。
			fmt.Fprintf(os.Stderr, "nput: apply %s failed: %v\n", name, err)
			continue
		}
		applied++
		if flagVerbose {
			reportResult(res, name)
		}
	}

	// 4. 集約表示と終了コード（error(1) > conflict(2) > 0 の優先・→ docs/spec.md・ADR-0024）。
	if flagVerbose {
		fmt.Fprintf(os.Stderr, "nput: apply --all done (applied %d / skipped %d / failed %d / selected %d)\n",
			applied, skipped, failures, len(selected))
	}
	// conflict(2) は --dryrun（#13）の読み取り専用経路でのみ生じる。非 dryrun の --all は error/0 のみ。
	code := applyAllExitCode(failures > 0, false)
	if code == 0 {
		return nil
	}
	return &exitCodeError{code: code, msg: fmt.Sprintf("nput: apply --all: %d config(s) failed", failures)}
}

// runApplyAllDryRun は apply --all --dryrun を駆動する。選んだ各 config を読み取り専用で build し
// plan を stdout へ集約出力する（FS 書込・flock・--set・pending gcroot いずれも取らない・→ ADR-0023）。
// 終了コードは error(1) > conflict(2) > 0 の優先（→ docs/spec.md・ADR-0024）で決め、empty msg の
// exitError で運ぶ（単体 apply --dryrun の conflict=2 と対称・main は code だけで終了する）。
func runApplyAllDryRun(ep *entrypoint, system string, selected []string, roots map[string]rootInfo) error {
	code := aggregateDryRun(selected, func(name string) (*engine.Result, error) {
		ri := roots[name]
		return engine.Apply(engine.Options{
			Name:         name,
			RootKind:     ri.RootKind,
			FixedRoot:    ri.Root,
			RootOverride: flagRoot,
			Recopy:       flagRecopy,
			DryRun:       true,
			Build:        dryBuildFunc(ep, system, name),
		})
	})
	if code == 0 {
		return nil
	}
	return &exitError{code: code}
}

// aggregateDryRun は selected 各 config を applyDry で読み取り専用に回し、plan を stdout へ出して
// error / conflict を集約し終了コードを返す（apply 実体を注入してテスト可能にする seam）。
// 一部 config の build / eval 失敗（error）は握り潰さず stderr に出して続行し、最終コードへ反映する。
func aggregateDryRun(selected []string, applyDry func(name string) (*engine.Result, error)) int {
	var anyError, anyConflict bool
	for _, name := range selected {
		res, err := applyDry(name)
		if err != nil {
			anyError = true
			// 部分失敗は握り潰さず stderr に出して続行する（→ docs/spec.md「一部失敗しても続行」）。
			fmt.Fprintf(os.Stderr, "nput: apply %s --dryrun failed: %v\n", name, err)
			continue
		}
		printApplyPlan(res)
		if len(res.Conflicts) > 0 {
			anyConflict = true
		}
	}
	return applyAllExitCode(anyError, anyConflict)
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
		return "", fmt.Errorf("nput: --project-root / --home-root / --system-root may be specified only one at a time")
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
		return fmt.Errorf("nput: --project-root / --home-root / --system-root are modifiers for %s", modifier)
	}
	return nil
}

// reportResult は配置レポートを stderr に出す（stdout は機械可読出力に専有・→ ADR-0023）。
func reportResult(res *engine.Result, name string) {
	fmt.Fprintf(os.Stderr, "nput: apply %s done (root=%s)\n", name, res.Root)
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
