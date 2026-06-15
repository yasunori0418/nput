package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// entrypoint は発見した flake entrypoint。本スライスは flake.nix のみ対応する
// （legacy shell.nix / default.nix は将来スライス・→ ADR-0007, ADR-0023 §4）。
type entrypoint struct {
	// flakeRef は `nix build`/`nix eval` に渡す flake ref（flake.nix を含むディレクトリの絶対パス）。
	flakeRef string
}

// discoverEntrypoint は -f 明示 → CWD 自動探索の順で entrypoint を発見する
// （→ docs/spec.md「entrypoint の発見」）。本スライスは flake.nix のみ受理する。
func discoverEntrypoint(fileFlag string) (*entrypoint, error) {
	if fileFlag != "" {
		abs, err := filepath.Abs(fileFlag)
		if err != nil {
			return nil, fmt.Errorf("nput: -f のパスを解決できません (%s): %w", fileFlag, err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("nput: -f のパスが見つかりません (%s): %w", fileFlag, err)
		}
		if info.IsDir() {
			if fileExists(filepath.Join(abs, "flake.nix")) {
				return &entrypoint{flakeRef: abs}, nil
			}
			return nil, fmt.Errorf("nput: -f ディレクトリに flake.nix がありません (%s)。本スライスは flake entrypoint のみ対応します", abs)
		}
		switch filepath.Base(abs) {
		case "flake.nix":
			return &entrypoint{flakeRef: filepath.Dir(abs)}, nil
		default:
			return nil, fmt.Errorf("nput: -f は flake.nix を指してください (%s)。shell.nix / default.nix は本スライス未対応です", abs)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("nput: cwd を取得できません: %w", err)
	}
	if fileExists(filepath.Join(cwd, "flake.nix")) {
		return &entrypoint{flakeRef: cwd}, nil
	}
	// legacy entrypoint を見つけたら未対応である旨を明示して停止する。
	for _, legacy := range []string{"shell.nix", "default.nix"} {
		if fileExists(filepath.Join(cwd, legacy)) {
			return nil, fmt.Errorf("nput: %s は本スライス未対応です（flake.nix のみ対応・-f で flake を指定してください）", legacy)
		}
	}
	return nil, errors.New("nput: entrypoint が見つかりません（CWD に flake.nix がありません。-f で指定してください）")
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// currentSystem は実行環境の nix system 名（例: aarch64-darwin）を返す。
// flake は `nput.<system>.<name>` で system 次元を持つため CLI が現行 system を差し込む（→ ADR-0007）。
func currentSystem() (string, error) {
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	default:
		return "", fmt.Errorf("nput: 未対応の GOARCH です: %s", runtime.GOARCH)
	}
	switch runtime.GOOS {
	case "linux", "darwin":
		return arch + "-" + runtime.GOOS, nil
	default:
		return "", fmt.Errorf("nput: 未対応の GOOS です: %s", runtime.GOOS)
	}
}

// installable は `nix build`/`nix eval` に渡す `<flakeRef>#nput.<system>.<name>` を組む。
func (e *entrypoint) installable(system, name string) string {
	return fmt.Sprintf("%s#nput.%s.%s", e.flakeRef, system, name)
}

// evalRoot は build 前に rootKind（+ fixed root のときは絶対パス）を安価な nix eval で先取りする
// （→ docs/spec.md 実行フロー 1・ADR-0023）。これで profileDir を確定し flock → build の順を成立させる。
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

// buildFunc は engine に注入する build コールバックを返す（→ engine.BuildFunc）。
// ロック内で `nix build <installable> --out-link <pending>` を実行し、out-link を読んで store path を返す。
func buildFunc(e *entrypoint, system, name string) func(pending string) (string, error) {
	inst := e.installable(system, name)
	return func(pending string) (string, error) {
		if err := runNixStream("build", inst, "--out-link", pending); err != nil {
			return "", err
		}
		store, err := os.Readlink(pending)
		if err != nil {
			return "", fmt.Errorf("nput: build 成果物の out-link を読めません (%s): %w", pending, err)
		}
		return store, nil
	}
}

// runNixCapture は nix の stdout を捕捉して返す（eval 等の機械可読出力用）。
func runNixCapture(args ...string) (string, error) {
	if flagVerbose {
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

// runNixStream は nix の出力を stderr へ流す（build 進捗用。stdout は機械可読出力に専有・→ ADR-0023）。
// build 前に eval が成功している＝nix-command/flakes は有効化済みなので experimental 検出は不要。
func runNixStream(args ...string) error {
	if flagVerbose {
		fmt.Fprintf(os.Stderr, "nput: + nix %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("nix", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nput: nix %s に失敗しました: %w", args[0], err)
	}
	return nil
}

// nixError は nix の失敗を分類する。experimental-features 未有効は前提条件を案内し、
// それ以外は生の nix stderr を握り潰さず添えて返す（→ ADR-0025 §1）。
func nixError(args []string, stderr string, runErr error) error {
	if isExperimentalDisabled(stderr) {
		return experimentalGuidance(stderr)
	}
	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" {
		return fmt.Errorf("nput: nix %s に失敗しました: %w", args[0], runErr)
	}
	return fmt.Errorf("nput: nix %s に失敗しました:\n%s", args[0], trimmed)
}

// isExperimentalDisabled は nix-command / flakes 未有効エラーを検出する（→ ADR-0025 §1）。
func isExperimentalDisabled(stderr string) bool {
	return strings.Contains(stderr, "experimental Nix feature") ||
		strings.Contains(stderr, "experimental-features") ||
		(strings.Contains(stderr, "flakes") && strings.Contains(stderr, "disabled"))
}

// experimentalGuidance は前提条件と有効化方法を案内するエラーを組む（生の nix エラーも添える）。
// CLI は --extra-experimental-features を自動付与しない（環境設定を黙って上書きしない・→ ADR-0025 §1）。
func experimentalGuidance(stderr string) error {
	return fmt.Errorf(`nput: nix の experimental-features が有効化されていません。
本コマンドは内部で `+"`nix eval`"+` / `+"`nix build`"+`（新 CLI）と flake を使うため、
experimental-features = nix-command flakes が必要です。

有効化方法（いずれか）:
  - ~/.config/nix/nix.conf または /etc/nix/nix.conf に追記:
      experimental-features = nix-command flakes
  - 一時的に環境変数で:
      export NIX_CONFIG="experimental-features = nix-command flakes"

nput は --extra-experimental-features を自動付与しません（環境設定を上書きしないため）。

元の nix エラー:
%s`, strings.TrimSpace(stderr))
}

// wrapEvalErr は eval の失敗のうち「nput.<name> が無い」ケースを分かりやすくする
// （experimental 等はそのまま）（→ docs/spec.md エラー仕様）。
func wrapEvalErr(err error, system, name string) error {
	msg := err.Error()
	if strings.Contains(msg, "does not provide attribute") ||
		(strings.Contains(msg, "attribute") && strings.Contains(msg, "missing")) {
		return fmt.Errorf("nput: entrypoint に nput.%s.%s が見つかりません（config 名と system を確認してください）\n%s", system, name, msg)
	}
	return err
}
