package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/blontic/swa/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var regionOverride string

var rootCmd = &cobra.Command{
	Use:   "swa",
	Short: "AWS CLI tool for SSO, RDS, and Secrets Manager",
	Long:  `SWA (AWS backwards) - A CLI tool for AWS SSO authentication, RDS port forwarding, and Secrets Manager operations.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := config.EnsureConfigExists(); err != nil {
			fmt.Printf("Error setting up configuration: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.swa.yaml)")
	rootCmd.PersistentFlags().StringVar(&regionOverride, "region", "", "AWS region to use (overrides config)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Look for config in ~/.swa/config.yaml
		viper.AddConfigPath(filepath.Join(home, ".swa"))
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	viper.ReadInConfig() // Ignore errors, config is optional

	// Set region override if provided
	if regionOverride != "" {
		viper.Set("default_region", regionOverride)
	}
}