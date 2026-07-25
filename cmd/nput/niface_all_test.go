package main

// Conformance and shape tests for the --all paths' multiple SubjectResult output (→ issue #164
// acceptance). Every emitted document must pass niface's conformance checker under the same
// schema as a single-config run — the point of the shape being identical is that N=1 and N>1 are
// not different documents, only different lengths of results[].

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yasunori0418/niface/go/conformance"

	"github.com/yasunori0418/nput/internal/engine"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/paths"
	"github.com/yasunori0418/nput/internal/planner"
)

// checkConformance fails the test unless buf holds a document niface's checker accepts (schema
// with format assertions + the schema-external lint MUSTs).
func checkConformance(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	checker, err := conformance.NewDefaultChecker()
	if err != nil {
		t.Fatalf("conformance.NewDefaultChecker: %v", err)
	}
	if findings := checker.Check(buf.Bytes()); len(findings) > 0 {
		t.Fatalf("conformance findings:\n%s\ndocument: %s", strings.Join(findings, "\n"), buf.String())
	}
}

// subjectResults decodes the envelope and returns its results[] keyed by subject name, alongside
// the ordered names — the --all documents are asserted per subject, and a map keeps each
// assertion pointing at the config it means rather than at a positional index.
func subjectResults(t *testing.T, buf *bytes.Buffer) (map[string]map[string]any, []string) {
	t.Helper()
	doc := decodeEnvelope(t, buf)
	byName := map[string]map[string]any{}
	var order []string
	for _, r := range doc["results"].([]any) {
		sr := r.(map[string]any)
		name := sr["subject"].(map[string]any)["name"].(string)
		if _, dup := byName[name]; dup {
			t.Fatalf("subject %q appears twice in results[]", name)
		}
		byName[name] = sr
		order = append(order, name)
	}
	return byName, order
}

// statusAndErrors returns the SubjectResult's status and its (possibly absent) errors[].
func statusAndErrors(t *testing.T, sr map[string]any) (string, []any) {
	t.Helper()
	errs, _ := sr["errors"].([]any)
	return sr["status"].(string), errs
}

// placedResult is one config's successful apply: a single placed entry, so its SubjectResult
// carries a real inventory rather than an empty one (a succeeded config surviving a sibling's
// failure has to be observable as more than a bare status). Each config has its own profile —
// what makes --all N separate atomic runs rather than one (→ ADR-0002).
func placedResult(name string) *engine.Result {
	return &engine.Result{
		Profile: "/state/nix/profiles/nput/" + name + "/profile",
		Entries: []manifest.Entry{{Target: "t/" + name}},
		Placed:  []string{"t/" + name},
	}
}

// TestApplyAllPartialFailureKeepsEverySubject is issue #164's first acceptance criterion: with one
// config failing among several, the envelope aggregates to error while still carrying the
// SubjectResult of every config that succeeded — a consumer must not lose the successes because a
// sibling failed. The failure stays on its own subject, and the top-level errors[] stays absent:
// once subjects exist, the top level is reserved for failures that precede their enumeration
// (→ ADR-0043 §6).
func TestApplyAllPartialFailureKeepsEverySubject(t *testing.T) {
	run, buf := newApplyTestRun()
	applied, skipped, failures := aggregateApply(run, []string{"a", "b", "c"}, func(name string) (*engine.Result, error) {
		if name == "b" {
			return nil, io.ErrUnexpectedEOF
		}
		return placedResult(name), nil
	})
	if applied != 2 || skipped != 0 || failures != 1 {
		t.Fatalf("counts = (%d, %d, %d), want (2, 0, 1)", applied, skipped, failures)
	}
	// main emits with the aggregate error the exit-code path builds from the same counts.
	cmdErr := &exitCodeError{code: applyAllExitCode(failures > 0, false), msg: "nput: apply --all: 1 config(s) failed"}
	if err := run.emit(cmdErr); err != nil {
		t.Fatalf("emit: %v", err)
	}
	checkConformance(t, buf)

	doc := decodeEnvelope(t, buf)
	if doc["status"] != "error" {
		t.Errorf("aggregate status = %v, want error (one subject failed)", doc["status"])
	}
	if topErrs, ok := doc["errors"]; ok {
		t.Errorf("top-level errors = %v, want absent — the failure belongs to subject b", topErrs)
	}
	if cmdErr.ExitCode() == 0 {
		t.Error("exit code = 0, want non-zero alongside the error status")
	}

	byName, order := subjectResults(t, buf)
	if want := []string{"a", "b", "c"}; !slices.Equal(order, want) {
		t.Errorf("results order = %v, want %v (selection order)", order, want)
	}
	for _, name := range []string{"a", "c"} {
		status, errs := statusAndErrors(t, byName[name])
		if status != "success" {
			t.Errorf("subject %s status = %s, want success (its own apply succeeded)", name, status)
		}
		if len(errs) != 0 {
			t.Errorf("subject %s errors = %v, want none — b's failure must not colour it", name, errs)
		}
		items := byName[name]["result"].(map[string]any)["items"].([]any)
		if len(items) != 1 {
			t.Errorf("subject %s items = %v, want the placed entry (the succeeded result is kept whole)", name, items)
		}
	}
	status, errs := statusAndErrors(t, byName["b"])
	if status != "error" {
		t.Errorf("subject b status = %s, want error", status)
	}
	if len(errs) != 1 {
		t.Fatalf("subject b errors = %v, want exactly one (subject-borne: no engine result exists)", errs)
	}
}

// TestApplyAllPartialFailureItemBorne is the acceptance criterion above on the path production
// actually takes: engine.Apply returns its partial result alongside the error (→ engine.apply's
// "return a.result, err"), so the failing config's subject gets a payload whose failed item already
// carries the error. That makes the failure item-borne, and its SubjectResult.errors[] must stay
// empty while the status is still error (niface §2 / ADR-0002). The res == nil variant above only
// covers the failures that precede any engine result (a pre-flight eval / lock rejection).
func TestApplyAllPartialFailureItemBorne(t *testing.T) {
	run, buf := newApplyTestRun()
	failure := errors.New("nput: symlink t/b: permission denied")
	_, _, failures := aggregateApply(run, []string{"a", "b", "c"}, func(name string) (*engine.Result, error) {
		if name == "b" {
			// The engine's partial result: the entry it stopped on, plus the planned entry it
			// never reached (→ niface ADR-0016's reached-state partition).
			res := placedResult(name)
			res.Placed = nil
			res.Entries = append(res.Entries, manifest.Entry{Target: "t/" + name + "-later"})
			res.FailedTarget = "t/" + name
			res.Unreached = []string{"t/" + name + "-later"}
			return res, failure
		}
		return placedResult(name), nil
	})
	if failures != 1 {
		t.Fatalf("failures = %d, want 1", failures)
	}
	if err := run.emit(&exitCodeError{code: 1, msg: "nput: apply --all: 1 config(s) failed"}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	checkConformance(t, buf)

	doc := decodeEnvelope(t, buf)
	if doc["status"] != "error" {
		t.Errorf("aggregate status = %v, want error", doc["status"])
	}
	if topErrs, ok := doc["errors"]; ok {
		t.Errorf("top-level errors = %v, want absent", topErrs)
	}
	byName, _ := subjectResults(t, buf)
	status, errs := statusAndErrors(t, byName["b"])
	if status != "error" {
		t.Errorf("failing subject status = %s, want error (its failed item makes it error)", status)
	}
	if len(errs) != 0 {
		t.Errorf("failing subject errors = %v, want none — the failure is item-borne (niface §2)", errs)
	}
	// The reached-state partition has to survive onto the items, not just the status.
	byTarget := map[string]map[string]any{}
	for _, it := range byName["b"]["result"].(map[string]any)["items"].([]any) {
		item := it.(map[string]any)
		byTarget[item["label"].(string)] = item
	}
	failed := byTarget["t/b"]
	if failed == nil || failed["status"] != "failed" {
		t.Fatalf("item t/b = %v, want status failed", failed)
	}
	if itemErr, ok := failed["error"].(map[string]any); !ok || itemErr["message"] != failure.Error() {
		t.Errorf("item t/b error = %v, want the config's own failure carried on the item", failed["error"])
	}
	if unreached := byTarget["t/b-later"]; unreached == nil || unreached["status"] != "skipped" {
		t.Errorf("item t/b-later = %v, want status skipped (planned but never attempted)", unreached)
	}
	// The sibling configs still carry their own successful inventories.
	for _, name := range []string{"a", "c"} {
		if s, e := statusAndErrors(t, byName[name]); s != "success" || len(e) != 0 {
			t.Errorf("subject %s = (%s, %v), want (success, none)", name, s, e)
		}
	}
}

// TestApplyAllSkipIsNotAFailure pins the try-lock skip's asymmetry with a failure: ErrSkipped is a
// normal skip (exit 0 for a named apply), so its subject succeeds and the aggregate stays success.
// Without this the skip would be indistinguishable from a failure in the envelope while the exit
// code says otherwise (→ docs/spec.md exit code table).
func TestApplyAllSkipIsNotAFailure(t *testing.T) {
	run, buf := newApplyTestRun()
	applied, skipped, failures := aggregateApply(run, []string{"a", "b"}, func(name string) (*engine.Result, error) {
		if name == "b" {
			// The engine returns its result alongside ErrSkipped (→ engine.apply's try-lock
			// arm sets Skipped and returns a.result), so the payload gets attached here too:
			// the skipped config's subject must still come out success, payload and all.
			res := placedResult(name)
			res.Placed = nil
			res.Skipped = true
			return res, engine.ErrSkipped
		}
		return placedResult(name), nil
	})
	if applied != 1 || skipped != 1 || failures != 0 {
		t.Fatalf("counts = (%d, %d, %d), want (1, 1, 0)", applied, skipped, failures)
	}
	if err := run.emit(nil); err != nil {
		t.Fatalf("emit: %v", err)
	}
	checkConformance(t, buf)

	if got := decodeEnvelope(t, buf)["status"]; got != "success" {
		t.Errorf("aggregate status = %v, want success (a skip is not a failure)", got)
	}
	byName, _ := subjectResults(t, buf)
	if status, errs := statusAndErrors(t, byName["b"]); status != "success" || len(errs) != 0 {
		t.Errorf("skipped subject = (%s, %v), want (success, none)", status, errs)
	}
}

// TestApplyAllEmptySelectionEmitsEmptyResults is issue #164's second acceptance criterion: an
// --all that matches no config emits results: [] with status success — the same shape, at N=0.
// Consumers may only rely on results[] always being present, so an empty selection must not
// degrade into an absent key or an error.
func TestApplyAllEmptySelectionEmitsEmptyResults(t *testing.T) {
	run, buf := newApplyTestRun()
	if applied, skipped, failures := aggregateApply(run, nil, func(string) (*engine.Result, error) {
		t.Fatal("apply must not run for an empty selection")
		return nil, nil
	}); applied+skipped+failures != 0 {
		t.Fatalf("counts = (%d, %d, %d), want all zero", applied, skipped, failures)
	}
	if err := run.emit(nil); err != nil {
		t.Fatalf("emit: %v", err)
	}
	checkConformance(t, buf)

	doc := decodeEnvelope(t, buf)
	if doc["status"] != "success" {
		t.Errorf("status = %v, want success (nothing selected is not a failure)", doc["status"])
	}
	results, ok := doc["results"].([]any)
	if !ok {
		t.Fatalf("results = %v, want the key present as an array", doc["results"])
	}
	if len(results) != 0 {
		t.Errorf("results = %v, want []", results)
	}
}

// TestApplyAllDryRunConflictIsItemBorne is issue #164's third acceptance criterion and the
// symmetry requirement against the named apply --dryrun: a conflicting config's entry becomes a
// failed item carrying E_NPUT_COLLISION (item-borne, so it must NOT be repeated in that
// SubjectResult's errors[] · niface §2), the subject's status is error (niface ADR-0002: a failed
// item makes the result error), the aggregate is error, and the exit code stays what
// applyAllExitCode decides — conflict 2, not the error 1 (→ nput ADR-0043 §6, ADR-0024).
func TestApplyAllDryRunConflictIsItemBorne(t *testing.T) {
	run, buf := newApplyTestRun()
	run.dryRun = true
	var code int
	captureStdout(t, func() {
		code = aggregateDryRun(run, []string{"a", "b"}, func(name string) (*engine.Result, error) {
			if name == "b" {
				conflicted := placedResult(name)
				conflicted.Placed = nil
				conflicted.Conflicts = []planner.Conflict{
					{Entry: manifest.Entry{Target: "t/" + name}, Reason: "occupied by a foreign entity"},
				}
				return conflicted, nil
			}
			return placedResult(name), nil
		})
	})
	if want := applyAllExitCode(false, true); code != want {
		t.Fatalf("aggregateDryRun code = %d, want %d (conflict, no error)", code, want)
	}
	// main turns that code into the exitError it exits with; emit sees it as the command error.
	if err := run.emit(&exitError{code: code}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	checkConformance(t, buf)

	doc := decodeEnvelope(t, buf)
	if doc["status"] != "error" {
		t.Errorf("aggregate status = %v, want error (a conflicting config is in error)", doc["status"])
	}
	if doc["dryRun"] != true {
		t.Errorf("dryRun = %v, want true", doc["dryRun"])
	}
	if topErrs, ok := doc["errors"]; ok {
		t.Errorf("top-level errors = %v, want absent (the conflict belongs to subject b's item)", topErrs)
	}

	byName, _ := subjectResults(t, buf)
	if status, errs := statusAndErrors(t, byName["a"]); status != "success" || len(errs) != 0 {
		t.Errorf("clean subject = (%s, %v), want (success, none)", status, errs)
	}
	status, errs := statusAndErrors(t, byName["b"])
	if status != "error" {
		t.Errorf("conflicting subject status = %s, want error", status)
	}
	if len(errs) != 0 {
		t.Errorf("conflicting subject errors = %v, want none — the conflict is item-borne (niface §2)", errs)
	}
	items := byName["b"]["result"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("conflicting subject items = %v, want the conflicting entry", items)
	}
	item := items[0].(map[string]any)
	if item["status"] != "failed" {
		t.Errorf("conflicting item status = %v, want failed", item["status"])
	}
	itemErr, ok := item["error"].(map[string]any)
	if !ok {
		t.Fatalf("conflicting item = %v, want an error object", item)
	}
	if itemErr["code"] != "E_NPUT_COLLISION" {
		t.Errorf("conflicting item error code = %v, want E_NPUT_COLLISION", itemErr["code"])
	}
}

// TestApplyAllDryRunMixedErrorAndConflict pins the two error layers side by side in one document,
// which is the combination the exit-code priority exists for: a config that failed outright carries
// a subject-level error, a config that merely conflicts carries none (its item does), and the exit
// code is error(1) — not the conflict's 2, which must never mask a real failure (→ ADR-0024).
func TestApplyAllDryRunMixedErrorAndConflict(t *testing.T) {
	run, buf := newApplyTestRun()
	run.dryRun = true
	buildFailure := errors.New("nput: nix build failed")
	var code int
	captureStdout(t, func() {
		code = aggregateDryRun(run, []string{"broken", "clashing", "clean"}, func(name string) (*engine.Result, error) {
			switch name {
			case "broken":
				return nil, buildFailure
			case "clashing":
				res := placedResult(name)
				res.Placed = nil
				res.Conflicts = []planner.Conflict{
					{Entry: manifest.Entry{Target: "t/" + name}, Reason: "occupied by a foreign entity"},
				}
				return res, nil
			}
			return placedResult(name), nil
		})
	})
	if want := applyAllExitCode(true, true); code != want {
		t.Fatalf("code = %d, want %d (error must win over conflict)", code, want)
	}
	if err := run.emit(&exitError{code: code}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	checkConformance(t, buf)

	doc := decodeEnvelope(t, buf)
	if doc["status"] != "error" {
		t.Errorf("aggregate status = %v, want error", doc["status"])
	}
	if topErrs, ok := doc["errors"]; ok {
		t.Errorf("top-level errors = %v, want absent (both failures belong to their configs)", topErrs)
	}
	byName, _ := subjectResults(t, buf)
	// The outright failure is subject-borne: no engine result exists, so nothing else can carry it.
	status, errs := statusAndErrors(t, byName["broken"])
	if status != "error" || len(errs) != 1 {
		t.Fatalf("failed subject = (%s, %v), want (error, exactly one subject-borne error)", status, errs)
	}
	if got := errs[0].(map[string]any)["code"]; got != "E_NPUT_FAILED" {
		t.Errorf("failed subject error code = %v, want E_NPUT_FAILED (the build error is unclassified here)", got)
	}
	// The conflict is item-borne: same error status, but errors[] stays empty (niface §2).
	status, errs = statusAndErrors(t, byName["clashing"])
	if status != "error" {
		t.Errorf("conflicting subject status = %s, want error", status)
	}
	if len(errs) != 0 {
		t.Errorf("conflicting subject errors = %v, want none — its item carries the conflict", errs)
	}
	if status, errs := statusAndErrors(t, byName["clean"]); status != "success" || len(errs) != 0 {
		t.Errorf("clean subject = (%s, %v), want (success, none)", status, errs)
	}
}

// TestApplyAllSingleConfigMatchesNamedApply is issue #164's fourth acceptance criterion, pinned as
// an equality rather than a description: a one-config --all and a named apply of the same config
// must emit byte-identical documents once the clocks agree. Any drift the append-based
// generalization could introduce (an extra results entry, a different error layer, a status the
// subject no longer decides) shows up here as a diff, which is what "single-config output is
// unchanged" actually means.
func TestApplyAllSingleConfigMatchesNamedApply(t *testing.T) {
	res := placedResult("default")

	// The named path: one subject, the payload attached, the command error carried by emit.
	named, namedBuf := newApplyTestRun()
	attachMutationPayload(named.beginSubject("default"), res, nil)
	if err := named.emit(nil); err != nil {
		t.Fatalf("emit named: %v", err)
	}

	// The --all path at N=1: the same subject, settled by the aggregator itself.
	all, allBuf := newApplyTestRun()
	aggregateApply(all, []string{"default"}, func(string) (*engine.Result, error) { return res, nil })
	if err := all.emit(nil); err != nil {
		t.Fatalf("emit all: %v", err)
	}

	if allBuf.String() != namedBuf.String() {
		t.Errorf("apply --all at N=1 diverged from the named apply\n--all: %s\nnamed: %s", allBuf, namedBuf)
	}
	checkConformance(t, allBuf)
}

// TestApplyAllSingleConfigFailureMatchesNamedApply is the equality above on the failing side, which
// is where the two paths actually diverge in mechanism: the named apply settles nothing and lets
// emit attribute the command error to its one subject, while --all settles the subject itself with
// that config's error and hands emit an aggregate error it must not re-apply. Both must still land
// on the same document — the invariant behind emit's first-wins finish (→ issue #164).
func TestApplyAllSingleConfigFailureMatchesNamedApply(t *testing.T) {
	failure := errors.New("nput: generation commit (nix-env --set) failed")
	// A commit failure: the engine returns a result but no failed target, so it is subject-borne
	// (not item-borne) and has to appear in the SubjectResult's errors[] on both paths.
	newRes := func() *engine.Result {
		res := placedResult("default")
		res.Placed = nil
		return res
	}

	named, namedBuf := newApplyTestRun()
	attachMutationPayload(named.beginSubject("default"), newRes(), failure)
	if err := named.emit(failure); err != nil {
		t.Fatalf("emit named: %v", err)
	}

	all, allBuf := newApplyTestRun()
	aggregateApply(all, []string{"default"}, func(string) (*engine.Result, error) { return newRes(), failure })
	// --all reports its own aggregate error, whose text differs from the config's; the subject is
	// already settled, so this must not reach it.
	if err := all.emit(&exitCodeError{code: 1, msg: "nput: apply --all: 1 config(s) failed"}); err != nil {
		t.Fatalf("emit all: %v", err)
	}

	if allBuf.String() != namedBuf.String() {
		t.Errorf("failing apply --all at N=1 diverged from the named apply\n--all: %s\nnamed: %s", allBuf, namedBuf)
	}
	checkConformance(t, allBuf)
	// Guard the equality against being satisfied by both sides losing the error.
	byName, _ := subjectResults(t, allBuf)
	if status, errs := statusAndErrors(t, byName["default"]); status != "error" || len(errs) != 1 {
		t.Fatalf("subject = (%s, %v), want (error, exactly one subject-borne error)", status, errs)
	}
}

// stubNixEnvListGenerations puts a fake nix-env first on PATH: it prints one generation line for
// every profile except failFor, for which it exits non-zero. That makes runListAllGenerations
// drivable in-process — the real one shells out to nix-env — so the subject wiring (register per
// config, attach that config's listing, settle it) is exercised through production code rather
// than restated by the test.
func stubNixEnvListGenerations(t *testing.T, failFor string) {
	t.Helper()
	bin := t.TempDir()
	script := "#!/bin/sh\n" +
		"for a in \"$@\"; do case \"$prev\" in --profile) prof=$a;; esac; prev=$a; done\n" +
		"case \"$prof\" in\n" +
		"*/" + failFor + "/profile) echo 'error: not a valid profile' >&2; exit 1;;\n" +
		"esac\n" +
		"printf '   1   2026-07-19 12:00:00   (current)\\n'\n"
	if err := os.WriteFile(filepath.Join(bin, "nix-env"), []byte(script), 0o755); err != nil {
		t.Fatalf("write nix-env stub: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// makeHomeProfiles creates the on-disk shape runListAllGenerations scans for: a <name> directory
// holding a "profile" link directly under <state>/nix/profiles/nput (the home-mode layout; the
// roothash family nests one level deeper and is skipped · → paths.Resolve, ADR-0024).
func makeHomeProfiles(t *testing.T, names ...string) {
	t.Helper()
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	for _, name := range names {
		dir := filepath.Join(paths.Base(state), name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		// The scan only lstats the link, so the destination need not exist.
		if err := os.Symlink("profile-1-link", filepath.Join(dir, "profile")); err != nil {
			t.Fatalf("symlink profile: %v", err)
		}
	}
}

// TestListGenerationsAllWiring drives the real runListAllGenerations, which is what actually
// registers and settles one subject per scanned config (→ issue #164). The hand-built variant below
// pins the document shape; this pins the wiring, so a forgotten beginSubject / setPayload / finish
// cannot pass by having the test restate what production should have done.
func TestListGenerationsAllWiring(t *testing.T) {
	origJSON := flagJSON
	defer func() { flagJSON = origJSON }()
	flagJSON = true

	t.Run("every scanned config becomes its own SubjectResult", func(t *testing.T) {
		makeHomeProfiles(t, "home", "work")
		stubNixEnvListGenerations(t, "")
		run, buf := newListGenerationsTestRun()
		if err := runListAllGenerations(run); err != nil {
			t.Fatalf("runListAllGenerations: %v", err)
		}
		if err := run.emit(nil); err != nil {
			t.Fatalf("emit: %v", err)
		}
		checkConformance(t, buf)

		byName, order := subjectResults(t, buf)
		want := []string{"home", "work"}
		if !slices.Equal(order, want) {
			t.Fatalf("results order = %v, want %v (lexical scan order)", order, want)
		}
		for _, name := range want {
			if status, errs := statusAndErrors(t, byName[name]); status != "success" || len(errs) != 0 {
				t.Errorf("subject %s = (%s, %v), want (success, none)", name, status, errs)
			}
			if got := infoArray(t, byName[name], "generations"); len(got) != 1 {
				t.Errorf("subject %s generations = %v, want its own listing", name, got)
			}
		}
	})

	t.Run("a mid-enumeration failure lands on its own subject", func(t *testing.T) {
		// "work" sorts after "home", so the scan lists home, then fails on work and stops.
		makeHomeProfiles(t, "home", "work", "zzz")
		stubNixEnvListGenerations(t, "work")
		run, buf := newListGenerationsTestRun()
		err := runListAllGenerations(run)
		if err == nil {
			t.Fatal("runListAllGenerations must fail when a profile cannot be listed")
		}
		if err := run.emit(err); err != nil {
			t.Fatalf("emit: %v", err)
		}
		checkConformance(t, buf)

		doc := decodeEnvelope(t, buf)
		if doc["status"] != "error" {
			t.Errorf("aggregate status = %v, want error", doc["status"])
		}
		// The failure belongs to the config it happened on, not to the top level — subjects exist,
		// so the top-level layer (pre-enumeration failures only) must stay empty (→ ADR-0043 §6).
		if topErrs, ok := doc["errors"]; ok {
			t.Errorf("top-level errors = %v, want absent (the failure belongs to subject work)", topErrs)
		}
		byName, order := subjectResults(t, buf)
		// The scan stops at the failure, so the untouched config never becomes a subject at all.
		if want := []string{"home", "work"}; !slices.Equal(order, want) {
			t.Fatalf("results = %v, want %v — the scan stops at the failure and zzz is never reached", order, want)
		}
		if status, errs := statusAndErrors(t, byName["home"]); status != "success" || len(errs) != 0 {
			t.Errorf("already-listed subject = (%s, %v), want (success, none) — it keeps its result", status, errs)
		}
		if status, errs := statusAndErrors(t, byName["work"]); status != "error" || len(errs) != 1 {
			t.Errorf("failing subject = (%s, %v), want (error, exactly one subject-borne error)", status, errs)
		}
	})
}

// TestReadAllCommandsEmitPerConfigResults covers the read-only --all paths' payload wiring
// (list-generations / gitignore): each config's own inventory rides its own result.info, so a
// consumer can tell which config declares what. gitignore --all deliberately does NOT dedup
// across configs in JSON — a path shared by two configs appears under both, because attributing
// it to one of them would be a lie about which config declares it (the text contract keeps its
// dedup+sort union · → ADR-0018, issue #164).
func TestReadAllCommandsEmitPerConfigResults(t *testing.T) {
	t.Run("list-generations", func(t *testing.T) {
		run, buf := newListGenerationsTestRun()
		run.beginSubject("home").setPayload(generationsPayload([]engine.Generation{
			{Number: 1, Date: "2026-07-19 12:00:00", Current: true},
		}))
		run.beginSubject("work").setPayload(generationsPayload(nil))
		if err := run.emit(nil); err != nil {
			t.Fatalf("emit: %v", err)
		}
		checkConformance(t, buf)

		byName, _ := subjectResults(t, buf)
		if got := infoArray(t, byName["home"], "generations"); len(got) != 1 {
			t.Errorf("home generations = %v, want its own single generation", got)
		}
		// An empty profile still lists the key as an array, exactly as the named listing does.
		if got := infoArray(t, byName["work"], "generations"); len(got) != 0 {
			t.Errorf("work generations = %v, want []", got)
		}
	})

	t.Run("gitignore keeps per-config paths undeduped", func(t *testing.T) {
		shared := ".nput-out/shared"
		run, buf := newGitignoreTestRun()
		run.beginSubject("docs").setPayload(gitignorePayload([]string{".claude/skills/nix", shared}))
		run.beginSubject("web").setPayload(gitignorePayload([]string{shared}))
		if err := run.emit(nil); err != nil {
			t.Fatalf("emit: %v", err)
		}
		checkConformance(t, buf)

		byName, _ := subjectResults(t, buf)
		docs := infoArray(t, byName["docs"], "paths")
		if len(docs) != 2 {
			t.Errorf("docs paths = %v, want both of its own targets", docs)
		}
		web := infoArray(t, byName["web"], "paths")
		if len(web) != 1 || web[0] != gitignoreAnchor(shared) {
			t.Errorf("web paths = %v, want the shared path kept under this config too (no cross-config dedup)", web)
		}
	})
}

// infoArray reads an array out of one SubjectResult's result.info (the read commands' inventory
// slot), failing the test if the key is missing or not an array. The key must be present even
// when empty — the read commands build non-nil slices precisely so it never marshals away.
func infoArray(t *testing.T, sr map[string]any, key string) []any {
	t.Helper()
	info, ok := sr["result"].(map[string]any)["info"].(map[string]any)
	if !ok {
		t.Fatalf("result.info = %v, want an object holding %s", sr["result"], key)
	}
	arr, ok := info[key].([]any)
	if !ok {
		t.Fatalf("info.%s = %v, want the key present as an array", key, info[key])
	}
	return arr
}
