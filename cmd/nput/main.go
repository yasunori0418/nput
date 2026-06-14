// Command nput は配置エンジン（internal/engine）を駆動する一次 UX の CLI（→ ADR-0007, ADR-0011）。
//
// 本スライス（#7）のスコープは project mode の `nput apply [<name>]` を flake entrypoint で
// end-to-end に通すことに絞る。実行順は docs/spec.md「実行フロー」の
// 「eval 先行（rootKind） → flock → ロック内 build → 配置 → --set → .pending 削除」に従い、
// flock〜build〜配置〜commit は engine が所有し、CLI は entrypoint 発見・nix eval/build の
// オーケストレーションを担う（→ ADR-0023, ADR-0025）。
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// グローバルフラグ（→ docs/spec.md「グローバルフラグ」）。本スライスは apply に必要な範囲のみ。
var (
	flagFile    string // -f/--file: entrypoint 明示
	flagRoot    string // --root: 解決 root の明示上書き
	flagNoWait  bool   // --no-wait: flock 競合時に待たず skip（shellHook 用）
	flagQuiet   bool   // --quiet: 進捗 / 配置レポートを抑制（warning / error は残す）
	flagVerbose bool   // --verbose: 内部実行する nix コマンドを開示
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "nput",
		Short: "Place fetched git repositories at arbitrary paths via symlink or copy.",
		// エラーは main で 1 度だけ出すため cobra の usage/error 自動表示は抑制する。
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	pf := root.PersistentFlags()
	pf.StringVarP(&flagFile, "file", "f", "", "entrypoint を明示（自動探索を上書き）")
	pf.StringVar(&flagRoot, "root", "", "解決 root を明示上書き（全モード共通）")
	pf.BoolVar(&flagNoWait, "no-wait", false, "flock 競合時に待たず skip（shellHook 用）")
	pf.BoolVar(&flagQuiet, "quiet", false, "進捗 / 配置レポートを抑制（warning / error は残す）")
	pf.BoolVar(&flagVerbose, "verbose", false, "内部実行する nix コマンド等の詳細を出力")

	root.AddCommand(newApplyCmd())
	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
