package main

import "testing"

// TestApplyBackupFlagParsing verifies apply --backup[=suffix]'s cobra optional-value wiring
// (→ ADR-0045, issue #169): bare --backup uses the NoOptDefVal default suffix, --backup=<suffix>
// (the "=" form) takes the given suffix, and a bare next token is NOT consumed as the flag's value
// (it is left as a positional argument) — the spec's "=" 区切り必須 requirement.
func TestApplyBackupFlagParsing(t *testing.T) {
	t.Run("absent: Changed is false", func(t *testing.T) {
		cmd := newApplyCmd()
		if err := cmd.ParseFlags([]string{"name"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		if cmd.Flags().Changed("backup") {
			t.Error("Changed(\"backup\") = true, want false when --backup is absent")
		}
	})

	t.Run("bare --backup uses the default suffix", func(t *testing.T) {
		cmd := newApplyCmd()
		if err := cmd.ParseFlags([]string{"--backup", "name"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		if !cmd.Flags().Changed("backup") {
			t.Error("Changed(\"backup\") = false, want true")
		}
		got, err := cmd.Flags().GetString("backup")
		if err != nil {
			t.Fatalf("GetString(backup): %v", err)
		}
		if got != "nput-backup" {
			t.Errorf("bare --backup value = %q, want %q (NoOptDefVal)", got, "nput-backup")
		}
		// The bare form must not swallow the next token as its value (space-separated form
		// rejected by design): "name" must remain a positional arg, not the flag's value.
		if args := cmd.Flags().Args(); len(args) != 1 || args[0] != "name" {
			t.Errorf("positional args after bare --backup = %v, want [name] (next token must not be consumed as the flag value)", args)
		}
	})

	t.Run("--backup=<suffix> takes the custom suffix", func(t *testing.T) {
		cmd := newApplyCmd()
		if err := cmd.ParseFlags([]string{"--backup=bak", "name"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		if !cmd.Flags().Changed("backup") {
			t.Error("Changed(\"backup\") = false, want true")
		}
		got, err := cmd.Flags().GetString("backup")
		if err != nil {
			t.Fatalf("GetString(backup): %v", err)
		}
		if got != "bak" {
			t.Errorf("--backup=bak value = %q, want %q", got, "bak")
		}
	})
}
