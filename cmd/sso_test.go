package cmd

import (
	"testing"
)

func TestSSOCommands(t *testing.T) {
	// Test that SSO commands are properly registered
	if loginCmd == nil {
		t.Error("loginCmd should not be nil")
	}
}

func TestLoginCommand(t *testing.T) {
	// Test command properties
	if loginCmd.Use != "login" {
		t.Errorf("Expected Use 'login', got '%s'", loginCmd.Use)
	}

	if loginCmd.Short == "" {
		t.Error("loginCmd should have Short description")
	}

	if loginCmd.Long == "" {
		t.Error("loginCmd should have Long description")
	}

	if loginCmd.Run == nil {
		t.Error("loginCmd should have Run function")
	}
}

func TestLoginCommandFlags(t *testing.T) {
	// Test that force flag is defined
	forceFlag := loginCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Error("--force flag should be defined for login command")
	}

	if forceFlag.DefValue != "false" {
		t.Errorf("Expected force flag default to be 'false', got '%s'", forceFlag.DefValue)
	}

	if forceFlag.Usage == "" {
		t.Error("Force flag should have usage description")
	}

	// Test that account flag is defined
	accountFlag := loginCmd.Flags().Lookup("account")
	if accountFlag == nil {
		t.Error("--account flag should be defined for login command")
	}

	if accountFlag.DefValue != "" {
		t.Errorf("Expected account flag default to be empty, got '%s'", accountFlag.DefValue)
	}

	if accountFlag.Usage == "" {
		t.Error("Account flag should have usage description")
	}

	// Test that role flag is defined
	roleFlag := loginCmd.Flags().Lookup("role")
	if roleFlag == nil {
		t.Error("--role flag should be defined for login command")
	}

	if roleFlag.DefValue != "" {
		t.Errorf("Expected role flag default to be empty, got '%s'", roleFlag.DefValue)
	}

	if roleFlag.Usage == "" {
		t.Error("Role flag should have usage description")
	}
}

func TestRunLogin(t *testing.T) {
	// Test that the function exists by checking command Run field
	if loginCmd.Run == nil {
		t.Error("loginCmd.Run should be defined")
	}

	// Business logic is tested in internal/aws package
	// This only tests CLI interface
}
