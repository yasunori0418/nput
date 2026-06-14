package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yasunori0418/nput/internal/manifest"
)

// place は新世代 manifest の各 entry を配置する（新規 / 張替を stale 除去より先に・→ ADR-0006）。
// 本スライスは store / out-of-store の symlink 配置のみ対応する（copy は将来スライス・→ Issue #6）。
func (a *applier) place(prev *manifest.Manifest) error {
	prevByTarget := byTarget(prev)
	for _, e := range a.manifest.Entries {
		switch e.Method {
		case manifest.MethodSymlink:
			if err := a.placeSymlink(e, prevByTarget); err != nil {
				return err
			}
		case manifest.MethodCopy:
			return fmt.Errorf("nput: method=copy は本スライスでは未実装です (target: %s)", e.Target)
		default:
			return fmt.Errorf("nput: 未知の method: %q (target: %s)", e.Method, e.Target)
		}
	}
	return nil
}

func (a *applier) placeSymlink(e manifest.Entry, prevByTarget map[string]manifest.Entry) error {
	targetAbs := a.targetAbs(e.Target)

	// 0. 祖先 component を lstat walk。symlink ならネスト不可で error 停止（→ ADR-0015）。
	if err := a.checkAncestors(e.Target); err != nil {
		return err
	}

	// 1. 親ディレクトリを作成（祖先 symlink は 0 で弾き済み）。
	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return fmt.Errorf("nput: 親ディレクトリを作成できません (%s): %w", filepath.Dir(targetAbs), err)
	}

	dest := linkDest(e)

	// 2. 既存 target の判定（→ docs/spec.md「symlink モード」・ADR-0015）。
	info, err := os.Lstat(targetAbs)
	switch {
	case err == nil && info.Mode()&os.ModeSymlink != 0:
		// 既存 symlink: 自身の前世代 manifest 記録通りなら silent、それ以外は foreign warning。
		if !a.recordedLink(e.Target, targetAbs, prevByTarget) {
			a.opts.Warnf("nput: 記録の無い symlink を上書きします (foreign・後勝ち): %s", e.Target)
		}
		if err := os.Remove(targetAbs); err != nil {
			return fmt.Errorf("nput: 既存 symlink を除去できません (%s): %w", targetAbs, err)
		}
		a.result.Replaced = append(a.result.Replaced, e.Target)
	case err == nil:
		// 通常ファイル / ディレクトリは上書きしない（→ docs/spec.md エラー仕様）。
		return fmt.Errorf("nput: target に既存のファイル/ディレクトリがあります（上書きしません）: %s", e.Target)
	case os.IsNotExist(err):
		a.result.Placed = append(a.result.Placed, e.Target)
	default:
		return fmt.Errorf("nput: target を lstat できません (%s): %w", targetAbs, err)
	}

	// 3. <配置元>/<subpath> を指す symlink を作成（張替は unlink + symlink・→ ADR-0017）。
	if err := os.Symlink(dest, targetAbs); err != nil {
		return fmt.Errorf("nput: symlink を作成できません (%s -> %s): %w", targetAbs, dest, err)
	}
	return nil
}

// checkAncestors は target の各祖先 component を root から順に lstat し、いずれかが
// symlink なら error を返す（全体 symlink 配置の配下にネストできない・→ ADR-0015）。
// 祖先が未作成ならそこで打ち切る（配下も存在しないため安全）。
func (a *applier) checkAncestors(target string) error {
	clean := filepath.Clean(target)
	comps := strings.Split(clean, string(os.PathSeparator))
	cur := a.root
	for i := 0; i < len(comps)-1; i++ {
		if comps[i] == "" {
			continue
		}
		cur = filepath.Join(cur, comps[i])
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("nput: 祖先を lstat できません (%s): %w", cur, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("nput: 祖先 %q が symlink です。配下にネストできません (target: %s・→ ADR-0015)", cur, target)
		}
	}
	return nil
}

// recordedLink は target が「自身の前世代 manifest が記録した symlink」かを判定する。
// 前世代に同 target の entry があり、かつ実体の symlink が記録通りの先を指すときのみ true
// （保守的不変条件・→ ADR-0002, ADR-0015）。
func (a *applier) recordedLink(target, targetAbs string, prevByTarget map[string]manifest.Entry) bool {
	pe, ok := prevByTarget[target]
	if !ok {
		return false
	}
	onDisk, err := os.Readlink(targetAbs)
	if err != nil {
		return false
	}
	return onDisk == linkDest(pe)
}

// removeStale は前世代にあって新世代から消えた entry を保守的に除去する（最後に実行・→ ADR-0006）。
// 削除するのは「記録通りの先を指す symlink」のみ。foreign / 内容不一致は警告して残す。
// copy entry は除去せず orphan を警告する（→ ADR-0002, ADR-0020）。
func (a *applier) removeStale(prev *manifest.Manifest) error {
	if prev == nil {
		return nil
	}
	newByTarget := byTarget(a.manifest)
	for _, pe := range prev.Entries {
		if _, kept := newByTarget[pe.Target]; kept {
			continue
		}
		if pe.Method == manifest.MethodCopy {
			a.opts.Warnf("nput: copy entry が消えましたが target は削除しません（orphan・reset で撤去）: %s", pe.Target)
			continue
		}

		targetAbs := a.targetAbs(pe.Target)
		info, err := os.Lstat(targetAbs)
		if err != nil {
			if os.IsNotExist(err) {
				continue // 既に無い = no-op。
			}
			return fmt.Errorf("nput: stale target を lstat できません (%s): %w", targetAbs, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			a.opts.Warnf("nput: stale target が symlink ではないため残します: %s", pe.Target)
			continue
		}
		onDisk, err := os.Readlink(targetAbs)
		if err != nil || onDisk != linkDest(pe) {
			a.opts.Warnf("nput: stale symlink が記録と不一致のため残します: %s", pe.Target)
			continue
		}
		if err := os.Remove(targetAbs); err != nil {
			return fmt.Errorf("nput: stale symlink を除去できません (%s): %w", targetAbs, err)
		}
		a.result.Removed = append(a.result.Removed, pe.Target)
	}
	return nil
}

// targetAbs は root 相対 target を絶対パスへ正規化する。
func (a *applier) targetAbs(target string) string {
	return filepath.Join(a.root, filepath.Clean(target))
}
