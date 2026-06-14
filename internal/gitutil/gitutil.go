// Package gitutil resolves the git toplevel for project mode root resolution
// (→ ADR-0005, docs/spec.md「root の解決」).
package gitutil

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNotInRepo は dir が git リポジトリ外で toplevel を解決できないときに返る。
var ErrNotInRepo = errors.New("nput: git リポジトリ外です（--root で root を明示してください）")

// ErrGitNotFound は git が PATH に無いときに返る。
var ErrGitNotFound = errors.New("nput: git が PATH にありません")

// Toplevel は dir を起点に `git rev-parse --show-toplevel` を実行し、
// 解決後の絶対 root パスを返す（→ ADR-0005）。どのサブディレクトリから叩いても
// 同じ root に解決される。git が無い／リポジトリ外なら明確なエラーで停止する。
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
			// git 自体は起動できたが非ゼロ終了 = リポジトリ外が代表ケース。
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
