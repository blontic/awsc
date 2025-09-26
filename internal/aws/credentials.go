package aws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/spf13/viper"
)

func SetupCredentials(accountId, roleName string, creds *types.RoleCredentials) error {
	// Write to config file with swa profile
	return writeConfigFile(accountId, roleName, creds)
}

func writeConfigFile(accountId, roleName string, creds *types.RoleCredentials) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	awsDir := filepath.Join(homeDir, ".aws")
	if err := os.MkdirAll(awsDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(awsDir, "config")
	
	// Read existing config file
	var existingContent string
	if data, err := os.ReadFile(configFile); err == nil {
		existingContent = string(data)
	}
	
	// Remove any existing swa profile sections
	lines := strings.Split(existingContent, "\n")
	var filteredLines []string
	inSwaProfile := false
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "[profile swa]" {
			inSwaProfile = true
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "[") && strings.HasSuffix(strings.TrimSpace(line), "]") {
			inSwaProfile = false
		}
		if !inSwaProfile {
			filteredLines = append(filteredLines, line)
		}
	}
	
	// Add new swa profile
	newContent := strings.Join(filteredLines, "\n")
	if newContent != "" && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += fmt.Sprintf(`[profile swa]
aws_access_key_id = %s
aws_secret_access_key = %s
aws_session_token = %s
`, *creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)

	// Write the updated config file
	return os.WriteFile(configFile, []byte(newContent), 0644)
}

// LoadSWAConfig loads AWS config with swa profile and region override
func LoadSWAConfig(ctx context.Context) (aws.Config, error) {
	// Use region override if provided, otherwise use default region from config
	region := viper.GetString("default_region")
	
	// Try to load with swa profile first
	options := []func(*config.LoadOptions) error{
		config.WithSharedConfigProfile("swa"),
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

// IsAuthError checks if an error is related to authentication/credentials
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	// Don't treat permission errors as auth errors
	if contains(errorStr, "is not authorized to perform") {
		return false
	}
	return contains(errorStr, "AuthFailure") ||
		contains(errorStr, "SignatureDoesNotMatch") ||
		contains(errorStr, "TokenRefreshRequired") ||
		contains(errorStr, "ExpiredToken") ||
		contains(errorStr, "InvalidToken") ||
		contains(errorStr, "get credentials") ||
		contains(errorStr, "no EC2 IMDS role found") ||
		contains(errorStr, "failed to refresh cached credentials")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CheckAWSSession verifies if there's a valid AWS session
func CheckAWSSession(ctx context.Context) error {
	cfg, err := LoadSWAConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)
	_, err = stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("invalid AWS session: %w", err)
	}

	return nil
}

