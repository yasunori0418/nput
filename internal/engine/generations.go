package engine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/yasunori0418/nput/internal/lock"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/paths"
	"github.com/yasunori0418/nput/internal/planner"
)

// 世代操作は `nix-env --profile <profileDir>/profile` 系で統一する（→ docs/spec.md 世代管理仕様・
// ADR-0015, ADR-0025）。コミット（--set）は commit.go、ここでは rollback（--switch-generation）・
// 一覧（--list-generations）を扱う。--set と同じく注入可能にして tmpdir テストでは nix を呼ばない。

// Generation は profile の 1 世代（nix-env --list-generations の 1 行）。
type Generation struct {
	Number  int    // 世代番号
	Date    string // 作成日時（nix-env の表示そのまま。解析せず原文を運ぶ）
	Current bool   // 現世代（profile リンクが指す）か
}

// ListGenerationsFunc は世代一覧の取得点（既定は nix-env --list-generations）。tmpdir テストで差し替える。
type ListGenerationsFunc func(profileLink string) ([]Generation, error)

// SwitchGenerationFunc は profile ポインタ移動点（既定は nix-env --switch-generation）。tmpdir テストで差し替える。
type SwitchGenerationFunc func(profileLink string, gen int) error

// ProfileOptions は profileDir を確定するための入力（Rollback / list-generations が共有）。
type ProfileOptions struct {
	Name         string
	RootKind     string
	FixedRoot    string
	RootOverride string
	WorkDir      string
	StateDir     string
	Git          GitFunc
}

// ProfileFor は root を解決し profileDir レイアウトを確定する（apply の前段と同型・→ ADR-0023, ADR-0024）。
// build をしない rollback / list-generations が flock / 世代読みのために共通で使う。
func ProfileFor(opts ProfileOptions) (paths.Profile, string, error) {
	root, err := resolveRoot(opts.RootKind, opts.FixedRoot, opts.RootOverride, opts.WorkDir, opts.Git)
	if err != nil {
		return paths.Profile{}, "", err
	}
	stateDir := opts.StateDir
	if stateDir == "" {
		stateDir, err = paths.StateDir()
		if err != nil {
			return paths.Profile{}, "", err
		}
	}
	prof := paths.Resolve(stateDir, opts.Name, opts.RootKind, root, opts.RootOverride != "")
	return prof, root, nil
}

// ListGenerations は profileLink の世代一覧を返す（既定の nix-env 実装）。CLI の list-generations が使う。
func ListGenerations(profileLink string) ([]Generation, error) {
	return nixEnvListGenerations(profileLink)
}

// RollbackOptions は Rollback の入力。Rollback は home mode 限定だが、その判定は CLI が担い
// engine は rootKind に依らず profileDir を解決して前世代へ収束させる。
type RollbackOptions struct {
	Name         string
	RootKind     string
	FixedRoot    string
	RootOverride string
	WorkDir      string
	StateDir     string

	// ListGenerations / SwitchGeneration は nix-env 呼び出しの差し替え（nil = 既定の nix-env 実装）。
	ListGenerations  ListGenerationsFunc
	SwitchGeneration SwitchGenerationFunc
	// Git は git toplevel 解決の差し替え（nil = gitutil.Toplevel）。home mode では未使用。
	Git GitFunc
	// Warnf は warning の出力先（nil = stderr）。
	Warnf func(format string, args ...any)
}

// RollbackResult は Rollback の結果レポート。Result（配置差分）に世代遷移 From→To を添える。
type RollbackResult struct {
	Result
	From int // 戻る前の現世代 N
	To   int // 戻った先の前世代 N-1
}

// Rollback は home mode の profile を 1 世代前へ戻す。profile ポインタ移動だけでは任意 root の FS は
// 変わらないため、現世代 N を baseline・前世代 N-1 を target として planner で diff し、N∖N-1 を保守的に
// stale 除去・N-1 の entry を再配置してから **最後に** profile ポインタを移す（→ docs/spec.md ロールバック・ADR-0015）。
// ポインタを先に動かすと baseline が N-2 へずれ stale 除去が誤るため、FS 収束を先・ポインタ移動を最後にする。
func Rollback(opts RollbackOptions) (*RollbackResult, error) {
	warnf := opts.Warnf
	if warnf == nil {
		warnf = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}
	listFn := opts.ListGenerations
	if listFn == nil {
		listFn = nixEnvListGenerations
	}
	switchFn := opts.SwitchGeneration
	if switchFn == nil {
		switchFn = nixEnvSwitchGeneration
	}

	// 1. profileDir 確定（root 解決 → レイアウト・apply と共通の前段）。
	prof, root, err := ProfileFor(ProfileOptions{
		Name: opts.Name, RootKind: opts.RootKind, FixedRoot: opts.FixedRoot,
		RootOverride: opts.RootOverride, WorkDir: opts.WorkDir, StateDir: opts.StateDir, Git: opts.Git,
	})
	if err != nil {
		return nil, err
	}

	// profileDir が無ければ一度も apply していない → 戻る世代が存在しない。
	if _, err := os.Stat(prof.Dir); err != nil {
		return nil, fmt.Errorf("nput: profile がありません（一度も apply していません）: %s", prof.Dir)
	}

	// 2. flock（blocking）で並行 apply / rollback と直列化する（→ ADR-0013）。
	l, err := lock.Acquire(prof.Dir, true)
	if err != nil {
		return nil, fmt.Errorf("nput: flock の取得に失敗しました (%s): %w", prof.Dir, err)
	}
	defer func() { _ = l.Release() }()

	// 3. 世代一覧から現世代 N と前世代 N-1 を特定する。
	gens, err := listFn(prof.Profile)
	if err != nil {
		return nil, err
	}
	curIdx := -1
	for i, g := range gens {
		if g.Current {
			curIdx = i
		}
	}
	if curIdx < 0 {
		return nil, fmt.Errorf("nput: 現世代を特定できません（profile: %s）", prof.Profile)
	}
	if curIdx == 0 {
		return nil, fmt.Errorf("nput: 前世代がありません（最古の世代のため rollback できません）")
	}
	cur := gens[curIdx]
	prev := gens[curIdx-1]

	// 4. baseline = 現世代 N の manifest（FS の現状）/ target = 前世代 N-1 の manifest。
	baseline, err := manifest.Load(prof.Profile)
	if err != nil {
		return nil, fmt.Errorf("nput: 現世代 manifest を読めません: %w", err)
	}
	target, err := manifest.Load(paths.GenerationLink(prof.Profile, prev.Number))
	if err != nil {
		return nil, fmt.Errorf("nput: 前世代 manifest を読めません (世代 %d): %w", prev.Number, err)
	}

	// 5. planner で N∖N-1 stale 除去・N-1 entry 再配置のプランを算出する（apply エンジンを (baseline, target) 差し替えで再利用）。
	plan, err := planner.Compute(baseline, target, root, planner.OSFS)
	if err != nil {
		return nil, err
	}
	if len(plan.Conflicts) > 0 {
		c := plan.Conflicts[0]
		return nil, fmt.Errorf("nput: %s (target: %s)", c.Reason, c.Entry.Target)
	}

	// 6. プランを実 FS に反映（新規/張替を先に、stale 除去を最後に・→ ADR-0006）。
	a := &applier{opts: Options{Warnf: warnf}, result: &Result{Root: root, ProfileDir: prof.Dir}}
	a.profile = prof
	a.root = root
	a.emitWarnings(plan.Warnings)
	if err := a.place(plan.Place); err != nil {
		return nil, err
	}
	if err := a.removeStale(plan.Remove); err != nil {
		return nil, err
	}

	// 7. 最後に profile ポインタを N-1 へ移す（→ docs/spec.md ロールバック手順 3）。
	if err := switchFn(prof.Profile, prev.Number); err != nil {
		return nil, fmt.Errorf("nput: profile ポインタの移動に失敗しました（--switch-generation %d）: %w", prev.Number, err)
	}

	return &RollbackResult{Result: *a.result, From: cur.Number, To: prev.Number}, nil
}

// nixEnvListGenerations は既定の世代一覧取得（nix-env --profile <p> --list-generations）。
// stdout を捕捉して解析する（機械可読出力の取り込み）。
func nixEnvListGenerations(profileLink string) ([]Generation, error) {
	if _, err := exec.LookPath("nix-env"); err != nil {
		return nil, fmt.Errorf("nix-env が PATH にありません: %w", err)
	}
	cmd := exec.Command("nix-env", "--profile", profileLink, "--list-generations")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		trimmed := strings.TrimSpace(stderr.String())
		if trimmed != "" {
			return nil, fmt.Errorf("nput: nix-env --list-generations に失敗しました: %w\n%s", err, trimmed)
		}
		return nil, fmt.Errorf("nput: nix-env --list-generations に失敗しました: %w", err)
	}
	return parseGenerations(stdout.String())
}

// nixEnvSwitchGeneration は既定の profile ポインタ移動（nix-env --profile <p> --switch-generation <gen>）。
// nix の出力は stderr に流す（stdout は機械可読出力に専有・→ docs/spec.md ストリーム規律）。
func nixEnvSwitchGeneration(profileLink string, gen int) error {
	if _, err := exec.LookPath("nix-env"); err != nil {
		return fmt.Errorf("nix-env が PATH にありません: %w", err)
	}
	cmd := exec.Command("nix-env", "--profile", profileLink, "--switch-generation", strconv.Itoa(gen))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// parseGenerations は `nix-env --list-generations` の出力を解析する。各行は
// 「<番号>   <日時>   [(current)]」形式（先頭・区切りは空白可変）。
func parseGenerations(out string) ([]Generation, error) {
	var gens []Generation
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("nput: 世代一覧の行を解析できません: %q", line)
		}
		g := Generation{Number: n}
		end := len(fields)
		if fields[end-1] == "(current)" {
			g.Current = true
			end--
		}
		g.Date = strings.Join(fields[1:end], " ")
		gens = append(gens, g)
	}
	return gens, nil
}
