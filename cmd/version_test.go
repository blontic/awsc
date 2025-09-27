package cmd

import (
	"testing"
)

func TestVersionCommand(t *testing.T) {
	// Test that version command is properly configured
	if versionCmd == nil {
		t.Error("versionCmd should not be nil")
	}

	if versionCmd.Use != "version" {
		t.Errorf("Expected Use 'version', got '%s'", versionCmd.Use)
	}

	if versionCmd.Short == "" {
		t.Error("versionCmd should have Short description")
	}

	if versionCmd.Run == nil {
		t.Error("versionCmd should have Run function")
	}
}

func TestVersionVariables(t *testing.T) {
	// Test that version variables are defined with default values
	if Version == "" {
		t.Error("Version should have a default value")
	}

	if Commit == "" {
		t.Error("Commit should have a default value")
	}

	if Date == "" {
		t.Error("Date should have a default value")
	}

	// Test default values
	if Version != "dev" {
		t.Errorf("Expected default Version 'dev', got '%s'", Version)
	}

	if Commit != "unknown" {
		t.Errorf("Expected default Commit 'unknown', got '%s'", Commit)
	}

	if Date != "unknown" {
		t.Errorf("Expected default Date 'unknown', got '%s'", Date)
	}
}
