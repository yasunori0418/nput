// niface.go is the --json machine-readable output foundation (→ ADR-0043, issue #130): the
// nput instantiation of the niface envelope (specVersion 1), the item-id derivation seam, and
// the emit helper that writes exactly one envelope document to stdout at command completion.
//
// The CLI output contract is closed inside the cmd layer: engine results are never marshaled
// directly — #131 / #132 convert them through the DTO types below, so engine-internal structure
// changes cannot break the output contract (→ ADR-0043 §8).
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	niface "github.com/yasunori0418/niface/go"

	"github.com/yasunori0418/nput/internal/lock"
)

// nifaceSpecVersion is the niface output-spec version nput produces. Independent of both the
// manifest.json schemaVersion (engine input contract) and tool.version (nput release)
// (→ ADR-0043 §2).
const nifaceSpecVersion = 1

// The nput instantiation of the niface generic envelope. The four type parameters are the
// tool-specific info slots (item / change / per-subject result / envelope-wide). The item /
// change slots are pinned to the concrete mutation DTOs shared by every command (pointers, so
// an absent info is omitted · #131); the per-subject result (TInfo) and envelope-wide
// (TEnvInfo) slots vary per command, so they stay type parameters here and each command
// instantiates them with its own info type (→ issue #196, niface ADR-0018).
//
// These stay aliases (parameterized aliases, Go 1.24+), so the emitted values remain niface's
// own types and keep niface's MarshalJSON — the required-array normalization must not be lost.
type (
	nputEnvelope[TInfo, TEnvInfo any] = niface.Envelope[*nifaceEntryInfo, *nifaceChangeInfo, TInfo, TEnvInfo]
	nputSubjectResult[TInfo any]      = niface.SubjectResult[*nifaceEntryInfo, *nifaceChangeInfo, TInfo]
	nputResult[TInfo any]             = niface.Result[*nifaceEntryInfo, *nifaceChangeInfo, TInfo]
	nputItem                          = niface.Item[*nifaceEntryInfo]
	nputChange                        = niface.Change[*nifaceChangeInfo]
)

// entryItemID derives the niface item id for a placement entry: identity kind="entry",
// key={target} (the root-relative target only — the config name stays out of the key; consumers
// resolve references by the (tool.name, subject, id) triple · → ADR-0043 §3, niface ADR-0024).
// Shared by the #131 / #132 payload builders; the identity shape is pinned against niface's
// id-vectors in TestEntryItemIDMatchesVectors.
func entryItemID(target string) (string, error) {
	return niface.DeriveID(niface.Identity{Kind: "entry", Key: map[string]any{"target": target}})
}

// nifaceTimestamp renders t for the envelope's startedAt/finishedAt: RFC 3339, "T" separator,
// explicit UTC offset (the local offset; UTC itself renders as "Z" — both satisfy niface
// ADR-0025's format assertion).
func nifaceTimestamp(t time.Time) string { return t.Format(time.RFC3339) }

// nifaceRun accumulates what the --json envelope needs across one command invocation. Each nput
// subcommand's RunE builds it first thing and begins it (command name + start time — flag parsing
// and cobra's argument validation have succeeded by then, and cobra's utility commands help /
// completion / __complete never reach a RunE of ours, so they keep stdout for their own text),
// commands register their subject as it becomes known, and main emits once after Execute
// returns: the single stdout write point (→ issue #130, ADR-0043 §2).
//
// TInfo / TEnvInfo are the command's own result.info / envelope-info types (→ issue #196): each
// command instantiates the run with its concrete pair in RunE and threads that concrete value
// through its run functions, because the info-typed setters cannot cross the emitter interface
// (Go forbids generic methods).
type nifaceRun[TInfo, TEnvInfo any] struct {
	now     func() time.Time // injectable clock; tests pin it to a fixed time
	out     io.Writer        // envelope sink (os.Stdout; tests substitute a buffer)
	command string           // executed subcommand name ("" until begin · → began)
	dryRun  bool             // envelope dryRun field, captured from flagDryrun at begin (tests set it directly)
	started time.Time
	// subjects are the run's registered subjects in registration order, one SubjectResult each
	// (→ issue #164). A single-config command registers exactly one (N=1, the shape #130 wired);
	// --all registers one per selected config; init and a pre-enumeration failure register none.
	subjects []*nifaceSubject[TInfo]
	info     TEnvInfo // envelope-wide tool info (init's run info · niface ADR-0018, issue #132)
}

// nifaceSubject is one subject's (config's) accumulation for its SubjectResult. Commands hold
// it as the handle beginSubject returns and attach everything through it, so a multi-subject run
// cannot mis-attribute one config's result to another — the binding is the handle, not the
// registration order (the precondition for #149's parallel apply --all · → issue #164).
type nifaceSubject[TInfo any] struct {
	name    string
	started time.Time
	// payload is the command-built items/changes/generation/warnings mapping (→ #131's
	// niface_payload.go builders). nil keeps the minimal #130 shape (empty items) — the
	// read-only commands until #132, and failures before any engine result exists.
	payload *nifacePayload[TInfo]
	// finished marks that this subject settled its own outcome through finish, so emit takes
	// its status and errors[] from err below instead of from the command-level error. A
	// multi-subject run (--all) finishes every subject — one config's failure must not colour
	// the ones that succeeded — while the single-config commands leave it false and keep the
	// #130 attribution (the command error IS that one subject's · → issue #164).
	finished bool
	err      error
}

// setPayload attaches this subject's result payload (→ #131 / #132 payload builders).
func (s *nifaceSubject[TInfo]) setPayload(p *nifacePayload[TInfo]) { s.payload = p }

// finish settles this subject's own outcome: err is that config's error (nil = success), which
// decides its SubjectResult status and errors[] independently of the other subjects and of the
// command's aggregate error (→ issue #164). Only the multi-subject paths call it.
func (s *nifaceSubject[TInfo]) finish(err error) {
	s.finished, s.err = true, err
}

// failed reports whether this subject settled on an error — the aggregate's error source
// alongside the command-level error (any one subject in error makes the envelope error ·
// → niface §2, ADR-0043 §6).
func (s *nifaceSubject[TInfo]) failed() bool { return s.finished && s.err != nil }

// setEnvelopeInfo registers the envelope-wide tool info (top-level info — run-scoped facts not
// tied to any subject; init's template expansion · niface ADR-0018, issue #132).
func (r *nifaceRun[TInfo, TEnvInfo]) setEnvelopeInfo(info TEnvInfo) { r.info = info }

// emitter is the type-erased face of a run: the two methods main needs after Execute returns,
// neither of which mentions the info types. Which subcommand runs is unknown before cobra
// dispatches, so the process-wide report cannot be a single static instantiation; the
// info-typed operations (begin / beginSubject / setPayload / setEnvelopeInfo) stay on the
// concrete run each command holds (→ issue #196).
type emitter interface {
	began() bool
	emit(error) error
}

// noopEmitter is nifaceReport's initial value: no RunE has run, so no envelope exists. It keeps
// main's emit gate total without a nil check on the paths that never reach a RunE (--help /
// --version / help / completion / flag- and argument-validation failures).
type noopEmitter struct{}

func (noopEmitter) began() bool      { return false }
func (noopEmitter) emit(error) error { return nil }

// nifaceReport is the process-wide run the CLI wires: each subcommand's RunE assigns its own
// concrete run here so main can emit it after Execute returns (tests build their own runs).
var nifaceReport emitter = noopEmitter{}

// beginNifaceRun starts a command's run: it builds the concrete instantiation against the real
// clock and stdout, begins it for command, and publishes it to nifaceReport so main can emit it
// after Execute returns — keeping build / begin / publish in one place, because a run that is
// begun but never published emits no envelope at all (main's gate reads nifaceReport, not the
// command's local variable · → issue #196).
//
// Commands do not call this directly with their type arguments; each one wraps it in a
// begin<Command>Run helper declared next to its run alias, so the command's (TInfo, TEnvInfo)
// pair is spelled exactly once per command and the alias stays the single source of truth.
func beginNifaceRun[TInfo, TEnvInfo any](command string) *nifaceRun[TInfo, TEnvInfo] {
	r := &nifaceRun[TInfo, TEnvInfo]{now: time.Now, out: os.Stdout}
	r.begin(command)
	nifaceReport = r
	return r
}

// begin records the executed subcommand, the run start time, and the parsed --dryrun value
// (cobra has finished flag parsing by RunE, where begin is called).
func (r *nifaceRun[TInfo, TEnvInfo]) begin(command string) {
	r.command = command
	r.dryRun = flagDryrun
	r.started = r.now()
}

// began satisfies emitter. A published run is always a begun one — beginNifaceRun is the only
// production constructor and it begins before publishing — so this is true for anything main
// finds in nifaceReport; the not-started state is noopEmitter's, which owns the list of paths
// that never reach a RunE. It still reads the field rather than returning true, because tests
// build runs directly and main's gate must stay honest for them too.
func (r *nifaceRun[TInfo, TEnvInfo]) began() bool { return r.command != "" }

// beginSubject registers a subject (config name) once it is known and returns its handle, which
// the command uses to attach that config's payload. From then on a command failure attaches to a
// subject's SubjectResult.errors[] rather than the top-level errors[] (top level = failures
// before/outside subject enumeration only · → ADR-0043 §6). A single-config command calls it once
// (N=1); --all calls it per selected config (→ issue #164); init registers none (results stays []).
func (r *nifaceRun[TInfo, TEnvInfo]) beginSubject(name string) *nifaceSubject[TInfo] {
	s := &nifaceSubject[TInfo]{name: name, started: r.now()}
	r.subjects = append(r.subjects, s)
	return s
}

// subjectResult renders this subject's SubjectResult. err is the error attributed to it — its
// own (finish) for a multi-subject run, the command-level one for a single-config command that
// never finished it — and decides both this result's status and whether it carries an errors[]
// entry (→ issue #164). finishedAt is the run's single finish timestamp, shared by every result.
func (s *nifaceSubject[TInfo]) subjectResult(err error, finishedAt string) nputSubjectResult[TInfo] {
	status := niface.StatusSuccess
	if err != nil {
		status = niface.StatusError
	}
	sr := nputSubjectResult[TInfo]{
		Subject:    niface.Subject{Name: s.name},
		Status:     status,
		StartedAt:  nifaceTimestamp(s.started),
		FinishedAt: finishedAt,
	}
	if p := s.payload; p != nil {
		sr.Generation = p.generation
		sr.Warnings = p.warnings
		sr.Result = nputResult[TInfo]{Items: p.items, Changes: p.changes, Info: p.info}
	}
	// An item-borne failure (entry failure / conflict) is already represented as a failed item;
	// duplicating it into errors[] would violate niface §2 (item 起因のエラーを errors[] に置いて
	// はならない). The failed item alone justifies the error status. Everything else that failed
	// with the subject established is subject-borne (build / lock / commit ...) and lands here
	// rather than at the top level (→ ADR-0043 §6).
	if err != nil && (s.payload == nil || !s.payload.itemBorne) {
		sr.Errors = append(sr.Errors, classifyError(err))
	}
	return sr
}

// emit writes the niface envelope — exactly one JSON document, trailing newline, nothing else —
// to r.out. cmdErr is the command's overall error: nil ⇔ exit 0 ⇔ status "success" (the only
// status contract consumers may rely on · → ADR-0043 §6).
//
// emit only aggregates (→ issue #164): each subject has already settled its own status and
// errors[] (finish), and emit folds them into the envelope's status — one subject in error makes
// the whole document error — while the top-level errors[] takes only a failure that belongs to no
// subject, i.e. one from before/outside subject enumeration. A single-config command settles no
// subject, so cmdErr attributes to the one it registered, which is #130's behaviour unchanged.
func (r *nifaceRun[TInfo, TEnvInfo]) emit(cmdErr error) error {
	finished := nifaceTimestamp(r.now())

	env := nputEnvelope[TInfo, TEnvInfo]{
		SpecVersion: nifaceSpecVersion,
		Tool:        niface.Tool{Name: "nput", Version: version},
		Command:     r.command,
		Status:      niface.StatusSuccess,
		DryRun:      r.dryRun,
		StartedAt:   nifaceTimestamp(r.started),
		FinishedAt:  finished,
		Info:        r.info,
	}
	if cmdErr != nil {
		env.Status = niface.StatusError
	}
	// unsettled is the subject cmdErr attributes to when no subject settled its own outcome: the
	// last registered one. A single-config command has exactly one and stops at its first failure,
	// so that is the subject it was on; a run whose subjects all called finish leaves it nil.
	var unsettled *nifaceSubject[TInfo]
	for _, s := range r.subjects {
		if s.failed() {
			env.Status = niface.StatusError
		}
		if !s.finished {
			unsettled = s
		}
	}
	for _, s := range r.subjects {
		err := s.err
		if s == unsettled {
			err = cmdErr
		}
		env.Results = append(env.Results, s.subjectResult(err, finished))
	}
	if cmdErr != nil && len(r.subjects) == 0 {
		// The failure belongs to no subject because none was ever enumerated: it happened before
		// or outside subject enumeration (entrypoint discovery, the batch eval, an argument
		// rejection), which is the only thing the top-level errors[] carries (→ ADR-0043 §6,
		// issue #164). Once subjects exist, the run's failure is theirs — a --all aggregate error
		// merely restates that some of them failed, and repeating it here would report the same
		// failure twice, at a layer that promises a different meaning.
		env.Errors = append(env.Errors, classifyError(cmdErr))
	}

	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(r.out, "%s\n", data)
	return err
}

// classifyError maps a command-level failure onto a niface error object (two-layer code naming ·
// niface §6, ADR-0043 §8): tool-specific E_NPUT_COLLISION (dryrun conflict exit) / E_NPUT_BUILD
// (internal nix eval / build invocation, via the nixCmdError marker), and the common registry
// codes E_LOCK / E_NOTFOUND / E_PERMISSION / E_IO. Specific sentinels win over the generic
// E_IO shape check, so a not-found PathError stays E_NOTFOUND. E_NPUT_FAILED is the
// tool-generic fallback for a command failure not otherwise classified.
func classifyError(err error) niface.Error {
	code := "E_NPUT_FAILED"
	message := err.Error()
	var ee *exitError
	var ne *nixCmdError
	switch {
	case errors.As(err, &ee) && ee.code == 2:
		// exit 2 is apply --dryrun's conflict detection (→ docs/spec.md exit code table). Its
		// exitError deliberately carries no message (the plan went to stdout), so supply one.
		code = "E_NPUT_COLLISION"
		if message == "" {
			message = "conflict(s) detected in dryrun"
		}
	case errors.Is(err, lock.ErrLocked):
		code = "E_LOCK"
	case errors.As(err, &ne):
		code = "E_NPUT_BUILD"
	case errors.Is(err, fs.ErrNotExist):
		code = "E_NOTFOUND"
	case errors.Is(err, fs.ErrPermission):
		code = "E_PERMISSION"
	case isIOError(err):
		code = "E_IO"
	}
	return niface.Error{Code: code, Message: message}
}

// isIOError reports whether err carries a filesystem / external-I/O failure shape (niface §6
// common code E_IO). Checked after the more specific fs.ErrNotExist / fs.ErrPermission
// sentinels, so it only catches the remaining I/O failures (EEXIST, ENOTEMPTY, EIO, ...).
func isIOError(err error) bool {
	var pe *fs.PathError
	var le *os.LinkError
	var se *os.SyscallError
	return errors.As(err, &pe) || errors.As(err, &le) || errors.As(err, &se)
}
