// Package engine is the placement core: it resolves root, fixes the profileDir
// layout, takes a flock, places store-symlinks via native FS ops, removes stale
// links conservatively, and commits a generation with `nix-env --set`
// (→ ADR-0002, ADR-0005, ADR-0006, ADR-0011, ADR-0013, ADR-0015, ADR-0025).
//
// 本スライス（#6）の最小核は project mode の store-symlink 配置に絞る。配置〜
// stale 除去（ネイティブ FS）は nix 不使用でユニット/統合テスト可能にし、commit
// （nix-env --set）は注入可能にして tmpdir テストでは nix を呼ばない（→ ADR-0006）。
package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yasunori0418/nput/internal/gitutil"
	"github.com/yasunori0418/nput/internal/lock"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/paths"
	"github.com/yasunori0418/nput/internal/planner"
)

// CommitFunc は配置成功後に世代を積むコミット点（→ ADR-0006, docs/spec.md 実行フロー f）。
// 既定は nix-env --profile <profileLink> --set <linkFarm>。tmpdir テストはこれを
// 差し替えて nix を呼ばずに検証する。
type CommitFunc func(profileLink, linkFarm string) error

// BuildFunc は flock 取得後・ロック内で link-farm を build するコールバック（→ docs/spec.md 実行フロー 2b・ADR-0011, ADR-0023）。
// 引数 pending は out-link を張る先（<profileDir>/.pending）。返り値は build された link-farm の
// store パス（os.Readlink の解決後）。CLI が `nix build <ep>#nput.<system>.<name> --out-link <pending>`
// を差し込む。nil のときは opts.LinkFarm を既ビルド済みとして使う（tmpdir テスト経路）。
type BuildFunc func(pending string) (linkFarm string, err error)

// GitFunc は project mode の git toplevel 解決（→ ADR-0005）。既定は gitutil.Toplevel。
type GitFunc func(dir string) (string, error)

// Options は Apply の入力。
type Options struct {
	// LinkFarm は manifest.json と GC アンカー symlink farm を含む link-farm ディレクトリ。
	// Build を渡さない経路（tmpdir テスト）でのみ使う既ビルド済み link-farm（→ ADR-0011）。
	LinkFarm string
	// Name は config 名（profile を一意特定する。entrypoint の nput.<name> 由来）。
	Name string
	// RootKind は eval 先取りで得た root kind（→ docs/spec.md 実行フロー 1・ADR-0023）。
	// Build 経路では manifest 未ビルドのため必須。空のときは LinkFarm の manifest から得る。
	RootKind string
	// FixedRoot は rootKind=fixed のときの絶対パス（eval 先取りの passthru.root 由来）。
	// 空かつ Build=nil のときは LinkFarm の manifest.Root.Root を使う。
	FixedRoot string
	// RootOverride は --root 上書き（空 = なし）。明示時は全モードで roothash キー（→ ADR-0023）。
	RootOverride string
	// WorkDir は project mode の git toplevel 解決の起点（空 = os.Getwd）。
	WorkDir string
	// StateDir は profile 基底 <state> の上書き（空 = paths.StateDir で解決・主にテスト用）。
	StateDir string
	// NoWait は flock を try-lock にする（shellHook 経路・保持中なら ErrSkipped・→ ADR-0013）。
	NoWait bool

	// Build はロック内 build の差し替え（nil = opts.LinkFarm を既ビルド済みとして使う）。
	Build BuildFunc
	// Git は git toplevel 解決の差し替え（nil = gitutil.Toplevel）。
	Git GitFunc
	// Commit は世代コミットの差し替え（nil = nix-env --set）。
	Commit CommitFunc
	// Warnf は warning の出力先（nil = stderr）。foreign symlink 等の可視化に使う（→ ADR-0015）。
	Warnf func(format string, args ...any)
}

// Result は Apply の結果レポート（dryrun / レポート表示・テスト検証用）。
type Result struct {
	Root       string   // 解決後の絶対 root パス
	ProfileDir string   // 確定した profileDir
	Placed     []string // 新規配置した target
	Replaced   []string // 既存 symlink を張り替えた target
	Removed    []string // stale 除去した target
	Skipped    bool     // try-lock 競合で skip した（NoWait 経路）
}

// ErrSkipped は NoWait 経路で他の apply が進行中のため skip したことを表す。
var ErrSkipped = lock.ErrLocked

// Apply は project mode の store-symlink 配置を行い、成功後に世代をコミットする。
// docs/spec.md「実行フロー」の engine 駆動部（2. engine を駆動）に対応し、
// 「flock → ロック内 build → 配置 → --set → .pending 削除」の順を engine が所有する。
// build は opts.Build（CLI が nix build を差し込む）にロック内で委譲し、未指定なら
// opts.LinkFarm を既ビルド済みとして使う（tmpdir テスト経路）。
func Apply(opts Options) (*Result, error) {
	a := &applier{opts: opts, result: &Result{}}
	if a.opts.Warnf == nil {
		a.opts.Warnf = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}

	// root kind: Build 経路では manifest 未ビルドのため eval 先取りの opts.RootKind を使う。
	// 既ビルド済み LinkFarm 経路（テスト）では先に manifest を読んで rootKind を得る。
	rootKind := opts.RootKind
	fixedRoot := opts.FixedRoot
	if opts.Build == nil {
		m, err := manifest.Load(opts.LinkFarm)
		if err != nil {
			return nil, err
		}
		a.manifest = m
		if rootKind == "" {
			rootKind = m.Root.RootKind
		}
		if fixedRoot == "" {
			fixedRoot = m.Root.Root
		}
	}

	// 1. root 解決 → profileDir 確定（→ docs/spec.md「root の解決」）。
	root, err := a.resolveRoot(rootKind, fixedRoot)
	if err != nil {
		return nil, err
	}
	a.root = root
	a.result.Root = root

	stateDir := opts.StateDir
	if stateDir == "" {
		stateDir, err = paths.StateDir()
		if err != nil {
			return nil, err
		}
	}
	a.profile = paths.Resolve(stateDir, opts.Name, rootKind, root, opts.RootOverride != "")
	a.result.ProfileDir = a.profile.Dir

	// 2. profileDir / backref を用意（flock は profileDir を開くため先に作る）。
	if err := a.ensureProfileDir(); err != nil {
		return nil, err
	}

	// 3. 解決後 profileDir 単位で flock を取得し直列化する（→ ADR-0013）。
	l, err := lock.Acquire(a.profile.Dir, !opts.NoWait)
	if err != nil {
		if opts.NoWait && err == lock.ErrLocked {
			a.result.Skipped = true
			return a.result, ErrSkipped
		}
		return nil, fmt.Errorf("nput: flock の取得に失敗しました (%s): %w", a.profile.Dir, err)
	}
	defer func() { _ = l.Release() }()

	// 4. ロック内で link-farm を build する（→ docs/spec.md 実行フロー 2b・ADR-0023）。
	//    build をロック内に閉じることで並行 apply の .pending 奪い合いが構造的に消える。
	if opts.Build != nil {
		linkFarm, err := opts.Build(a.profile.Pending)
		if err != nil {
			return nil, err
		}
		m, err := manifest.Load(linkFarm)
		if err != nil {
			return nil, err
		}
		a.opts.LinkFarm = linkFarm
		a.manifest = m
	}

	// 5. 前世代 manifest を読む（無ければ初回 = stale 除去ゼロ）。
	prev := a.loadPrevManifest()

	// 6. planner で place/replace/remove プランを算出する（純ロジック・→ internal/planner）。
	plan, err := planner.Compute(prev, a.manifest, a.root, planner.OSFS)
	if err != nil {
		return nil, err
	}
	if len(plan.Conflicts) > 0 {
		c := plan.Conflicts[0]
		return nil, fmt.Errorf("nput: %s (target: %s)", c.Reason, c.Entry.Target)
	}
	a.emitWarnings(plan.Warnings)

	// 6.5 out-of-store のリンク先存在を配置直前に検査（dangling 禁止・→ ADR-0001, ADR-0013）。
	//     FS 変更前に閉じるため、不在なら何も配置せずエラー停止する。
	if err := a.checkOutOfStore(); err != nil {
		return nil, err
	}

	// 7. プランを実 FS に反映（新規 / 張替を先に、stale 除去を最後に・→ ADR-0006）。
	if err := a.place(plan.Place); err != nil {
		return nil, err
	}
	if err := a.removeStale(plan.Remove); err != nil {
		return nil, err
	}

	// 8. 世代コミット（→ docs/spec.md 実行フロー 2f）。
	commit := opts.Commit
	if commit == nil {
		commit = nixEnvCommit
	}
	if err := commit(a.profile.Profile, a.opts.LinkFarm); err != nil {
		return nil, fmt.Errorf("nput: 世代コミット（nix-env --set）に失敗しました: %w", err)
	}

	// 9. --set 成功後に .pending を削除する（世代リンクが gcroot を引き継ぐ・→ ADR-0011, ADR-0025）。
	//    build 経路でのみ pending を張るため、その経路でのみ削除する。
	if opts.Build != nil {
		if err := os.Remove(a.profile.Pending); err != nil && !os.IsNotExist(err) {
			a.opts.Warnf("nput: .pending out-link を削除できませんでした (%s): %v", a.profile.Pending, err)
		}
	}

	return a.result, nil
}

type applier struct {
	opts     Options
	manifest *manifest.Manifest
	profile  paths.Profile
	root     string
	result   *Result
}

func (a *applier) resolveRoot(rootKind, fixedRoot string) (string, error) {
	if a.opts.RootOverride != "" {
		return filepath.Abs(a.opts.RootOverride)
	}
	switch rootKind {
	case manifest.RootKindProject:
		dir := a.opts.WorkDir
		if dir == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("nput: cwd を取得できません: %w", err)
			}
			dir = cwd
		}
		git := a.opts.Git
		if git == nil {
			git = gitutil.Toplevel
		}
		return git(dir)
	case manifest.RootKindHome:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("nput: $HOME を解決できません: %w", err)
		}
		return home, nil
	case manifest.RootKindFixed:
		if fixedRoot == "" {
			return "", fmt.Errorf("nput: rootKind=fixed なのに root パスがありません")
		}
		return filepath.Abs(fixedRoot)
	case manifest.RootKindSystem:
		return "", fmt.Errorf("nput: root = systemRoot (system mode) は未実装です（→ ADR-0013）")
	case "":
		return "", fmt.Errorf("nput: rootKind が未確定です（eval 先取りまたは manifest が必要）")
	default:
		return "", fmt.Errorf("nput: 未知の rootKind: %q", rootKind)
	}
}

func (a *applier) ensureProfileDir() error {
	if err := os.MkdirAll(a.profile.Dir, 0o755); err != nil {
		return fmt.Errorf("nput: profileDir を作成できません (%s): %w", a.profile.Dir, err)
	}
	// backref .root を <roothash> 階層に置く（孤児 profile の逆引き seam・→ ADR-0013）。
	if a.profile.Backref != "" {
		if err := os.MkdirAll(a.profile.BackrefDir, 0o755); err != nil {
			return fmt.Errorf("nput: backref ディレクトリを作成できません (%s): %w", a.profile.BackrefDir, err)
		}
		if err := os.WriteFile(a.profile.Backref, []byte(a.root+"\n"), 0o644); err != nil {
			return fmt.Errorf("nput: backref を書けません (%s): %w", a.profile.Backref, err)
		}
	}
	return nil
}

// loadPrevManifest は profileDir/profile（前世代 link-farm への symlink）が指す
// manifest.json を読む。初回（profile 不在）は nil を返す（削除対象ゼロ・→ ADR-0006）。
func (a *applier) loadPrevManifest() *manifest.Manifest {
	if _, err := os.Stat(a.profile.Profile); err != nil {
		return nil
	}
	m, err := manifest.Load(a.profile.Profile)
	if err != nil {
		// 前世代が読めなくても新規配置は妨げない（stale 除去だけ諦める）。
		a.opts.Warnf("nput: 前世代 manifest を読めませんでした。stale 除去をスキップします: %v", err)
		return nil
	}
	return m
}

// emitWarnings は planner が算出した非致命 warning を stderr（opts.Warnf）へ出す。
// warning は --quiet でも消えない（→ docs/spec.md ストリーム規律・ADR-0015, ADR-0024）。
func (a *applier) emitWarnings(ws []planner.Warning) {
	for _, w := range ws {
		switch w.Kind {
		case planner.WarnForeignReplace:
			a.opts.Warnf("nput: 記録の無い symlink を上書きします (foreign・後勝ち): %s", w.Target)
		case planner.WarnStaleMismatch:
			a.opts.Warnf("nput: stale symlink が記録と不一致のため残します: %s", w.Target)
		case planner.WarnStaleNonSymlink:
			a.opts.Warnf("nput: stale target が symlink ではないため残します: %s", w.Target)
		case planner.WarnCopyOrphan:
			a.opts.Warnf("nput: copy entry が消えましたが target は削除しません（orphan・reset で撤去）: %s", w.Target)
		}
	}
}
