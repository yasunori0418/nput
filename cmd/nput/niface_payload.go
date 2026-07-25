// niface_payload.go maps the mutation commands' engine results (apply / reset / rollback,
// single config) onto the niface SubjectResult payload: full-inventory items, diff-only
// changes, the generation observation, and structured warnings (→ issue #131, ADR-0043).
//
// The builders read the same engine.Result / engine.ResetResult the -v text report reads, so
// the JSON and the human report can never disagree on the outcome (single result source ·
// → issue #131 acceptance). They are pure data mapping: no FS access, no engine calls.
package main

import (
	"fmt"
	"os"

	niface "github.com/yasunori0418/niface/go"

	"github.com/yasunori0418/nput/internal/engine"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/planner"
)

// nifaceEntryInfo is the item.info DTO for an entry item: the entry's declarative identity
// plus its placement shape ({target, method, subpath} · → issue #131). The volatile src
// store path stays out — it belongs to change.info (the diff's concrete content), not to
// the item's identity-adjacent info.
type nifaceEntryInfo struct {
	Target  string `json:"target"`
	Method  string `json:"method,omitempty"`
	Subpath string `json:"subpath,omitempty"`
}

// nifaceChangeInfo is the change.info DTO: the transition's concrete old / new values
// (symlink destinations, copy sources · → issue #131, niface §4). Either side is omitted
// when unknowable (a fresh add has no old; a recopy overwrite's old content is untracked).
type nifaceChangeInfo struct {
	Old string `json:"old,omitempty"`
	New string `json:"new,omitempty"`
}

// nifacePayload is one subject's accumulated result payload, attached to the run by the
// mutation commands and folded into the SubjectResult at emit time (→ nifaceRun.emit).
// TInfo is the owning command's result.info type (→ issue #196).
type nifacePayload[TInfo any] struct {
	items      []nputItem
	changes    []nputChange
	generation *niface.Generation
	warnings   []niface.Warning // subject-level (not item-borne) warnings
	// info is the per-subject tool info (result.info): the read-only enumeration inventories
	// (list-generations の generations / gitignore の paths · → issue #132, ADR-0043 §5).
	// The mutation commands leave it at its zero value (a nil seat pointer, omitted from the
	// document) — their record lives in items / changes.
	info TInfo
	// itemBorne marks that the command error is already represented by a failed item
	// (entry failure / conflict), so emit must not duplicate it into subjectResult.errors[]
	// (niface §2: item 起因のエラーを errors[] に置いてはならない).
	itemBorne bool
}

// attachMutationPayload builds the apply / rollback payload from res and registers it on the
// command's run. A payload-build failure (id derivation — practically impossible for string
// targets) is reported to stderr and the envelope falls back to the minimal #130 shape rather
// than emitting a half-mapped document.
func attachMutationPayload[TInfo, TEnvInfo any](run *nifaceRun[TInfo, TEnvInfo], res *engine.Result, cmdErr error) {
	p, err := mutationPayload[TInfo](res, cmdErr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nput: could not build the --json payload: %v\n", err)
		return
	}
	run.setPayload(p)
}

// attachResetPayload is attachMutationPayload's reset counterpart.
func attachResetPayload[TInfo, TEnvInfo any](run *nifaceRun[TInfo, TEnvInfo], res *engine.ResetResult, cmdErr error) {
	p, err := resetPayload[TInfo](res, cmdErr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nput: could not build the --json payload: %v\n", err)
		return
	}
	run.setPayload(p)
}

// itemStatuses is the reached-state partition shared by the builders (→ niface ADR-0016 /
// ADR-0020): the failed entry, the planned-but-never-attempted entries (skipped — the only
// use of skipped), and per-target conflicts (failed + E_NPUT_COLLISION). Everything else is
// success — including policy inaction (a kept stale target, a place-once copy skip), which
// carries warnings instead of a non-success status.
type itemStatuses struct {
	failed    string
	failedErr *niface.Error
	unreached map[string]bool
	conflicts map[string]planner.Conflict
}

// statusFor resolves one target's item status and error under the partition.
func (s *itemStatuses) statusFor(target string) (niface.ItemStatus, *niface.Error) {
	if c, ok := s.conflicts[target]; ok {
		return niface.ItemFailed, &niface.Error{Code: "E_NPUT_COLLISION", Message: c.Reason}
	}
	if target == s.failed {
		return niface.ItemFailed, s.failedErr
	}
	if s.unreached[target] {
		return niface.ItemSkipped, nil
	}
	return niface.ItemSuccess, nil
}

// newItemStatuses assembles the partition from the engine's reached-state fields. cmdErr is
// the command's overall error: for an entry-scoped failure it IS the failed entry's error
// (the engine stops at the first entry failure), so it becomes that item's error object.
func newItemStatuses(failedTarget string, unreached []string, conflicts []planner.Conflict, cmdErr error) *itemStatuses {
	s := &itemStatuses{failed: failedTarget, unreached: map[string]bool{}}
	for _, t := range unreached {
		s.unreached[t] = true
	}
	if len(conflicts) > 0 {
		s.conflicts = make(map[string]planner.Conflict, len(conflicts))
		for _, c := range conflicts {
			s.conflicts[c.Entry.Target] = c
		}
	}
	if failedTarget != "" && cmdErr != nil {
		e := classifyError(cmdErr)
		s.failedErr = &e
	}
	return s
}

// entryItem renders one manifest entry as a niface item under the status partition.
func entryItem(e manifest.Entry, statuses *itemStatuses) (nputItem, error) {
	id, err := entryItemID(e.Target)
	if err != nil {
		return nputItem{}, err
	}
	status, itemErr := statuses.statusFor(e.Target)
	return nputItem{
		ID:     id,
		Kind:   "entry",
		Label:  e.Target,
		Status: status,
		Error:  itemErr,
		Info:   &nifaceEntryInfo{Target: e.Target, Method: e.Method, Subpath: e.Subpath},
	}, nil
}

// entryChange renders one change for an entry item, deriving the itemId from the target.
func entryChange(target string, kind niface.ChangeKind, reversible bool, info *nifaceChangeInfo) (nputChange, error) {
	id, err := entryItemID(target)
	if err != nil {
		return nputChange{}, err
	}
	return nputChange{Kind: kind, ItemID: id, Reversible: reversible, Info: info}, nil
}

// changeInfoOrNil packs old/new into a change info, or nil when both are unknowable.
func changeInfoOrNil(old, new string) *nifaceChangeInfo {
	if old == "" && new == "" {
		return nil
	}
	return &nifaceChangeInfo{Old: old, New: new}
}

// mutationPayload maps an apply / rollback engine.Result onto the niface payload:
//
//   - items = full inventory: every new-manifest entry plus the previous-generation entries
//     behind planned removals (stale-removed old entries), each with the reached-state status
//     partition applied (→ niface ADR-0016 / ADR-0020).
//   - changes = the diffs that actually happened, in op order: place → add, replace → modify,
//     new copy → add, recopy → modify (irreversible), stale removal → remove. A target both
//     unlinked and re-placed in the same run (a symlink→copy method change) coalesces into one
//     modify — the item's actual old→new transition, not its mechanical op pair.
//   - generation = the run's before/after observation ({profile, before?, after?} · → niface
//     ADR-0015). The first apply has no before; a failed run observes an unmoved pointer.
//   - warnings = the planner's structured warnings, attached to the warned target's item when
//     it is in the inventory and to the subject otherwise (→ niface ADR-0019). An unwound run
//     (→ ADR-0044) additionally carries W_NPUT_UNWOUND at the subject level: the changes list
//     stays the record of what happened up to the failure, and the warning tells consumers
//     those diffs were rolled back rather than left on disk.
func mutationPayload[TInfo any](res *engine.Result, cmdErr error) (*nifacePayload[TInfo], error) {
	statuses := newItemStatuses(res.FailedTarget, res.Unreached, res.Conflicts, cmdErr)
	p := &nifacePayload[TInfo]{itemBorne: res.FailedTarget != "" || len(res.Conflicts) > 0}

	// Items: the new manifest's inventory first, then the stale-removed old entries not
	// shadowed by it (a method-change target lives in both; the new entry wins).
	inventory := map[string]bool{}
	for _, e := range res.Entries {
		item, err := entryItem(e, statuses)
		if err != nil {
			return nil, err
		}
		p.items = append(p.items, item)
		inventory[e.Target] = true
	}
	oldEntries := map[string]manifest.Entry{}
	for _, e := range res.RemovalEntries {
		if inventory[e.Target] {
			continue
		}
		if _, seen := oldEntries[e.Target]; seen {
			continue
		}
		oldEntries[e.Target] = e
		item, err := entryItem(e, statuses)
		if err != nil {
			return nil, err
		}
		p.items = append(p.items, item)
		inventory[e.Target] = true
	}

	// Changes. oldDest resolves what a target pointed at before this run touched it: the
	// pre-re-link readlink for replaces (exact, covers foreign dests), the recorded dest for
	// removals (re-verified on disk immediately before the unlink).
	removalByTarget := map[string]manifest.Entry{}
	for _, e := range res.RemovalEntries {
		removalByTarget[e.Target] = e
	}
	oldDest := func(target string) string {
		if d, ok := res.ReplacedDests[target]; ok {
			return d
		}
		if e, ok := removalByTarget[target]; ok {
			return planner.LinkDest(e)
		}
		return ""
	}
	newDest := map[string]string{}
	for _, e := range res.Entries {
		newDest[e.Target] = planner.LinkDest(e)
	}
	removed := map[string]bool{}
	for _, t := range res.Removed {
		removed[t] = true
	}
	rePlaced := map[string]bool{}
	addChange := func(target string, kind niface.ChangeKind, reversible bool, info *nifaceChangeInfo) error {
		c, err := entryChange(target, kind, reversible, info)
		if err != nil {
			return err
		}
		p.changes = append(p.changes, c)
		return nil
	}
	place := func(targets []string, modifyAlways, reversible, skipNoop bool) error {
		for _, t := range targets {
			rePlaced[t] = true
			kind := niface.ChangeAdd
			old := ""
			if modifyAlways || removed[t] {
				// Re-link, or unlinked-then-re-placed in the same run: the item's actual
				// transition is old → new, one modify.
				kind = niface.ChangeModify
				old = oldDest(t)
			}
			if skipNoop && old != "" && old == newDest[t] {
				// A re-link back to the recorded dest (apply re-links every planned symlink
				// mechanically) leaves the state identical: a noop, which changes must not
				// contain (niface §4). The -v report still shows the mechanical op.
				continue
			}
			if err := addChange(t, kind, reversible, changeInfoOrNil(old, newDest[t])); err != nil {
				return err
			}
		}
		return nil
	}
	if err := place(res.Placed, false, true, false); err != nil {
		return nil, err
	}
	if err := place(res.Replaced, true, true, true); err != nil {
		return nil, err
	}
	if err := place(res.Copied, false, true, false); err != nil {
		return nil, err
	}
	// A recopy overwrite discards content nput never tracked: irreversible, no old value
	// (→ ADR-0020, ADR-0043 §4). Not a noop even when content happens to match — the
	// overwrite itself happened and the pre-state is unknowable.
	if err := place(res.Recopied, true, false, false); err != nil {
		return nil, err
	}
	for _, t := range res.Removed {
		if rePlaced[t] {
			continue // coalesced into the modify above
		}
		if err := addChange(t, niface.ChangeRemove, true, changeInfoOrNil(oldDest(t), "")); err != nil {
			return nil, err
		}
	}

	// Generation: the run's observation (→ niface ADR-0015). Reset never comes through here —
	// it has its own builder without a generation slot (an FS-only teardown moves nothing).
	p.generation = &niface.Generation{Profile: res.Profile, Before: res.GenBefore, After: res.GenAfter}

	p.warnings = attachWarnings(p.items, res.Warnings)
	if res.Unwound {
		p.warnings = append(p.warnings, niface.Warning{
			Code:    "W_NPUT_UNWOUND",
			Message: "the undo journal rolled this run's filesystem changes back after the failure; the listed changes did not survive on disk",
		})
	}
	return p, nil
}

// resetPayload maps an engine.ResetResult onto the niface payload: items = the selected
// teardown entries (reset's full inventory), changes = the removals that actually happened
// (symlink removals reversible, copy deletions irreversible — copy content is untracked ·
// → ADR-0043 §4), no generation slot (reset is an FS-only teardown: the profile pointer and
// the generations are untouched, so there is no transition to observe · → issue #131).
// An aborted run (confirmation declined — unreachable under --json, which never prompts)
// changed nothing: items stay success, changes stay empty.
func resetPayload[TInfo any](res *engine.ResetResult, cmdErr error) (*nifacePayload[TInfo], error) {
	statuses := newItemStatuses(res.FailedTarget, res.Unreached, nil, cmdErr)
	p := &nifacePayload[TInfo]{itemBorne: res.FailedTarget != ""}

	byTarget := map[string]manifest.Entry{}
	for _, e := range res.Entries {
		item, err := entryItem(e, statuses)
		if err != nil {
			return nil, err
		}
		p.items = append(p.items, item)
		byTarget[e.Target] = e
	}
	if !res.Aborted {
		for _, t := range res.RemovedSymlinks {
			info := changeInfoOrNil(planner.LinkDest(byTarget[t]), "")
			c, err := entryChange(t, niface.ChangeRemove, true, info)
			if err != nil {
				return nil, err
			}
			p.changes = append(p.changes, c)
		}
		for _, t := range res.RemovedCopies {
			// No info: what a copy deletion destroys is the on-disk content, which nput does
			// not track (the recorded src is not what was lost · → ADR-0020).
			c, err := entryChange(t, niface.ChangeRemove, false, nil)
			if err != nil {
				return nil, err
			}
			p.changes = append(p.changes, c)
		}
	}

	p.warnings = attachWarnings(p.items, res.Warnings)
	return p, nil
}

// attachWarnings maps the planner's structured warnings onto the payload: a warning whose
// target is an inventory item lands in that item's warnings, anything else (a target outside
// the inventory — e.g. a kept stale symlink or a copy orphan whose entry left the config) is
// returned as a subject-level warning (→ niface ADR-0019). items is mutated in place.
func attachWarnings(items []nputItem, warnings []planner.Warning) []niface.Warning {
	itemIdx := map[string]int{}
	for i, it := range items {
		itemIdx[it.Info.Target] = i
	}
	var subject []niface.Warning
	for _, w := range warnings {
		nw := nifaceWarning(w)
		if i, ok := itemIdx[w.Target]; ok {
			items[i].Warnings = append(items[i].Warnings, nw)
			continue
		}
		subject = append(subject, nw)
	}
	return subject
}

// nifaceWarning translates one planner warning into the niface warning vocabulary
// (tool-specific W_NPUT_* codes · niface §6 two-layer naming). The messages mirror the
// stderr text (→ engine.emitWarnings) without the "nput: " prefix and target suffix — the
// target rides in detail (and in the carrying item) instead.
func nifaceWarning(w planner.Warning) niface.Warning {
	var code, msg string
	switch w.Kind {
	case planner.WarnForeignReplace:
		code, msg = "W_NPUT_FOREIGN_SYMLINK", "overwriting an unrecorded symlink (foreign; last-wins)"
	case planner.WarnStaleMismatch:
		code, msg = "W_NPUT_STALE_MISMATCH", "keeping stale symlink because it mismatches the record"
	case planner.WarnStaleNonSymlink:
		code, msg = "W_NPUT_STALE_NON_SYMLINK", "keeping stale target because it is not a symlink"
	case planner.WarnCopyOrphan:
		code, msg = "W_NPUT_COPY_ORPHAN", "copy entry vanished but the target is not removed (orphan; clear it with reset)"
	case planner.WarnCopyForeign:
		code, msg = "W_NPUT_COPY_FOREIGN", "skipped copy because a real file already exists at the copy target (foreign; place-once)"
	default:
		// Unreachable today — the switch covers every planner.WarnKind. Kept as a defensive
		// fallback so a future kind surfaces visibly instead of being silently mis-coded.
		code, msg = "W_NPUT_WARNING", "unclassified planner warning"
	}
	return niface.Warning{Code: code, Message: msg, Detail: map[string]any{"target": w.Target}}
}
