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

// FS abstracts the lstat/readlink/readdir probes the planner needs, so diff
// classification is a pure function over (manifests, FS state) and can be
// table-tested with a fake FS without touching the real filesystem
// (→ ADR-0006, ADR-0047, docs/spec.md "targets and safety invariant of stale removal").
type FS interface {
	Lstat(path string) (os.FileInfo, error)
	Readlink(path string) (string, error)
	// ReadDir lists path's immediate children (a real directory target). Used only
	// to classify an occupying real directory for PreRemove migration (→ ADR-0047).
	ReadDir(path string) ([]os.DirEntry, error)
}

// osFS is the real-filesystem FS used at engine runtime.
type osFS struct{}

func (osFS) Lstat(path string) (os.FileInfo, error)     { return os.Lstat(path) }
func (osFS) Readlink(path string) (string, error)       { return os.Readlink(path) }
func (osFS) ReadDir(path string) ([]os.DirEntry, error) { return os.ReadDir(path) }

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

// RemoveKind distinguishes what a RemoveAction removes: an entry-recorded symlink
// (Unlink) versus an empty directory left behind by removals (Rmdir; → ADR-0047).
type RemoveKind int

const (
	// RemoveUnlink removes a stale symlink recorded by Entry (the original, entry-driven case).
	RemoveUnlink RemoveKind = iota
	// RemoveRmdir removes an empty directory that occupies a placement target or is left
	// empty by a migration. Entry is the zero value: rmdir has no manifest record to
	// re-verify against, so the engine re-checks emptiness at unlink time instead (→ ADR-0047).
	RemoveRmdir
)

// RemoveAction is a stale filesystem object that satisfies the conservative invariant
// at plan time. For Kind == RemoveUnlink: a symlink recorded by prev AND on-disk points
// to the recorded dest (Entry is populated; the stale-remover re-verifies this against
// the real FS before unlinking). For Kind == RemoveRmdir: an empty directory (Entry is
// the zero value; the engine re-verifies emptiness immediately before rmdir · → ADR-0047).
//
// Execution order is expressed by slice order: the planner appends children before their
// parents, so consumers that walk a RemoveAction slice front-to-back naturally unlink
// leaves first and rmdir from the deepest directory upward (bottom-up).
type RemoveAction struct {
	Kind      RemoveKind
	Entry     manifest.Entry
	TargetAbs string
}

// Conflict is a placement target the engine must stop on: occupied by a non-symlink
// (regular file / directory) or nested under a symlinked ancestor (→ ADR-0015).
type Conflict struct {
	Entry     manifest.Entry
	TargetAbs string
	Reason    string
	Kind      ConflictKind
}

// ConflictKind classifies a Conflict for caller-side guidance (→ docs/spec.md エラー仕様 ·
// grilling 2026-07-12 D6). Kept distinct from the free-form Reason string so guidance
// selection does not depend on message text.
type ConflictKind int

const (
	// ConflictUnspecified is the zero value: a Kind left unset. It must never be produced by
	// Compute (every Conflict{} literal below sets Kind explicitly); guarded against by falling
	// through to a generic guidance rather than silently reading as ConflictForeignEntity.
	ConflictUnspecified ConflictKind = iota
	// ConflictForeignEntity is a regular file/directory occupying a symlink target (→ ADR-0006).
	ConflictForeignEntity
	// ConflictForeignAncestor is a symlinked ancestor component not recorded by this profile's
	// own previous generation (unrecorded / mismatched dest / no previous generation · → ADR-0015 §4).
	ConflictForeignAncestor
	// ConflictSelfContradictoryAncestor is a symlinked ancestor still kept by the new generation
	// while a descendant entry also targets beneath it (→ ADR-0015 §4, ADR-0046).
	ConflictSelfContradictoryAncestor
	// ConflictCopyStructureMismatch is a copy entry whose src structure (dir/file) mismatches the
	// existing target kind (→ ADR-0020).
	ConflictCopyStructureMismatch
)

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
// fatal conflicts. The engine executes PreRemove → Place / Copies → Remove
// ("new/re-link first, stale removal last"; PreRemove is a local ordering exception
// that unlinks self-recorded stale ancestor symlinks *before* placement so children
// nest into a real directory; → ADR-0006, ADR-0046); a non-empty Conflicts means apply must stop.
type Plan struct {
	Place  []PlaceAction
	Copies []CopyAction
	Remove []RemoveAction
	// PreRemove unlinks self-recorded stale ancestor symlinks before placement, so a
	// previous-generation whole-tree symlink can migrate to nested child entries without a
	// manual rm. Populated only for ancestors the previous generation recorded and the new
	// generation drops (recorded ∧ stale); foreign or still-kept ancestors stay Conflicts (→ ADR-0046).
	PreRemove []RemoveAction
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
	nextByTarget := byTarget(next)
	// preRemoved dedups ancestors scheduled for pre-removal migration: several children can
	// detect the same ancestor symlink, but it is unlinked once (→ ADR-0046).
	preRemoved := map[string]bool{}
	for _, e := range entriesOf(next) {
		targetAbs := filepath.Join(root, filepath.Clean(e.Target))

		// If an ancestor component is a symlink, nesting is normally forbidden (→ ADR-0015). The one
		// exception is a symlink this profile's own previous generation recorded and the new generation
		// drops (recorded ∧ stale): migrate it — schedule a pre-removal and place the child as new —
		// instead of stopping (→ ADR-0046).
		offenderAbs, offenderRel, err := ancestorSymlink(root, e.Target, fs)
		if err != nil {
			return Plan{}, err
		}
		if offenderAbs != "" {
			// offenderRel is the cleaned root-relative ancestor path; matching it against
			// nextByTarget / prevByTarget (keyed by the raw manifest target) relies on targets being
			// canonical — the same convention the rest of the planner already assumes (byTarget keys,
			// placement's filepath.Clean). A non-canonical ancestor target simply fails to match here
			// and degrades safely to a conflict; it never mis-migrates.
			_, keptInNext := nextByTarget[offenderRel]
			if !keptInNext && recordedLink(offenderRel, offenderAbs, prevByTarget, fs) {
				if !preRemoved[offenderRel] {
					preRemoved[offenderRel] = true
					plan.PreRemove = append(plan.PreRemove, RemoveAction{Entry: prevByTarget[offenderRel], TargetAbs: offenderAbs})
				}
				// The child currently resolves *through* the ancestor symlink into the previous farm,
				// so an lstat here would misclassify it against store content (the pollution ADR-0015 §4
				// guarded). After PreRemove the ancestor is gone and the child is absent, so place it as
				// new unconditionally without probing the FS (→ ADR-0046).
				if err := appendAbsentPlacement(&plan, e, targetAbs); err != nil {
					return Plan{}, err
				}
				continue
			}
			// foreign ancestor, or the new generation still keeps the ancestor (self-contradictory):
			// the ancestor symlink cannot be removed, so nesting stays a conflict (→ ADR-0015, ADR-0046).
			kind := ConflictForeignAncestor
			if keptInNext {
				kind = ConflictSelfContradictoryAncestor
			}
			plan.Conflicts = append(plan.Conflicts, Conflict{
				Entry:     e,
				TargetAbs: targetAbs,
				Reason:    fmt.Sprintf("ancestor %q is a symlink; cannot nest beneath it (→ ADR-0015)", offenderAbs),
				Kind:      kind,
			})
			continue
		}

		// Branch the place-once / re-link classification per method.
		if e.Method == manifest.MethodCopy {
			if err := classifyCopy(&plan, e, targetAbs, prevByTarget, preRemoved, fs); err != nil {
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
		case err == nil && info.IsDir():
			// A real directory occupies the target: fully migratable only when every leaf beneath
			// it is a self-recorded stale symlink or an empty subdirectory (→ ADR-0047, issue #175).
			// A copy-placed target is out of scope (copy targets never appear here — this arm only
			// runs for the symlink-method branch).
			if err := classifyRealDirTarget(&plan, e, targetAbs, prevByTarget, nextByTarget, preRemoved, fs); err != nil {
				return Plan{}, err
			}
		case err == nil:
			// A regular file is not overwritten (→ docs/spec.md error spec).
			plan.Conflicts = append(plan.Conflicts, Conflict{
				Entry:     e,
				TargetAbs: targetAbs,
				Reason:    "target already has an existing file/directory (will not overwrite)",
				Kind:      ConflictForeignEntity,
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
		for _, pe := range prev.Entries {
			if _, kept := nextByTarget[pe.Target]; kept {
				continue
			}
			if preRemoved[pe.Target] {
				// Already scheduled for pre-removal migration; do not remove it twice (→ ADR-0046).
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
//	target absent                     → CopyAction (new place-once copy)
//	target is a self-recorded stale symlink (method changed symlink→copy) → PreRemove(Unlink) + CopyAction (→ ADR-0047 D5)
//	target exists, structure mismatch → conflict (subpath dir × target file / subpath file × target dir)
//	target exists, recorded           → no-op (placed by nput in a previous generation; place-once leaves it untouched)
//	target exists, foreign            → skip + WarnCopyForeign (unrecorded real file; masking prevention)
//
// recopy (apply --recopy) is a separate path that breaks place-once: the engine
// overwrites the manifest's copy entry directly. The planner only does the
// normal place-once classification (→ ADR-0020).
func classifyCopy(plan *Plan, e manifest.Entry, targetAbs string, prevByTarget map[string]manifest.Entry, preRemoved map[string]bool, fs FS) error {
	info, err := fs.Lstat(targetAbs)
	switch {
	case err == nil && info.Mode()&os.ModeSymlink != 0 && prevByTarget[e.Target].Method == manifest.MethodSymlink && recordedLink(e.Target, targetAbs, prevByTarget, fs):
		// The previous generation placed a symlink here and the new generation wants a copy at the
		// same target (method changed symlink→copy): pre-remove the recorded symlink and place a
		// fresh place-once copy — zero data loss, since the symlink carried no user data (→ ADR-0047
		// D5). A readlink mismatch (on-disk drifted from the record) falls through to the ordinary
		// foreign-symlink handling below instead (copy→symlink direction stays a structure
		// mismatch/conflict; this arm never fires for it since Method would be "copy").
		if !preRemoved[e.Target] {
			preRemoved[e.Target] = true
			plan.PreRemove = append(plan.PreRemove, RemoveAction{Kind: RemoveUnlink, Entry: prevByTarget[e.Target], TargetAbs: targetAbs})
		}
		plan.Copies = append(plan.Copies, CopyAction{Entry: e, TargetAbs: targetAbs, Src: LinkDest(e)})
		return nil
	case err == nil:
		// target exists: check whether the src structure and kind match. A symlink target that
		// fell through the method-change arm above (unrecorded or drifted) is treated as a foreign,
		// non-directory occupant here — consistent with copyStructureMismatch's IsDir()==false
		// handling for symlinks.
		mismatch, err := copyStructureMismatch(e, info, fs)
		if err != nil {
			return err
		}
		if mismatch {
			plan.Conflicts = append(plan.Conflicts, Conflict{
				Entry:     e,
				TargetAbs: targetAbs,
				Reason:    "copy src structure and target kind mismatch (dir↔file; will not overwrite)",
				Kind:      ConflictCopyStructureMismatch,
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

// appendAbsentPlacement records the placement for an entry whose target is known to be absent
// (a child nesting under a to-be-pre-removed ancestor symlink): a new symlink, or a place-once
// copy. It mirrors the "target absent" arms of the normal per-method classification without
// probing the FS, which would misread store content through the still-present ancestor symlink
// (→ ADR-0046).
func appendAbsentPlacement(plan *Plan, e manifest.Entry, targetAbs string) error {
	switch e.Method {
	case manifest.MethodSymlink:
		plan.Place = append(plan.Place, PlaceAction{Entry: e, TargetAbs: targetAbs, Dest: LinkDest(e), Kind: PlaceNew})
	case manifest.MethodCopy:
		plan.Copies = append(plan.Copies, CopyAction{Entry: e, TargetAbs: targetAbs, Src: LinkDest(e)})
	default:
		return fmt.Errorf("nput: unknown method: %q (target: %s)", e.Method, e.Target)
	}
	return nil
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

// classifyRealDirTarget classifies a symlink-method entry whose target is occupied by a real
// directory (→ ADR-0047, issue #175). Mirrors classifyCopy's role for the copy-method branch:
// it owns the full decision — walk the tree via classifyDirMigration, emit a Conflict on
// failure, or on success append the PreRemove actions (children before the target itself),
// dedup the unlinked children against the remove-side loop's preRemoved map (→ ADR-0046 dedup
// convention: each unlinked child is also a stale entry in prev.Entries and must not be
// scheduled there a second time), and append the new-symlink Place action.
func classifyRealDirTarget(plan *Plan, e manifest.Entry, targetAbs string, prevByTarget, nextByTarget map[string]manifest.Entry, preRemoved map[string]bool, fs FS) error {
	dirActions, reason, err := classifyDirMigration(filepath.Clean(e.Target), targetAbs, prevByTarget, nextByTarget, fs)
	if err != nil {
		return err
	}
	if reason != "" {
		plan.Conflicts = append(plan.Conflicts, Conflict{
			Entry:     e,
			TargetAbs: targetAbs,
			Reason:    fmt.Sprintf("target directory cannot be fully migrated: %s (→ ADR-0047)", reason),
		})
		return nil
	}
	plan.PreRemove = append(plan.PreRemove, dirActions...)
	plan.PreRemove = append(plan.PreRemove, RemoveAction{Kind: RemoveRmdir, TargetAbs: targetAbs})
	for _, da := range dirActions {
		if da.Kind == RemoveUnlink {
			preRemoved[da.Entry.Target] = true
		}
	}
	plan.Place = append(plan.Place, PlaceAction{Entry: e, TargetAbs: targetAbs, Dest: LinkDest(e), Kind: PlaceNew})
	return nil
}

// classifyDirMigration walks an occupying real directory (dirAbs, root-relative dirRel) that a
// symlink-method entry wants to place at, and decides whether the whole tree beneath it is
// safely removable so the entry can be placed as a new symlink (→ ADR-0047, issue #175, #172 D2).
//
// A directory is fully migratable only when every leaf beneath it, at any depth, is one of:
//   - a symlink this profile's own previous generation recorded and the new generation drops
//     (recorded ∧ ¬kept) — scheduled as a RemoveUnlink
//   - an empty subdirectory, regardless of provenance (rmdir only ever succeeds on empty, so this
//     is data-loss-free even for dirs nput never created) — scheduled as a RemoveRmdir
//
// Any other leaf — a regular file, a foreign or record-mismatched symlink, or a symlink the new
// generation still keeps at the same target (self-contradictory manifest) — makes the *whole*
// directory a conflict; no partial removal is scheduled (the caller discards dirActions when
// reason != ""). The walk is lstat-based and never descends into a symlink (a symlink is
// classified as a leaf, matching the ancestor-walk safety rule in ancestorSymlink · → ADR-0046 §2).
//
// The returned actions are ordered children-before-parents (leaves and inner-dir rmdirs appended
// depth-first before the walk returns to its caller), so executing them in slice order naturally
// unlinks leaves first and rmdirs from the deepest directory upward.
func classifyDirMigration(dirRel, dirAbs string, prevByTarget, nextByTarget map[string]manifest.Entry, fs FS) (actions []RemoveAction, reason string, err error) {
	children, err := fs.ReadDir(dirAbs)
	if err != nil {
		return nil, "", fmt.Errorf("nput: cannot read directory (%s): %w", dirAbs, err)
	}
	for _, de := range children {
		childAbs := filepath.Join(dirAbs, de.Name())
		childRel := filepath.Join(dirRel, de.Name())

		info, lerr := fs.Lstat(childAbs)
		if lerr != nil {
			return nil, "", fmt.Errorf("nput: cannot lstat (%s): %w", childAbs, lerr)
		}

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			if _, kept := nextByTarget[childRel]; kept {
				return nil, fmt.Sprintf("%q is a symlink the new generation still keeps (self-contradictory manifest)", childRel), nil
			}
			if !recordedLink(childRel, childAbs, prevByTarget, fs) {
				return nil, fmt.Sprintf("%q is a foreign or record-mismatched symlink", childRel), nil
			}
			actions = append(actions, RemoveAction{Kind: RemoveUnlink, Entry: prevByTarget[childRel], TargetAbs: childAbs})
		case info.IsDir():
			sub, subReason, serr := classifyDirMigration(childRel, childAbs, prevByTarget, nextByTarget, fs)
			if serr != nil {
				return nil, "", serr
			}
			if subReason != "" {
				return nil, subReason, nil
			}
			actions = append(actions, sub...)
			actions = append(actions, RemoveAction{Kind: RemoveRmdir, TargetAbs: childAbs})
		default:
			return nil, fmt.Sprintf("%q is a regular file", childRel), nil
		}
	}
	return actions, "", nil
}

// ancestorSymlink walks the target's ancestor components under root and returns the first
// existing ancestor that is a symlink, as both its absolute path and its root-relative
// (cleaned) target. The caller needs the relative target to look the offender up in the
// prev/next manifests and decide whether it is a self-recorded stale link eligible for
// pre-removal migration (→ ADR-0015, ADR-0046). A non-existent ancestor stops the walk (its
// descendants don't exist either), returning "", "" with no error.
func ancestorSymlink(root, target string, fs FS) (abs, rel string, err error) {
	clean := filepath.Clean(target)
	comps := strings.Split(clean, string(os.PathSeparator))
	cur := root
	for i := 0; i < len(comps)-1; i++ {
		if comps[i] == "" {
			continue
		}
		cur = filepath.Join(cur, comps[i])
		if rel == "" {
			rel = comps[i]
		} else {
			rel = filepath.Join(rel, comps[i])
		}
		info, lerr := fs.Lstat(cur)
		if lerr != nil {
			if os.IsNotExist(lerr) {
				return "", "", nil
			}
			return "", "", fmt.Errorf("nput: cannot lstat ancestor (%s): %w", cur, lerr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return cur, rel, nil
		}
	}
	return "", "", nil
}
