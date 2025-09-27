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
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		return nil // Config exists
	}

	fmt.Printf("Configuration file not found. Let's set up SWA.\n\n")
	return InitializeConfig()
}

func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".swa", "config.yaml")
}

func InitializeConfig() error {
	reader := bufio.NewReader(os.Stdin)

	// Get SSO Start URL
	fmt.Print("SSO Start URL: ")
	ssoStartURL, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	ssoStartURL = strings.TrimSpace(ssoStartURL)

	// Get SSO Region
	fmt.Print("SSO Region (e.g., us-east-1): ")
	ssoRegion, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	ssoRegion = strings.TrimSpace(ssoRegion)

	// Get Default Region
	fmt.Print("Default AWS Region (e.g., us-east-1): ")
	defaultRegion, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	defaultRegion = strings.TrimSpace(defaultRegion)

	// Create config directory
	configDir := filepath.Dir(GetConfigPath())
	if err := os.MkdirAll(configDir, 0755); err != nil {
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

// ShowConfig displays the current configuration
func ShowConfig() error {
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("No configuration file found. Run 'swa config init' to create one.\n")
		return nil
	}

	fmt.Printf("Configuration file: %s\n\n", configPath)
	fmt.Printf("SSO Start URL: %s\n", viper.GetString("sso.start_url"))
	fmt.Printf("SSO Region: %s\n", viper.GetString("sso.region"))
	fmt.Printf("Default Region: %s\n", viper.GetString("default_region"))
	return nil
}
