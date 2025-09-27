package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath should return non-empty path")
	}

	if !filepath.IsAbs(path) {
		t.Error("GetConfigPath should return absolute path")
	}

	if !strings.HasSuffix(path, ".swa/config.yaml") {
		t.Errorf("Expected path to end with .swa/config.yaml, got %s", path)
	}
}

func TestShowConfig_NoFile(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// ShowConfig should handle missing file gracefully
	err := ShowConfig()
	if err != nil {
		t.Errorf("ShowConfig should not return error for missing file, got: %v", err)
	}
}

func TestShowConfig_WithFile(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Create .swa directory and config file
	swaDir := filepath.Join(tempDir, ".swa")
	os.MkdirAll(swaDir, 0755)

	configFile := filepath.Join(swaDir, "config.yaml")
	configContent := `sso:
  start_url: https://test.awsapps.com/start
  region: us-east-1
default_region: us-east-1`

	os.WriteFile(configFile, []byte(configContent), 0644)

	// Set viper values to match file
	viper.Set("sso.start_url", "https://test.awsapps.com/start")
	viper.Set("sso.region", "us-east-1")
	viper.Set("default_region", "us-east-1")

	// ShowConfig should work with existing file
	err := ShowConfig()
	if err != nil {
		t.Errorf("ShowConfig should not return error with valid file, got: %v", err)
	}

	// Clean up
	viper.Reset()
}

func TestInitializeConfigWithPrompt_NoFile(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test that InitializeConfigWithPrompt doesn't panic when no file exists
	// Skip actual execution as it requires user input
	t.Skip("Skipping InitializeConfigWithPrompt test - requires user input")
}

func TestInitializeConfigWithPrompt_ExistingFile(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Create existing config file
	swaDir := filepath.Join(tempDir, ".swa")
	os.MkdirAll(swaDir, 0755)
	configFile := filepath.Join(swaDir, "config.yaml")
	os.WriteFile(configFile, []byte("existing: config"), 0644)

	// Verify file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file should exist for this test")
	}

	// Skip actual execution as it requires user input
	t.Skip("Skipping InitializeConfigWithPrompt test - requires user input")
}

func TestEnsureConfigExists_FileExists(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Create .swa directory and config file
	swaDir := filepath.Join(tempDir, ".swa")
	os.MkdirAll(swaDir, 0755)

	configFile := filepath.Join(swaDir, "config.yaml")
	os.WriteFile(configFile, []byte("test: value"), 0644)

	// EnsureConfigExists should return nil when file exists
	err := EnsureConfigExists()
	if err != nil {
		t.Errorf("EnsureConfigExists should return nil when file exists, got: %v", err)
	}
}
