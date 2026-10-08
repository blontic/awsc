package cmd

import "testing"

func TestLogoutCommand(t *testing.T) {
	if logoutCmd.Use != "logout" || logoutCmd.Run == nil {
		t.Errorf("logout command not set up: %+v", logoutCmd)
	}
	if logoutCmd.Flags().Lookup("all") == nil {
		t.Error("missing --all flag")
	}
}
