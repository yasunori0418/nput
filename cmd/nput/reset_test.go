package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yasunori0418/nput/internal/engine"
)

// confirmPolicy は --yes / TTY 状態から確認方針を決める（→ ADR-0025 §5）。
// 非対話 + --yes 無しの破壊的 reset 拒否（#13 AC-5）の回帰防止。
func TestConfirmPolicy(t *testing.T) {
	cases := []struct {
		name        string
		yes         bool
		interactive bool
		wantPrompt  bool
		wantErr     bool
	}{
		{"--yes はスキップ（非対話でも）", true, false, false, false},
		{"--yes はスキップ（対話でも）", true, true, false, false},
		{"対話 + --yes 無しはプロンプト", false, true, true, false},
		{"非対話 + --yes 無しは拒否", false, false, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotPrompt, err := confirmPolicy(c.yes, c.interactive)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if gotPrompt != c.wantPrompt {
				t.Errorf("needPrompt = %v, want %v", gotPrompt, c.wantPrompt)
			}
		})
	}
}

// promptYesNo は yes 系入力のときだけ true を返す（既定 No）。stdin を差し替えて検証する。
func TestPromptYesNo(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"yes\n", true},
		{"Y\n", true},
		{"YES\n", true},
		{"n\n", false},
		{"\n", false},      // 既定 No
		{"maybe\n", false}, // 不明な入力は No
		{"", false},        // EOF（パイプ閉じ）も No
	}
	for _, c := range cases {
		t.Run(strings.TrimSpace(c.input)+"->"+boolStr(c.want), func(t *testing.T) {
			restore := withStdin(t, c.input)
			defer restore()
			got, err := promptYesNo("続行しますか？")
			if err != nil {
				t.Fatalf("promptYesNo err = %v", err)
			}
			if got != c.want {
				t.Errorf("promptYesNo(%q) = %v, want %v", c.input, got, c.want)
			}
		})
	}
}

// isInteractive はパイプ / リダイレクト stdin では false を返す（非 TTY 経路・→ ADR-0025 §5）。
// TTY=true 経路は実端末が要るため CI では検証できず、false 経路のみ検証する。
func TestIsInteractiveNonTTY(t *testing.T) {
	restore := withStdin(t, "")
	defer restore()
	if isInteractive() {
		t.Error("パイプ stdin では isInteractive() は false であるべき")
	}
}

// reset の出力ストリーム規律（#13 AC-9）: dryrun plan は stdout 専有、
// 削除予定 / 結果レポートは stderr。
func TestResetOutputStreams(t *testing.T) {
	res := &engine.ResetResult{
		Root:            "/root",
		RemovedSymlinks: []string{"/root/a"},
		RemovedCopies:   []string{"/root/b"},
		KeptForeign:     []string{"/root/c"},
	}

	t.Run("printResetPlan は stdout 専有", func(t *testing.T) {
		out, errOut := captureOutErr(t, func() { printResetPlan(res) })
		if !strings.Contains(out, "remove-symlink\t/root/a") ||
			!strings.Contains(out, "remove-copy\t/root/b") ||
			!strings.Contains(out, "keep-foreign\t/root/c") {
			t.Errorf("plan が stdout に出ていない: %q", out)
		}
		if errOut != "" {
			t.Errorf("plan で stderr に出力があるべきでない: %q", errOut)
		}
	})

	t.Run("reportResetResult は stderr 専有", func(t *testing.T) {
		out, errOut := captureOutErr(t, func() { reportResetResult(res, "name") })
		if out != "" {
			t.Errorf("レポートが stdout を汚している: %q", out)
		}
		if !strings.Contains(errOut, "removed-symlink") || !strings.Contains(errOut, "/root/a") {
			t.Errorf("レポートが stderr に出ていない: %q", errOut)
		}
	})

	t.Run("reportResetTargets は stderr 専有", func(t *testing.T) {
		out, errOut := captureOutErr(t, func() { reportResetTargets(res, "name") })
		if out != "" {
			t.Errorf("確認表示が stdout を汚している: %q", out)
		}
		if !strings.Contains(errOut, "削除対象") {
			t.Errorf("確認表示が stderr に出ていない: %q", errOut)
		}
	})
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// withStdin は os.Stdin を input を返すパイプに差し替え、復元する関数を返す。
func withStdin(t *testing.T, input string) func() {
	t.Helper()
	old := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	go func() {
		_, _ = io.WriteString(w, input)
		w.Close()
	}()
	os.Stdin = r
	return func() { os.Stdin = old; r.Close() }
}

// captureOutErr は f 実行中の stdout / stderr を捕捉して返す（ストリーム規律の検証用）。
func captureOutErr(t *testing.T, f func()) (stdout, stderr string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout, os.Stderr = wOut, wErr
	outCh, errCh := make(chan string, 1), make(chan string, 1)
	go func() { b, _ := io.ReadAll(rOut); outCh <- string(b) }()
	go func() { b, _ := io.ReadAll(rErr); errCh <- string(b) }()
	f()
	wOut.Close()
	wErr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	return <-outCh, <-errCh
}
