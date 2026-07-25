package main

// Tests for the #131 mutation payloads (apply / reset / rollback): the engine-result → niface
// items/changes/generation/warnings mapping, the partial-failure partition (niface ADR-0016 /
// ADR-0020), the error-layer placement (item-borne vs subject-borne · niface §2), and
// conformance of every emitted shape against niface's checker.

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	niface "github.com/yasunori0418/niface/go"
	"github.com/yasunori0418/niface/go/conformance"

	"github.com/yasunori0418/nput/internal/engine"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/planner"
)

func ip(n int) *int { return &n }

// mustItemID resolves the entry item id or fails the test.
func mustItemID(t *testing.T, target string) string {
	t.Helper()
	id, err := entryItemID(target)
	if err != nil {
		t.Fatalf("entryItemID(%q): %v", target, err)
	}
	return id
}

// findItem returns the item whose info.target matches, failing when absent.
func findItem(t *testing.T, items []nputItem, target string) nputItem {
	t.Helper()
	for _, it := range items {
		if it.Info != nil && it.Info.Target == target {
			return it
		}
	}
	t.Fatalf("no item for target %q in %+v", target, items)
	return nputItem{}
}

// changesFor returns every change whose itemId belongs to target.
func changesFor(t *testing.T, changes []nputChange, target string) []nputChange {
	t.Helper()
	id := mustItemID(t, target)
	var out []nputChange
	for _, c := range changes {
		if c.ItemID == id {
			out = append(out, c)
		}
	}
	return out
}

// emitPayloadDoc runs the payload through the real emit path (command + subject + payload),
// checks the document against niface's conformance checker, and returns it decoded. newRun is
// the emitting command's test-run constructor (newApplyTestRun, ...), which carries both the
// command name and its info pair from the production alias — so no call site re-spells the type
// arguments (→ issue #196). The mutation commands' pairs are empty seat types, so the emitted
// document carries no info key on either level.
func emitPayloadDoc[TInfo, TEnvInfo any](t *testing.T, newRun func() (*nifaceRun[TInfo, TEnvInfo], *bytes.Buffer), p *nifacePayload[TInfo], cmdErr error) map[string]any {
	t.Helper()
	checker, err := conformance.NewDefaultChecker()
	if err != nil {
		t.Fatalf("conformance.NewDefaultChecker: %v", err)
	}
	r, buf := newRun()
	r.beginSubject("default").setPayload(p)
	if err := r.emit(cmdErr); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if findings := checker.Check(buf.Bytes()); len(findings) > 0 {
		t.Fatalf("conformance findings:\n%s\ndocument: %s", strings.Join(findings, "\n"), buf.String())
	}
	return decodeEnvelope(t, buf)
}

// assertNoInfoKeys fails when the document carries an info key at the envelope level or inside
// any result. The seat types are empty structs held as nil pointers precisely so omitempty
// drops both keys (→ issue #196 §4); a value struct, or a seat accidentally filled with &T{},
// would emit "info":{} — schema-valid (envelope.schema.json types info as a bare object) and
// therefore invisible to the conformance checker, so it has to be pinned here.
//
// The results loop is empty for init — the one command that registers no subject — leaving the
// envelope-level check as the whole assertion there, which is the only level its documents have.
func assertNoInfoKeys(t *testing.T, doc map[string]any) {
	t.Helper()
	if v, ok := doc["info"]; ok {
		t.Errorf("envelope info = %v, want the key absent (the seat type is a nil pointer)", v)
	}
	results, _ := doc["results"].([]any)
	for i, r := range results {
		res, _ := r.(map[string]any)["result"].(map[string]any)
		if v, ok := res["info"]; ok {
			t.Errorf("results[%d].result.info = %v, want the key absent (the seat type is a nil pointer)", i, v)
		}
	}
}

// TestMutationSeatInfoKeysStayAbsent pins the output-invariance requirement that made the
// mutation info slots empty seat types behind pointers (→ issue #196 §4): apply / reset /
// rollback must emit no info key at either level, both with a payload attached and on the
// payload-less path (a failure before any engine result exists). Without this the seats could
// silently start emitting "info":{} — the conformance checker accepts it, so no other test
// in the suite would notice.
func TestMutationSeatInfoKeysStayAbsent(t *testing.T) {
	res := &engine.Result{
		Profile: "/p",
		Entries: []manifest.Entry{{SrcKind: "store", Src: "/nix/store/a", Target: "a", Method: "symlink"}},
		Placed:  []string{"a"},
	}
	resetRes := &engine.ResetResult{
		Entries:         []manifest.Entry{{SrcKind: "store", Src: "/nix/store/s", Target: "s", Method: "symlink"}},
		RemovedSymlinks: []string{"s"},
	}

	t.Run("apply with payload", func(t *testing.T) {
		p, err := mutationPayload[*applyResultInfo](res, nil)
		if err != nil {
			t.Fatalf("mutationPayload: %v", err)
		}
		assertNoInfoKeys(t, emitPayloadDoc(t, newApplyTestRun, p, nil))
	})
	t.Run("rollback with payload", func(t *testing.T) {
		p, err := mutationPayload[*rollbackResultInfo](res, nil)
		if err != nil {
			t.Fatalf("mutationPayload: %v", err)
		}
		assertNoInfoKeys(t, emitPayloadDoc(t, newRollbackTestRun, p, nil))
	})
	t.Run("reset with payload", func(t *testing.T) {
		p, err := resetPayload[*resetResultInfo](resetRes, nil)
		if err != nil {
			t.Fatalf("resetPayload: %v", err)
		}
		assertNoInfoKeys(t, emitPayloadDoc(t, newResetTestRun, p, nil))
	})
	// The subject-registered-but-payload-less path: the command failed before any engine result
	// existed, so Result.Info stays the zero value. A non-pointer seat would surface here.
	t.Run("apply without payload", func(t *testing.T) {
		r, buf := newApplyTestRun()
		r.beginSubject("default")
		if err := r.emit(errors.New("nput: no entrypoint found")); err != nil {
			t.Fatalf("emit: %v", err)
		}
		assertNoInfoKeys(t, decodeEnvelope(t, buf))
	})
}

// subjectResultOf digs results[0] out of a decoded envelope.
func subjectResultOf(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	results := doc["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results length = %d, want 1", len(results))
	}
	return results[0].(map[string]any)
}

// TestMutationPayloadFullInventory pins the success mapping: every new-manifest entry plus the
// stale-removed old entry as items (all success), diff-only changes with kind/reversible/info,
// warnings attached to the warned item, and the generation observation (first apply: before
// omitted · → issue #131).
func TestMutationPayloadFullInventory(t *testing.T) {
	res := &engine.Result{
		Profile: "/state/nput/default/profile",
		Entries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/aaa", Subpath: "conf", Target: ".config/tool", Method: "symlink"},
			{SrcKind: "store", Src: "/nix/store/bbb", Target: ".config/relinked", Method: "symlink"},
			{SrcKind: "store", Src: "/nix/store/ccc", Target: ".local/copy", Method: "copy"},
			{SrcKind: "store", Src: "/nix/store/ddd", Target: ".local/recopied", Method: "copy"},
			{SrcKind: "store", Src: "/nix/store/eee", Target: ".config/noop", Method: "symlink"},
		},
		RemovalEntries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/old", Subpath: "sub", Target: ".config/old", Method: "symlink"},
		},
		Placed:        []string{".config/tool"},
		Replaced:      []string{".config/relinked"},
		ReplacedDests: map[string]string{".config/relinked": "/nix/store/prev-bbb"},
		Copied:        []string{".local/copy"},
		Recopied:      []string{".local/recopied"},
		Removed:       []string{".config/old"},
		GenAfter:      ip(1),
		Warnings:      []planner.Warning{{Kind: planner.WarnForeignReplace, Target: ".config/relinked"}},
	}
	p, err := mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}

	if len(p.items) != 6 {
		t.Fatalf("items = %d, want 6 (5 new entries + 1 stale-removed old entry)", len(p.items))
	}
	for _, it := range p.items {
		if it.Status != niface.ItemSuccess {
			t.Errorf("item %s status = %s, want success", it.Info.Target, it.Status)
		}
	}
	old := findItem(t, p.items, ".config/old")
	if old.Info.Method != "symlink" || old.Info.Subpath != "sub" || old.Label != ".config/old" {
		t.Errorf("stale-removed old entry item = %+v, want method/subpath from the previous generation", old.Info)
	}

	if len(p.changes) != 5 {
		t.Fatalf("changes = %d, want 5 (the no-op entry has none)", len(p.changes))
	}
	assertChange := func(target string, kind niface.ChangeKind, reversible bool, old, new string) {
		t.Helper()
		cs := changesFor(t, p.changes, target)
		if len(cs) != 1 {
			t.Fatalf("changes for %s = %d, want 1", target, len(cs))
		}
		c := cs[0]
		if c.Kind != kind || c.Reversible != reversible {
			t.Errorf("%s change = kind %s reversible %v, want %s/%v", target, c.Kind, c.Reversible, kind, reversible)
		}
		gotOld, gotNew := "", ""
		if c.Info != nil {
			gotOld, gotNew = c.Info.Old, c.Info.New
		}
		if gotOld != old || gotNew != new {
			t.Errorf("%s change info = {old:%q new:%q}, want {old:%q new:%q}", target, gotOld, gotNew, old, new)
		}
	}
	assertChange(".config/tool", niface.ChangeAdd, true, "", "/nix/store/aaa/conf")
	assertChange(".config/relinked", niface.ChangeModify, true, "/nix/store/prev-bbb", "/nix/store/bbb")
	assertChange(".local/copy", niface.ChangeAdd, true, "", "/nix/store/ccc")
	assertChange(".local/recopied", niface.ChangeModify, false, "", "/nix/store/ddd")
	assertChange(".config/old", niface.ChangeRemove, true, "/nix/store/old/sub", "")

	relinked := findItem(t, p.items, ".config/relinked")
	if len(relinked.Warnings) != 1 || relinked.Warnings[0].Code != "W_NPUT_FOREIGN_SYMLINK" {
		t.Errorf("relinked item warnings = %+v, want one W_NPUT_FOREIGN_SYMLINK", relinked.Warnings)
	}
	if len(p.warnings) != 0 {
		t.Errorf("subject warnings = %+v, want none (the warned target is an item)", p.warnings)
	}

	doc := emitPayloadDoc(t, newApplyTestRun, p, nil)
	sr := subjectResultOf(t, doc)
	gen, ok := sr["generation"].(map[string]any)
	if !ok {
		t.Fatalf("generation missing in %v", sr)
	}
	if _, hasBefore := gen["before"]; hasBefore {
		t.Errorf("generation.before = %v, want omitted on the first apply", gen["before"])
	}
	if gen["after"] != json.Number("1") || gen["profile"] != res.Profile {
		t.Errorf("generation = %v, want after=1 profile=%s", gen, res.Profile)
	}
}

// TestMutationPayloadMethodChangeCoalesces pins the symlink→copy method change: the same
// target unlinked and re-placed in one run yields one item (the new entry) and one modify
// change carrying the old symlink dest → new copy source transition (→ issue #131).
func TestMutationPayloadMethodChangeCoalesces(t *testing.T) {
	res := &engine.Result{
		Profile: "/p",
		Entries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/new", Target: ".config/x", Method: "copy"},
		},
		RemovalEntries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/old", Target: ".config/x", Method: "symlink"},
		},
		Removed: []string{".config/x"},
		Copied:  []string{".config/x"},
	}
	p, err := mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	if len(p.items) != 1 {
		t.Fatalf("items = %d, want 1 (the new entry shadows the removed old one)", len(p.items))
	}
	if got := findItem(t, p.items, ".config/x").Info.Method; got != "copy" {
		t.Errorf("item method = %s, want the new entry's copy", got)
	}
	if len(p.changes) != 1 {
		t.Fatalf("changes = %+v, want the coalesced single modify", p.changes)
	}
	c := p.changes[0]
	if c.Kind != niface.ChangeModify || !c.Reversible || c.Info.Old != "/nix/store/old" || c.Info.New != "/nix/store/new" {
		t.Errorf("change = %+v info %+v, want reversible modify old→new", c, c.Info)
	}
}

// TestMutationPayloadNoopRelinkSuppressed pins the idempotent-re-apply case: apply
// mechanically re-links a planned symlink back to its recorded dest, which is not a state
// transition — such a Replaced target must produce no change (niface §4 noop MUST NOT),
// while a re-link to a different dest still does.
func TestMutationPayloadNoopRelinkSuppressed(t *testing.T) {
	res := &engine.Result{
		Profile: "/p",
		Entries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/same", Target: ".same", Method: "symlink"},
			{SrcKind: "store", Src: "/nix/store/new", Target: ".moved", Method: "symlink"},
		},
		Replaced: []string{".same", ".moved"},
		ReplacedDests: map[string]string{
			".same":  "/nix/store/same", // re-linked back to the recorded dest = noop
			".moved": "/nix/store/old",  // genuinely moved
		},
	}
	p, err := mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	if cs := changesFor(t, p.changes, ".same"); len(cs) != 0 {
		t.Errorf(".same changes = %+v, want none (noop re-link must be suppressed)", cs)
	}
	if cs := changesFor(t, p.changes, ".moved"); len(cs) != 1 || cs[0].Kind != niface.ChangeModify {
		t.Errorf(".moved changes = %+v, want one modify", cs)
	}
	if it := findItem(t, p.items, ".same"); it.Status != niface.ItemSuccess {
		t.Errorf(".same item status = %s, want success (still in the inventory)", it.Status)
	}
}

// TestMutationPayloadOrphanSubjectWarning pins attachWarnings' subject-side branch: a
// planner warning whose target is outside the inventory (a vanished copy entry's orphan —
// neither a new-manifest entry nor a planned removal) lands in subjectResult.warnings, not
// on any item (→ niface ADR-0019).
func TestMutationPayloadOrphanSubjectWarning(t *testing.T) {
	res := &engine.Result{
		Profile: "/p",
		Entries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/a", Target: "a", Method: "symlink"},
		},
		Warnings: []planner.Warning{{Kind: planner.WarnCopyOrphan, Target: ".gone/copy"}},
	}
	p, err := mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	if len(p.warnings) != 1 || p.warnings[0].Code != "W_NPUT_COPY_ORPHAN" || p.warnings[0].Detail["target"] != ".gone/copy" {
		t.Fatalf("subject warnings = %+v, want one W_NPUT_COPY_ORPHAN carrying the target", p.warnings)
	}
	if it := findItem(t, p.items, "a"); len(it.Warnings) != 0 {
		t.Errorf("item warnings = %+v, want none (the orphan is not this item's)", it.Warnings)
	}
	doc := emitPayloadDoc(t, newApplyTestRun, p, nil)
	sr := subjectResultOf(t, doc)
	warns, ok := sr["warnings"].([]any)
	if !ok || len(warns) != 1 {
		t.Fatalf("subjectResult.warnings = %v, want the orphan warning", sr["warnings"])
	}
}

// TestMutationPayloadRollbackGeneration pins the rollback reuse of mutationPayload: the
// From→To transition rides generation.before/after (nothing in result.info), and a failed
// rollback observes the pinned, unmoved generation (before == after == current).
func TestMutationPayloadRollbackGeneration(t *testing.T) {
	rr := &engine.RollbackResult{
		Result: engine.Result{
			Profile: "/p",
			Entries: []manifest.Entry{
				{SrcKind: "store", Src: "/nix/store/a", Target: "a", Method: "symlink"},
			},
			Replaced:      []string{"a"},
			ReplacedDests: map[string]string{"a": "/nix/store/newer"},
			GenBefore:     ip(5),
			GenAfter:      ip(4),
		},
		From: 5, To: 4,
	}
	p, err := mutationPayload[*rollbackResultInfo](&rr.Result, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	doc := emitPayloadDoc(t, newRollbackTestRun, p, nil)
	sr := subjectResultOf(t, doc)
	gen := sr["generation"].(map[string]any)
	if gen["before"] != json.Number("5") || gen["after"] != json.Number("4") {
		t.Errorf("generation = %v, want the 5→4 rollback transition", gen)
	}

	// A failed rollback pins the pointer at the unmoved current generation.
	rr.GenBefore, rr.GenAfter = ip(5), ip(5)
	p, err = mutationPayload[*rollbackResultInfo](&rr.Result, errors.New("nput: failed to move the profile pointer"))
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	if p.generation.Before == nil || *p.generation.Before != 5 || p.generation.After == nil || *p.generation.After != 5 {
		t.Errorf("failed rollback generation = %+v, want the unmoved 5→5 observation", p.generation)
	}
}

// TestNifaceWarningMapping pins every planner WarnKind → W_NPUT_* code pair, plus the
// defensive fallback for an unknown kind.
func TestNifaceWarningMapping(t *testing.T) {
	cases := []struct {
		kind planner.WarnKind
		code string
	}{
		{planner.WarnForeignReplace, "W_NPUT_FOREIGN_SYMLINK"},
		{planner.WarnStaleMismatch, "W_NPUT_STALE_MISMATCH"},
		{planner.WarnStaleNonSymlink, "W_NPUT_STALE_NON_SYMLINK"},
		{planner.WarnCopyOrphan, "W_NPUT_COPY_ORPHAN"},
		{planner.WarnCopyForeign, "W_NPUT_COPY_FOREIGN"},
		{planner.WarnKind(99), "W_NPUT_WARNING"},
	}
	for _, c := range cases {
		w := nifaceWarning(planner.Warning{Kind: c.kind, Target: "t"})
		if w.Code != c.code {
			t.Errorf("kind %v: code = %s, want %s", c.kind, w.Code, c.code)
		}
		if w.Message == "" {
			t.Errorf("kind %v: message must not be empty", c.kind)
		}
		if w.Detail["target"] != "t" {
			t.Errorf("kind %v: detail = %v, want {target: t}", c.kind, w.Detail)
		}
	}
}

// TestMutationPayloadPartialFailure pins the reached-state partition (niface ADR-0016 /
// ADR-0020): the failed entry carries the classified command error, unreached entries are
// skipped (and only those), completed entries stay success with their changes, the unwound
// run carries W_NPUT_UNWOUND at the subject, and the item-borne failure is NOT duplicated
// into subjectResult.errors[] (niface §2).
func TestMutationPayloadPartialFailure(t *testing.T) {
	res := &engine.Result{
		Profile: "/p",
		Entries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/a", Target: "a", Method: "symlink"},
			{SrcKind: "store", Src: "/nix/store/b", Target: "b", Method: "symlink"},
			{SrcKind: "store", Src: "/nix/store/c", Target: "c", Method: "symlink"},
		},
		Placed:       []string{"a"},
		FailedTarget: "b",
		Unreached:    []string{"c"},
		Unwound:      true,
		GenBefore:    ip(3),
		GenAfter:     ip(3),
	}
	cmdErr := &os.PathError{Op: "symlink", Path: "/root/b", Err: fs.ErrPermission}
	p, err := mutationPayload[*applyResultInfo](res, cmdErr)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	if !p.itemBorne {
		t.Error("itemBorne = false, want true for an entry-scoped failure")
	}

	if it := findItem(t, p.items, "a"); it.Status != niface.ItemSuccess {
		t.Errorf("completed item status = %s, want success", it.Status)
	}
	failed := findItem(t, p.items, "b")
	if failed.Status != niface.ItemFailed || failed.Error == nil || failed.Error.Code != "E_PERMISSION" {
		t.Errorf("failed item = status %s error %+v, want failed + E_PERMISSION", failed.Status, failed.Error)
	}
	if it := findItem(t, p.items, "c"); it.Status != niface.ItemSkipped {
		t.Errorf("unreached item status = %s, want skipped", it.Status)
	}
	if cs := changesFor(t, p.changes, "a"); len(cs) != 1 || cs[0].Kind != niface.ChangeAdd {
		t.Errorf("completed entry changes = %+v, want its add present despite the failure", cs)
	}
	var unwound bool
	for _, w := range p.warnings {
		if w.Code == "W_NPUT_UNWOUND" {
			unwound = true
		}
	}
	if !unwound {
		t.Errorf("subject warnings = %+v, want W_NPUT_UNWOUND for the unwound run", p.warnings)
	}

	doc := emitPayloadDoc(t, newApplyTestRun, p, cmdErr)
	if doc["status"] != "error" {
		t.Errorf("status = %v, want error", doc["status"])
	}
	sr := subjectResultOf(t, doc)
	if sr["status"] != "error" {
		t.Errorf("subject status = %v, want error", sr["status"])
	}
	if errList, ok := sr["errors"]; ok {
		t.Errorf("subjectResult.errors = %v, want absent (the failure is item-borne)", errList)
	}
	gen := sr["generation"].(map[string]any)
	if gen["before"] != json.Number("3") || gen["after"] != json.Number("3") {
		t.Errorf("generation = %v, want an unmoved 3→3 observation", gen)
	}
}

// TestMutationPayloadSubjectBorneFailure pins the non-entry-scoped failure (commit / build):
// no failed item, the full changes stay, and the classified error lands in
// subjectResult.errors[] (→ ADR-0043 §6).
func TestMutationPayloadSubjectBorneFailure(t *testing.T) {
	res := &engine.Result{
		Profile: "/p",
		Entries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/a", Target: "a", Method: "symlink"},
		},
		Placed:   []string{"a"},
		GenAfter: ip(2), GenBefore: ip(2),
	}
	cmdErr := errors.New("nput: generation commit (nix-env --set) failed: exit status 1")
	p, err := mutationPayload[*applyResultInfo](res, cmdErr)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	if p.itemBorne {
		t.Error("itemBorne = true, want false for a commit failure")
	}
	if it := findItem(t, p.items, "a"); it.Status != niface.ItemSuccess {
		t.Errorf("placed item status = %s, want success (the commit, not the entry, failed)", it.Status)
	}
	doc := emitPayloadDoc(t, newApplyTestRun, p, cmdErr)
	sr := subjectResultOf(t, doc)
	errList, ok := sr["errors"].([]any)
	if !ok || len(errList) != 1 {
		t.Fatalf("subjectResult.errors = %v, want exactly one subject-borne error", sr["errors"])
	}
	if code := errList[0].(map[string]any)["code"]; code != "E_NPUT_FAILED" {
		t.Errorf("error code = %v, want the generic fallback for a commit failure", code)
	}
	items := sr["result"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Errorf("items = %v, want the full inventory alongside the subject error", items)
	}
}

// TestMutationPayloadConflicts pins the conflict mapping: each conflicted entry is a failed
// item with E_NPUT_COLLISION and the planner reason, everything else planned is skipped, and
// the aggregate command error is not duplicated at the subject layer (→ ADR-0043 §6).
func TestMutationPayloadConflicts(t *testing.T) {
	res := &engine.Result{
		Profile: "/p",
		Entries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/a", Target: "a", Method: "symlink"},
			{SrcKind: "store", Src: "/nix/store/b", Target: "b", Method: "symlink"},
		},
		Conflicts: []planner.Conflict{
			{Entry: manifest.Entry{Target: "a"}, Reason: "a regular file occupies the symlink target", Kind: planner.ConflictForeignEntity},
		},
		Unreached: []string{"b"},
	}
	cmdErr := errors.New("nput: 1 conflict(s) detected; stopped without placing (see above)")
	p, err := mutationPayload[*applyResultInfo](res, cmdErr)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	if !p.itemBorne {
		t.Error("itemBorne = false, want true for a conflict stop")
	}
	conflicted := findItem(t, p.items, "a")
	if conflicted.Status != niface.ItemFailed || conflicted.Error == nil ||
		conflicted.Error.Code != "E_NPUT_COLLISION" || conflicted.Error.Message != "a regular file occupies the symlink target" {
		t.Errorf("conflicted item = %+v error %+v, want failed + E_NPUT_COLLISION with the planner reason", conflicted, conflicted.Error)
	}
	if it := findItem(t, p.items, "b"); it.Status != niface.ItemSkipped {
		t.Errorf("non-conflicted item status = %s, want skipped (nothing ran)", it.Status)
	}
	if len(p.changes) != 0 {
		t.Errorf("changes = %+v, want none (the run stopped before any FS action)", p.changes)
	}
	doc := emitPayloadDoc(t, newApplyTestRun, p, cmdErr)
	sr := subjectResultOf(t, doc)
	if errList, ok := sr["errors"]; ok {
		t.Errorf("subjectResult.errors = %v, want absent (conflicts are item-borne)", errList)
	}
}

// TestResetPayload pins reset's mapping: items = the selected teardown entries, symlink
// removals are reversible remove changes with the recorded dest, copy deletions are
// irreversible removes without info, a kept-foreign target stays success with the
// W_NPUT_STALE_MISMATCH warning on its item, and no generation slot is emitted (reset never
// moves the profile pointer · → issue #131).
func TestResetPayload(t *testing.T) {
	res := &engine.ResetResult{
		Entries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/s1", Target: "s1", Method: "symlink"},
			{SrcKind: "store", Src: "/nix/store/s2", Target: "s2", Method: "symlink"},
			{SrcKind: "store", Src: "/nix/store/c1", Target: "c1", Method: "copy"},
		},
		RemovedSymlinks: []string{"s1"},
		RemovedCopies:   []string{"c1"},
		KeptForeign:     []string{"s2"},
		Warnings:        []planner.Warning{{Kind: planner.WarnStaleMismatch, Target: "s2"}},
	}
	p, err := resetPayload[*resetResultInfo](res, nil)
	if err != nil {
		t.Fatalf("resetPayload: %v", err)
	}
	if p.generation != nil {
		t.Fatalf("generation = %+v, want none for reset", p.generation)
	}
	for _, target := range []string{"s1", "s2", "c1"} {
		if it := findItem(t, p.items, target); it.Status != niface.ItemSuccess {
			t.Errorf("item %s status = %s, want success", target, it.Status)
		}
	}
	kept := findItem(t, p.items, "s2")
	if len(kept.Warnings) != 1 || kept.Warnings[0].Code != "W_NPUT_STALE_MISMATCH" {
		t.Errorf("kept item warnings = %+v, want one W_NPUT_STALE_MISMATCH", kept.Warnings)
	}
	if cs := changesFor(t, p.changes, "s1"); len(cs) != 1 || cs[0].Kind != niface.ChangeRemove ||
		!cs[0].Reversible || cs[0].Info == nil || cs[0].Info.Old != "/nix/store/s1" {
		t.Errorf("symlink removal change = %+v, want reversible remove with the recorded dest", cs)
	}
	if cs := changesFor(t, p.changes, "c1"); len(cs) != 1 || cs[0].Kind != niface.ChangeRemove ||
		cs[0].Reversible || cs[0].Info != nil {
		t.Errorf("copy removal change = %+v, want irreversible remove without info", cs)
	}
	if cs := changesFor(t, p.changes, "s2"); len(cs) != 0 {
		t.Errorf("kept target changes = %+v, want none (policy inaction is not a diff)", cs)
	}

	doc := emitPayloadDoc(t, newResetTestRun, p, nil)
	sr := subjectResultOf(t, doc)
	if gen, ok := sr["generation"]; ok {
		t.Errorf("generation = %v, want the slot absent for reset", gen)
	}
}

// TestResetPayloadPartialFailure pins reset's reached-state partition: removed-so-far keeps
// its changes, the failing target carries the classified error, and the never-attempted rest
// is skipped (→ issue #131, niface ADR-0020).
func TestResetPayloadPartialFailure(t *testing.T) {
	res := &engine.ResetResult{
		Entries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/s1", Target: "s1", Method: "symlink"},
			{SrcKind: "store", Src: "/nix/store/c1", Target: "c1", Method: "copy"},
			{SrcKind: "store", Src: "/nix/store/c2", Target: "c2", Method: "copy"},
		},
		RemovedSymlinks: []string{"s1"},
		FailedTarget:    "c1",
		Unreached:       []string{"c2"},
	}
	cmdErr := &os.PathError{Op: "removeall", Path: "/root/c1", Err: fs.ErrPermission}
	p, err := resetPayload[*resetResultInfo](res, cmdErr)
	if err != nil {
		t.Fatalf("resetPayload: %v", err)
	}
	if !p.itemBorne {
		t.Error("itemBorne = false, want true")
	}
	if it := findItem(t, p.items, "s1"); it.Status != niface.ItemSuccess {
		t.Errorf("removed item status = %s, want success", it.Status)
	}
	failed := findItem(t, p.items, "c1")
	if failed.Status != niface.ItemFailed || failed.Error == nil || failed.Error.Code != "E_PERMISSION" {
		t.Errorf("failed item = %s / %+v, want failed + E_PERMISSION", failed.Status, failed.Error)
	}
	if it := findItem(t, p.items, "c2"); it.Status != niface.ItemSkipped {
		t.Errorf("unreached item status = %s, want skipped", it.Status)
	}
	if cs := changesFor(t, p.changes, "s1"); len(cs) != 1 {
		t.Errorf("removed-so-far changes = %+v, want the remove present despite the failure", cs)
	}
	doc := emitPayloadDoc(t, newResetTestRun, p, cmdErr)
	if doc["status"] != "error" {
		t.Errorf("status = %v, want error", doc["status"])
	}
	sr := subjectResultOf(t, doc)
	if sr["status"] != "error" {
		t.Errorf("subject status = %v, want error", sr["status"])
	}
	if errList, ok := sr["errors"]; ok {
		t.Errorf("subjectResult.errors = %v, want absent (the failure is item-borne)", errList)
	}
}

// --- end-to-end through the real engine on a tmpdir ---

// genCommit fakes nix-env --set with a real generation-link layout, so observeGeneration
// reads a numeric generation the same way it does under nix.
func genCommit(gen int) engine.CommitFunc {
	return func(profileLink, linkFarm string) error {
		genLink := profileLink + "-" + strconv.Itoa(gen) + "-link"
		_ = os.Remove(genLink)
		if err := os.Symlink(linkFarm, genLink); err != nil {
			return err
		}
		_ = os.Remove(profileLink)
		return os.Symlink(filepath.Base(genLink), profileLink)
	}
}

// writeTestLinkFarm writes a manifest.json-only link-farm for the engine's pre-built path.
func writeTestLinkFarm(t *testing.T, entries ...manifest.Entry) string {
	t.Helper()
	dir := t.TempDir()
	m := manifest.Manifest{
		SchemaVersion: 1,
		Root:          manifest.Root{RootKind: manifest.RootKindHome},
		Entries:       entries,
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, manifest.FileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestJSONEndToEndApplyAndResetPayload drives the real engine (tmpdir, injected commit) and
// checks the emitted envelopes: first apply (add changes, before omitted / after observed),
// second apply with a dropped entry (stale-removed old entry item + remove change, 1→2
// generation), then reset (remove changes, no generation slot). The payload reads the same
// engine result the -v report reads — this is the single-result-source path end to end
// (→ issue #131 acceptance).
func TestJSONEndToEndApplyAndResetPayload(t *testing.T) {
	root := t.TempDir()
	state := t.TempDir()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	keep := manifest.Entry{SrcKind: "store", Src: src, Subpath: "f", Target: ".keep", Method: "symlink"}
	drop := manifest.Entry{SrcKind: "store", Src: src, Subpath: "f", Target: ".drop", Method: "symlink"}
	apply := func(gen int, entries ...manifest.Entry) *engine.Result {
		t.Helper()
		res, err := engine.Apply(engine.Options{
			LinkFarm: writeTestLinkFarm(t, entries...), Name: "cfg",
			RootOverride: root, StateDir: state, Commit: genCommit(gen),
			Warnf: func(string, ...any) {},
		})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		return res
	}

	// First apply: two adds, no previous generation to observe.
	res := apply(1, keep, drop)
	p, err := mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	doc := emitPayloadDoc(t, newApplyTestRun, p, nil)
	sr := subjectResultOf(t, doc)
	gen := sr["generation"].(map[string]any)
	if _, hasBefore := gen["before"]; hasBefore || gen["after"] != json.Number("1") {
		t.Errorf("first apply generation = %v, want before omitted / after 1", gen)
	}
	if items := sr["result"].(map[string]any)["items"].([]any); len(items) != 2 {
		t.Errorf("items = %v, want both entries", items)
	}
	if changes := sr["result"].(map[string]any)["changes"].([]any); len(changes) != 2 {
		t.Errorf("changes = %v, want two adds", changes)
	}

	// Second apply drops .drop: its old entry is stale-removed and stays in the inventory.
	res = apply(2, keep)
	p, err = mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	doc = emitPayloadDoc(t, newApplyTestRun, p, nil)
	sr = subjectResultOf(t, doc)
	gen = sr["generation"].(map[string]any)
	if gen["before"] != json.Number("1") || gen["after"] != json.Number("2") {
		t.Errorf("second apply generation = %v, want 1→2", gen)
	}
	if len(p.items) != 2 {
		t.Fatalf("second apply items = %+v, want the kept entry + the stale-removed old entry", p.items)
	}
	if cs := changesFor(t, p.changes, ".drop"); len(cs) != 1 || cs[0].Kind != niface.ChangeRemove ||
		cs[0].Info == nil || cs[0].Info.Old != filepath.Join(src, "f") {
		t.Errorf(".drop changes = %+v, want one remove with the recorded dest", cs)
	}
	if cs := changesFor(t, p.changes, ".keep"); len(cs) != 0 {
		t.Errorf(".keep changes = %+v, want none (unchanged entry)", cs)
	}

	// Reset tears the remaining placement down: a reversible remove, no generation slot.
	resetRes, err := engine.Reset(engine.ResetOptions{
		Name: "cfg", RootKind: manifest.RootKindHome, RootOverride: root, StateDir: state,
		Warnf: func(string, ...any) {},
	})
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	rp, err := resetPayload[*resetResultInfo](resetRes, nil)
	if err != nil {
		t.Fatalf("resetPayload: %v", err)
	}
	doc = emitPayloadDoc(t, newResetTestRun, rp, nil)
	sr = subjectResultOf(t, doc)
	if gen, ok := sr["generation"]; ok {
		t.Errorf("reset generation = %v, want the slot absent", gen)
	}
	if cs := changesFor(t, rp.changes, ".keep"); len(cs) != 1 || cs[0].Kind != niface.ChangeRemove || !cs[0].Reversible {
		t.Errorf("reset changes = %+v, want one reversible remove for .keep", cs)
	}
}

// TestDryrunPayloadFirstPlanOmitsGenerationNumbers pins apply --dryrun over a not-yet-created
// profile (niface ADR-0015 · → issue #132): the dryrun rides mutationPayload, and with neither
// generation number observable the emitted generation carries the profile path alone — no
// before / after keys (nil pointers must marshal away, never as 0 or null).
func TestDryrunPayloadFirstPlanOmitsGenerationNumbers(t *testing.T) {
	res := &engine.Result{
		Profile: "/state/nix/profiles/nput/home/profile",
		DryRun:  true,
		Entries: []manifest.Entry{{SrcKind: "store", Src: "/nix/store/z", Target: ".zshrc", Method: "symlink"}},
		Placed:  []string{".zshrc"},
	}
	p, err := mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	doc := emitPayloadDoc(t, newApplyTestRun, p, nil)
	gen := subjectResultOf(t, doc)["generation"].(map[string]any)
	if gen["profile"] != res.Profile {
		t.Errorf("generation.profile = %v, want %s", gen["profile"], res.Profile)
	}
	for _, key := range []string{"before", "after"} {
		if v, ok := gen[key]; ok {
			t.Errorf("generation.%s present (= %v), want omitted before the first apply", key, v)
		}
	}
}

// TestDryrunPayloadConflictKeepsEnvelopeBesideExit2 pins the dryrun conflict contract
// (→ issue #132 acceptance): the CLI attaches the payload with cmdErr nil (the exit-2
// exitError is decided after the plan is printed), the conflicted entry is a failed item with
// E_NPUT_COLLISION, and the envelope emitted alongside exit 2 stays conformant with status
// error, dryRun true, and no subject-level duplication of the item-borne error.
func TestDryrunPayloadConflictKeepsEnvelopeBesideExit2(t *testing.T) {
	res := &engine.Result{
		Profile: "/state/nix/profiles/nput/home/profile",
		DryRun:  true,
		Entries: []manifest.Entry{
			{SrcKind: "store", Src: "/nix/store/a", Target: ".zshrc", Method: "symlink"},
			{SrcKind: "store", Src: "/nix/store/b", Target: ".config/nvim", Method: "symlink"},
		},
		Placed: []string{".config/nvim"},
		Conflicts: []planner.Conflict{
			{Entry: manifest.Entry{Target: ".zshrc"}, Reason: "target already has an existing file/directory (will not overwrite)", Kind: planner.ConflictForeignEntity},
		},
	}
	p, err := mutationPayload[*applyResultInfo](res, nil) // the dryrun wiring passes cmdErr nil (→ runApply)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	if !p.itemBorne {
		t.Error("itemBorne = false, want true (the conflict is fully represented by its item)")
	}
	if it := findItem(t, p.items, ".config/nvim"); it.Status != niface.ItemSuccess {
		t.Errorf("sibling item status = %s, want success (a dryrun attempts nothing)", it.Status)
	}

	r, buf := newApplyTestRun()
	r.dryRun = true
	r.beginSubject("default").setPayload(p)
	if err := r.emit(&exitError{code: 2}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	checker, err := conformance.NewDefaultChecker()
	if err != nil {
		t.Fatalf("conformance.NewDefaultChecker: %v", err)
	}
	if findings := checker.Check(buf.Bytes()); len(findings) > 0 {
		t.Fatalf("conformance findings:\n%s\ndocument: %s", strings.Join(findings, "\n"), buf.String())
	}
	doc := decodeEnvelope(t, buf)
	if doc["status"] != "error" || doc["dryRun"] != true {
		t.Errorf("status/dryRun = %v/%v, want error/true", doc["status"], doc["dryRun"])
	}
	sr := subjectResultOf(t, doc)
	if errList, ok := sr["errors"]; ok {
		t.Errorf("subjectResult.errors = %v, want absent (the failed item carries the collision)", errList)
	}
	item := findItem(t, mustDecodeItems(t, sr), ".zshrc")
	if item.Status != niface.ItemFailed || item.Error == nil || item.Error.Code != "E_NPUT_COLLISION" {
		t.Errorf("conflicted item = %+v, want failed with E_NPUT_COLLISION", item)
	}
}

// mustDecodeItems re-decodes a subjectResult's items into typed nputItem values so the typed
// helpers (findItem) work on emitted documents too.
func mustDecodeItems(t *testing.T, sr map[string]any) []nputItem {
	t.Helper()
	raw, err := json.Marshal(sr["result"].(map[string]any)["items"])
	if err != nil {
		t.Fatalf("re-marshal items: %v", err)
	}
	var items []nputItem
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	return items
}

// TestDryrunPayloadRelinkNotSuppressed pins the dryrun side of the noop rule (→ issue #132):
// a dryrun never executes the re-link, so no pre-relink dest is observed (ReplacedDests
// stays empty) and the planned re-link cannot be proven a noop — it stays a modify (mirroring
// the text plan's replace line), unlike the real apply's suppression
// (→ TestMutationPayloadNoopRelinkSuppressed).
func TestDryrunPayloadRelinkNotSuppressed(t *testing.T) {
	res := &engine.Result{
		Profile:  "/state/nix/profiles/nput/home/profile",
		DryRun:   true,
		Entries:  []manifest.Entry{{SrcKind: "store", Src: "/nix/store/same", Target: ".same", Method: "symlink"}},
		Replaced: []string{".same"},
	}
	p, err := mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	cs := changesFor(t, p.changes, ".same")
	if len(cs) != 1 || cs[0].Kind != niface.ChangeModify || !cs[0].Reversible {
		t.Fatalf(".same changes = %+v, want one reversible modify (unobserved old dest ⇒ not provably a noop)", cs)
	}
	if cs[0].Info == nil || cs[0].Info.New != "/nix/store/same" || cs[0].Info.Old != "" {
		t.Errorf("change info = %+v, want new only (the old dest is unobserved in a dryrun)", cs[0].Info)
	}
}

// TestMutationPayloadKeptStaleSubjectWarning pins the apply-side routing of the conservative
// keep warnings (→ issue #132 review follow-up): a kept-stale target (record mismatch / not a
// symlink) is neither in the new manifest nor in the removal plan, so no item exists for it —
// its W_NPUT_STALE_* warning lands on the subject, self-contained via detail.target.
func TestMutationPayloadKeptStaleSubjectWarning(t *testing.T) {
	res := &engine.Result{
		Profile: "/p",
		DryRun:  true,
		Entries: []manifest.Entry{{SrcKind: "store", Src: "/nix/store/a", Target: ".zshrc", Method: "symlink"}},
		Warnings: []planner.Warning{
			{Kind: planner.WarnStaleMismatch, Target: ".config/drifted"},
			{Kind: planner.WarnStaleNonSymlink, Target: ".config/solidified"},
		},
	}
	p, err := mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	want := []struct{ code, target string }{
		{"W_NPUT_STALE_MISMATCH", ".config/drifted"},
		{"W_NPUT_STALE_NON_SYMLINK", ".config/solidified"},
	}
	if len(p.warnings) != len(want) {
		t.Fatalf("subject warnings = %+v, want the %d kept-stale warnings", p.warnings, len(want))
	}
	for i, w := range want {
		if p.warnings[i].Code != w.code || p.warnings[i].Detail["target"] != w.target {
			t.Errorf("subject warnings[%d] = %+v, want %s on %s", i, p.warnings[i], w.code, w.target)
		}
	}
	if it := findItem(t, p.items, ".zshrc"); len(it.Warnings) != 0 {
		t.Errorf("inventory item warnings = %+v, want none (the stale warnings are subject-borne)", it.Warnings)
	}
}

// TestMutationPayloadConflictWithWarningStaysConformant pins the failed-item × warnings
// combination: an item that failed on a conflict can still carry an entry-borne warning, and
// the emitted envelope (error + warnings side by side on one item) stays conformant.
func TestMutationPayloadConflictWithWarningStaysConformant(t *testing.T) {
	res := &engine.Result{
		Profile: "/p",
		DryRun:  true,
		Entries: []manifest.Entry{{SrcKind: "store", Src: "/nix/store/a", Target: ".zshrc", Method: "symlink"}},
		Conflicts: []planner.Conflict{
			{Entry: manifest.Entry{Target: ".zshrc"}, Reason: "target already has an existing file", Kind: planner.ConflictForeignEntity},
		},
		Warnings: []planner.Warning{{Kind: planner.WarnForeignReplace, Target: ".zshrc"}},
	}
	p, err := mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	it := findItem(t, p.items, ".zshrc")
	if it.Status != niface.ItemFailed || it.Error == nil || len(it.Warnings) != 1 {
		t.Fatalf("item = %+v, want failed with both error and the entry-borne warning", it)
	}
	doc := emitPayloadDoc(t, newApplyTestRun, p, &exitError{code: 2})
	if doc["status"] != "error" {
		t.Errorf("status = %v, want error", doc["status"])
	}
}

// TestMutationPayloadCopyForeignItemWarning completes the warning kind × in/out-of-inventory
// table (→ issue #132 review follow-up): the place-once copy skip is the one copy-family
// warning whose entry stays in the manifest, so W_NPUT_COPY_FOREIGN rides on its item — not
// the subject.
func TestMutationPayloadCopyForeignItemWarning(t *testing.T) {
	res := &engine.Result{
		Profile:  "/p",
		DryRun:   true,
		Entries:  []manifest.Entry{{SrcKind: "store", Src: "/nix/store/c", Target: ".config/copydir", Method: "copy"}},
		Warnings: []planner.Warning{{Kind: planner.WarnCopyForeign, Target: ".config/copydir"}},
	}
	p, err := mutationPayload[*applyResultInfo](res, nil)
	if err != nil {
		t.Fatalf("mutationPayload: %v", err)
	}
	it := findItem(t, p.items, ".config/copydir")
	if len(it.Warnings) != 1 || it.Warnings[0].Code != "W_NPUT_COPY_FOREIGN" {
		t.Errorf("item warnings = %+v, want W_NPUT_COPY_FOREIGN on the inventory item", it.Warnings)
	}
	if len(p.warnings) != 0 {
		t.Errorf("subject warnings = %+v, want none", p.warnings)
	}
	if it.Status != niface.ItemSuccess {
		t.Errorf("item status = %s, want success (a skip is policy inaction, not a failure)", it.Status)
	}
}
