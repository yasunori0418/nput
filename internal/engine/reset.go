package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yasunori0418/nput/internal/lock"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/planner"
)

// reset は配置物を無い状態へ戻す FS-only teardown（→ ADR-0020, ADR-0021, ADR-0025,
// docs/spec.md「recopy・reset」）。
//
//   - symlink: stale 除去と同じ保守的不変条件（前世代 manifest が記録し、かつ現状もその記録通りの
//     先を指す symlink のみ削除。foreign / 記録不一致は warning で残す）。planner.Compute（next=nil）+
//     staleremove を再利用する（完成済みモジュール・→ planner, staleremove.go）。
//   - copy target: 削除する（copy を消す唯一の明示手段・事前存在ファイルを消すリスクは CLI の確認で守る）。
//   - profile / 世代は触らない。config が entry を残す限り次の apply で再配置される（transient）。
//
// 撤去対象 entry の源は **前世代 manifest（profileDir/profile が指す link-farm の manifest.json）**。
// これが「nput が実際に配置した（記録した）真実」で、保守的不変条件の "recorded dest" と一致する
// （config を build し直すと src ドリフト時に記録 dest とズレ誤判定するため、記録済み前世代を使う）。
// rootKind 先取り eval（profileDir 確定）は CLI が担い、entries はこの前世代 manifest から読む。

// ResetOptions は Reset の入力。Reset は build しない（前世代 manifest を読む）。
type ResetOptions struct {
	Name         string
	RootKind     string
	FixedRoot    string
	RootOverride string
	WorkDir      string
	StateDir     string
	Git          GitFunc

	// Targets は撤去対象 target の絞り込み（root 相対・空 = 全 entry）。
	// 前世代 manifest に存在しない target を指定するとエラーになる。
	Targets []string
	// DryRun は副作用ゼロのプレビュー（削除対象を算出して返すだけ・flock / confirm / FS 削除なし・→ ADR-0021）。
	DryRun bool
	// Confirm は削除実行前の確認コールバック（nil = 確認なしで実行・--yes 経路 / dryrun）。
	// 算出済みプランを渡し、false を返すと中断する（Result.Aborted = true）。CLI が TTY プロンプトを担う。
	Confirm func(*ResetResult) (bool, error)
	// Warnf は warning の出力先（nil = stderr）。foreign symlink を残す等の可視化に使う。
	Warnf func(format string, args ...any)
}

// ResetResult は Reset の結果（dryrun はプレビュー・実行時は実削除結果）。
type ResetResult struct {
	Root            string   // 解決後の絶対 root
	ProfileDir      string   // 確定した profileDir
	RemovedSymlinks []string // 削除した（dryrun は削除予定の）symlink target
	RemovedCopies   []string // 削除した（dryrun は削除予定の）copy target
	KeptForeign     []string // 保守的不変条件を満たさず残した symlink target（foreign / 記録不一致）
	DryRun          bool     // 読み取り専用プレビューだった
	Aborted         bool     // 確認プロンプトで中断した
}

// Reset は対象 entry の配置物を無い状態へ戻す。docs/spec.md「実行フロー」の非 build コマンド前段
// （rootKind 先取り eval → root 解決 → profileDir 確定）を CLI と分担し、engine 側は profileDir 解決・
// blocking flock・前世代 manifest 読み・保守的 symlink 除去 + copy 削除を所有する（→ ADR-0021, ADR-0024）。
func Reset(opts ResetOptions) (*ResetResult, error) {
	warnf := opts.Warnf
	if warnf == nil {
		warnf = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}

	// 1. profileDir 確定（root 解決 → レイアウト・apply / rollback と共通の前段・→ ADR-0024）。
	prof, root, err := ProfileFor(ProfileOptions{
		Name: opts.Name, RootKind: opts.RootKind, FixedRoot: opts.FixedRoot,
		RootOverride: opts.RootOverride, WorkDir: opts.WorkDir, StateDir: opts.StateDir, Git: opts.Git,
	})
	if err != nil {
		return nil, err
	}
	res := &ResetResult{Root: root, ProfileDir: prof.Dir, DryRun: opts.DryRun}

	// profile（前世代リンク）が無ければ一度も apply していない = 撤去対象ゼロの no-op。
	if _, err := os.Stat(prof.Profile); err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return nil, fmt.Errorf("nput: profile を確認できません (%s): %w", prof.Profile, err)
	}

	// 2. 実行時は blocking flock で並行 apply / reset と直列化する（→ ADR-0013, ADR-0021）。
	//    dryrun は読み取り専用なので flock を取らない。
	if !opts.DryRun {
		l, err := lock.Acquire(prof.Dir, true)
		if err != nil {
			return nil, fmt.Errorf("nput: flock の取得に失敗しました (%s): %w", prof.Dir, err)
		}
		defer func() { _ = l.Release() }()
	}

	// 3. 前世代 manifest（記録済みの真実）を読み、対象 entry を絞り込む。
	prev, err := manifest.Load(prof.Profile)
	if err != nil {
		return nil, err
	}
	entries, err := selectResetEntries(prev.Entries, opts.Targets)
	if err != nil {
		return nil, err
	}

	// 4. symlink は保守的不変条件で除去するプランを planner で算出（next=nil で全対象が remove 候補）。
	//    copy は planner が決して消さない領域なので分離して扱う。
	var symEntries, copyEntries []manifest.Entry
	for _, e := range entries {
		if e.Method == manifest.MethodCopy {
			copyEntries = append(copyEntries, e)
		} else {
			symEntries = append(symEntries, e)
		}
	}
	symManifest := &manifest.Manifest{SchemaVersion: prev.SchemaVersion, Root: prev.Root, Entries: symEntries}
	plan, err := planner.Compute(symManifest, nil, root, planner.OSFS)
	if err != nil {
		return nil, err
	}

	// 5. copy target のうち実在するものを削除候補にする（不在は no-op・→ docs/spec.md エラー仕様）。
	copyTargets := make([]string, 0, len(copyEntries))
	for _, e := range copyEntries {
		targetAbs := filepath.Join(root, filepath.Clean(e.Target))
		if _, err := os.Lstat(targetAbs); err == nil {
			copyTargets = append(copyTargets, targetAbs)
			res.RemovedCopies = append(res.RemovedCopies, e.Target)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("nput: copy target を lstat できません (%s): %w", targetAbs, err)
		}
	}

	// プレビュー（dryrun / confirm 表示）用に symlink 削除予定・残す foreign を詰める。
	for _, a := range plan.Remove {
		res.RemovedSymlinks = append(res.RemovedSymlinks, a.Entry.Target)
	}
	for _, w := range plan.Warnings {
		if w.Kind == planner.WarnStaleMismatch || w.Kind == planner.WarnStaleNonSymlink {
			res.KeptForeign = append(res.KeptForeign, w.Target)
		}
	}

	// 6. dryrun は算出済みプランを返して終了（FS 削除・confirm なし・→ ADR-0021）。
	if opts.DryRun {
		return res, nil
	}

	// 7. 確認（データ損失リスク・→ ADR-0020, ADR-0025）。CLI が TTY プロンプト / --yes を担う。
	if opts.Confirm != nil {
		proceed, err := opts.Confirm(res)
		if err != nil {
			return nil, err
		}
		if !proceed {
			res.Aborted = true
			return res, nil
		}
	}

	// 8. 実 FS に反映する。symlink は staleremove（plan 後ドリフト再検証つき）を再利用し、
	//    copy target は削除する。残す foreign の warning を出す。
	a := &applier{opts: Options{Warnf: warnf}, result: &Result{Root: root, ProfileDir: prof.Dir}}
	a.profile = prof
	a.root = root
	a.emitWarnings(plan.Warnings, false)
	if err := a.removeStale(plan.Remove); err != nil {
		return nil, err
	}
	res.RemovedSymlinks = a.result.Removed // 実削除分（ドリフトで残ったものは除外）

	removedCopies := make([]string, 0, len(copyTargets))
	for i, targetAbs := range copyTargets {
		if err := os.RemoveAll(targetAbs); err != nil {
			return nil, fmt.Errorf("nput: copy target を削除できません (%s): %w", targetAbs, err)
		}
		removedCopies = append(removedCopies, res.RemovedCopies[i])
	}
	res.RemovedCopies = removedCopies

	return res, nil
}

// selectResetEntries は前世代 manifest の entry を Targets で絞り込む（空 = 全 entry）。
// 指定 target が前世代に存在しなければエラー（nput が配置していない target は撤去対象にならない）。
func selectResetEntries(entries []manifest.Entry, targets []string) ([]manifest.Entry, error) {
	if len(targets) == 0 {
		return entries, nil
	}
	byTarget := make(map[string]manifest.Entry, len(entries))
	for _, e := range entries {
		byTarget[e.Target] = e
	}
	out := make([]manifest.Entry, 0, len(targets))
	var unknown []string
	for _, t := range targets {
		key := filepath.Clean(t)
		e, ok := byTarget[key]
		if !ok {
			unknown = append(unknown, t)
			continue
		}
		out = append(out, e)
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("nput: reset 対象が前世代 manifest に見つかりません（nput が配置した target ではありません）: %v", unknown)
	}
	return out, nil
}
