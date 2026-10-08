package cmd

import (
	"testing"
)

func TestConfigCommands(t *testing.T) {
	// Test that config commands are properly registered
	if configCmd == nil {
		t.Error("configCmd should not be nil")
	}

	if configAddCmd == nil {
		t.Error("configAddCmd should not be nil")
	}

	if configShowCmd == nil {
		t.Error("configShowCmd should not be nil")
	}
}

func TestConfigAddCommand(t *testing.T) {
	if configAddCmd.Use != "add [name]" {
		t.Errorf("Expected Use 'add [name]', got '%s'", configAddCmd.Use)
	}
	if configAddCmd.Run == nil {
		t.Error("configAddCmd should have Run function")
	}
	// Config commands must skip the root pre-run (no active org required)
	if configCmd.PersistentPreRun == nil {
		t.Error("configCmd should have PersistentPreRun to skip root pre-run checks")
	}
	if !configAddCmd.HasAlias("init") {
		t.Error("'config init' should remain as an alias of 'config add' for backward compatibility")
	}
}

func TestConfigShowCommand(t *testing.T) {
	// Test command properties
	if configShowCmd.Use != "show [name]" {
		t.Errorf("Expected Use 'show', got '%s'", configShowCmd.Use)
	}

	if configShowCmd.Short == "" {
		t.Error("configShowCmd should have Short description")
	}

	if configShowCmd.Run == nil {
		t.Error("configShowCmd should have Run function")
	}
}
