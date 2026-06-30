package engine

// lock_test.go fills the under-covered lock-semantics / sentinel-error observations of the
// public engine API (Apply / Rollback / Reset · → ADR-0013, docs/spec.md execution flow):
//
//   - Apply NoWait: a try-lock while the profileDir flock is held returns ErrSkipped.
//   - Rollback / Reset: a blocking flock waits until the held lock is released, then proceeds.
//   - Observation order: the lock is held for the whole operation and released on completion
//     (observed blackbox via a concurrent NoWait Apply that skips mid-operation, then succeeds
//     once the lock is free).
//   - ErrSkipped is errors.Is-transparent through wrapping (the sentinel stays reachable even
//     after fmt.Errorf("%w") wrapping · a watchdog for #90 not breaking sentinel identity).
//
// Everything is written blackbox through the exported engine functions; it does not couple to
// internal helpers. Blocking-lock progress is synchronized deterministically with channels —
// no time.Sleep for timing. "Stays blocked while held" is asserted with a short-timeout select
// (the only place a timer appears); "proceeds after release" is asserted by waiting on a
// completion channel that the operation closes only after it returns.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yasunori0418/nput/internal/lock"
	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/paths"
)

// lockTest_blockedGrace is the window a held-lock operation is given to (not) make progress
// while we confirm it stays blocked. A false positive here can only be a *missed* block
// (the operation racing ahead), never a flaky failure, so a small grace is safe.
const lockTest_blockedGrace = 200 * time.Millisecond

// lockTest_progressTimeout bounds the wait for a post-release operation to finish. It is a
// safety net to fail loudly instead of hanging the suite; the happy path returns well under it.
const lockTest_progressTimeout = 10 * time.Second

// TestLockApplyNoWaitSkipsWhenHeld verifies the NoWait (shellHook) path: while the profileDir
// flock is held, a try-lock apply skips with ErrSkipped and places nothing. Asserted via
// errors.Is (not ==) so the contract survives future context wrapping of the sentinel (#90).
func TestLockApplyNoWaitSkipsWhenHeld(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	// Create the profileDir and hold its flock from a separate descriptor.
	prof := profileDirFor(t, state, root, "c")
	if err := os.MkdirAll(prof, 0o755); err != nil {
		t.Fatal(err)
	}
	held, err := lock.Acquire(prof, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Release() }()

	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, NoWait: true, Commit: fakeCommit(nil),
	})
	if !errors.Is(err, ErrSkipped) {
		t.Fatalf("NoWait while locked: err = %v, want errors.Is(err, ErrSkipped)", err)
	}
	if res == nil || !res.Skipped {
		t.Errorf("expected res.Skipped = true, got %+v", res)
	}
	if _, err := os.Lstat(filepath.Join(root, ".config", "foo")); !os.IsNotExist(err) {
		t.Errorf("skip should place nothing: lstat err = %v", err)
	}
}

// TestLockErrSkippedIsTransparentThroughWrapping pins the contract that ErrSkipped stays
// reachable via errors.Is after fmt.Errorf("%w") wrapping (single and nested). This is the
// watchdog for #90: adding context to the error must not sever sentinel identity.
func TestLockErrSkippedIsTransparentThroughWrapping(t *testing.T) {
	single := fmt.Errorf("nput: apply skipped: %w", ErrSkipped)
	if !errors.Is(single, ErrSkipped) {
		t.Errorf("single wrap: errors.Is(%v, ErrSkipped) = false, want true", single)
	}

	nested := fmt.Errorf("nput: outer: %w", fmt.Errorf("nput: inner: %w", ErrSkipped))
	if !errors.Is(nested, ErrSkipped) {
		t.Errorf("nested wrap: errors.Is(%v, ErrSkipped) = false, want true", nested)
	}

	// A wrapped *unrelated* error must not masquerade as ErrSkipped.
	other := fmt.Errorf("nput: unrelated: %w", errors.New("boom"))
	if errors.Is(other, ErrSkipped) {
		t.Errorf("unrelated wrap: errors.Is(%v, ErrSkipped) = true, want false", other)
	}
}

// TestLockApplyHoldsLockThroughOperationThenReleases observes the lock lifecycle blackbox:
// the lock is held for the whole operation and released on completion. A blocking Commit
// parks Apply inside the lock; while parked, a concurrent NoWait apply must skip (lock held
// during the operation), and once unblocked-and-returned, a NoWait apply succeeds (lock
// released after the operation). Order is fixed deterministically with channels.
func TestLockApplyHoldsLockThroughOperationThenReleases(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	src := makeSrc(t, "x")
	lf := writeLinkFarm(t, projectManifest(storeEntry(src, ".", ".config/foo")))

	inLock := make(chan struct{})  // closed once Apply is inside the lock (Commit entered).
	proceed := make(chan struct{}) // gates the in-lock operation's completion.
	commit := fakeCommit(nil)
	blockingCommit := func(profileLink, linkFarm string) error {
		close(inLock)
		<-proceed
		return commit(profileLink, linkFarm)
	}

	done := make(chan error, 1)
	go func() {
		_, err := Apply(Options{
			LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, Commit: blockingCommit,
		})
		done <- err
	}()

	// The operation now holds the lock (parked inside Commit).
	<-inLock

	// While held, a concurrent NoWait apply must skip.
	if _, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, NoWait: true, Commit: fakeCommit(nil),
	}); !errors.Is(err, ErrSkipped) {
		t.Fatalf("NoWait during in-lock operation: err = %v, want errors.Is(err, ErrSkipped)", err)
	}

	// Let the operation finish and release the lock.
	close(proceed)
	if err := <-done; err != nil {
		t.Fatalf("in-lock Apply: %v", err)
	}

	// Lock is now free → a NoWait apply succeeds (no skip).
	res, err := Apply(Options{
		LinkFarm: lf, Name: "c", RootOverride: root, StateDir: state, NoWait: true, Commit: fakeCommit(nil),
	})
	if err != nil {
		t.Fatalf("NoWait after release: %v", err)
	}
	if res.Skipped {
		t.Errorf("after release the lock is free, NoWait should not skip: %+v", res)
	}
}

// TestLockResetWaitsForHeldLock verifies Reset's blocking flock: while the lock is held, Reset
// does not proceed; once released it runs to completion. The "stays blocked" half uses a
// short-timeout select; the "proceeds" half waits on a completion channel.
func TestLockResetWaitsForHeldLock(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	symSrc := makeSrc(t, "sub/file")

	// Precondition: one committed generation so prof.Profile exists (Reset locks only when it does).
	applyForReset(t, root, state, homeManifest(storeEntry(symSrc, "sub", ".link")))
	linkAbs := filepath.Join(root, ".link")
	if _, err := os.Lstat(linkAbs); err != nil {
		t.Fatalf("setup: .link missing: %v", err)
	}

	prof := paths.Resolve(state, "cfg", manifest.RootKindHome, root, true)
	held, err := lock.Acquire(prof.Dir, true)
	if err != nil {
		t.Fatal(err)
	}

	type resetOutcome struct {
		res *ResetResult
		err error
	}
	started := make(chan struct{})
	done := make(chan resetOutcome, 1)
	go func() {
		close(started)
		res, err := Reset(resetOpts(root, state, nil, false, nil))
		done <- resetOutcome{res, err}
	}()

	// Confirm Reset is blocked while the lock is held.
	<-started
	select {
	case out := <-done:
		_ = held.Release()
		t.Fatalf("Reset proceeded while the lock was held: %+v / %v", out.res, out.err)
	case <-time.After(lockTest_blockedGrace):
		// Still blocked — expected.
	}

	// Release the lock; Reset must now proceed to completion.
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	select {
	case out := <-done:
		if out.err != nil {
			t.Fatalf("Reset after release: %v", out.err)
		}
		if _, err := os.Lstat(linkAbs); !os.IsNotExist(err) {
			t.Errorf(".link should be removed after Reset, lstat err = %v", err)
		}
		if got := out.res.RemovedSymlinks; len(got) != 1 || got[0] != ".link" {
			t.Errorf("RemovedSymlinks = %v, want [.link]", got)
		}
	case <-time.After(lockTest_progressTimeout):
		t.Fatal("Reset did not proceed after the lock was released")
	}
}

// TestLockRollbackWaitsForHeldLock verifies Rollback's blocking flock: while the lock is held
// Rollback waits, and once released it re-converges and moves the profile pointer. Setup mirrors
// the rollback re-convergence layout (gen1={a}, gen2(current)={a,b}); rollback removes b.
func TestLockRollbackWaitsForHeldLock(t *testing.T) {
	root := realTempDir(t)
	state := realTempDir(t)
	srcA := makeSrc(t, "x")
	srcB := makeSrc(t, "x")

	prof := paths.Resolve(state, "vim", manifest.RootKindHome, root, true)
	if err := os.MkdirAll(prof.Dir, 0o755); err != nil {
		t.Fatal(err)
	}

	lf1 := writeLinkFarm(t, homeManifest(storeEntry(srcA, ".", "a")))
	lf2 := writeLinkFarm(t, homeManifest(
		storeEntry(srcA, ".", "a"),
		storeEntry(srcB, ".", "b"),
	))
	if err := os.Symlink(lf1, paths.GenerationLink(prof.Profile, 1)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(lf2, paths.GenerationLink(prof.Profile, 2)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(lf2, prof.Profile); err != nil {
		t.Fatal(err)
	}
	// Current FS = gen2: a→srcA, b→srcB.
	if err := os.Symlink(srcA, filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(srcB, filepath.Join(root, "b")); err != nil {
		t.Fatal(err)
	}

	held, err := lock.Acquire(prof.Dir, true)
	if err != nil {
		t.Fatal(err)
	}

	type rollbackOutcome struct {
		res *RollbackResult
		err error
	}
	var switched int
	started := make(chan struct{})
	done := make(chan rollbackOutcome, 1)
	go func() {
		close(started)
		res, err := Rollback(RollbackOptions{
			Name:         "vim",
			RootKind:     manifest.RootKindHome,
			RootOverride: root,
			StateDir:     state,
			ListGenerations: func(string) ([]Generation, error) {
				return []Generation{{Number: 1}, {Number: 2, Current: true}}, nil
			},
			SwitchGeneration: func(_ string, gen int) error { switched = gen; return nil },
		})
		done <- rollbackOutcome{res, err}
	}()

	// Confirm Rollback is blocked while the lock is held.
	<-started
	select {
	case out := <-done:
		_ = held.Release()
		t.Fatalf("Rollback proceeded while the lock was held: %+v / %v", out.res, out.err)
	case <-time.After(lockTest_blockedGrace):
		// Still blocked — expected.
	}

	// Release the lock; Rollback must now re-converge to gen1.
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	select {
	case out := <-done:
		if out.err != nil {
			t.Fatalf("Rollback after release: %v", out.err)
		}
		if out.res.From != 2 || out.res.To != 1 {
			t.Errorf("From/To = %d/%d, want 2/1", out.res.From, out.res.To)
		}
		if switched != 1 {
			t.Errorf("switched generation = %d, want 1", switched)
		}
		if _, err := os.Lstat(filepath.Join(root, "b")); !os.IsNotExist(err) {
			t.Errorf("b should be removed after rollback, lstat err = %v", err)
		}
		if dest, err := os.Readlink(filepath.Join(root, "a")); err != nil || dest != srcA {
			t.Errorf("a: dest=%q err=%v, want %q", dest, err, srcA)
		}
	case <-time.After(lockTest_progressTimeout):
		t.Fatal("Rollback did not proceed after the lock was released")
	}
}
