package cmd

import (
	"testing"
)

func TestRDSCommands(t *testing.T) {
	// Test that RDS commands are properly registered
	if rdsCmd == nil {
		t.Error("rdsCmd should not be nil")
	}

	if rdsConnectCmd == nil {
		t.Error("rdsConnectCmd should not be nil")
	}
}

func TestRDSCommand(t *testing.T) {
	// Test command properties
	if rdsCmd.Use != "rds" {
		t.Errorf("Expected Use 'rds', got '%s'", rdsCmd.Use)
	}

	if rdsCmd.Short == "" {
		t.Error("rdsCmd should have Short description")
	}

	if rdsCmd.Long == "" {
		t.Error("rdsCmd should have Long description")
	}
}

func TestRDSConnectCommand(t *testing.T) {
	// Test command properties
	if rdsConnectCmd.Use != "connect" {
		t.Errorf("Expected Use 'connect', got '%s'", rdsConnectCmd.Use)
	}

	if rdsConnectCmd.Short == "" {
		t.Error("rdsConnectCmd should have Short description")
	}

	if rdsConnectCmd.Long == "" {
		t.Error("rdsConnectCmd should have Long description")
	}

	if rdsConnectCmd.Run == nil {
		t.Error("rdsConnectCmd should have Run function")
	}
}

func TestRunRDSConnect(t *testing.T) {
	// Test that the function exists by checking command Run field
	if rdsConnectCmd.Run == nil {
		t.Error("rdsConnectCmd.Run should be defined")
	}

	// Business logic is tested in internal/aws package
	// This only tests CLI interface
}

func TestRDSConnectFlags(t *testing.T) {
	// Test that required flags are present
	localPortFlag := rdsConnectCmd.Flags().Lookup("local-port")
	if localPortFlag == nil {
		t.Error("rdsConnectCmd should have --local-port flag")
	}

	nameFlag := rdsConnectCmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Error("rdsConnectCmd should have --name flag")
	}

	// Test flag properties
	if nameFlag != nil {
		if nameFlag.Usage == "" {
			t.Error("--name flag should have usage description")
		}
	}

	if localPortFlag != nil {
		if localPortFlag.Usage == "" {
			t.Error("--local-port flag should have usage description")
		}
	}
}
