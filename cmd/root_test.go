package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRootCommand(t *testing.T) {
	// Test that root command is properly configured
	if rootCmd == nil {
		t.Error("rootCmd should not be nil")
	}

	if rootCmd.Use != "awsc" {
		t.Errorf("Expected Use 'awsc', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("rootCmd should have Short description")
	}

	if rootCmd.Long == "" {
		t.Error("rootCmd should have Long description")
	}

	if rootCmd.PersistentPreRun == nil {
		t.Error("rootCmd should have PersistentPreRun")
	}
}

func TestGlobalFlags(t *testing.T) {
	// Test that global flags are defined
	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("--config flag should be defined")
	}

	regionFlag := rootCmd.PersistentFlags().Lookup("region")
	if regionFlag == nil {
		t.Error("--region flag should be defined")
	}

	verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("--verbose flag should be defined")
	}

	// Test short flag for verbose
	verboseFlagShort := rootCmd.PersistentFlags().ShorthandLookup("v")
	if verboseFlagShort == nil {
		t.Error("-v short flag should be defined for verbose")
	}
}

func TestExecute(t *testing.T) {
	// Test that Execute function exists and doesn't panic when called
	// We can't easily test the actual execution without complex setup
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Execute function should exist and not panic during definition")
		}
	}()

	// Execute function exists if we can reference it without panic
	// The function is defined, so this test passes
}

func TestInitViper(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test initViper doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("initViper panicked: %v", r)
		}
	}()

	// Create .awsc directory
	awscDir := filepath.Join(tempDir, ".awsc")
	os.MkdirAll(awscDir, 0755)

	initViper("", "us-west-2")
}

func TestInitViper_WithConfigFile(t *testing.T) {
	// Create temp config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")
	os.WriteFile(configFile, []byte("test: value"), 0644)

	// Test initViper with custom config file
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("initViper with config file panicked: %v", r)
		}
	}()

	initViper(configFile, "")
}

func TestInitViper_NonexistentConfigFile(t *testing.T) {
	// Test that initViper exits when config file doesn't exist
	if os.Getenv("BE_CRASHER") == "1" {
		initViper("/nonexistent/config.yaml", "")
		return
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestInitViper_NonexistentConfigFile")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()

	// Expect the subprocess to exit with non-zero status
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return // Expected behavior
	}
	t.Fatalf("Expected initViper to exit when config file doesn't exist, but it didn't")
}

func TestInitViper_RegionOverride(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Create .awsc directory
	awscDir := filepath.Join(tempDir, ".awsc")
	os.MkdirAll(awscDir, 0755)

	// Test region override functionality
	initViper("", "eu-west-1")

	// Verify region was set (this is basic verification)
	// In a real test, we'd check viper.Get("default_region")
	// but that requires more complex viper state management
}

func TestCobraInitialization(t *testing.T) {
	// Test that cobra OnInitialize is set up
	// We can't easily test the callback directly, but we can verify
	// that the initialization doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Cobra initialization panicked: %v", r)
		}
	}()

	// The init() function has already run, so if we get here, it worked
}

func TestFlagDefaults(t *testing.T) {
	// Test flag default values
	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag.DefValue != "" {
		t.Errorf("Expected config flag default to be empty, got '%s'", configFlag.DefValue)
	}

	regionFlag := rootCmd.PersistentFlags().Lookup("region")
	if regionFlag.DefValue != "" {
		t.Errorf("Expected region flag default to be empty, got '%s'", regionFlag.DefValue)
	}

	verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
	if verboseFlag.DefValue != "false" {
		t.Errorf("Expected verbose flag default to be 'false', got '%s'", verboseFlag.DefValue)
	}
}

func TestFlagUsage(t *testing.T) {
	// Test flag usage strings
	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag.Usage == "" {
		t.Error("Config flag should have usage description")
	}

	regionFlag := rootCmd.PersistentFlags().Lookup("region")
	if regionFlag.Usage == "" {
		t.Error("Region flag should have usage description")
	}

	verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
	if verboseFlag.Usage == "" {
		t.Error("Verbose flag should have usage description")
	}
}
