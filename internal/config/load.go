package config

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/viper"
)

// LoadAWSConfig loads AWS config with region settings (no profile)
func LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	// Use region override if provided, otherwise use SSO region from config
	region := viper.GetString("default_region")
	if region == "" {
		region = viper.GetString("sso.region")
	}

	if region != "" {
		return config.LoadDefaultConfig(ctx, config.WithRegion(region))
	}
	return config.LoadDefaultConfig(ctx)
}

// LoadAWSConfigWithProfile loads AWS config with awsc profile and region override
func LoadAWSConfigWithProfile(ctx context.Context) (aws.Config, error) {
	// Use region override if provided, otherwise use default region from config
	region := viper.GetString("default_region")

	// Try to load with awsc profile first
	options := []func(*config.LoadOptions) error{
		config.WithSharedConfigProfile("awsc"),
	}

	if region != "" {
		options = append(options, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, options...)
	if err != nil {
		// Only fall back if profile doesn't exist, not for credential errors
		if strings.Contains(err.Error(), "failed to get shared config profile") {
			if region != "" {
				return config.LoadDefaultConfig(ctx, config.WithRegion(region))
			}
			return config.LoadDefaultConfig(ctx)
		}
		// For other errors (like credential issues), return the error
		return cfg, err
	}

	return cfg, nil
}
