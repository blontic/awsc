package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
)

func TestWriteProfile(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Test credentials
	creds := &types.RoleCredentials{
		AccessKeyId:     aws.String("AKIATEST123456789"),
		SecretAccessKey: aws.String("test-secret-key"),
		SessionToken:    aws.String("test-session-token"),
	}

	// Write profile
	profileName, err := WriteProfile("test-account", "123456789012", "TestRole", creds)
	if err != nil {
		t.Fatalf("WriteProfile failed: %v", err)
	}

	// Verify profile name
	expectedProfileName := "awsc-test-account"
	if profileName != expectedProfileName {
		t.Errorf("Expected profile name %s, got %s", expectedProfileName, profileName)
	}

	// Verify config file was created
	configFile := filepath.Join(tempDir, ".aws", "config")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Verify file permissions
	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", info.Mode().Perm())
	}

	// Read and verify config content
	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(content)
	if !strings.Contains(configStr, "[profile awsc-test-account]") {
		t.Error("Config file does not contain profile header")
	}
	if !strings.Contains(configStr, "# Account: test-account (123456789012)") {
		t.Error("Config file does not contain account comment")
	}
	if !strings.Contains(configStr, "# Role: TestRole") {
		t.Error("Config file does not contain role comment")
	}
	if !strings.Contains(configStr, "AKIATEST123456789") {
		t.Error("Config file does not contain access key")
	}
	if !strings.Contains(configStr, "test-secret-key") {
		t.Error("Config file does not contain secret key")
	}
	if !strings.Contains(configStr, "test-session-token") {
		t.Error("Config file does not contain session token")
	}
}

func TestWriteProfile_UpdateExisting(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Write initial profile
	creds1 := &types.RoleCredentials{
		AccessKeyId:     aws.String("AKIAOLD123456789"),
		SecretAccessKey: aws.String("old-secret-key"),
		SessionToken:    aws.String("old-session-token"),
	}

	_, err := WriteProfile("test-account", "123456789012", "OldRole", creds1)
	if err != nil {
		t.Fatalf("WriteProfile failed: %v", err)
	}

	// Write updated profile with same account name
	creds2 := &types.RoleCredentials{
		AccessKeyId:     aws.String("AKIANEW123456789"),
		SecretAccessKey: aws.String("new-secret-key"),
		SessionToken:    aws.String("new-session-token"),
	}

	profileName, err := WriteProfile("test-account", "123456789012", "NewRole", creds2)
	if err != nil {
		t.Fatalf("WriteProfile update failed: %v", err)
	}

	// Verify profile name is the same
	if profileName != "awsc-test-account" {
		t.Errorf("Expected profile name awsc-test-account, got %s", profileName)
	}

	// Read config file
	configFile := filepath.Join(tempDir, ".aws", "config")
	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(content)

	// Verify old credentials are gone
	if strings.Contains(configStr, "AKIAOLD123456789") {
		t.Error("Old credentials still present in config file")
	}
	if strings.Contains(configStr, "old-secret-key") {
		t.Error("Old secret key still present in config file")
	}

	// Verify new credentials are present
	if !strings.Contains(configStr, "AKIANEW123456789") {
		t.Error("New credentials not found in config file")
	}
	if !strings.Contains(configStr, "new-secret-key") {
		t.Error("New secret key not found in config file")
	}
	if !strings.Contains(configStr, "# Role: NewRole") {
		t.Error("New role not found in config file")
	}

	// Verify profile only appears once
	count := strings.Count(configStr, "[profile awsc-test-account]")
	if count != 1 {
		t.Errorf("Expected profile to appear once, found %d times", count)
	}
}

func TestRemoveProfileSection(t *testing.T) {
	content := `[profile awsc-account1]
aws_access_key_id = KEY1
aws_secret_access_key = SECRET1

[profile awsc-account2]
aws_access_key_id = KEY2
aws_secret_access_key = SECRET2

[profile awsc-account3]
aws_access_key_id = KEY3
aws_secret_access_key = SECRET3
`

	// Remove middle profile
	result := removeProfileSection(content, "awsc-account2")

	// Verify account2 is removed
	if strings.Contains(result, "[profile awsc-account2]") {
		t.Error("Profile awsc-account2 was not removed")
	}
	if strings.Contains(result, "KEY2") {
		t.Error("KEY2 was not removed")
	}

	// Verify other profiles remain
	if !strings.Contains(result, "[profile awsc-account1]") {
		t.Error("Profile awsc-account1 was incorrectly removed")
	}
	if !strings.Contains(result, "[profile awsc-account3]") {
		t.Error("Profile awsc-account3 was incorrectly removed")
	}
	if !strings.Contains(result, "KEY1") {
		t.Error("KEY1 was incorrectly removed")
	}
	if !strings.Contains(result, "KEY3") {
		t.Error("KEY3 was incorrectly removed")
	}
}
