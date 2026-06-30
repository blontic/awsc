package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sso/types"
)

// WriteProfile writes AWS credentials to ~/.aws/config with the profile name awsc-{accountName}
func WriteProfile(accountName, accountID, roleName string, creds *types.RoleCredentials) (string, error) {
	// Guard against INI injection: account/role names are interpolated into the
	// profile header and comments, so reject values containing newlines or
	// section brackets that could forge additional config sections or keys.
	if err := validateProfileField("account name", accountName); err != nil {
		return "", err
	}
	if err := validateProfileField("account ID", accountID); err != nil {
		return "", err
	}
	if err := validateProfileField("role name", roleName); err != nil {
		return "", err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	awsDir := filepath.Join(homeDir, ".aws")
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create .aws directory: %w", err)
	}

	if creds == nil || creds.AccessKeyId == nil || creds.SecretAccessKey == nil || creds.SessionToken == nil {
		return "", fmt.Errorf("incomplete role credentials returned by AWS")
	}

	configPath := filepath.Join(awsDir, "config")
	profileName := fmt.Sprintf("awsc-%s", accountName)

	// Read existing config
	var existingContent string
	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}
	if err == nil {
		existingContent = string(data)
	}

	// Remove old profile section if exists
	existingContent = removeProfileSection(existingContent, profileName)

	// Ensure existing content ends with newline if not empty
	if existingContent != "" && !strings.HasSuffix(existingContent, "\n") {
		existingContent += "\n"
	}

	// Build new profile section
	profileSection := fmt.Sprintf(`[profile %s]
# Account: %s (%s)
# Role: %s
aws_access_key_id = %s
aws_secret_access_key = %s
aws_session_token = %s

`, profileName, accountName, accountID, roleName,
		*creds.AccessKeyId,
		*creds.SecretAccessKey,
		*creds.SessionToken)

	// Append new profile
	newContent := existingContent + profileSection

	// Write back to file
	if err := os.WriteFile(configPath, []byte(newContent), 0600); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return profileName, nil
}

// removeProfileSection removes a profile section from the config content
func removeProfileSection(content, profileName string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inTargetProfile := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is the start of our target profile
		if trimmed == fmt.Sprintf("[profile %s]", profileName) {
			inTargetProfile = true
			continue
		}

		// Check if this is the start of a different profile
		if strings.HasPrefix(trimmed, "[profile ") && trimmed != fmt.Sprintf("[profile %s]", profileName) {
			inTargetProfile = false
		}

		// Skip lines that are part of the target profile
		if inTargetProfile {
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// validateProfileField rejects values that would break the structure of the
// ~/.aws/config INI file (newlines, carriage returns, or section brackets).
func validateProfileField(label, value string) error {
	if strings.ContainsAny(value, "\r\n[]") {
		return fmt.Errorf("invalid %s %q: contains illegal characters", label, value)
	}
	return nil
}
