package engine

import (
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"

	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/planner"
)

// materializeCopies は copy entry を実 FS に反映する分岐点。--recopy のときは全 copy target を
// 無条件上書き（recopyAll）、通常は planner の place-once 分類（target 不在のみ新規コピー）に従う
// （→ ADR-0020）。通常 apply と世代スキップ時のドリフト修復が共有する。
func (a *applier) materializeCopies(plan planner.Plan, recopy bool) error {
	if recopy {
		return a.recopyAll()
	}
	return a.placeCopies(plan.Copies)
}

// placeCopies materializes the planner's place-once CopyActions（target 不在の copy のみ・
// → ADR-0002, ADR-0016）。既存 target（記録あり / foreign）は planner が CopyAction を
// 生まないため、ここは「新規コピー」だけを実 FS に反映する薄い executor に徹する。
func (a *applier) placeCopies(actions []planner.CopyAction) error {
	for _, act := range actions {
		if err := os.MkdirAll(filepath.Dir(act.TargetAbs), 0o755); err != nil {
			return fmt.Errorf("nput: 親ディレクトリを作成できません (%s): %w", filepath.Dir(act.TargetAbs), err)
		}
		if err := copyTree(act.Src, act.TargetAbs); err != nil {
			return fmt.Errorf("nput: copy 配置に失敗しました (%s -> %s): %w", act.Src, act.TargetAbs, err)
		}
		a.result.Copied = append(a.result.Copied, act.Entry.Target)
	}
	return nil
}

// recopyAll は apply --recopy の copy 上書き経路（→ ADR-0020, docs/spec.md「recopy」）。
// config 内の全 copy entry について、target が在れば削除してから無条件に再コピーする
// （差分判定なし・ローカル編集は破棄）。place-once 分類（planner.Copies）は使わず manifest を
// 直接走査するが、祖先 symlink / 構造不一致は planner の conflict ゲートで apply 前に弾かれている。
func (a *applier) recopyAll() error {
	for _, e := range a.manifest.Entries {
		if e.Method != manifest.MethodCopy {
			continue
		}
		targetAbs := filepath.Join(a.root, filepath.Clean(e.Target))
		existed := false
		if _, err := os.Lstat(targetAbs); err == nil {
			existed = true
			if err := os.RemoveAll(targetAbs); err != nil {
				return fmt.Errorf("nput: recopy 対象を削除できません (%s): %w", targetAbs, err)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("nput: recopy target を lstat できません (%s): %w", targetAbs, err)
		}

		if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
			return fmt.Errorf("nput: 親ディレクトリを作成できません (%s): %w", filepath.Dir(targetAbs), err)
		}
		if err := copyTree(planner.LinkDest(e), targetAbs); err != nil {
			return fmt.Errorf("nput: recopy に失敗しました (%s -> %s): %w", planner.LinkDest(e), targetAbs, err)
		}
		if existed {
			a.result.Recopied = append(a.result.Recopied, e.Target)
		} else {
			a.result.Copied = append(a.result.Copied, e.Target)
		}
	}
	return nil
}

// copyTree は src（ファイル / ディレクトリ / symlink）を dst へネイティブにコピーする
// （→ ADR-0016, docs/spec.md「copy モード」）。
//   - mode は保存しつつ owner-write（0o200）を付与する（store の read-only 0444/0555 →
//     編集可能な 0644/0755）。
//   - src ツリー内の symlink は deref せず symlink のまま複製する（循環 / サイズ膨張回避）。
func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("copy src を lstat できません (%s): %w", src, err)
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return copySymlink(src, dst)
	case !info.IsDir():
		return copyFile(src, dst, info.Mode())
	}

	return filepath.WalkDir(src, func(path string, d iofs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, rel)
		fi, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case fi.Mode()&os.ModeSymlink != 0:
			return copySymlink(path, dstPath)
		case d.IsDir():
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			// umask に依らず最終 mode を確定する（owner-write 付与）。
			return os.Chmod(dstPath, fi.Mode().Perm()|0o200)
		default:
			return copyFile(path, dstPath, fi.Mode())
		}
	})
}

// copyFile は単一ファイルを内容コピーし、mode 保存 + owner-write を付与する。
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	perm := mode.Perm() | 0o200
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	// OpenFile の mode は umask でマスクされるため、最終 perm を明示確定する。
	return os.Chmod(dst, perm)
}

// copySymlink は symlink を deref せず、同じリンク先で複製する（→ ADR-0016）。
func copySymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	// 再コピー時の残骸対策（recopy は親を RemoveAll 済みだが防御的に消す）。
	_ = os.Remove(dst)
	return os.Symlink(target, dst)
}
