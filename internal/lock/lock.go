// Package lock serializes concurrent apply / reset / rollback on a profileDir
// via an advisory flock (→ ADR-0011, ADR-0013).
//
// The advisory lock from syscall.Flock is released automatically by the OS on
// process exit, so no stale lock remains even after a crash. Supports both
// linux and darwin (→ ADR-0011).
package lock

import (
	"errors"
	"os"
	"syscall"
)

// ErrLocked is returned by a non-blocking acquisition (try-lock) when another holder is active.
// Used for the skip decision on the shellHook path (--no-wait) (→ ADR-0013).
var ErrLocked = errors.New("nput: profileDir is locked by another process")

// Lock is an exclusive flock acquired on a profileDir.
type Lock struct {
	f *os.File
}

// Acquire takes an exclusive flock on dir (the profileDir).
// blocking=true uses LOCK_EX for explicit apply (waits until acquired);
// blocking=false uses LOCK_NB for shellHook (returns ErrLocked if held; → ADR-0013).
// dir must already exist.
func Acquire(dir string, blocking bool) (*Lock, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}

	how := syscall.LOCK_EX
	if !blocking {
		how |= syscall.LOCK_NB
	}
	if err := syscall.Flock(int(f.Fd()), how); err != nil {
		_ = f.Close()
		if !blocking && (errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN)) {
			return nil, ErrLocked
		}
		return nil, err
	}
	return &Lock{f: f}, nil
}

// Release releases the flock and closes the file.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	if cerr := l.f.Close(); err == nil {
		err = cerr
	}
	l.f = nil
	return err
}
