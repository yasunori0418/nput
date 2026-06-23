package engine

import (
	"path/filepath"

	"github.com/yasunori0418/nput/internal/planner"
)

// Generation skip + lstat drift repair (project mode only · → ADR-0005, ADR-0017,
// docs/spec.md "project mode generations").
//
// The devShell / direnv shellHook runs on every shell re-entry, so if the new
// link-farm derivation is identical to the previous generation, no new generation
// is committed (--set is omitted · avoiding unbounded generation growth). However,
// "same derivation ⇒ same FS" breaks under foreign tool rewrites, so instead of a
// full no-op each entry's target is lstat-checked and only drifted entries are re-linked.

// generationUnchanged reports whether the new link-farm is the same store derivation
// as the previous generation (the one the profile link points at). It resolves the
// chain profile link → generation link → link-farm store path and compares it with the
// new link-farm's store path. If either cannot be resolved it returns an error and the
// caller falls back to the safe side (normal apply = new generation commit).
func generationUnchanged(profileLink, newLinkFarm string) (bool, error) {
	prev, err := filepath.EvalSymlinks(profileLink)
	if err != nil {
		return false, err
	}
	next, err := filepath.EvalSymlinks(newLinkFarm)
	if err != nil {
		return false, err
	}
	return prev == next, nil
}

// repairDrift performs the lstat drift repair during a generation skip. Of the plan the
// planner computed from the current FS state, it re-converges **only** the drifted entries,
// performing neither stale removal nor a generation commit.
//
//   - symlink: re-links only PlaceNew (target gone) and PlaceForeign (foreign rewrite).
//     PlaceReplace is "a symlink pointing at the recorded dest" = no drift, so it is left
//     untouched (→ ADR-0017). The foreign-rewrite warning has already been emitted by the
//     caller via emitWarnings(plan.Warnings).
//   - copy: planner.Copies contains only place-once actions for absent targets, so reflecting
//     it gives "copy reverts place-once only when the target is absent". Content diffs (user
//     edits) produce no CopyAction and are left untouched (→ docs/spec.md "project mode
//     generations" · ADR-0022). On --recopy, recopyAll overwrites unconditionally (recopy is
//     an opt-in outside generations).
func (a *applier) repairDrift(plan planner.Plan, recopy bool) error {
	drifted := make([]planner.PlaceAction, 0, len(plan.Place))
	for _, act := range plan.Place {
		if act.Kind == planner.PlaceReplace {
			continue // symlink pointing at the recorded dest = in-sync. No drift, no re-link needed.
		}
		drifted = append(drifted, act)
	}
	if err := a.place(drifted); err != nil {
		return err
	}
	return a.materializeCopies(plan, recopy)
}
