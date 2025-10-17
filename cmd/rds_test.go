package cmd

import (
	"strings"
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

func TestRDSConnectSwitchAccountFlag(t *testing.T) {
	// Test that switch-account flag is properly defined
	switchAccountFlag := rdsConnectCmd.Flags().Lookup("switch-account")
	if switchAccountFlag == nil {
		t.Error("--switch-account flag should be defined for RDS connect command")
	}

	if switchAccountFlag.Shorthand != "s" {
		t.Errorf("Expected shorthand 's' for switch-account flag, got '%s'", switchAccountFlag.Shorthand)
	}

	if switchAccountFlag.DefValue != "false" {
		t.Errorf("Expected switch-account flag default to be false, got '%s'", switchAccountFlag.DefValue)
	}

	// Test that existing flags are still present
	nameFlag := rdsConnectCmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Error("--name flag should still be defined for RDS connect command")
	}

	localPortFlag := rdsConnectCmd.Flags().Lookup("local-port")
	if localPortFlag == nil {
		t.Error("--local-port flag should still be defined for RDS connect command")
	}
}

func TestRDSConnectFlagUsage(t *testing.T) {
	// Test that the flag has proper usage description
	switchAccountFlag := rdsConnectCmd.Flags().Lookup("switch-account")
	if switchAccountFlag != nil {
		if switchAccountFlag.Usage == "" {
			t.Error("--switch-account flag should have usage description")
		}
		if !strings.Contains(switchAccountFlag.Usage, "Switch AWS account") {
			t.Errorf("Expected usage to mention 'Switch AWS account', got '%s'", switchAccountFlag.Usage)
		}
	}
}
