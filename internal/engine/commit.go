package engine

import (
	"fmt"
	"os"
	"os/exec"
)

// nixEnvCommit is the default commit point. It performs an atomic swap of one
// generation = one link-farm via `nix-env --profile <profileLink> --set <linkFarm>`
// (→ ADR-0002, ADR-0006, ADR-0015). stdout is reserved for machine-readable output,
// so nix output is routed to stderr (→ docs/spec.md stream discipline).
func nixEnvCommit(profileLink, linkFarm string) error {
	if _, err := exec.LookPath("nix-env"); err != nil {
		return fmt.Errorf("nix-env is not on PATH: %w", err)
	}
	cmd := exec.Command("nix-env", "--profile", profileLink, "--set", linkFarm)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
