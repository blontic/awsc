package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestViperConfigLoading(t *testing.T) {
	// Save original viper state
	originalViper := viper.GetViper()
	defer func() {
		viper.Reset()
		// Restore original viper instance
		for key, value := range originalViper.AllSettings() {
			viper.Set(key, value)
		}
	}()

	// Create temp directory and config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `sso:
  start_url: https://test.awsapps.com/start
  region: us-east-1
default_region: us-west-2`

	os.WriteFile(configFile, []byte(configContent), 0644)

	// Test viper loads config correctly
	viper.Reset()
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Verify config values
	if viper.GetString("sso.start_url") != "https://test.awsapps.com/start" {
		t.Errorf("Expected SSO start URL to be loaded from config")
	}

	if viper.GetString("default_region") != "us-west-2" {
		t.Errorf("Expected default region to be loaded from config")
	}
}

func TestViperRegionOverride(t *testing.T) {
	// Save and reset viper state
	defer viper.Reset()

	// Set initial config
	viper.Set("default_region", "us-east-1")

	// Test region override
	viper.Set("default_region", "eu-west-1")

	if viper.GetString("default_region") != "eu-west-1" {
		t.Errorf("Expected region override to work, got %s", viper.GetString("default_region"))
	}
}

func TestViperEnvironmentVariables(t *testing.T) {
	// Save and reset viper state
	defer viper.Reset()

	// Test that viper can read environment variables
	viper.AutomaticEnv()

	// Set a test environment variable
	os.Setenv("SWA_TEST_VAR", "test-value")
	defer os.Unsetenv("SWA_TEST_VAR")

	// Viper should be able to read it (though we don't use env vars in our app)
	// This tests that AutomaticEnv() is working
	viper.BindEnv("test_var", "SWA_TEST_VAR")

	if viper.GetString("test_var") != "test-value" {
		t.Error("Viper should be able to read environment variables")
	}
}

func TestViperConfigPaths(t *testing.T) {
	// Save and reset viper state
	defer viper.Reset()

	// Create temp directory structure
	tempDir := t.TempDir()
	swaDir := filepath.Join(tempDir, ".swa")
	os.MkdirAll(swaDir, 0755)

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test viper config path setup
	viper.AddConfigPath(swaDir)
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	// Create config file
	configFile := filepath.Join(swaDir, "config.yaml")
	os.WriteFile(configFile, []byte("test: value"), 0644)

	// Test that viper can find and read the config
	err := viper.ReadInConfig()
	if err != nil {
		t.Errorf("Viper should be able to find config in .swa directory: %v", err)
	}

	if viper.GetString("test") != "value" {
		t.Error("Viper should load config from .swa directory")
	}
}
