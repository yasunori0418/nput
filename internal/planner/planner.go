// Package planner is the diff/plan deep module of the placement engine: given
// the previous-generation manifest, the new manifest, the resolved root, and an
// FS prober, it computes a place/replace/remove plan as pure logic. The
// conservative stale-removal invariant lives here — a stale symlink is only
// scheduled for removal when the previous generation recorded it AND the on-disk
// link still points to the recorded destination. Regular files, foreign links,
// and record/reality mismatches are never removed; copy entries are never
// removed (orphan warning only); the first apply (no previous manifest) removes
// nothing (→ ADR-0002, ADR-0006, ADR-0015, docs/spec.md「stale 除去の対象と安全不変条件」).
//
// The plan is computed without mutating the filesystem. The engine consumes the
// plan: it materializes Place actions and hands Remove actions to the
// conservative stale-remover, which re-verifies the invariant against the real
// FS immediately before unlinking.
package planner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yasunori0418/nput/internal/manifest"
)

// FS abstracts the lstat/readlink probes the planner needs, so diff
// classification is a pure function over (manifests, FS state) and can be
// table-tested with a fake FS without touching the real filesystem
// (→ ADR-0006, docs/spec.md「stale 除去の対象と安全不変条件」).
type FS interface {
	Lstat(path string) (os.FileInfo, error)
	Readlink(path string) (string, error)
}

// osFS is the real-filesystem FS used at engine runtime.
type osFS struct{}

func (osFS) Lstat(path string) (os.FileInfo, error) { return os.Lstat(path) }
func (osFS) Readlink(path string) (string, error)   { return os.Readlink(path) }

// OSFS probes the real filesystem (engine runtime).
var OSFS FS = osFS{}

// PlaceKind classifies how a new-generation entry maps onto the current FS.
type PlaceKind int

const (
	// PlaceNew は target が不在で新規 symlink を作る。
	PlaceNew PlaceKind = iota
	// PlaceReplace は自身の前世代 manifest が記録した symlink を silent に張り替える。
	PlaceReplace
	// PlaceForeign は記録の無い symlink（foreign）を warning 付きで後勝ち置換する（→ ADR-0015）。
	PlaceForeign
)

// PlaceAction is a symlink to materialize at TargetAbs pointing to Dest.
type PlaceAction struct {
	Entry     manifest.Entry
	TargetAbs string
	Dest      string // LinkDest(Entry): <src>/<subpath>
	Kind      PlaceKind
}

// RemoveAction is a stale symlink that satisfies the conservative invariant at
// plan time (recorded by prev AND on-disk points to the recorded dest). The
// stale-remover re-verifies this against the real FS before unlinking.
type RemoveAction struct {
	Entry     manifest.Entry
	TargetAbs string
}

// Conflict is a placement target the engine must stop on: occupied by a non-symlink
// (regular file / directory) or nested under a symlinked ancestor (→ ADR-0015).
type Conflict struct {
	Entry     manifest.Entry
	TargetAbs string
	Reason    string
}

// WarnKind enumerates non-fatal conditions the planner surfaces to the user.
type WarnKind int

const (
	// WarnForeignReplace は記録の無い symlink を上書きする（place・後勝ち・→ ADR-0015）。
	WarnForeignReplace WarnKind = iota
	// WarnStaleMismatch は stale target が記録と不一致な symlink のため残す（→ ADR-0002）。
	WarnStaleMismatch
	// WarnStaleNonSymlink は stale target が symlink でない（通常ファイル等）ため残す。
	WarnStaleNonSymlink
	// WarnCopyOrphan は消えた copy entry の orphan（削除はしない・reset で撤去・→ ADR-0020）。
	WarnCopyOrphan
)

// Warning is a non-fatal condition surfaced to the user for a given target.
type Warning struct {
	Kind   WarnKind
	Target string
}

// Plan is the computed place/replace/remove plan plus non-fatal warnings and
// fatal conflicts. The engine executes Place then Remove (「新規/張替を先に、stale
// 除去を最後に」・→ ADR-0006); a non-empty Conflicts means apply must stop.
type Plan struct {
	Place     []PlaceAction
	Remove    []RemoveAction
	Conflicts []Conflict
	Warnings  []Warning
}

// LinkDest は entry の symlink が指すべき先（<src>/<subpath>）を返す。
func LinkDest(e manifest.Entry) string {
	if e.Subpath == "" || e.Subpath == "." {
		return e.Src
	}
	return filepath.Join(e.Src, e.Subpath)
}

// Compute は前世代 manifest（prev・nil なら初回）と新 manifest（next）を root と FS
// 状態に照らして diff し、place/replace/remove プランを算出する純ロジック。
// 副作用は持たず、FS への反映は engine（place + stale-remover）が行う。
func Compute(prev, next *manifest.Manifest, root string, fs FS) (Plan, error) {
	var plan Plan

	// --- place / replace 側: 新世代の各 entry を現 FS に照らして分類する ---
	prevByTarget := byTarget(prev)
	for _, e := range entriesOf(next) {
		switch e.Method {
		case manifest.MethodSymlink:
			// 本スライスは symlink のみ対応。
		case manifest.MethodCopy:
			return Plan{}, fmt.Errorf("nput: method=copy は本スライスでは未実装です (target: %s)", e.Target)
		default:
			return Plan{}, fmt.Errorf("nput: 未知の method: %q (target: %s)", e.Method, e.Target)
		}

		targetAbs := filepath.Join(root, filepath.Clean(e.Target))

		// 祖先 component が symlink ならネスト不可で conflict（→ ADR-0015）。
		offender, err := ancestorSymlink(root, e.Target, fs)
		if err != nil {
			return Plan{}, err
		}
		if offender != "" {
			plan.Conflicts = append(plan.Conflicts, Conflict{
				Entry:     e,
				TargetAbs: targetAbs,
				Reason:    fmt.Sprintf("祖先 %q が symlink です。配下にネストできません (→ ADR-0015)", offender),
			})
			continue
		}

		info, err := fs.Lstat(targetAbs)
		switch {
		case err == nil && info.Mode()&os.ModeSymlink != 0:
			kind := PlaceForeign
			if recordedLink(e.Target, targetAbs, prevByTarget, fs) {
				kind = PlaceReplace
			} else {
				plan.Warnings = append(plan.Warnings, Warning{Kind: WarnForeignReplace, Target: e.Target})
			}
			plan.Place = append(plan.Place, PlaceAction{Entry: e, TargetAbs: targetAbs, Dest: LinkDest(e), Kind: kind})
		case err == nil:
			// 通常ファイル / ディレクトリは上書きしない（→ docs/spec.md エラー仕様）。
			plan.Conflicts = append(plan.Conflicts, Conflict{
				Entry:     e,
				TargetAbs: targetAbs,
				Reason:    "target に既存のファイル/ディレクトリがあります（上書きしません）",
			})
		case os.IsNotExist(err):
			plan.Place = append(plan.Place, PlaceAction{Entry: e, TargetAbs: targetAbs, Dest: LinkDest(e), Kind: PlaceNew})
		default:
			return Plan{}, fmt.Errorf("nput: target を lstat できません (%s): %w", targetAbs, err)
		}
	}

	// --- remove 側: stale entry（prev ∖ next）を保守的不変条件下で算出する ---
	// 初回（prev == nil）は何も消さない（→ ADR-0006）。
	if prev != nil {
		nextByTarget := byTarget(next)
		for _, pe := range prev.Entries {
			if _, kept := nextByTarget[pe.Target]; kept {
				continue
			}
			if pe.Method == manifest.MethodCopy {
				// copy はユーザー所有データ。削除せず orphan を警告（→ ADR-0002, ADR-0020）。
				plan.Warnings = append(plan.Warnings, Warning{Kind: WarnCopyOrphan, Target: pe.Target})
				continue
			}

			targetAbs := filepath.Join(root, filepath.Clean(pe.Target))
			info, err := fs.Lstat(targetAbs)
			switch {
			case err != nil && os.IsNotExist(err):
				continue // 既に無い = no-op（警告なし）。
			case err != nil:
				return Plan{}, fmt.Errorf("nput: stale target を lstat できません (%s): %w", targetAbs, err)
			case info.Mode()&os.ModeSymlink == 0:
				// 通常ファイル / ディレクトリには触れない（→ docs/spec.md 安全不変条件）。
				plan.Warnings = append(plan.Warnings, Warning{Kind: WarnStaleNonSymlink, Target: pe.Target})
				continue
			}

			onDisk, err := fs.Readlink(targetAbs)
			if err != nil || onDisk != LinkDest(pe) {
				// 記録と実体が不一致（foreign / ユーザー差し替え）→ 削除せず警告（→ ADR-0002）。
				plan.Warnings = append(plan.Warnings, Warning{Kind: WarnStaleMismatch, Target: pe.Target})
				continue
			}
			plan.Remove = append(plan.Remove, RemoveAction{Entry: pe, TargetAbs: targetAbs})
		}
	}

	return plan, nil
}

func entriesOf(m *manifest.Manifest) []manifest.Entry {
	if m == nil {
		return nil
	}
	return m.Entries
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

// recordedLink は target が「自身の前世代 manifest が記録した symlink」かを判定する。
// 前世代に同 target の entry があり、かつ実体の symlink が記録通りの先を指すときのみ true
// （保守的不変条件・→ ADR-0002, ADR-0015）。
func recordedLink(target, targetAbs string, prevByTarget map[string]manifest.Entry, fs FS) bool {
	pe, ok := prevByTarget[target]
	if !ok {
		return false
	}
	onDisk, err := fs.Readlink(targetAbs)
	if err != nil {
		return false
	}
	return onDisk == LinkDest(pe)
}

// ancestorSymlink walks the target's ancestor components under root and returns
// the first existing ancestor that is a symlink (全体 symlink 配置の配下にネスト不可・
// → ADR-0015). A non-existent ancestor stops the walk (its descendants don't exist
// either), returning "" with no error.
func ancestorSymlink(root, target string, fs FS) (string, error) {
	clean := filepath.Clean(target)
	comps := strings.Split(clean, string(os.PathSeparator))
	cur := root
	for i := 0; i < len(comps)-1; i++ {
		if comps[i] == "" {
			continue
		}
		cur = filepath.Join(cur, comps[i])
		info, err := fs.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", fmt.Errorf("nput: 祖先を lstat できません (%s): %w", cur, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return cur, nil
		}
	}
	return "", nil
}
