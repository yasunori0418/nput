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

// materializeCopies is the branch point that reflects copy entries onto the real FS.
// On --recopy it overwrites all copy targets unconditionally (recopyAll); normally it
// follows the planner's place-once classification (new copy only when the target is
// absent) (→ ADR-0020). Shared by normal apply and generation-skip drift repair.
func (a *applier) materializeCopies(plan planner.Plan, recopy bool) error {
	if recopy {
		return a.recopyAll()
	}
	return a.placeCopies(plan.Copies)
}

// placeCopies materializes the planner's place-once CopyActions (only copies whose
// target is absent · → ADR-0002, ADR-0016). The planner emits no CopyAction for an
// existing target (recorded / foreign), so this stays a thin executor that reflects
// only "new copies" onto the real FS.
func (a *applier) placeCopies(actions []planner.CopyAction) error {
	for _, act := range actions {
		if err := os.MkdirAll(filepath.Dir(act.TargetAbs), 0o755); err != nil {
			return fmt.Errorf("nput: cannot create parent directory (%s): %w", filepath.Dir(act.TargetAbs), err)
		}
		if err := copyTree(act.Src, act.TargetAbs); err != nil {
			return fmt.Errorf("nput: copy placement failed (%s -> %s): %w", act.Src, act.TargetAbs, err)
		}
		a.result.Copied = append(a.result.Copied, act.Entry.Target)
	}
	return nil
}

// recopyAll is the copy overwrite path of apply --recopy (→ ADR-0020, docs/spec.md "recopy").
// For every copy entry in the config, if the target exists it is removed and then
// re-copied unconditionally (no diff check · local edits are discarded). It does not use
// the place-once classification (planner.Copies) but scans the manifest directly; ancestor
// symlinks / structural mismatches are already rejected by the planner's conflict gate before apply.
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
				return fmt.Errorf("nput: cannot remove recopy target (%s): %w", targetAbs, err)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("nput: cannot lstat recopy target (%s): %w", targetAbs, err)
		}

		if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
			return fmt.Errorf("nput: cannot create parent directory (%s): %w", filepath.Dir(targetAbs), err)
		}
		if err := copyTree(planner.LinkDest(e), targetAbs); err != nil {
			return fmt.Errorf("nput: recopy failed (%s -> %s): %w", planner.LinkDest(e), targetAbs, err)
		}
		if existed {
			a.result.Recopied = append(a.result.Recopied, e.Target)
		} else {
			a.result.Copied = append(a.result.Copied, e.Target)
		}
	}
	return nil
}

// copyTree natively copies src (file / directory / symlink) to dst
// (→ ADR-0016, docs/spec.md "copy mode").
//   - Preserves mode while adding owner-write (0o200) (store's read-only 0444/0555 →
//     editable 0644/0755).
//   - Symlinks inside the src tree are duplicated as symlinks without deref (avoids
//     cycles / size blow-up).
func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("cannot lstat copy src (%s): %w", src, err)
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
			// Fix the final mode regardless of umask (add owner-write).
			return os.Chmod(dstPath, fi.Mode().Perm()|0o200)
		default:
			return copyFile(path, dstPath, fi.Mode())
		}
	})
}

// copyFile copies a single file's contents, preserving mode and adding owner-write.
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
	// OpenFile's mode is masked by umask, so set the final perm explicitly.
	return os.Chmod(dst, perm)
}

// copySymlink duplicates a symlink without deref, pointing at the same link target (→ ADR-0016).
func copySymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	// Guard against leftovers on re-copy (recopy has already RemoveAll'd the parent, but remove defensively).
	_ = os.Remove(dst)
	return os.Symlink(target, dst)
}
