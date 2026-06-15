package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yasunori0418/nput/internal/planner"
)

// place materializes the planner's Place actions as native symlinks
// （新規 / 張替を stale 除去より先に・→ ADR-0006）。プランは planner.Compute が
// 現 FS 状態から算出済みで、ここは plan を実 FS に反映する薄い executor に徹する。
// 本スライスは store / out-of-store の symlink 配置のみ（copy は将来スライス・→ Issue #6）。
func (a *applier) place(actions []planner.PlaceAction) error {
	for _, act := range actions {
		// 親ディレクトリを作成（祖先 symlink は planner が conflict として弾き済み・→ ADR-0015）。
		if err := os.MkdirAll(filepath.Dir(act.TargetAbs), 0o755); err != nil {
			return fmt.Errorf("nput: 親ディレクトリを作成できません (%s): %w", filepath.Dir(act.TargetAbs), err)
		}

		switch act.Kind {
		case planner.PlaceNew:
			a.result.Placed = append(a.result.Placed, act.Entry.Target)
		case planner.PlaceReplace, planner.PlaceForeign:
			// 張替は unlink + symlink（rename ベースの atomic swap は採らない・→ ADR-0017）。
			// foreign 上書きの warning は planner.Warnings 経由で emitWarnings 済み（→ ADR-0015）。
			if err := os.Remove(act.TargetAbs); err != nil {
				return fmt.Errorf("nput: 既存 symlink を除去できません (%s): %w", act.TargetAbs, err)
			}
			a.result.Replaced = append(a.result.Replaced, act.Entry.Target)
		}

		if err := os.Symlink(act.Dest, act.TargetAbs); err != nil {
			return fmt.Errorf("nput: symlink を作成できません (%s -> %s): %w", act.TargetAbs, act.Dest, err)
		}
	}
	return nil
}
