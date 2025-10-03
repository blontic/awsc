package cmd

import (
	"fmt"
	"os"

	"github.com/blontic/awsc/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage awsc configuration",
	Long:  `Configure SSO settings for awsc`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize awsc configuration",
	Long:  `Create a new configuration file with SSO settings`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Skip the persistent pre-run that calls EnsureConfigExists
		// We handle config setup ourselves in this command
	},
	Run: runConfigInit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current awsc configuration",
	Long:  `Display the current configuration settings`,
	Run:   runConfigShow,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) {
	if err := config.InitializeConfigWithPrompt(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func runConfigShow(cmd *cobra.Command, args []string) {
	if err := config.ShowConfig(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
