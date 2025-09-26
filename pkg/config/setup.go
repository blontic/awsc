package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func EnsureConfigExists() error {
	// Check if config file exists
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		return nil // Config exists
	}

	fmt.Printf("Configuration file not found. Let's set up SWA.\n\n")
	return createConfig()
}

func createConfig() error {
	reader := bufio.NewReader(os.Stdin)

	// Get SSO Start URL
	fmt.Printf("Enter your SSO Start URL (e.g., https://your-org.awsapps.com/start): ")
	ssoStartURL, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	ssoStartURL = strings.TrimSpace(ssoStartURL)

	// Get SSO Region
	fmt.Printf("Enter your SSO Region (e.g., us-east-1): ")
	ssoRegion, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	ssoRegion = strings.TrimSpace(ssoRegion)

	// Get Default Region
	fmt.Printf("Enter your default AWS Region (e.g., us-east-1): ")
	defaultRegion, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	defaultRegion = strings.TrimSpace(defaultRegion)

	// Create config directory
	configDir := filepath.Dir(getConfigPath())
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Set viper values
	viper.Set("sso.start_url", ssoStartURL)
	viper.Set("sso.region", ssoRegion)
	viper.Set("default_region", defaultRegion)

	// Write config file
	if err := viper.WriteConfigAs(getConfigPath()); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	fmt.Printf("\nConfiguration saved to %s\n", getConfigPath())

	return nil
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".swa", "config.yaml")
}