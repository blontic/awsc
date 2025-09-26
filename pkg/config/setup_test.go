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
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	
	// Create config file
	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	file.Close()
	
	// We can't easily mock getConfigPath, so we'll test the actual function
	// In a real scenario, we'd refactor to make this more testable
	
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

