package cmd

import (
	"github.com/blontic/awsc/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

func TestApplyRegionOverride(t *testing.T) {
	defer config.SetActive(config.Settings{})
	org := config.Settings{Org: "alpha", StartURL: "https://alpha.awsapps.com/start", SSORegion: "us-east-1", DefaultRegion: "us-east-1"}
	config.SetActive(org)
	regionOverride = "eu-west-1"
	defer func() { regionOverride = "" }()

	applyRegionOverride()

	want := org
	want.DefaultRegion = "eu-west-1"
	if got := config.Active(); got != want {
		t.Errorf("--region should only change the default region: got %+v, want %+v", got, want)
	}
}

// Guards against reintroducing viper: settings live in config.Settings.
func TestNoViperDependency(t *testing.T) {
	data, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "spf13/viper") {
		t.Error("go.mod must not depend on spf13/viper; use config.Settings")
	}
}

func TestSetupOrg_RemovedConfigFlag(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		removedConfigFlag = "/some/config.yaml"
		setupOrg()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestSetupOrg_RemovedConfigFlag")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	out, err := cmd.CombinedOutput()
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %v", err)
	}
	if !strings.Contains(string(out), "--config has been removed") {
		t.Errorf("expected removal message, got:\n%s", out)
	}
}

func TestNeedsOrg(t *testing.T) {
	rootCmd.InitDefaultCompletionCmd()
	completion, _, err := rootCmd.Find([]string{"completion", "zsh"})
	if err != nil {
		t.Fatal(err)
	}
	cases := map[*cobra.Command]bool{
		versionCmd: false,
		completion: false,
		loginCmd:   true,
		configCmd:  true,
	}
	for cmd, want := range cases {
		if got := needsOrg(cmd); got != want {
			t.Errorf("needsOrg(%s) = %v, want %v", cmd.CommandPath(), got, want)
		}
	}
}

func TestVersionDoesNotTouchConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	defer func() { config.SetActive(config.Settings{}) }()

	rootCmd.SetArgs([]string{"version"})
	defer rootCmd.SetArgs(nil)
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{".awsc", ".aws"} {
		if _, err := os.Stat(filepath.Join(home, dir)); !os.IsNotExist(err) {
			t.Errorf("'awsc version' must not create ~/%s", dir)
		}
	}
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
	orgFlag := rootCmd.PersistentFlags().Lookup("org")
	if orgFlag.Usage == "" {
		t.Error("Org flag should have usage description")
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

func TestValidateLocalPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{name: "zero means default", port: 0, wantErr: false},
		{name: "valid low", port: 1, wantErr: false},
		{name: "valid typical", port: 5432, wantErr: false},
		{name: "valid high", port: 65535, wantErr: false},
		{name: "negative", port: -1, wantErr: true},
		{name: "too high", port: 65536, wantErr: true},
		{name: "way too high (int32 wrap range)", port: 70000, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLocalPort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLocalPort(%d) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}
