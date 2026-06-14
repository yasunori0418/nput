// Package lock serializes concurrent apply / reset / rollback on a profileDir
// via an advisory flock (→ ADR-0011, ADR-0013).
//
// syscall.Flock の advisory ロックはプロセス終了時に OS が自動解放するため、
// クラッシュしても stale lock が残らない。linux / darwin 両対応（→ ADR-0011）。
package lock

import (
	"errors"
	"os"
	"syscall"
)

// ErrLocked は非ブロッキング取得（try-lock）で他者が保持中のときに返る。
// shellHook 経路（--no-wait）の skip 判定に使う（→ ADR-0013）。
var ErrLocked = errors.New("nput: profileDir is locked by another process")

// Lock は profileDir 上に取得した排他 flock。
type Lock struct {
	f *os.File
}

// Acquire は dir（profileDir）に対し排他 flock を取得する。
// blocking=true は明示 apply 用の LOCK_EX（取得まで待つ）、
// blocking=false は shellHook 用の LOCK_NB（保持中なら ErrLocked・→ ADR-0013）。
// dir は事前に存在している必要がある。
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

// Release は flock を解放しファイルを閉じる。
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
