package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yasunori0418/nput/internal/manifest"
)

func newGitignoreCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "gitignore [name]",
		Short: "配置 target を .gitignore 向けに stdout 出力（project mode 限定・書き込みなし）",
		Long: "nput.<name> の配置 target を .gitignore 向けに stdout へ列挙する（ファイルは書き込まない）。" +
			"出力は root 相対 target に先頭 / を付けたアンカー形式（例: /.claude/skills/nix）で 1 行 1 件、" +
			"method（symlink / copy）を区別せず全 target を対象とする。project mode 限定で、" +
			"--all は projectRoot の全 config の target をソート + 重複除去して出力する。",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				if len(args) > 0 {
					return fmt.Errorf("nput: gitignore は <name> と --all を併用できません")
				}
				return runGitignoreAll()
			}
			if len(args) != 1 {
				return fmt.Errorf("nput: gitignore は <name> または --all が必要です")
			}
			return runGitignore(args[0])
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "projectRoot の全 config の target をソート + 重複除去して出力")
	return cmd
}

// runGitignore は単体 config の配置 target を列挙する。project mode 限定で、
// 非 project config（home / fixed）を指定したらエラー停止する（アンカー形式が git toplevel を前提とするため・→ ADR-0023）。
func runGitignore(name string) error {
	ep, err := discoverEntrypoint(flagFile)
	if err != nil {
		return err
	}
	system, err := currentSystem()
	if err != nil {
		return err
	}

	// rootKind 先取り eval で project mode を確認する（build より先に安価に弾く）。
	rootKind, _, err := evalRoot(ep, system, name)
	if err != nil {
		return err
	}
	if rootKind != manifest.RootKindProject {
		return fmt.Errorf("nput: gitignore は project mode 限定です（nput.%s は rootKind=%q・home / fixed では .gitignore のアンカーが意味を成しません）", name, rootKind)
	}

	targets, err := configTargets(ep, system, name)
	if err != nil {
		return err
	}
	printGitignore(targets)
	return nil
}

// runGitignoreAll は projectRoot の全 config の target をソート + 重複除去して列挙する
// （repo の .gitignore は 1 つなので一括列挙が自然・→ docs/spec.md・ADR-0018）。
// 非 project config は対象から除外する（--all は projectRoot の config のみ拾う）。
func runGitignoreAll() error {
	ep, err := discoverEntrypoint(flagFile)
	if err != nil {
		return err
	}
	system, err := currentSystem()
	if err != nil {
		return err
	}

	roots, err := evalAllRoots(ep, system)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(roots))
	for name := range roots {
		names = append(names, name)
	}
	sort.Strings(names)

	var all []string
	for _, name := range names {
		if roots[name].RootKind != manifest.RootKindProject {
			continue
		}
		targets, err := configTargets(ep, system, name)
		if err != nil {
			return err
		}
		all = append(all, targets...)
	}
	printGitignore(dedupeSorted(all))
	return nil
}

// configTargets は config をビルドして manifest.json を読み、配置 target を列挙する
// （method を区別せず全 entry・→ ADR-0019）。
func configTargets(ep *entrypoint, system, name string) ([]string, error) {
	store, err := buildManifestStorePath(ep, system, name)
	if err != nil {
		return nil, err
	}
	m, err := manifest.Load(store)
	if err != nil {
		return nil, err
	}
	targets := make([]string, 0, len(m.Entries))
	for _, e := range m.Entries {
		targets = append(targets, e.Target)
	}
	return targets, nil
}

// dedupeSorted はソート + 重複除去する（--all の一括列挙用）。
func dedupeSorted(in []string) []string {
	sort.Strings(in)
	out := in[:0]
	var prev string
	for i, s := range in {
		if i == 0 || s != prev {
			out = append(out, s)
		}
		prev = s
	}
	return out
}

// printGitignore は target を /-anchor 形式（先頭 /・末尾 / なし）で stdout へ 1 行 1 件出力する
// （→ docs/spec.md・ADR-0013）。stdout 専有原則のためパイプ安全（`nput gitignore <name> >> .gitignore`）。
func printGitignore(targets []string) {
	for _, t := range targets {
		fmt.Println(gitignoreAnchor(t))
	}
}

// gitignoreAnchor は root 相対 target を /-anchor 形式に整える。target は eval で絶対パス / 末尾 / を
// 持たないが、防御的に整形する（→ ADR-0013: 先頭 / でアンカー・ディレクトリ / ファイルとも末尾 / なし）。
func gitignoreAnchor(target string) string {
	t := strings.TrimSuffix(target, "/")
	t = strings.TrimPrefix(t, "/")
	return "/" + t
}
