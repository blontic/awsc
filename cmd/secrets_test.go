package cmd

import (
	"testing"
)

func TestSecretsCommands(t *testing.T) {
	// Test that Secrets commands are properly registered
	if secretsCmd == nil {
		t.Error("secretsCmd should not be nil")
	}

	if secretsShowCmd == nil {
		t.Error("secretsShowCmd should not be nil")
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

func TestSecretsShowCommand(t *testing.T) {
	// Test command properties
	if secretsShowCmd.Use != "show" {
		t.Errorf("Expected Use 'show', got '%s'", secretsShowCmd.Use)
	}

	if secretsShowCmd.Short == "" {
		t.Error("secretsShowCmd should have Short description")
	}

	// secretsShowCmd doesn't have Long description, only Short

	if secretsShowCmd.Run == nil {
		t.Error("secretsShowCmd should have Run function")
	}
}

func TestRunSecretsShow(t *testing.T) {
	// Test that the function exists by checking command Run field
	if secretsShowCmd.Run == nil {
		t.Error("secretsShowCmd.Run should be defined")
	}

	// Business logic is tested in internal/aws package
	// This only tests CLI interface
}

func TestSecretsShowFlags(t *testing.T) {
	// Test that required flags are present
	nameFlag := secretsShowCmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Error("secretsShowCmd should have --name flag")
	}

	// Test flag properties
	if nameFlag != nil {
		if nameFlag.Usage == "" {
			t.Error("--name flag should have usage description")
		}
	}
}
