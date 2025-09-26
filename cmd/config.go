package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage swa configuration",
	Long:  `Configure SSO settings for swa`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize swa configuration",
	Long:  `Create a new configuration file with SSO settings`,
	Run:   runConfigInit,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) {
	var ssoStartURL, ssoRegion, defaultRegion string

	fmt.Print("SSO Start URL: ")
	fmt.Scanln(&ssoStartURL)

	fmt.Print("SSO Region (e.g., us-east-1): ")
	fmt.Scanln(&ssoRegion)

	fmt.Print("Default AWS Region (e.g., us-east-1): ")
	fmt.Scanln(&defaultRegion)

	// Create config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	configDir := filepath.Join(homeDir, ".swa")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	// Write config file
	configFile := filepath.Join(configDir, "config.yaml")
	viper.Set("sso.start_url", ssoStartURL)
	viper.Set("sso.region", ssoRegion)
	viper.Set("default_region", defaultRegion)

	if err := viper.WriteConfigAs(configFile); err != nil {
		fmt.Printf("Error writing config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration saved to %s\n", configFile)
}