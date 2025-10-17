package cmd

import (
	"testing"
)

func TestEC2Commands(t *testing.T) {
	// Test that EC2 commands are properly registered
	if ec2Cmd == nil {
		t.Error("ec2Cmd should not be nil")
	}

	if ec2ConnectCmd == nil {
		t.Error("ec2ConnectCmd should not be nil")
	}

	if ec2RdpCmd == nil {
		t.Error("ec2RdpCmd should not be nil")
	}
}

func TestEC2Command(t *testing.T) {
	// Test command properties
	if ec2Cmd.Use != "ec2" {
		t.Errorf("Expected Use 'ec2', got '%s'", ec2Cmd.Use)
	}

	if ec2Cmd.Short == "" {
		t.Error("ec2Cmd should have Short description")
	}

	if ec2Cmd.Long == "" {
		t.Error("ec2Cmd should have Long description")
	}
}

func TestEC2ConnectCommand(t *testing.T) {
	// Test command properties
	if ec2ConnectCmd.Use != "connect" {
		t.Errorf("Expected Use 'connect', got '%s'", ec2ConnectCmd.Use)
	}

	if ec2ConnectCmd.Short == "" {
		t.Error("ec2ConnectCmd should have Short description")
	}

	if ec2ConnectCmd.Long == "" {
		t.Error("ec2ConnectCmd should have Long description")
	}

	if ec2ConnectCmd.Run == nil {
		t.Error("ec2ConnectCmd should have Run function")
	}

	// Test instance-id flag
	instanceIdFlag := ec2ConnectCmd.Flags().Lookup("instance-id")
	if instanceIdFlag == nil {
		t.Error("--instance-id flag should be defined for connect command")
	}

	if instanceIdFlag.DefValue != "" {
		t.Errorf("Expected instance-id flag default to be empty, got '%s'", instanceIdFlag.DefValue)
	}
}

func TestRunEC2Connect(t *testing.T) {
	// Test that the function exists by checking command Run field
	if ec2ConnectCmd.Run == nil {
		t.Error("ec2ConnectCmd.Run should be defined")
	}

	// Business logic is tested in internal/aws package
	// This only tests CLI interface
}

func TestEC2RdpCommand(t *testing.T) {
	// Test RDP command properties
	if ec2RdpCmd.Use != "rdp" {
		t.Errorf("Expected Use 'rdp', got '%s'", ec2RdpCmd.Use)
	}

	if ec2RdpCmd.Short == "" {
		t.Error("ec2RdpCmd should have Short description")
	}

	if ec2RdpCmd.Long == "" {
		t.Error("ec2RdpCmd should have Long description")
	}

	if ec2RdpCmd.Run == nil {
		t.Error("ec2RdpCmd should have Run function")
	}

	// Test instance-id flag
	instanceIdFlag := ec2RdpCmd.Flags().Lookup("instance-id")
	if instanceIdFlag == nil {
		t.Error("--instance-id flag should be defined for rdp command")
	}

	if instanceIdFlag.DefValue != "" {
		t.Errorf("Expected instance-id flag default to be empty, got '%s'", instanceIdFlag.DefValue)
	}
}

func TestRunEC2RDP(t *testing.T) {
	// Test that the RDP function exists by checking command Run field
	if ec2RdpCmd.Run == nil {
		t.Error("ec2RdpCmd.Run should be defined")
	}

	// Business logic is tested in internal/aws package
	// This only tests CLI interface
}
func TestEC2ConnectSwitchAccountFlag(t *testing.T) {
	// Test that switch-account flag is properly defined for connect command
	switchAccountFlag := ec2ConnectCmd.Flags().Lookup("switch-account")
	if switchAccountFlag == nil {
		t.Error("--switch-account flag should be defined for EC2 connect command")
	}

	if switchAccountFlag.Shorthand != "s" {
		t.Errorf("Expected shorthand 's' for switch-account flag, got '%s'", switchAccountFlag.Shorthand)
	}

	if switchAccountFlag.DefValue != "false" {
		t.Errorf("Expected switch-account flag default to be false, got '%s'", switchAccountFlag.DefValue)
	}

	if switchAccountFlag.Usage == "" {
		t.Error("--switch-account flag should have usage description")
	}
}

func TestEC2RdpSwitchAccountFlag(t *testing.T) {
	// Test that switch-account flag is properly defined for RDP command
	switchAccountFlag := ec2RdpCmd.Flags().Lookup("switch-account")
	if switchAccountFlag == nil {
		t.Error("--switch-account flag should be defined for EC2 RDP command")
	}

	if switchAccountFlag.Shorthand != "s" {
		t.Errorf("Expected shorthand 's' for switch-account flag, got '%s'", switchAccountFlag.Shorthand)
	}

	if switchAccountFlag.DefValue != "false" {
		t.Errorf("Expected switch-account flag default to be false, got '%s'", switchAccountFlag.DefValue)
	}

	if switchAccountFlag.Usage == "" {
		t.Error("--switch-account flag should have usage description")
	}
}

func TestEC2FlagsStillPresent(t *testing.T) {
	// Ensure existing flags are still present after adding switch-account
	instanceIdFlag := ec2ConnectCmd.Flags().Lookup("instance-id")
	if instanceIdFlag == nil {
		t.Error("--instance-id flag should still be defined for EC2 connect command")
	}

	rdpInstanceIdFlag := ec2RdpCmd.Flags().Lookup("instance-id")
	if rdpInstanceIdFlag == nil {
		t.Error("--instance-id flag should still be defined for EC2 RDP command")
	}

	localPortFlag := ec2RdpCmd.Flags().Lookup("local-port")
	if localPortFlag == nil {
		t.Error("--local-port flag should still be defined for EC2 RDP command")
	}
}
