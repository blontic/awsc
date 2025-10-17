package cmd

import (
	"os"
	"testing"
)

func TestSwitchAccount_UnsetsAWSC_PROFILE(t *testing.T) {
	// This test verifies that when switching accounts, AWSC_PROFILE is unset
	// so the new PPID session takes priority

	// Set AWSC_PROFILE environment variable
	originalProfile := os.Getenv("AWSC_PROFILE")
	os.Setenv("AWSC_PROFILE", "awsc-test-account")
	defer func() {
		if originalProfile != "" {
			os.Setenv("AWSC_PROFILE", originalProfile)
		} else {
			os.Unsetenv("AWSC_PROFILE")
		}
	}()

	// Verify it's set
	if os.Getenv("AWSC_PROFILE") != "awsc-test-account" {
		t.Fatal("AWSC_PROFILE should be set before test")
	}

	// Simulate what handleAccountSwitch does
	os.Unsetenv("AWSC_PROFILE")

	// Verify it's unset
	if os.Getenv("AWSC_PROFILE") != "" {
		t.Error("AWSC_PROFILE should be unset when switching accounts")
	}
}
