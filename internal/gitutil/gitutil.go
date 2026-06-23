// Package gitutil resolves the git toplevel for project mode root resolution
// (→ ADR-0005, docs/spec.md "root resolution").
package gitutil

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNotInRepo is returned when dir is outside a git repository and the toplevel cannot be resolved.
var ErrNotInRepo = errors.New("nput: git リポジトリ外です（--root で root を明示してください）")

// ErrGitNotFound is returned when git is not on PATH.
var ErrGitNotFound = errors.New("nput: git が PATH にありません")

// Toplevel runs `git rev-parse --show-toplevel` rooted at dir and returns the
// resolved absolute root path (→ ADR-0005). It resolves to the same root no matter
// which subdirectory it is invoked from. It stops with a clear error if git is
// missing or dir is outside a repository.
func Toplevel(dir string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", ErrGitNotFound
	}

	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// git started but exited non-zero; the representative case is being outside a repository.
			return "", fmt.Errorf("%w: %s", ErrNotInRepo, strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("nput: git rev-parse の実行に失敗しました: %w", err)
	}

	top := strings.TrimSpace(stdout.String())
	if top == "" {
		return "", ErrNotInRepo
	}
	return top, nil
}
