package engine

import (
	"fmt"
	"os"

	"github.com/yasunori0418/nput/internal/manifest"
	"github.com/yasunori0418/nput/internal/planner"
)

// checkOutOfStore checks, just before placement, that the link target of out-of-store symlink
// entries (the marker's local absolute path + subpath = planner.LinkDest) actually exists. If
// absent, placing it would create a dangling symlink, so it stops with an engine runtime error
// ("an absent link target is a runtime error" · → ADR-0001, ADR-0013, docs/spec.md "out-of-store symlink").
//
// The check is closed to out-of-store only: store links are guaranteed to exist by the farm
// derivation build, and copy goes through a different path (place-once · outside this check).
// method=copy × out-of-store marker is rejected by normalizeManifest at eval time, so no copy
// out-of-store entry reaches here (→ ADR-0013).
func (a *applier) checkOutOfStore() error {
	for _, e := range a.manifest.Entries {
		if e.SrcKind != manifest.SrcKindOutOfStore {
			continue
		}
		dest := planner.LinkDest(e)
		if _, err := os.Lstat(dest); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("nput: out-of-store link target does not exist (target: %s -> %s); will not create a dangling symlink (→ ADR-0001)", e.Target, dest)
			}
			return fmt.Errorf("nput: cannot check out-of-store link target (target: %s -> %s): %w", e.Target, dest, err)
		}
	}
	return nil
}
