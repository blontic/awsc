package config

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// LoadAWSConfig loads AWS config for IAM Identity Center (SSO/OIDC) calls,
// which need no credentials and must use the org's SSO region.
func LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	region := active.SSORegion

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
	region := active.DefaultRegion

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
		// The terminal is logged in to a different org than the one requested
		// (e.g. via --org): treat as not logged in so login runs for that org.
		if session.Org != active.Org {
			return aws.Config{}, fmt.Errorf("no active session")
		}
		if profileName, err = sessionProfile(session); err != nil {
			return aws.Config{}, err
		}
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

// sessionProfile returns the profile for the terminal's session. A profile
// missing from ~/.aws/config (e.g. the file was deleted) is recreated, so
// commands keep working without a new login while the SSO token is cached. If
// the profile now points at a different account or role (another terminal
// logged in to the same account with another role), the session is treated as
// inactive so the user logs in again rather than silently switching role.
func sessionProfile(session *SessionInfo) (string, error) {
	accountID, roleName, found, err := lookupProfile(session.ProfileName)
	if err != nil {
		return "", err
	}
	if found {
		if accountID != session.AccountID || roleName != session.RoleName {
			return "", fmt.Errorf("no active session")
		}
		return session.ProfileName, nil
	}

	cfg, err := ReadFileConfig()
	if err != nil {
		return "", err
	}
	name, err := WriteProfile(cfg.Orgs, Profile{
		Org:         session.Org,
		AccountName: session.AccountName,
		AccountID:   session.AccountID,
		RoleName:    session.RoleName,
	})
	if err != nil {
		return "", err
	}
	if name != session.ProfileName {
		err = SaveSession(os.Getppid(), name, session.AccountID, session.AccountName, session.RoleName, session.Org)
	}
	return name, err
}
