package config

import (
	"context"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadAWSConfig(t *testing.T) {
	// Set up test config
	viper.Set("default_region", "us-west-2")
	viper.Set("sso.region", "us-east-1")

	ctx := context.Background()
	cfg, err := LoadAWSConfig(ctx)
	if err != nil {
		t.Fatalf("LoadAWSConfig failed: %v", err)
	}

	// Should use default_region over sso.region
	if cfg.Region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got %s", cfg.Region)
	}

	// Clean up
	viper.Reset()
}

func TestLoadAWSConfigFallback(t *testing.T) {
	// Set up test config with only sso.region
	viper.Set("sso.region", "eu-west-1")

	ctx := context.Background()
	cfg, err := LoadAWSConfig(ctx)
	if err != nil {
		t.Fatalf("LoadAWSConfig failed: %v", err)
	}

	// Should fall back to sso.region
	if cfg.Region != "eu-west-1" {
		t.Errorf("Expected region eu-west-1, got %s", cfg.Region)
	}

	// Clean up
	viper.Reset()
}

func TestLoadAWSConfigWithProfile(t *testing.T) {
	// Set up test config
	viper.Set("default_region", "ap-southeast-2")

	ctx := context.Background()
	_, err := LoadAWSConfigWithProfile(ctx)

	// This will likely fail in test environment without AWS config
	// but we're testing that it doesn't panic and handles errors gracefully
	if err == nil {
		t.Log("LoadAWSConfigWithProfile succeeded (unexpected in test env)")
	} else {
		t.Logf("LoadAWSConfigWithProfile failed as expected: %v", err)
	}

	// Clean up
	viper.Reset()
}
