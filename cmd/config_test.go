package cmd

import (
	"testing"
)

func TestConfigCommands(t *testing.T) {
	// Test that config commands are properly registered
	if configCmd == nil {
		t.Error("configCmd should not be nil")
	}

	if configInitCmd == nil {
		t.Error("configInitCmd should not be nil")
	}

	if configShowCmd == nil {
		t.Error("configShowCmd should not be nil")
	}
}

func TestConfigInitCommand(t *testing.T) {
	// Test command properties
	if configInitCmd.Use != "init" {
		t.Errorf("Expected Use 'init', got '%s'", configInitCmd.Use)
	}

	if configInitCmd.Short == "" {
		t.Error("configInitCmd should have Short description")
	}

	if configInitCmd.Run == nil {
		t.Error("configInitCmd should have Run function")
	}

	// Test that PreRun is set to skip persistent pre-run
	if configInitCmd.PreRun == nil {
		t.Error("configInitCmd should have PreRun to skip persistent pre-run")
	}
}

func TestConfigShowCommand(t *testing.T) {
	// Test command properties
	if configShowCmd.Use != "show" {
		t.Errorf("Expected Use 'show', got '%s'", configShowCmd.Use)
	}

	if configShowCmd.Short == "" {
		t.Error("configShowCmd should have Short description")
	}

	if configShowCmd.Run == nil {
		t.Error("configShowCmd should have Run function")
	}
}
