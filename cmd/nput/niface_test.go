package main

// Conformance tests for the --json niface envelope foundation (→ issue #130 acceptance):
// every emitted document passes niface's conformance checker (embedded schema with format
// assertions + the schema-external lint MUSTs), and the item-id derivation matches niface's
// id-vectors byte-for-byte (decoded with UseNumber, per the niface godoc input contract).

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	niface "github.com/yasunori0418/niface/go"
	"github.com/yasunori0418/niface/go/conformance"

	"github.com/yasunori0418/nput/internal/engine"
	"github.com/yasunori0418/nput/internal/lock"
)

// fixedClock returns a clock that yields t0 and advances one second per call, so
// startedAt / finishedAt are deterministic yet distinct.
func fixedClock(t0 time.Time) func() time.Time {
	n := 0
	return func() time.Time {
		t := t0.Add(time.Duration(n) * time.Second)
		n++
		return t
	}
}

// newTestRun returns a nifaceRun with a pinned clock and buffer sink, already begun for command.
// The info type arguments are the command's own pair (→ issue #196); callers spell them out
// because the command name is a plain string and cannot drive inference.
func newTestRun[TInfo, TEnvInfo any](command string) (*nifaceRun[TInfo, TEnvInfo], *bytes.Buffer) {
	var buf bytes.Buffer
	r := &nifaceRun[TInfo, TEnvInfo]{
		now: fixedClock(time.Date(2026, 7, 19, 12, 0, 0, 0, time.FixedZone("JST", 9*3600))),
		out: &buf,
	}
	r.begin(command)
	return r, &buf
}

// decodeEnvelope asserts buf holds exactly one JSON document with a trailing newline and
// returns it decoded (UseNumber, so nothing degrades to float64).
func decodeEnvelope(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	s := buf.String()
	if !strings.HasSuffix(s, "\n") {
		t.Fatalf("envelope must end with a newline, got %q", s)
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		t.Fatalf("stdout must hold exactly one JSON document, found a second: %v", extra)
	}
	return doc
}

// TestNifaceEnvelopeConformance drives the emit helper through the #130 shapes — success with /
// without a subject, pre-subject and subject-borne failures, dryrun conflict — and checks every
// document against niface's conformance checker (schema + lint MUSTs · issue #130 acceptance).
func TestNifaceEnvelopeConformance(t *testing.T) {
	checker, err := conformance.NewDefaultChecker()
	if err != nil {
		t.Fatalf("conformance.NewDefaultChecker: %v", err)
	}

	cases := []struct {
		name        string
		subject     string // "" = no subject registered
		cmdErr      error
		wantStatus  string
		wantResults int
		wantTopErrs bool // error attached at the top level (vs the subject)
		wantCode    string
	}{
		{name: "success with subject", subject: "default", wantStatus: "success", wantResults: 1},
		{name: "success without subject", wantStatus: "success", wantResults: 0},
		{name: "pre-subject failure", cmdErr: errors.New("nput: no entrypoint found"),
			wantStatus: "error", wantResults: 0, wantTopErrs: true, wantCode: "E_NPUT_FAILED"},
		{name: "subject-borne failure", subject: "web", cmdErr: errors.New("nput: build failed"),
			wantStatus: "error", wantResults: 1, wantCode: "E_NPUT_FAILED"},
		{name: "lock failure", subject: "web", cmdErr: lock.ErrLocked,
			wantStatus: "error", wantResults: 1, wantCode: "E_LOCK"},
		{name: "dryrun conflict", subject: "default", cmdErr: &exitError{code: 2},
			wantStatus: "error", wantResults: 1, wantCode: "E_NPUT_COLLISION"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, buf := newTestRun[*applyResultInfo, *applyEnvInfo]("apply")
			if c.subject != "" {
				r.beginSubject(c.subject)
			}
			if err := r.emit(c.cmdErr); err != nil {
				t.Fatalf("emit: %v", err)
			}

			if findings := checker.Check(buf.Bytes()); len(findings) > 0 {
				t.Fatalf("conformance findings:\n%s\ndocument: %s", strings.Join(findings, "\n"), buf.String())
			}

			doc := decodeEnvelope(t, buf)
			if doc["status"] != c.wantStatus {
				t.Errorf("status = %v, want %v", doc["status"], c.wantStatus)
			}
			results := doc["results"].([]any)
			if len(results) != c.wantResults {
				t.Fatalf("results length = %d, want %d", len(results), c.wantResults)
			}
			tool := doc["tool"].(map[string]any)
			if tool["name"] != "nput" || tool["version"] != version {
				t.Errorf("tool = %v, want name=nput version=%s (main.version)", tool, version)
			}
			if doc["command"] != "apply" {
				t.Errorf("command = %v, want apply", doc["command"])
			}

			if c.subject != "" {
				sr := results[0].(map[string]any)
				if sr["subject"].(map[string]any)["name"] != c.subject {
					t.Errorf("subject = %v, want %s", sr["subject"], c.subject)
				}
				if sr["status"] != c.wantStatus {
					t.Errorf("subject status = %v, want %v", sr["status"], c.wantStatus)
				}
				// The minimal #130 SubjectResult carries an empty items array (payloads are #131/#132).
				items := sr["result"].(map[string]any)["items"].([]any)
				if len(items) != 0 {
					t.Errorf("items = %v, want empty in the #130 minimal envelope", items)
				}
			}

			if c.cmdErr != nil {
				var errList []any
				if c.wantTopErrs {
					errList, _ = doc["errors"].([]any)
				} else {
					errList, _ = results[0].(map[string]any)["errors"].([]any)
				}
				if len(errList) != 1 {
					t.Fatalf("error layer mismatch (wantTopErrs=%v): %v", c.wantTopErrs, doc)
				}
				e := errList[0].(map[string]any)
				if e["code"] != c.wantCode {
					t.Errorf("error code = %v, want %s", e["code"], c.wantCode)
				}
				if e["message"] == "" {
					t.Error("error message must not be empty")
				}
			}
		})
	}
}

// TestNifaceEnvelopeDryRunFlag pins the envelope's dryRun field to the run's captured --dryrun
// value (begin snapshots flagDryrun; tests set the field directly).
func TestNifaceEnvelopeDryRunFlag(t *testing.T) {
	checker, err := conformance.NewDefaultChecker()
	if err != nil {
		t.Fatalf("conformance.NewDefaultChecker: %v", err)
	}
	for _, dryRun := range []bool{false, true} {
		r, buf := newTestRun[*applyResultInfo, *applyEnvInfo]("apply")
		r.dryRun = dryRun
		r.beginSubject("default")
		if err := r.emit(nil); err != nil {
			t.Fatalf("emit: %v", err)
		}
		if findings := checker.Check(buf.Bytes()); len(findings) > 0 {
			t.Fatalf("conformance findings: %v", findings)
		}
		doc := decodeEnvelope(t, buf)
		if doc["dryRun"] != dryRun {
			t.Errorf("dryRun = %v, want %v", doc["dryRun"], dryRun)
		}
	}
}

// failingWriter simulates a broken stdout (EPIPE etc.) for the emit write-failure path.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

// TestNifaceEmitWriteFailure pins that a failed envelope write surfaces as an error from emit —
// main then exits non-zero even for a succeeded command, so a missing/partial document is never
// read as success (→ docs/spec.md emit タイミングと成立条件).
func TestNifaceEmitWriteFailure(t *testing.T) {
	r := &applyRun{now: fixedClock(time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)), out: failingWriter{}}
	r.begin("apply")
	r.beginSubject("default")
	if err := r.emit(nil); err == nil {
		t.Fatal("emit must propagate the writer failure")
	}
}

// TestNifaceTimestampOffset pins the timestamp shape: RFC 3339, "T" separator, explicit offset
// (niface ADR-0025 format assertion; local zone renders its UTC offset, UTC renders "Z").
func TestNifaceTimestampOffset(t *testing.T) {
	jst := time.Date(2026, 7, 19, 12, 0, 0, 0, time.FixedZone("JST", 9*3600))
	if got, want := nifaceTimestamp(jst), "2026-07-19T12:00:00+09:00"; got != want {
		t.Errorf("nifaceTimestamp(JST) = %q, want %q", got, want)
	}
	utc := time.Date(2026, 7, 19, 3, 0, 0, 0, time.UTC)
	if got, want := nifaceTimestamp(utc), "2026-07-19T03:00:00Z"; got != want {
		t.Errorf("nifaceTimestamp(UTC) = %q, want %q", got, want)
	}
}

// TestEntryItemIDMatchesVectors verifies the id derivation against niface's embedded id-vectors
// (decoded with UseNumber — the niface godoc input contract; issue #130 acceptance): every vector
// must reproduce its expected id through niface.DeriveID, and the entry-kind vectors must equally
// reproduce through nput's entryItemID seam (pinning nput's identity shape: kind="entry",
// key={target} · → ADR-0043 §3).
func TestEntryItemIDMatchesVectors(t *testing.T) {
	var doc struct {
		Vectors []struct {
			Identity struct {
				Kind string          `json:"kind"`
				Key  json.RawMessage `json:"key"`
			} `json:"identity"`
			Expected string `json:"expected"`
		} `json:"vectors"`
	}
	dec := json.NewDecoder(bytes.NewReader(niface.IDVectorsV1()))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("decode id-vectors: %v", err)
	}
	if len(doc.Vectors) == 0 {
		t.Fatal("id-vectors has no vectors")
	}

	entryVectors := 0
	for i, v := range doc.Vectors {
		var key any
		kd := json.NewDecoder(bytes.NewReader(v.Identity.Key))
		kd.UseNumber()
		if err := kd.Decode(&key); err != nil {
			t.Fatalf("vector %d: decode key: %v", i, err)
		}
		got, err := niface.DeriveID(niface.Identity{Kind: v.Identity.Kind, Key: key})
		if err != nil {
			t.Errorf("vector %d (%s): DeriveID: %v", i, v.Identity.Kind, err)
			continue
		}
		if got != v.Expected {
			t.Errorf("vector %d (%s): id = %s, want %s", i, v.Identity.Kind, got, v.Expected)
		}

		// The entry-kind, single-target-key vectors must also reproduce through nput's seam.
		if m, ok := key.(map[string]any); ok && v.Identity.Kind == "entry" && len(m) == 1 {
			if target, ok := m["target"].(string); ok {
				entryVectors++
				got, err := entryItemID(target)
				if err != nil {
					t.Errorf("entryItemID(%q): %v", target, err)
				} else if got != v.Expected {
					t.Errorf("entryItemID(%q) = %s, want %s", target, got, v.Expected)
				}
			}
		}
	}
	if entryVectors == 0 {
		t.Error("no entry-kind vectors exercised nput's entryItemID seam")
	}
}

// TestClassifyErrorCodes pins the classifyError table directly, one case per code (the #131
// refinement): the nixCmdError marker → E_NPUT_BUILD, including its survival through
// wrapEvalErr / wrapEvalAllErr's re-wraps (the %w chain is the classification's lifeline);
// the specific fs sentinels beating the generic E_IO shape check; and the residual-I/O and
// fallback arms.
func TestClassifyErrorCodes(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nix invocation failure", &nixCmdError{err: errors.New("nput: nix build failed")}, "E_NPUT_BUILD"},
		{"marker survives wrapEvalErr attr-missing rewrap",
			wrapEvalErr(&nixCmdError{err: errors.New("error: flake does not provide attribute nput")}, "nput.x86_64-linux.web"),
			"E_NPUT_BUILD"},
		{"marker survives wrapEvalErr passthrough",
			wrapEvalErr(&nixCmdError{err: errors.New("error: something else")}, "nput.x86_64-linux.web"),
			"E_NPUT_BUILD"},
		{"marker survives wrapEvalAllErr attr-missing rewrap",
			wrapEvalAllErr(&nixCmdError{err: errors.New("error: flake does not provide attribute nput")}, "nput.x86_64-linux"),
			"E_NPUT_BUILD"},
		{"lock sentinel", lock.ErrLocked, "E_LOCK"},
		{"not-found beats the IO shape", &fs.PathError{Op: "stat", Path: "/x", Err: fs.ErrNotExist}, "E_NOTFOUND"},
		{"permission beats the IO shape", &fs.PathError{Op: "open", Path: "/x", Err: fs.ErrPermission}, "E_PERMISSION"},
		{"residual IO PathError", &fs.PathError{Op: "rmdir", Path: "/x", Err: syscall.ENOTEMPTY}, "E_IO"},
		{"residual IO LinkError", &os.LinkError{Op: "symlink", Old: "/a", New: "/b", Err: syscall.EEXIST}, "E_IO"},
		{"unclassified fallback", errors.New("nput: generation commit (nix-env --set) failed"), "E_NPUT_FAILED"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := classifyError(c.err)
			if e.Code != c.want {
				t.Errorf("classifyError(%v).Code = %s, want %s", c.err, e.Code, c.want)
			}
			if e.Message == "" {
				t.Error("message must not be empty")
			}
		})
	}
}

// TestJSONSuppressesLineOrientedStdout pins the --json stdout-ownership contract at its single
// chokepoints: every line-oriented printer emits nothing under --json (the envelope owns stdout)
// and everything under the default contract (→ ADR-0043 §2; issue #130 acceptance "--json 指定時、
// stdout には niface エンベロープ 1 文書以外何も出ない").
func TestJSONSuppressesLineOrientedStdout(t *testing.T) {
	origJSON := flagJSON
	defer func() { flagJSON = origJSON }()

	applyRes := &engine.Result{Placed: []string{"a"}, Removed: []string{"b"}}
	resetRes := &engine.ResetResult{RemovedSymlinks: []string{"s"}, RemovedCopies: []string{"c"}, KeptForeign: []string{"k"}}
	gens := []engine.Generation{{Number: 1, Date: "2026-07-19", Current: true}}
	targets := []string{".claude/skills"}

	printers := []struct {
		name  string
		print func()
	}{
		{"printApplyPlan", func() { printApplyPlan(applyRes) }},
		{"printResetPlan", func() { printResetPlan(resetRes) }},
		{"printGenerations", func() { printGenerations(gens) }},
		{"printGitignore", func() { printGitignore(targets) }},
	}
	for _, p := range printers {
		t.Run(p.name, func(t *testing.T) {
			flagJSON = false
			if out := captureStdout(t, p.print); out == "" {
				t.Errorf("%s must print under the default contract", p.name)
			}
			flagJSON = true
			if out := captureStdout(t, p.print); out != "" {
				t.Errorf("%s must print nothing under --json, got %q", p.name, out)
			}
		})
	}
}

// TestJSONEmptyResetPlanKeepsStderrNotice covers the empty-result quadrants of printResetPlan's
// contract: with nothing to remove, the stderr "nothing to remove" notice is printed under BOTH
// contracts — human diagnostics coexist with --json — while stdout stays empty (under --json it
// must stay empty for the envelope; under the default contract there are simply no plan lines).
func TestJSONEmptyResetPlanKeepsStderrNotice(t *testing.T) {
	origJSON := flagJSON
	defer func() { flagJSON = origJSON }()

	empty := &engine.ResetResult{}
	for _, jsonMode := range []bool{false, true} {
		flagJSON = jsonMode
		out, errOut := captureOutErr(t, func() { printResetPlan(empty) })
		if out != "" {
			t.Errorf("flagJSON=%v: stdout = %q, want empty for an empty plan", jsonMode, out)
		}
		if !strings.Contains(errOut, "nothing to remove") {
			t.Errorf("flagJSON=%v: stderr = %q, want the nothing-to-remove notice", jsonMode, errOut)
		}
	}
}

// TestResetPromptAllowed pins reset's prompt-permission composition: --json forbids prompting
// even on a TTY, so reset --json without --yes goes down confirmPolicy's refuse path and fails
// fast (→ ADR-0043 §8, docs/spec.md "reset --json は --yes 必須").
func TestResetPromptAllowed(t *testing.T) {
	cases := []struct{ interactive, jsonMode, want bool }{
		{true, false, true},   // TTY, default contract → prompting allowed
		{true, true, false},   // TTY + --json → machine consumption never prompts
		{false, false, false}, // non-TTY → refuse path (unchanged)
		{false, true, false},
	}
	for _, c := range cases {
		if got := resetPromptAllowed(c.interactive, c.jsonMode); got != c.want {
			t.Errorf("resetPromptAllowed(%v, %v) = %v, want %v", c.interactive, c.jsonMode, got, c.want)
		}
	}
	// The composed contract: --json without --yes refuses; --json with --yes runs promptless.
	if _, err := confirmPolicy(false, resetPromptAllowed(true, true)); err == nil {
		t.Error("reset --json without --yes must refuse (fail fast)")
	}
	if needPrompt, err := confirmPolicy(true, resetPromptAllowed(true, true)); err != nil || needPrompt {
		t.Errorf("reset --json --yes: needPrompt=%v err=%v, want promptless success", needPrompt, err)
	}
}

// TestJSONFlagRegistered pins --json as a root persistent flag, so every subcommand accepts it
// (issue #130 acceptance: --json errors on no subcommand).
func TestJSONFlagRegistered(t *testing.T) {
	root := newRootCmd()
	f := root.PersistentFlags().Lookup("json")
	if f == nil {
		t.Fatal("--json is not registered as a persistent flag")
	}
	if f.Shorthand != "" {
		t.Errorf("--json shorthand = %q, want none", f.Shorthand)
	}
}

// TestJSONUtilityCommandsDoNotBegin pins that cobra's auto-added utility commands (help /
// completion) never begin a niface run: they own stdout with their own text, so emitting an
// envelope there would corrupt both contracts (→ issue #130, docs/spec.md).
func TestJSONUtilityCommandsDoNotBegin(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"completion", "bash"}} {
		origReport := nifaceReport
		// The process-start state: only a RunE of ours replaces it with a begun run, so a
		// still-noop report after Execute proves the utility command emitted nothing.
		nifaceReport = noopEmitter{}
		root := newRootCmd()
		root.SetArgs(args)
		_ = captureStdout(t, func() {
			if err := root.Execute(); err != nil {
				t.Errorf("%v: Execute: %v", args, err)
			}
		})
		if nifaceReport.began() {
			t.Errorf("%v began a niface run; utility commands must not emit an envelope", args)
		}
		nifaceReport = origReport
	}
}

// TestJSONEndToEndSubjectBorneFailure exercises the real wiring — apply's RunE begins the run,
// runApply registers the subject, main-style emit after Execute — in-process: an apply in an
// entrypoint-less directory fails after the subject is known, so the envelope must be
// conformant, status error, with the error attached to results[0] and stdout holding nothing else.
func TestJSONEndToEndSubjectBorneFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	origReport := nifaceReport
	origJSON, origDryrun := flagJSON, flagDryrun
	defer func() {
		nifaceReport = origReport
		flagJSON, flagDryrun = origJSON, origDryrun
	}()

	nifaceReport = noopEmitter{}

	// RunE builds its own concrete run against os.Stdout (→ issue #196), so the document is
	// captured off the real sink: Execute and the main-style emit both run inside the capture.
	var execErr error
	out := captureStdout(t, func() {
		root := newRootCmd()
		root.SetArgs([]string{"apply", "--json"})
		execErr = root.Execute()
		if execErr == nil {
			return
		}
		if !nifaceReport.began() {
			return
		}
		if err := nifaceReport.emit(execErr); err != nil {
			t.Errorf("emit: %v", err)
		}
	})
	if execErr == nil {
		t.Fatal("apply in an entrypoint-less directory must fail")
	}
	if !nifaceReport.began() {
		t.Fatal("apply's RunE did not publish a begun niface run")
	}
	buf := bytes.NewBufferString(out)

	checker, err := conformance.NewDefaultChecker()
	if err != nil {
		t.Fatalf("conformance.NewDefaultChecker: %v", err)
	}
	if findings := checker.Check(buf.Bytes()); len(findings) > 0 {
		t.Fatalf("conformance findings:\n%s\ndocument: %s", strings.Join(findings, "\n"), buf.String())
	}
	doc := decodeEnvelope(t, buf)
	if doc["status"] != "error" || doc["command"] != "apply" {
		t.Errorf("status/command = %v/%v, want error/apply", doc["status"], doc["command"])
	}
	results := doc["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results = %v, want the registered subject (default)", results)
	}
	sr := results[0].(map[string]any)
	if sr["subject"].(map[string]any)["name"] != "default" {
		t.Errorf("subject = %v, want default", sr["subject"])
	}
	if errList, _ := sr["errors"].([]any); len(errList) != 1 {
		t.Errorf("subject errors = %v, want exactly one (subject-borne)", sr["errors"])
	}
	if topErrs, ok := doc["errors"]; ok {
		t.Errorf("top-level errors present = %v, want the failure attached to the subject", topErrs)
	}
}
