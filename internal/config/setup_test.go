package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigPath(t *testing.T) {
	path := getConfigPath()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".swa", "config.yaml")
	
	if path != expected {
		t.Errorf("Expected config path %s, got %s", expected, path)
	}
}

func TestEnsureConfigExists_ConfigExists(t *testing.T) {
	// Create actual config file in home directory for test
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".swa")
	configPath := filepath.Join(configDir, "config.yaml")
	
	// Ensure config directory exists
	os.MkdirAll(configDir, 0755)
	
	// Create minimal config file
	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	file.WriteString("default_region: us-east-1\nsso:\n  region: us-east-1\n  start_url: https://test.awsapps.com/start\n")
	file.Close()
	
	// Clean up after test
	defer os.Remove(configPath)
	
	err = EnsureConfigExists()
	if err != nil {
		t.Errorf("Expected no error when config exists, got %v", err)
	}
}

func TestEnsureConfigExists_ConfigMissing(t *testing.T) {
	// This test would require mocking stdin, which is complex
	// In a real scenario, we'd refactor createConfig to be more testable
	t.Skip("Skipping interactive test - requires stdin mocking")
	
	// This should trigger interactive config creation, which we can't easily test
	// In a real scenario, we'd mock the input or make createConfig testable
	err := EnsureConfigExists()
	
	// We expect this to fail in test environment due to no stdin
	if err == nil {
		t.Error("Expected error when config missing and no stdin available")
	}
}

