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
// tool-specific info slots (item / change / per-subject result / envelope-wide); #130 leaves
// them as open JSON objects — #131 / #132 pin them down to concrete DTO structs when the
// payloads are populated (nil maps marshal as omitted, keeping the minimal envelope minimal).
type (
	nputEnvelope      = niface.Envelope[map[string]any, map[string]any, map[string]any, map[string]any]
	nputSubjectResult = niface.SubjectResult[map[string]any, map[string]any, map[string]any]
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
// subcommand's RunE begins it first thing (command name + start time — flag parsing and cobra's
// argument validation have succeeded by then, and cobra's utility commands help / completion /
// __complete never reach a RunE of ours, so they keep stdout for their own text), commands
// register their subject as it becomes known, and main emits once after Execute returns: the
// single stdout write point (→ issue #130, ADR-0043 §2).
type nifaceRun struct {
	now     func() time.Time // injectable clock; tests pin it to a fixed time
	out     io.Writer        // envelope sink (os.Stdout; tests substitute a buffer)
	command string           // executed subcommand name ("" until begin — no envelope for --help / --version)
	started time.Time
	subject *nifaceSubject // the single subject #130 wires (nil = none; #164 generalizes to N)
}

// nifaceSubject is one subject's (config's) accumulation for its SubjectResult.
type nifaceSubject struct {
	name    string
	started time.Time
}

// nifaceReport is the process-wide run the CLI wires (tests build their own nifaceRun instances).
var nifaceReport = &nifaceRun{now: time.Now, out: os.Stdout}

// begin records the executed subcommand and the run start time.
func (r *nifaceRun) begin(command string) {
	r.command = command
	r.started = r.now()
}

// began reports whether an nput subcommand's RunE actually started (false on the --help /
// --version / help / completion paths and on flag or argument validation failures — none of
// those emit an envelope; exit code + stderr are the only signals there).
func (r *nifaceRun) began() bool { return r.command != "" }

// beginSubject registers the command's subject (config name) once it is known. From then on a
// command failure attaches to this subject's SubjectResult.errors[] rather than the top-level
// errors[] (top level = failures before/outside subject resolution only · → ADR-0043 §6).
// #130 wires the single-config commands; --all / init leave no subject (results stays []).
func (r *nifaceRun) beginSubject(name string) {
	r.subject = &nifaceSubject{name: name, started: r.now()}
}

// emit writes the niface envelope — exactly one JSON document, trailing newline, nothing else —
// to r.out. cmdErr is the command's overall error: nil ⇔ exit 0 ⇔ status "success" (the only
// status contract consumers may rely on · → ADR-0043 §6). The #130 envelope is minimal: the
// registered subject yields one SubjectResult with empty items; payload population is #131 / #132.
func (r *nifaceRun) emit(cmdErr error) error {
	finished := nifaceTimestamp(r.now())
	status := niface.StatusSuccess
	if cmdErr != nil {
		status = niface.StatusError
	}

	env := nputEnvelope{
		SpecVersion: nifaceSpecVersion,
		Tool:        niface.Tool{Name: "nput", Version: version},
		Command:     r.command,
		Status:      status,
		DryRun:      flagDryrun,
		StartedAt:   nifaceTimestamp(r.started),
		FinishedAt:  finished,
	}
	if r.subject != nil {
		env.Results = append(env.Results, nputSubjectResult{
			Subject:    niface.Subject{Name: r.subject.name},
			Status:     status,
			StartedAt:  nifaceTimestamp(r.subject.started),
			FinishedAt: finished,
		})
	}
	if cmdErr != nil {
		e := classifyError(cmdErr)
		if r.subject != nil {
			// The failure happened with the subject established, so it is subject-borne
			// (build / lock / engine ...), not a pre-enumeration failure (→ ADR-0043 §6).
			env.Results[0].Errors = append(env.Results[0].Errors, e)
		} else {
			env.Errors = append(env.Errors, e)
		}
	}

	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(r.out, "%s\n", data)
	return err
}

// classifyError maps a command-level failure onto a niface error object (two-layer code naming ·
// niface §6). #130 classifies only what sentinel errors already distinguish; the finer per-site
// codes (E_NPUT_BUILD at build call sites, per-item E_NPUT_COLLISION) arrive with the payloads
// in #131 / #132 (→ ADR-0043 §8). E_NPUT_FAILED is the tool-generic fallback for a command
// failure not otherwise classified.
func classifyError(err error) niface.Error {
	code := "E_NPUT_FAILED"
	message := err.Error()
	var ee *exitError
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
	case errors.Is(err, fs.ErrNotExist):
		code = "E_NOTFOUND"
	case errors.Is(err, fs.ErrPermission):
		code = "E_PERMISSION"
	}
	return niface.Error{Code: code, Message: message}
}
