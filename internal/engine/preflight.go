package engine

import (
	"fmt"
	"os"

	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/planner"
)

// checkOutOfStore は out-of-store symlink entry のリンク先（marker のローカル絶対パス +
// subpath = planner.LinkDest）が実在することを配置直前に検査する。不在なら dangling symlink
// を張ることになるため engine 実行時エラーで停止する（「リンク先不在は実行時エラー」・→ ADR-0001,
// ADR-0013, docs/spec.md「out-of-store symlink」）。
//
// 検査は out-of-store のみに閉じる: store link は farm derivation のビルドで実在が保証され、
// copy は別経路（place-once・本検査の対象外）。method=copy × out-of-store marker は
// normalizeManifest が eval 時に弾くため、ここに copy の out-of-store entry は来ない（→ ADR-0013）。
func (a *applier) checkOutOfStore() error {
	for _, e := range a.manifest.Entries {
		if e.SrcKind != manifest.SrcKindOutOfStore {
			continue
		}
		dest := planner.LinkDest(e)
		if _, err := os.Lstat(dest); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("nput: out-of-store のリンク先が存在しません (target: %s -> %s)。dangling symlink は張りません（→ ADR-0001）", e.Target, dest)
			}
			return fmt.Errorf("nput: out-of-store のリンク先を確認できません (target: %s -> %s): %w", e.Target, dest, err)
		}
	}
	return nil
}
