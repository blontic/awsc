package cmd

import (
	"testing"
)

func TestOpenSearchCommand(t *testing.T) {
	// Test that opensearch command is properly initialized
	if opensearchCmd == nil {
		t.Fatal("opensearchCmd should not be nil")
	}

	if opensearchCmd.Use != "opensearch" {
		t.Errorf("Expected opensearch command use to be 'opensearch', got %s", opensearchCmd.Use)
	}

	// Test that connect subcommand exists
	connectCmd := opensearchCmd.Commands()[0]
	if connectCmd.Use != "connect" {
		t.Errorf("Expected connect subcommand use to be 'connect', got %s", connectCmd.Use)
	}
}
