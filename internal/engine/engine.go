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
)

// CommitFunc は配置成功後に世代を積むコミット点（→ ADR-0006, docs/spec.md 実行フロー f）。
// 既定は nix-env --profile <profileLink> --set <linkFarm>。tmpdir テストはこれを
// 差し替えて nix を呼ばずに検証する（commit の e2e は 1c・→ Issue #6）。
type CommitFunc func(profileLink, linkFarm string) error

// GitFunc は project mode の git toplevel 解決（→ ADR-0005）。既定は gitutil.Toplevel。
type GitFunc func(dir string) (string, error)

// Options は Apply の入力。
type Options struct {
	// LinkFarm は manifest.json と GC アンカー symlink farm を含む link-farm ディレクトリ。
	// 実運用では CLI が `nix build --out-link <profileDir>/.pending` で得る（→ ADR-0011）。
	LinkFarm string
	// Name は config 名（profile を一意特定する。entrypoint の nput.<name> 由来）。
	Name string
	// RootOverride は --root 上書き（空 = なし）。明示時は全モードで roothash キー（→ ADR-0023）。
	RootOverride string
	// WorkDir は project mode の git toplevel 解決の起点（空 = os.Getwd）。
	WorkDir string
	// StateDir は profile 基底 <state> の上書き（空 = paths.StateDir で解決・主にテスト用）。
	StateDir string
	// NoWait は flock を try-lock にする（shellHook 経路・保持中なら ErrSkipped・→ ADR-0013）。
	NoWait bool

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

// Apply は manifest.json を入力に project mode の store-symlink 配置を行い、
// 成功後に世代をコミットする。docs/spec.md「実行フロー」の engine 駆動部に対応する
// （eval 先行・build は CLI 責務でここでは LinkFarm を受け取る）。
func Apply(opts Options) (*Result, error) {
	m, err := manifest.Load(opts.LinkFarm)
	if err != nil {
		return nil, err
	}

	a := &applier{opts: opts, manifest: m, result: &Result{}}
	if a.opts.Warnf == nil {
		a.opts.Warnf = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}

	// 1. root 解決 → profileDir 確定（→ docs/spec.md「root の解決」）。
	root, err := a.resolveRoot()
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
	a.profile = paths.Resolve(stateDir, opts.Name, m.Root.RootKind, root, opts.RootOverride != "")
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

	// 4. 前世代 manifest を読む（無ければ初回 = stale 除去ゼロ）。
	prev := a.loadPrevManifest()

	// 5. 配置 → stale 除去（新規 / 張替を先に、stale 除去を最後に・→ ADR-0006）。
	if err := a.place(prev); err != nil {
		return nil, err
	}
	if err := a.removeStale(prev); err != nil {
		return nil, err
	}

	// 6. 世代コミット（commit の e2e は 1c・→ Issue #6）。
	commit := opts.Commit
	if commit == nil {
		commit = nixEnvCommit
	}
	if err := commit(a.profile.Profile, opts.LinkFarm); err != nil {
		return nil, fmt.Errorf("nput: 世代コミット（nix-env --set）に失敗しました: %w", err)
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

func (a *applier) resolveRoot() (string, error) {
	if a.opts.RootOverride != "" {
		return filepath.Abs(a.opts.RootOverride)
	}
	switch a.manifest.Root.RootKind {
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
		if a.manifest.Root.Root == "" {
			return "", fmt.Errorf("nput: rootKind=fixed なのに root パスがありません")
		}
		return filepath.Abs(a.manifest.Root.Root)
	case manifest.RootKindSystem:
		return "", fmt.Errorf("nput: root = systemRoot (system mode) は未実装です（→ ADR-0013）")
	default:
		return "", fmt.Errorf("nput: 未知の rootKind: %q", a.manifest.Root.RootKind)
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

func byTarget(m *manifest.Manifest) map[string]manifest.Entry {
	if m == nil {
		return nil
	}
	out := make(map[string]manifest.Entry, len(m.Entries))
	for _, e := range m.Entries {
		out[e.Target] = e
	}
	return out
}

// linkDest は entry の symlink が指すべき先（<src>/<subpath>）を返す。
func linkDest(e manifest.Entry) string {
	if e.Subpath == "" || e.Subpath == "." {
		return e.Src
	}
	return filepath.Join(e.Src, e.Subpath)
}
