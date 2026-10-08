package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath should return non-empty path")
	}

	if !filepath.IsAbs(path) {
		t.Error("GetConfigPath should return absolute path")
	}

	if !strings.HasSuffix(path, ".awsc/config.yaml") {
		t.Errorf("Expected path to end with .awsc/config.yaml, got %s", path)
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
	err := ShowConfig(io.Discard, "")
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

	// Create .awsc directory and config file
	awscDir := filepath.Join(tempDir, ".awsc")
	os.MkdirAll(awscDir, 0755)

	configFile := filepath.Join(awscDir, "config.yaml")
	configContent := `default_org: test
orgs:
  test:
    sso:
      start_url: https://test.awsapps.com/start
      region: us-east-1
    default_region: us-east-1`

	os.WriteFile(configFile, []byte(configContent), 0644)
	active.Org = "test"

	// ShowConfig should work with existing file
	err := ShowConfig(io.Discard, "")
	if err != nil {
		t.Errorf("ShowConfig should not return error with valid file, got: %v", err)
	}

	// Clean up
	active = Settings{}
}

func TestEnsureConfigExists_OrgConfigured(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Create .awsc directory and config file
	awscDir := filepath.Join(tempDir, ".awsc")
	os.MkdirAll(awscDir, 0755)

	configFile := filepath.Join(awscDir, "config.yaml")
	os.WriteFile(configFile, []byte("orgs:\n  test:\n    sso:\n      start_url: https://test.awsapps.com/start\n      region: us-east-1\n"), 0600)

	// EnsureConfigExists should return nil when an org is configured
	err := EnsureConfigExists()
	if err != nil {
		t.Errorf("EnsureConfigExists should return nil when file exists, got: %v", err)
	}
}

func TestValidateRegion(t *testing.T) {
	tests := []struct {
		region string
		valid  bool
	}{
		{"us-east-1", true},
		{"us-west-2", true},
		{"eu-west-1", true},
		{"ap-southeast-1", true},
		{"invalid-region", false},
		{"us-east-99", false},
		{"", false},
		{"US-EAST-1", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result := ValidateRegion(tt.region)
			if result != tt.valid {
				t.Errorf("ValidateRegion(%q) = %v, want %v", tt.region, result, tt.valid)
			}
		})
	}
}

func TestValidateSSOURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"https://myorg.awsapps.com/start", true},
		{"https://my-org.awsapps.com/start", true},
		{"https://test123.awsapps.com/start/", true}, // trailing slash
		{"https://myorg.awsapps.com/start/extra", false},
		{"http://myorg.awsapps.com/start", false},    // http not https
		{"https://myorg.amazonaws.com/start", false}, // wrong domain
		{"https://my_org.awsapps.com/start", false},  // underscore not allowed
		{"myorg.awsapps.com/start", false},           // missing https
		{"", false},
		{"https://.awsapps.com/start", false}, // empty subdomain
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := validateSSOURL(tt.url)
			if result != tt.valid {
				t.Errorf("validateSSOURL(%q) = %v, want %v", tt.url, result, tt.valid)
			}
		})
	}
}
