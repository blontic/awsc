package cmd

import "testing"

func TestConsoleCommand(t *testing.T) {
	if consoleCmd.Use != "console" || consoleCmd.Short == "" || consoleCmd.Run == nil {
		t.Errorf("console command not set up: %+v", consoleCmd)
	}
	for _, name := range []string{"service", "switch-account"} {
		if consoleCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing --%s flag", name)
		}
	}
	if f := consoleCmd.Flags().Lookup("switch-account"); f != nil && f.Shorthand != "s" {
		t.Errorf("--switch-account shorthand = %q, want s", f.Shorthand)
	}
	found := false
	for _, c := range rootCmd.Commands() {
		found = found || c == consoleCmd
	}
	if !found {
		t.Error("console command not registered on root")
	}
}
