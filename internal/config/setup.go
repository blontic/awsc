package config

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

// AWS regions list
var awsRegions = map[string]bool{
	"us-east-1":      true,
	"us-east-2":      true,
	"us-west-1":      true,
	"us-west-2":      true,
	"af-south-1":     true,
	"ap-east-1":      true,
	"ap-south-1":     true,
	"ap-south-2":     true,
	"ap-southeast-1": true,
	"ap-southeast-2": true,
	"ap-southeast-3": true,
	"ap-southeast-4": true,
	"ap-northeast-1": true,
	"ap-northeast-2": true,
	"ap-northeast-3": true,
	"ca-central-1":   true,
	"ca-west-1":      true,
	"eu-central-1":   true,
	"eu-central-2":   true,
	"eu-west-1":      true,
	"eu-west-2":      true,
	"eu-west-3":      true,
	"eu-north-1":     true,
	"eu-south-1":     true,
	"eu-south-2":     true,
	"il-central-1":   true,
	"me-central-1":   true,
	"me-south-1":     true,
	"sa-east-1":      true,
}

// SSO URL regex pattern
var ssoURLPattern = regexp.MustCompile(`^https://[a-zA-Z0-9-]+\.awsapps\.com/start/?$`)

// validateRegion checks if the region is a valid AWS region
func validateRegion(region string) bool {
	return awsRegions[region]
}

// validateSSOURL checks if the SSO URL matches the expected pattern
func validateSSOURL(url string) bool {
	return ssoURLPattern.MatchString(url)
}

func EnsureConfigExists() error {
	// Check if config file exists
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		return nil // Config exists
	}

	fmt.Printf("Configuration file not found. Let's set up AWSC.\n\n")
	return InitializeConfig()
}

func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".awsc", "config.yaml")
}

func InitializeConfig() error {
	reader := bufio.NewReader(os.Stdin)

	// Get SSO Start URL
	var ssoStartURL string
	for {
		fmt.Print("SSO Start URL: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		ssoStartURL = strings.TrimSpace(input)
		if validateSSOURL(ssoStartURL) {
			break
		}
		fmt.Printf("Invalid SSO URL format. Expected: https://your-org.awsapps.com/start\n")
	}

	// Get SSO Region
	var ssoRegion string
	for {
		fmt.Print("SSO Region (e.g., us-east-1): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		ssoRegion = strings.TrimSpace(input)
		if validateRegion(ssoRegion) {
			break
		}
		fmt.Printf("Invalid AWS region. Please enter a valid region like us-east-1, us-west-2, etc.\n")
	}

	// Get Default Region
	var defaultRegion string
	for {
		fmt.Print("Default AWS Region (e.g., us-east-1): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		defaultRegion = strings.TrimSpace(input)
		if validateRegion(defaultRegion) {
			break
		}
		fmt.Printf("Invalid AWS region. Please enter a valid region like us-east-1, us-west-2, etc.\n")
	}

	// Create config directory with secure permissions
	configDir := filepath.Dir(GetConfigPath())
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Set viper values
	viper.Set("sso.start_url", ssoStartURL)
	viper.Set("sso.region", ssoRegion)
	viper.Set("default_region", defaultRegion)

	// Write config file
	if err := viper.WriteConfigAs(GetConfigPath()); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	// Set secure permissions on config file
	if err := os.Chmod(GetConfigPath(), 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %v", err)
	}

	fmt.Printf("Configuration saved to %s\n", GetConfigPath())
	return nil
}

// InitializeConfigWithPrompt checks for existing config and prompts user before overwriting
func InitializeConfigWithPrompt() error {
	// Check if config already exists
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuration file already exists at %s\n", configPath)
		fmt.Print("Do you want to overwrite it? (y/N): ")

		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Println("Configuration initialization cancelled.")
			return nil
		}
	}

	return InitializeConfig()
}

// EnsureConfigAndAuth checks for config and prompts for setup/login if needed
func EnsureConfigAndAuth(ctx context.Context) error {
	// Check if config exists
	if viper.GetString("sso.start_url") == "" {
		fmt.Printf("No SSO configuration found.\n")
		fmt.Print("Would you like to set up configuration now? (y/N): ")

		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Printf("Configuration required. Run 'awsc config init' to set up.\n")
			return fmt.Errorf("configuration required")
		}

		if err := InitializeConfig(); err != nil {
			return fmt.Errorf("failed to initialize config: %v", err)
		}

		fmt.Printf("\nConfiguration complete. Now let's authenticate...\n")
	}

	// Check if we have valid credentials by trying to load AWS config
	_, err := LoadAWSConfigWithProfile(ctx)
	if err != nil {
		// If we can't load config, we need to authenticate
		fmt.Printf("Authentication required.\n")
		fmt.Print("Would you like to login now? (y/N): ")

		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Printf("Authentication required. Run 'awsc login' to authenticate.\n")
			return fmt.Errorf("authentication required")
		}

		// Import the aws package to run login
		// We'll need to handle this differently since we can't import aws here
		return fmt.Errorf("please run 'awsc login' to authenticate")
	}

	return nil
}

// ShowConfig displays the current configuration
func ShowConfig() error {
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("No configuration file found. Run 'awsc config init' to create one.\n")
		return nil
	}

	fmt.Printf("Configuration file: %s\n\n", configPath)
	fmt.Printf("SSO Start URL: %s\n", viper.GetString("sso.start_url"))
	fmt.Printf("SSO Region: %s\n", viper.GetString("sso.region"))
	fmt.Printf("Default Region: %s\n", viper.GetString("default_region"))
	return nil
}
