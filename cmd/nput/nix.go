package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// entrypoint is a discovered flake entrypoint. This slice supports only flake.nix
// (legacy shell.nix / default.nix are a future slice; → ADR-0007, ADR-0023 §4).
type entrypoint struct {
	// flakeRef is the flake ref passed to `nix build`/`nix eval` (the absolute path of the directory containing flake.nix).
	flakeRef string
}

// discoverEntrypoint discovers the entrypoint in the order -f explicit → CWD autodiscovery
// (→ docs/spec.md "entrypoint discovery"). This slice accepts only flake.nix.
func discoverEntrypoint(fileFlag string) (*entrypoint, error) {
	if fileFlag != "" {
		abs, err := filepath.Abs(fileFlag)
		if err != nil {
			return nil, fmt.Errorf("nput: cannot resolve the -f path (%s): %w", fileFlag, err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("nput: -f path not found (%s): %w", fileFlag, err)
		}
		if info.IsDir() {
			if fileExists(filepath.Join(abs, "flake.nix")) {
				return &entrypoint{flakeRef: abs}, nil
			}
			return nil, fmt.Errorf("nput: no flake.nix in the -f directory (%s). This slice supports only a flake entrypoint", abs)
		}
		switch filepath.Base(abs) {
		case "flake.nix":
			return &entrypoint{flakeRef: filepath.Dir(abs)}, nil
		default:
			return nil, fmt.Errorf("nput: -f must point to a flake.nix (%s). shell.nix / default.nix are not supported in this slice", abs)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("nput: cannot get the current working directory: %w", err)
	}
	if fileExists(filepath.Join(cwd, "flake.nix")) {
		return &entrypoint{flakeRef: cwd}, nil
	}
	// If a legacy entrypoint is found, stop and make clear it is unsupported.
	for _, legacy := range []string{"shell.nix", "default.nix"} {
		if fileExists(filepath.Join(cwd, legacy)) {
			return nil, fmt.Errorf("nput: %s is not supported in this slice (only flake.nix is supported; pass a flake with -f)", legacy)
		}
	}
	return nil, errors.New("nput: no entrypoint found (no flake.nix in the CWD; specify one with -f)")
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// currentSystem returns the nix system name of the runtime environment (e.g. aarch64-darwin).
// Because the flake has a system dimension in `nput.<system>.<name>`, the CLI injects the current system (→ ADR-0007).
func currentSystem() (string, error) {
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	default:
		return "", fmt.Errorf("nput: unsupported GOARCH: %s", runtime.GOARCH)
	}
	switch runtime.GOOS {
	case "linux", "darwin":
		return arch + "-" + runtime.GOOS, nil
	default:
		return "", fmt.Errorf("nput: unsupported GOOS: %s", runtime.GOOS)
	}
}

// installable builds the `<flakeRef>#nput.<system>.<name>` passed to `nix build`/`nix eval`.
func (e *entrypoint) installable(system, name string) string {
	return fmt.Sprintf("%s#nput.%s.%s", e.flakeRef, system, name)
}

// namespace builds `<flakeRef>#nput.<system>` (the config set) without a config name.
// It is used for the batch eval of apply --all / gitignore --all (config name → rootKind map; → ADR-0024).
func (e *entrypoint) namespace(system string) string {
	return fmt.Sprintf("%s#nput.%s", e.flakeRef, system)
}

// rootInfo is one config's root info (the value from the batch eval). It has Root only when fixed.
type rootInfo struct {
	RootKind string `json:"rootKind"`
	Root     string `json:"root"`
}

// evalAllRoots gets the config name → rootInfo map for `apply --all` / `gitignore --all`
// in a single `nix eval` (fixing eval process launches at N→1; → docs/spec.md execution flow, ADR-0024).
// It is a cheap eval that does no build and reads only the passthru rootKind (+ root for fixed).
func evalAllRoots(e *entrypoint, system string) (map[string]rootInfo, error) {
	// Extract only rootKind (+ root if fixed) from each config under nput.<system>.
	apply := `cs: builtins.mapAttrs (_: c: { rootKind = c.rootKind; } // (if c ? root then { root = c.root; } else {})) cs`
	out, err := runNixCapture("eval", e.namespace(system), "--apply", apply, "--json")
	if err != nil {
		return nil, wrapEvalAllErr(err, system)
	}
	var roots map[string]rootInfo
	if err := json.Unmarshal([]byte(out), &roots); err != nil {
		return nil, fmt.Errorf("nput: cannot parse the batch eval result for nput.%s: %w", system, err)
	}
	return roots, nil
}

// buildManifestStorePath builds the config and returns the link-farm's store path (a read-only path).
// Because gitignore does no placement, it gets only the store path via `--no-link --print-out-paths` without laying down an out-link gcroot.
// Progress goes to stderr and the store path to stdout (→ docs/spec.md output stream discipline).
func buildManifestStorePath(e *entrypoint, system, name string) (string, error) {
	out, err := runNixCapture("build", e.installable(system, name), "--no-link", "--print-out-paths")
	if err != nil {
		return "", err
	}
	store := strings.TrimSpace(out)
	if store == "" {
		return "", fmt.Errorf("nput: cannot obtain the build output path for nput.%s.%s", system, name)
	}
	return store, nil
}

// evalRoot pre-resolves rootKind (+ the absolute path when fixed root) via a cheap nix eval before build
// (→ docs/spec.md execution flow 1, ADR-0023). This resolves profileDir and establishes the order flock → build.
func evalRoot(e *entrypoint, system, name string) (rootKind, fixedRoot string, err error) {
	inst := e.installable(system, name)
	out, err := runNixCapture("eval", inst+".rootKind", "--raw")
	if err != nil {
		return "", "", wrapEvalErr(err, system, name)
	}
	rootKind = strings.TrimSpace(out)
	if rootKind == "fixed" {
		out, err := runNixCapture("eval", inst+".root", "--raw")
		if err != nil {
			return "", "", wrapEvalErr(err, system, name)
		}
		fixedRoot = strings.TrimSpace(out)
	}
	return rootKind, fixedRoot, nil
}

// buildFunc returns the build callback injected into the engine (→ engine.BuildFunc).
// Inside the lock it runs `nix build <installable> --out-link <pending>`, reads the out-link, and returns the store path.
func buildFunc(e *entrypoint, system, name string) func(pending string) (string, error) {
	inst := e.installable(system, name)
	return func(pending string) (string, error) {
		if err := runNixStream("build", inst, "--out-link", pending); err != nil {
			return "", err
		}
		store, err := os.Readlink(pending)
		if err != nil {
			return "", fmt.Errorf("nput: cannot read the build output out-link (%s): %w", pending, err)
		}
		return store, nil
	}
}

// dryBuildFunc returns the build callback for --dryrun (→ engine.BuildFunc). Unlike a normal build,
// it gets the link-farm's store path via `nix build --no-link --print-out-paths` **without laying down a gcroot (out-link)**
// (dryrun is side-effect-free and creates no pending out-link; → ADR-0011, ADR-0023). The pending argument is unused.
func dryBuildFunc(e *entrypoint, system, name string) func(pending string) (string, error) {
	inst := e.installable(system, name)
	return func(string) (string, error) {
		out, err := runNixCapture("build", inst, "--no-link", "--print-out-paths")
		if err != nil {
			return "", err
		}
		store := strings.TrimSpace(out)
		if store == "" {
			return "", fmt.Errorf("nput: nix build --print-out-paths was empty (%s)", inst)
		}
		// --print-out-paths may return multiple lines (multi-output). The link-farm is a single output, so take the last line.
		lines := strings.Split(store, "\n")
		return strings.TrimSpace(lines[len(lines)-1]), nil
	}
}

// runNixCapture captures and returns nix's stdout (for machine-readable output such as eval).
func runNixCapture(args ...string) (string, error) {
	if flagDebug {
		fmt.Fprintf(os.Stderr, "nput: + nix %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("nix", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", nixError(args, stderr.String(), err)
	}
	return stdout.String(), nil
}

// runNixStream streams nix's output to stderr (for build progress; stdout is reserved for machine-readable output; → ADR-0023).
// eval succeeded before build = nix-command/flakes are already enabled, so experimental detection is unnecessary.
func runNixStream(args ...string) error {
	if flagDebug {
		fmt.Fprintf(os.Stderr, "nput: + nix %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("nix", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nput: nix %s failed: %w", args[0], err)
	}
	return nil
}

// nixError classifies a nix failure. For experimental-features not enabled it guides the prerequisites,
// and otherwise it returns the raw nix stderr attached without swallowing it (→ ADR-0025 §1).
func nixError(args []string, stderr string, runErr error) error {
	if isExperimentalDisabled(stderr) {
		return experimentalGuidance(stderr)
	}
	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" {
		return fmt.Errorf("nput: nix %s failed: %w", args[0], runErr)
	}
	return fmt.Errorf("nput: nix %s failed:\n%s", args[0], trimmed)
}

// isExperimentalDisabled detects the nix-command / flakes not-enabled error (→ ADR-0025 §1).
func isExperimentalDisabled(stderr string) bool {
	return strings.Contains(stderr, "experimental Nix feature") ||
		strings.Contains(stderr, "experimental-features") ||
		(strings.Contains(stderr, "flakes") && strings.Contains(stderr, "disabled"))
}

// experimentalGuidance builds an error that guides the prerequisites and how to enable them (attaching the raw nix error too).
// The CLI does not add --extra-experimental-features automatically (it will not silently override environment settings; → ADR-0025 §1).
func experimentalGuidance(stderr string) error {
	return fmt.Errorf(`nput: nix's experimental-features are not enabled.
This command internally uses `+"`nix eval`"+` / `+"`nix build`"+` (the new CLI) and flakes,
so experimental-features = nix-command flakes is required.

How to enable (either one):
  - Append to ~/.config/nix/nix.conf or /etc/nix/nix.conf:
      experimental-features = nix-command flakes
  - Temporarily via an environment variable:
      export NIX_CONFIG="experimental-features = nix-command flakes"

nput does not add --extra-experimental-features automatically (it will not override your environment settings).

Original nix error:
%s`, strings.TrimSpace(stderr))
}

// wrapEvalErr makes the "nput.<name> does not exist" case of an eval failure clearer
// (experimental etc. are passed through as-is) (→ docs/spec.md error spec).
func wrapEvalErr(err error, system, name string) error {
	msg := err.Error()
	if strings.Contains(msg, "does not provide attribute") ||
		(strings.Contains(msg, "attribute") && strings.Contains(msg, "missing")) {
		return fmt.Errorf("nput: nput.%s.%s not found in the entrypoint (check the config name and system)\n%s", system, name, msg)
	}
	return err
}

// wrapEvalAllErr makes the "nput.<system> does not exist" case of a batch eval failure clearer.
func wrapEvalAllErr(err error, system string) error {
	msg := err.Error()
	if strings.Contains(msg, "does not provide attribute") ||
		(strings.Contains(msg, "attribute") && strings.Contains(msg, "missing")) {
		return fmt.Errorf("nput: nput.%s not found in the entrypoint (no configs for this system)\n%s", system, msg)
	}
	return err
}
