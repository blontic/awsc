package cmd

import (
	"testing"
)

func TestSecretsCommands(t *testing.T) {
	// Test that Secrets commands are properly registered
	if secretsCmd == nil {
		t.Error("secretsCmd should not be nil")
	}

	if secretsListCmd == nil {
		t.Error("secretsListCmd should not be nil")
	}
}

func TestSecretsCommand(t *testing.T) {
	// Test command properties
	if secretsCmd.Use != "secrets" {
		t.Errorf("Expected Use 'secrets', got '%s'", secretsCmd.Use)
	}

	if secretsCmd.Short == "" {
		t.Error("secretsCmd should have Short description")
	}

	// secretsCmd doesn't have Long description, only Short
}

func TestSecretsListCommand(t *testing.T) {
	// Test command properties
	if secretsListCmd.Use != "list" {
		t.Errorf("Expected Use 'list', got '%s'", secretsListCmd.Use)
	}

	if secretsListCmd.Short == "" {
		t.Error("secretsListCmd should have Short description")
	}

	// secretsListCmd doesn't have Long description, only Short

	if secretsListCmd.Run == nil {
		t.Error("secretsListCmd should have Run function")
	}
}

func TestRunSecretsList(t *testing.T) {
	// Test that the function exists by checking command Run field
	if secretsListCmd.Run == nil {
		t.Error("secretsListCmd.Run should be defined")
	}

	// Business logic is tested in internal/aws package
	// This only tests CLI interface
}
