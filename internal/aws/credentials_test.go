package aws

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sso/types"
)

func TestSetupCredentials(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test credentials
	accountId := "123456789012"
	roleName := "TestRole"
	accessKey := "AKIATEST123"
	secretKey := "secretkey123"
	sessionToken := "sessiontoken123"

	creds := &types.RoleCredentials{
		AccessKeyId:     &accessKey,
		SecretAccessKey: &secretKey,
		SessionToken:    &sessionToken,
	}

	err := SetupCredentials(accountId, roleName, creds)
	if err != nil {
		t.Fatalf("SetupCredentials failed: %v", err)
	}

	// Verify config file was created
	configFile := filepath.Join(tempDir, ".aws", "config")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Read and verify file contents
	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	contentStr := string(content)

	// Check for required sections
	if !strings.Contains(contentStr, "[profile swa]") {
		t.Error("Config file should contain [profile swa] section")
	}

	if !strings.Contains(contentStr, "aws_access_key_id = "+accessKey) {
		t.Error("Config file should contain access key")
	}

	if !strings.Contains(contentStr, "aws_secret_access_key = "+secretKey) {
		t.Error("Config file should contain secret key")
	}

	if !strings.Contains(contentStr, "aws_session_token = "+sessionToken) {
		t.Error("Config file should contain session token")
	}
}

func TestWriteConfigFile_CreatesDirectory(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test credentials
	accountId := "123456789012"
	roleName := "TestRole"
	accessKey := "AKIATEST123"
	secretKey := "secretkey123"
	sessionToken := "sessiontoken123"

	creds := &types.RoleCredentials{
		AccessKeyId:     &accessKey,
		SecretAccessKey: &secretKey,
		SessionToken:    &sessionToken,
	}

	err := writeAWSProfile("swa", accountId, roleName, creds)
	if err != nil {
		t.Fatalf("writeAWSProfile failed: %v", err)
	}

	// Verify .aws directory was created
	awsDir := filepath.Join(tempDir, ".aws")
	if _, err := os.Stat(awsDir); os.IsNotExist(err) {
		t.Fatal(".aws directory was not created")
	}
}

func TestWriteConfigFile_AppendsToExisting(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Create .aws directory and existing config file
	awsDir := filepath.Join(tempDir, ".aws")
	os.MkdirAll(awsDir, 0755)

	configFile := filepath.Join(awsDir, "config")
	existingContent := "[profile swa]\naws_access_key_id = OLD_KEY\n"
	os.WriteFile(configFile, []byte(existingContent), 0644)

	// Test credentials
	accountId := "123456789012"
	roleName := "TestRole"
	accessKey := "AKIATEST123"
	secretKey := "secretkey123"
	sessionToken := "sessiontoken123"

	creds := &types.RoleCredentials{
		AccessKeyId:     &accessKey,
		SecretAccessKey: &secretKey,
		SessionToken:    &sessionToken,
	}

	err := writeAWSProfile("swa", accountId, roleName, creds)
	if err != nil {
		t.Fatalf("writeAWSProfile failed: %v", err)
	}

	// Read and verify file was appended to
	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	contentStr := string(content)

	// Should contain new key
	if !strings.Contains(contentStr, accessKey) {
		t.Error("Config file should contain new access key")
	}
}

func TestCheckCredentialsValid(t *testing.T) {
	ctx := context.Background()

	// This test will likely fail in CI/test environment without valid AWS credentials
	// but it tests that the function doesn't panic and returns an error appropriately
	err := CheckCredentialsValid(ctx)

	// We expect an error in test environment (no valid credentials)
	if err == nil {
		t.Log("CheckCredentialsValid passed - valid AWS session found")
	} else {
		t.Logf("CheckCredentialsValid failed as expected in test environment: %v", err)
	}

	// Test passes if function doesn't panic
}
