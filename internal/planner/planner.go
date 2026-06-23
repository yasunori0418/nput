// Package planner is the diff/plan deep module of the placement engine: given
// the previous-generation manifest, the new manifest, the resolved root, and an
// FS prober, it computes a place/replace/remove plan as pure logic. The
// conservative stale-removal invariant lives here — a stale symlink is only
// scheduled for removal when the previous generation recorded it AND the on-disk
// link still points to the recorded destination. Regular files, foreign links,
// and record/reality mismatches are never removed; copy entries are never
// removed (orphan warning only); the first apply (no previous manifest) removes
// nothing (→ ADR-0002, ADR-0006, ADR-0015, docs/spec.md "targets and safety invariant of stale removal").
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
// (→ ADR-0006, docs/spec.md "targets and safety invariant of stale removal").
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
	// PlaceNew creates a new symlink when target is absent.
	PlaceNew PlaceKind = iota
	// PlaceReplace silently re-links a symlink recorded by this profile's own previous-generation manifest.
	PlaceReplace
	// PlaceForeign last-wins replaces an unrecorded symlink (foreign) with a warning (→ ADR-0015).
	PlaceForeign
)

// PlaceAction is a symlink to materialize at TargetAbs pointing to Dest.
type PlaceAction struct {
	Entry     manifest.Entry
	TargetAbs string
	Dest      string // LinkDest(Entry): <src>/<subpath>
	Kind      PlaceKind
}

// CopyAction is a place-once copy to materialize: copy Src (= <src>/<subpath>)
// into TargetAbs (only when target is absent; place-once; → ADR-0002, ADR-0016).
// An existing target (recorded / foreign) is left untouched, so no CopyAction is emitted for it.
type CopyAction struct {
	Entry     manifest.Entry
	TargetAbs string
	Src       string // LinkDest(Entry): <src>/<subpath> (copy source)
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
	// WarnForeignReplace overwrites an unrecorded symlink (place; last-wins; → ADR-0015).
	WarnForeignReplace WarnKind = iota
	// WarnStaleMismatch keeps a stale target because its symlink mismatches the record (→ ADR-0002).
	WarnStaleMismatch
	// WarnStaleNonSymlink keeps a stale target because it is not a symlink (regular file, etc.).
	WarnStaleNonSymlink
	// WarnCopyOrphan is the orphan of a vanished copy entry (not removed; cleared by reset; → ADR-0020).
	WarnCopyOrphan
	// WarnCopyForeign skips a copy target under place-once because an unrecorded real file exists there
	// (no overwrite; surfaced to prevent masking; symmetric with the symlink foreign warning; → ADR-0022).
	WarnCopyForeign
)

// Warning is a non-fatal condition surfaced to the user for a given target.
type Warning struct {
	Kind   WarnKind
	Target string
}

// Plan is the computed place/replace/remove plan plus non-fatal warnings and
// fatal conflicts. The engine executes Place / Copies then Remove ("new/re-link
// first, stale removal last"; → ADR-0006); a non-empty Conflicts means apply must stop.
type Plan struct {
	Place     []PlaceAction
	Copies    []CopyAction
	Remove    []RemoveAction
	Conflicts []Conflict
	Warnings  []Warning
}

// LinkDest returns the destination the entry's symlink should point to (<src>/<subpath>).
func LinkDest(e manifest.Entry) string {
	if e.Subpath == "" || e.Subpath == "." {
		return e.Src
	}
	return filepath.Join(e.Src, e.Subpath)
}

// Compute diffs the previous-generation manifest (prev; nil means first apply)
// against the new manifest (next), relative to root and FS state, and computes
// the place/replace/remove plan as pure logic. It has no side effects; the FS
// changes are applied by the engine (place + stale-remover).
func Compute(prev, next *manifest.Manifest, root string, fs FS) (Plan, error) {
	var plan Plan

	// --- place / replace side: classify each new-generation entry against the current FS ---
	prevByTarget := byTarget(prev)
	for _, e := range entriesOf(next) {
		targetAbs := filepath.Join(root, filepath.Clean(e.Target))

		// If an ancestor component is a symlink, nesting is forbidden: conflict (common to all methods; → ADR-0015).
		offender, err := ancestorSymlink(root, e.Target, fs)
		if err != nil {
			return Plan{}, err
		}
		if offender != "" {
			plan.Conflicts = append(plan.Conflicts, Conflict{
				Entry:     e,
				TargetAbs: targetAbs,
				Reason:    fmt.Sprintf("ancestor %q is a symlink; cannot nest beneath it (→ ADR-0015)", offender),
			})
			continue
		}

		// Branch the place-once / re-link classification per method.
		if e.Method == manifest.MethodCopy {
			if err := classifyCopy(&plan, e, targetAbs, prevByTarget, fs); err != nil {
				return Plan{}, err
			}
			continue
		}
		if e.Method != manifest.MethodSymlink {
			return Plan{}, fmt.Errorf("nput: unknown method: %q (target: %s)", e.Method, e.Target)
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
			// A regular file / directory is not overwritten (→ docs/spec.md error spec).
			plan.Conflicts = append(plan.Conflicts, Conflict{
				Entry:     e,
				TargetAbs: targetAbs,
				Reason:    "target already has an existing file/directory (will not overwrite)",
			})
		case os.IsNotExist(err):
			plan.Place = append(plan.Place, PlaceAction{Entry: e, TargetAbs: targetAbs, Dest: LinkDest(e), Kind: PlaceNew})
		default:
			return Plan{}, fmt.Errorf("nput: cannot lstat target (%s): %w", targetAbs, err)
		}
	}

	// --- remove side: compute stale entries (prev ∖ next) under the conservative invariant ---
	// On first apply (prev == nil) nothing is removed (→ ADR-0006).
	if prev != nil {
		nextByTarget := byTarget(next)
		for _, pe := range prev.Entries {
			if _, kept := nextByTarget[pe.Target]; kept {
				continue
			}
			if pe.Method == manifest.MethodCopy {
				// copy is user-owned data: not removed, warn as orphan (→ ADR-0002, ADR-0020).
				plan.Warnings = append(plan.Warnings, Warning{Kind: WarnCopyOrphan, Target: pe.Target})
				continue
			}

			targetAbs := filepath.Join(root, filepath.Clean(pe.Target))
			info, err := fs.Lstat(targetAbs)
			switch {
			case err != nil && os.IsNotExist(err):
				continue // already gone = no-op (no warning).
			case err != nil:
				return Plan{}, fmt.Errorf("nput: cannot lstat stale target (%s): %w", targetAbs, err)
			case info.Mode()&os.ModeSymlink == 0:
				// A regular file / directory is left untouched (→ docs/spec.md safety invariant).
				plan.Warnings = append(plan.Warnings, Warning{Kind: WarnStaleNonSymlink, Target: pe.Target})
				continue
			}

			onDisk, err := fs.Readlink(targetAbs)
			if err != nil || onDisk != LinkDest(pe) {
				// Record and reality mismatch (foreign / user-replaced) → not removed, warn (→ ADR-0002).
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

// recordedLink reports whether target is "a symlink recorded by this profile's
// own previous-generation manifest". True only when the previous generation has
// an entry for the same target AND the on-disk symlink points to the recorded
// destination (conservative invariant; → ADR-0002, ADR-0015).
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

// classifyCopy classifies a copy entry under place-once semantics (→ ADR-0002,
// ADR-0016, ADR-0022, docs/spec.md "copy mode").
//
//	target absent              → CopyAction (new place-once copy)
//	target exists, structure mismatch → conflict (subpath dir × target file / subpath file × target dir)
//	target exists, recorded    → no-op (placed by nput in a previous generation; place-once leaves it untouched)
//	target exists, foreign     → skip + WarnCopyForeign (unrecorded real file; masking prevention)
//
// recopy (apply --recopy) is a separate path that breaks place-once: the engine
// overwrites the manifest's copy entry directly. The planner only does the
// normal place-once classification (→ ADR-0020).
func classifyCopy(plan *Plan, e manifest.Entry, targetAbs string, prevByTarget map[string]manifest.Entry, fs FS) error {
	info, err := fs.Lstat(targetAbs)
	switch {
	case err == nil:
		// target exists: first check whether the src structure and kind match.
		mismatch, err := copyStructureMismatch(e, info, fs)
		if err != nil {
			return err
		}
		if mismatch {
			plan.Conflicts = append(plan.Conflicts, Conflict{
				Entry:     e,
				TargetAbs: targetAbs,
				Reason:    "copy src structure and target kind mismatch (dir↔file; will not overwrite)",
			})
			return nil
		}
		// place-once: leave a copy recorded by the previous generation untouched. An unrecorded real file gets a foreign warning.
		if pe, ok := prevByTarget[e.Target]; ok && pe.Method == manifest.MethodCopy {
			return nil
		}
		plan.Warnings = append(plan.Warnings, Warning{Kind: WarnCopyForeign, Target: e.Target})
		return nil
	case os.IsNotExist(err):
		plan.Copies = append(plan.Copies, CopyAction{Entry: e, TargetAbs: targetAbs, Src: LinkDest(e)})
		return nil
	default:
		return fmt.Errorf("nput: cannot lstat copy target (%s): %w", targetAbs, err)
	}
}

// copyStructureMismatch reports whether the dir/file kind of src (<src>/<subpath>)
// disagrees with the kind of the existing target (subpath dir × target file /
// subpath file × target dir; → docs/spec.md). A symlink target has IsDir()=false
// and is treated as the "file side".
func copyStructureMismatch(e manifest.Entry, targetInfo os.FileInfo, fs FS) (bool, error) {
	srcInfo, err := fs.Lstat(LinkDest(e))
	if err != nil {
		return false, fmt.Errorf("nput: cannot lstat copy src (%s): %w", LinkDest(e), err)
	}
	return srcInfo.IsDir() != targetInfo.IsDir(), nil
}

// ancestorSymlink walks the target's ancestor components under root and returns
// the first existing ancestor that is a symlink (cannot nest under a whole-tree
// symlink placement; → ADR-0015). A non-existent ancestor stops the walk (its descendants don't exist
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
			return "", fmt.Errorf("nput: cannot lstat ancestor (%s): %w", cur, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return cur, nil
		}
	}
	return "", nil
}
