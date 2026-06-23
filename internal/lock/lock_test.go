package lock

import (
	"testing"
	"time"
)

func TestTryLockConflict(t *testing.T) {
	dir := t.TempDir()

	l1, err := Acquire(dir, true)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer func() { _ = l1.Release() }()

	// A try-lock while held returns ErrLocked.
	if _, err := Acquire(dir, false); err != ErrLocked {
		t.Fatalf("try-lock while held: got %v, want ErrLocked", err)
	}
}

func TestReleaseAllowsReacquire(t *testing.T) {
	dir := t.TempDir()

	l1, err := Acquire(dir, false)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := l1.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}

	l2, err := Acquire(dir, false)
	if err != nil {
		t.Fatalf("re-Acquire after release: %v", err)
	}
	_ = l2.Release()
}

func TestBlockingWaitsForRelease(t *testing.T) {
	dir := t.TempDir()

	l1, err := Acquire(dir, true)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	acquired := make(chan struct{})
	go func() {
		l2, err := Acquire(dir, true) // blocking: wait until acquired.
		if err != nil {
			t.Errorf("blocking Acquire: %v", err)
			close(acquired)
			return
		}
		_ = l2.Release()
		close(acquired)
	}()

	select {
	case <-acquired:
		t.Fatal("blocking Acquire returned before lock was released")
	case <-time.After(100 * time.Millisecond):
		// Still not acquired = blocking is in effect.
	}

	if err := l1.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	select {
	case <-acquired:
		// Acquired after release.
	case <-time.After(2 * time.Second):
		t.Fatal("blocking Acquire did not proceed after release")
	}
}
