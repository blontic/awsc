package cmd

import (
	"fmt"
	"os"

	"github.com/blontic/swa/internal/config"
	"github.com/blontic/swa/internal/debug"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var regionOverride string
var verbose bool

var rootCmd = &cobra.Command{
	Use:   "swa",
	Short: "AWS CLI tool for SSO, RDS, and Secrets Manager",
	Long:  `SWA (AWS backwards) - A CLI tool for AWS SSO authentication, RDS port forwarding, and Secrets Manager operations.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		debug.SetVerbose(verbose)
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
	cobra.OnInitialize(func() {
		initViper(cfgFile, regionOverride)
	})
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.swa/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&regionOverride, "region", "", "AWS region to use (overrides config)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

// initViper initializes viper configuration
func initViper(cfgFile, regionOverride string) {
	if cfgFile != "" {
		// Check if custom config file exists
		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			fmt.Printf("Error: config file '%s' does not exist\n", cfgFile)
			os.Exit(1)
		}
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Look for config in ~/.swa/config.yaml
		viper.AddConfigPath(home + "/.swa")
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
