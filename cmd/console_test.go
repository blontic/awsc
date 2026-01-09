package cmd

import (
	"testing"
)

func TestConsoleCommand(t *testing.T) {
	// Test that console command is properly registered
	if consoleCmd == nil {
		t.Error("consoleCmd should not be nil")
	}

	// Test command properties
	if consoleCmd.Use != "console" {
		t.Errorf("Expected Use 'console', got '%s'", consoleCmd.Use)
	}

	if consoleCmd.Short == "" {
		t.Error("consoleCmd should have Short description")
	}

	if consoleCmd.Long == "" {
		t.Error("consoleCmd should have Long description")
	}

	if consoleCmd.Run == nil {
		t.Error("consoleCmd should have Run function")
	}
}

func TestConsoleCommandFlags(t *testing.T) {
	// Test service flag
	serviceFlag := consoleCmd.Flags().Lookup("service")
	if serviceFlag == nil {
		t.Error("--service flag should be defined for console command")
	}

	if serviceFlag.DefValue != "" {
		t.Errorf("Expected service flag default to be empty, got '%s'", serviceFlag.DefValue)
	}

	// Test switch-account flag
	switchAccountFlag := consoleCmd.Flags().Lookup("switch-account")
	if switchAccountFlag == nil {
		t.Error("--switch-account flag should be defined for console command")
	}

	if switchAccountFlag.Shorthand != "s" {
		t.Errorf("Expected switch-account flag shorthand to be 's', got '%s'", switchAccountFlag.Shorthand)
	}

	if switchAccountFlag.DefValue != "false" {
		t.Errorf("Expected switch-account flag default to be 'false', got '%s'", switchAccountFlag.DefValue)
	}
}

func TestRunConsole(t *testing.T) {
	// Test that the function exists by checking command Run field
	if consoleCmd.Run == nil {
		t.Error("consoleCmd.Run should be defined")
	}

	// Business logic is tested in internal/aws package
	// This only tests CLI interface
}
