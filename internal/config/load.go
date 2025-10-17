package config

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/viper"
)

// LoadAWSConfig loads AWS config with region settings (no profile)
// This is used for SSO operations which don't need credentials
func LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	// Use region override if provided, otherwise use SSO region from config
	region := viper.GetString("default_region")
	if region == "" {
		region = viper.GetString("sso.region")
	}

	// Explicitly use empty profile to ignore AWS_PROFILE environment variable
	options := []func(*config.LoadOptions) error{
		config.WithSharedConfigProfile(""),
	}

	if region != "" {
		options = append(options, config.WithRegion(region))
	}

	return config.LoadDefaultConfig(ctx, options...)
}

// LoadAWSConfigWithProfile loads AWS config using hybrid approach:
// 1. AWSC_PROFILE environment variable (explicit override)
// 2. PPID session tracking (automatic per-terminal)
// 3. Error if neither exists
func LoadAWSConfigWithProfile(ctx context.Context) (aws.Config, error) {
	// Use region override if provided, otherwise use default region from config
	region := viper.GetString("default_region")

	var profileName string

	// Priority 1: Check AWSC_PROFILE environment variable
	envProfile := os.Getenv("AWSC_PROFILE")
	if envProfile != "" {
		profileName = envProfile
	} else {
		// Priority 2: Check PPID session
		session, err := GetCurrentSession()
		if err != nil {
			// No session found
			return aws.Config{}, fmt.Errorf("no active session")
		}
		profileName = session.ProfileName
	}

	// Load config with the determined profile
	options := []func(*config.LoadOptions) error{
		config.WithSharedConfigProfile(profileName),
	}

	if region != "" {
		options = append(options, config.WithRegion(region))
	}

	return config.LoadDefaultConfig(ctx, options...)
}
