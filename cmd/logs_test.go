package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestLogsCommands(t *testing.T) {
	// Test logs command exists
	if logsCmd.Use != "logs" {
		t.Errorf("Expected logs command use to be 'logs', got '%s'", logsCmd.Use)
	}

	// Test subcommands exist
	subcommands := logsCmd.Commands()
	if len(subcommands) != 1 {
		t.Errorf("Expected 1 subcommand, got %d", len(subcommands))
	}

	// Test tail command and flags
	var tailCmd *cobra.Command
	for _, cmd := range subcommands {
		if cmd.Use == "tail" {
			tailCmd = cmd
			break
		}
	}
	if tailCmd == nil {
		t.Error("Expected tail subcommand to exist")
	}

	if tailCmd != nil {
		// Test flags exist
		groupFlag := tailCmd.Flags().Lookup("group")
		if groupFlag == nil {
			t.Error("Expected --group flag to exist")
		}

		sinceFlag := tailCmd.Flags().Lookup("since")
		if sinceFlag == nil {
			t.Error("Expected --since flag to exist")
		}

		followFlag := tailCmd.Flags().Lookup("follow")
		if followFlag == nil {
			t.Error("Expected --follow flag to exist")
		}
	}
}
