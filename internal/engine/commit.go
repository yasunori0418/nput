package engine

import (
	"fmt"
	"os"
	"os/exec"
)

// nixEnvCommit は既定のコミット点。`nix-env --profile <profileLink> --set <linkFarm>`
// で 1 世代 = 1 link-farm の atomic 置換を行う（→ ADR-0002, ADR-0006, ADR-0015）。
// stdout は機械可読出力を専有するため nix の出力は stderr に流す（→ docs/spec.md ストリーム規律）。
func nixEnvCommit(profileLink, linkFarm string) error {
	if _, err := exec.LookPath("nix-env"); err != nil {
		return fmt.Errorf("nix-env が PATH にありません: %w", err)
	}
	cmd := exec.Command("nix-env", "--profile", profileLink, "--set", linkFarm)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
